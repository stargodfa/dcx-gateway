package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"dcx-gateway/internal/model/protocol/ue"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

func TestFastAPIClientPushSuccessStatsAndAuthHeader(t *testing.T) {
	received := make(chan ue.WSMessage, 1)
	errCh := make(chan error, 4)
	release := make(chan struct{})
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer ws-secret" {
			errCh <- fmt.Errorf("Authorization header = %q", got)
			http.Error(w, "bad auth", http.StatusUnauthorized)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			errCh <- fmt.Errorf("upgrade failed: %w", err)
			return
		}
		defer conn.Close()
		var msg ue.WSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			errCh <- fmt.Errorf("ReadJSON failed: %w", err)
			return
		}
		received <- msg
		<-release
	}))
	defer server.Close()

	client := NewFastAPIClient(toWSURL(server.URL), zap.NewNop(),
		WithAuthHeader("Authorization", "Bearer ws-secret"),
		WithTimings(time.Second, time.Hour, time.Millisecond, 10*time.Millisecond),
	)
	defer client.Close()

	err := client.PushMessage(context.Background(), ue.WSMessage{Type: "chart_data_push", Seq: 1, Payload: map[string]any{"rows": [][]string{{"a"}}}})
	if err != nil {
		t.Fatalf("PushMessage returned error: %v", err)
	}
	select {
	case err := <-errCh:
		t.Fatal(err)
	case msg := <-received:
		if msg.Type != "chart_data_push" {
			t.Fatalf("message type = %s", msg.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ws message")
	}

	stats := client.Stats()
	if !stats.Enabled || !stats.Connected || stats.SuccessCount != 1 || stats.MessageCountByType["chart_data_push"] != 1 {
		t.Fatalf("stats = %#v", stats)
	}
	b, _ := json.Marshal(stats)
	if strings.Contains(string(b), "ws-secret") || strings.Contains(string(b), "rows") {
		t.Fatalf("stats leaked token or payload: %s", string(b))
	}
	close(release)
	if err := client.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if client.Stats().Connected {
		t.Fatal("connected = true after Close")
	}
}

func TestFastAPIClientPushFailureStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not websocket", http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewFastAPIClient(toWSURL(server.URL), zap.NewNop(),
		WithTimings(time.Second, time.Hour, time.Millisecond, 10*time.Millisecond),
	)
	defer client.Close()

	err := client.PushMessage(context.Background(), ue.WSMessage{Type: "chart_data_push", Seq: 1})
	if err == nil {
		t.Fatal("expected push failure")
	}
	stats := client.Stats()
	if stats.Connected || stats.FailedCount != 1 || stats.LastError == "" {
		t.Fatalf("stats = %#v", stats)
	}
}

func TestFastAPIClientConcurrentPushesUseSingleDial(t *testing.T) {
	const pushes = 12
	var connCount int32
	received := make(chan ue.WSMessage, pushes)
	errCh := make(chan error, pushes)
	release := make(chan struct{})
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&connCount, 1)
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			errCh <- fmt.Errorf("upgrade failed: %w", err)
			return
		}
		defer conn.Close()
		for {
			var msg ue.WSMessage
			if err := conn.ReadJSON(&msg); err != nil {
				select {
				case <-release:
					return
				default:
					errCh <- fmt.Errorf("ReadJSON failed: %w", err)
					return
				}
			}
			received <- msg
		}
	}))
	defer server.Close()

	client := NewFastAPIClient(toWSURL(server.URL), zap.NewNop(),
		WithTimings(time.Second, time.Hour, time.Millisecond, 10*time.Millisecond),
	)
	defer client.Close()

	var wg sync.WaitGroup
	pushErrCh := make(chan error, pushes)
	for i := 0; i < pushes; i++ {
		wg.Add(1)
		go func(seq int) {
			defer wg.Done()
			if err := client.PushMessage(context.Background(), ue.WSMessage{Type: "chart_data_push", Seq: int64(seq)}); err != nil {
				pushErrCh <- err
			}
		}(i)
	}
	wg.Wait()
	close(pushErrCh)
	for err := range pushErrCh {
		t.Fatalf("PushMessage returned error: %v", err)
	}
	waitForMessages(t, received, pushes)
	if got := atomic.LoadInt32(&connCount); got != 1 {
		t.Fatalf("connection count = %d, want 1", got)
	}
	close(release)
	if err := client.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	select {
	case err := <-errCh:
		t.Fatal(err)
	default:
	}
}

func TestFastAPIClientMarkDisconnectedClosesNonCurrentConn(t *testing.T) {
	release := make(chan struct{})
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		<-release
	}))
	defer server.Close()
	defer close(release)

	connA, _, err := websocket.DefaultDialer.Dial(toWSURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial connA failed: %v", err)
	}
	defer connA.Close()
	connB, _, err := websocket.DefaultDialer.Dial(toWSURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial connB failed: %v", err)
	}

	client := &fastAPIClient{
		conn:    connA,
		safeURL: toWSURL(server.URL),
		log:     zap.NewNop(),
		stats: FastAPIClientStats{
			Connected:          true,
			MessageCountByType: map[string]int64{},
		},
	}

	client.markDisconnected(connB, "test.noncurrent", fmt.Errorf("old conn"))

	client.mu.Lock()
	current := client.conn
	connected := client.stats.Connected
	lastDisconnect := client.stats.LastDisconnectTime
	client.mu.Unlock()
	if current != connA {
		t.Fatal("current conn was changed by non-current disconnect")
	}
	if !connected || lastDisconnect != "" {
		t.Fatalf("stats changed unexpectedly: connected=%v lastDisconnect=%q", connected, lastDisconnect)
	}
	if err := connB.WriteMessage(websocket.TextMessage, []byte("should fail")); err == nil {
		t.Fatal("non-current conn was not closed")
	}
}

func TestFastAPIClientAuthHeaderModes(t *testing.T) {
	tests := []struct {
		name       string
		option     FastAPIClientOption
		wantHeader string
		wantValue  string
	}{
		{name: "api key", option: WithAuthHeader("X-API-Key", "secret"), wantHeader: "X-API-Key", wantValue: "secret"},
		{name: "custom", option: WithAuthHeader("X-FastAPI-Token", "secret"), wantHeader: "X-FastAPI-Token", wantValue: "secret"},
		{name: "none", option: nil, wantHeader: "Authorization", wantValue: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errCh := make(chan error, 4)
			received := make(chan struct{}, 1)
			release := make(chan struct{})
			upgrader := websocket.Upgrader{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get(tt.wantHeader); got != tt.wantValue {
					errCh <- fmt.Errorf("header %s = %q, want %q", tt.wantHeader, got, tt.wantValue)
					http.Error(w, "bad header", http.StatusUnauthorized)
					return
				}
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					errCh <- fmt.Errorf("upgrade failed: %w", err)
					return
				}
				defer conn.Close()
				if _, _, err := conn.ReadMessage(); err != nil {
					errCh <- fmt.Errorf("ReadMessage failed: %w", err)
					return
				}
				received <- struct{}{}
				<-release
			}))
			defer server.Close()

			opts := []FastAPIClientOption{WithTimings(time.Second, time.Hour, time.Millisecond, 10*time.Millisecond)}
			if tt.option != nil {
				opts = append(opts, tt.option)
			}
			client := NewFastAPIClient(toWSURL(server.URL), zap.NewNop(), opts...)
			defer client.Close()
			if err := client.PushMessage(context.Background(), ue.WSMessage{Type: "chart_data_push"}); err != nil {
				t.Fatalf("PushMessage returned error: %v", err)
			}
			select {
			case err := <-errCh:
				t.Fatal(err)
			case <-received:
			case <-time.After(time.Second):
				t.Fatal("timed out waiting for ws message")
			}
			close(release)
		})
	}
}

func waitForMessages(t *testing.T, ch <-chan ue.WSMessage, want int) {
	t.Helper()
	for i := 0; i < want; i++ {
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for message %d/%d", i+1, want)
		}
	}
}

func toWSURL(httpURL string) string {
	return "ws" + strings.TrimPrefix(httpURL, "http")
}
