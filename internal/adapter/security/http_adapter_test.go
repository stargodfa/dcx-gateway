package security

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"dcx-gateway/internal/config"

	"go.uber.org/zap"
)

func TestHTTPAdapterGetAccessSummarySuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/access" {
			t.Fatalf("path = %s, want /access", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"in_count":1800,"out_count":1480,"total_count":3280,"timestamp":"2026-06-09T10:00:00Z"}`))
	}))
	defer server.Close()

	adapter := NewHTTPAdapter(server.URL, time.Second, "/access", "/fire", zap.NewNop())
	summary, err := adapter.GetAccessSummary(context.Background())
	if err != nil {
		t.Fatalf("GetAccessSummary returned error: %v", err)
	}
	if summary.InCount != 1800 || summary.OutCount != 1480 || summary.TotalCount != 3280 || summary.Timestamp == "" {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestHTTPAdapterGetFireAlertsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/fire" {
			t.Fatalf("path = %s, want /fire", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"alerts":[{"alert_id":"FIRE-001","level":"normal","location":"实验楼","message":"无异常","timestamp":"2026-06-09T10:00:00Z"}]}`))
	}))
	defer server.Close()

	adapter := NewHTTPAdapter(server.URL, time.Second, "/access", "/fire", zap.NewNop())
	alerts, err := adapter.GetFireAlerts(context.Background())
	if err != nil {
		t.Fatalf("GetFireAlerts returned error: %v", err)
	}
	if len(alerts) != 1 || alerts[0].AlertID != "FIRE-001" || alerts[0].Level != "normal" {
		t.Fatalf("alerts = %#v", alerts)
	}
}

func TestHTTPAdapterNon2xxReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadGateway)
	}))
	defer server.Close()

	adapter := NewHTTPAdapter(server.URL, time.Second, "/access", "/fire", zap.NewNop())
	if _, err := adapter.GetAccessSummary(context.Background()); err == nil {
		t.Fatal("expected non-2xx error")
	}
}

func TestHTTPAdapterInvalidJSONReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{`))
	}))
	defer server.Close()

	adapter := NewHTTPAdapter(server.URL, time.Second, "/access", "/fire", zap.NewNop())
	if _, err := adapter.GetAccessSummary(context.Background()); err == nil {
		t.Fatal("expected invalid json error")
	}
}

func TestHTTPAdapterTimeoutReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte(`{"in_count":1}`))
	}))
	defer server.Close()

	adapter := NewHTTPAdapter(server.URL, time.Millisecond, "/access", "/fire", zap.NewNop())
	if _, err := adapter.GetAccessSummary(context.Background()); err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestHTTPAdapterContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"in_count":1,"out_count":2,"total_count":3}`))
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	adapter := NewHTTPAdapter(server.URL, time.Second, "/access", "/fire", zap.NewNop())
	if _, err := adapter.GetAccessSummary(ctx); err == nil {
		t.Fatal("expected context canceled error")
	}
}

func TestHTTPAdapterAuthHeaders(t *testing.T) {
	tests := []struct {
		name       string
		auth       config.OutboundAuthConfig
		wantHeader string
		wantValue  string
	}{
		{name: "bearer", auth: config.OutboundAuthConfig{Type: "bearer", Token: "secret"}, wantHeader: "Authorization", wantValue: "Bearer secret"},
		{name: "api key default", auth: config.OutboundAuthConfig{Type: "api_key", Token: "secret"}, wantHeader: "X-API-Key", wantValue: "secret"},
		{name: "custom header", auth: config.OutboundAuthConfig{Type: "header", HeaderName: "X-Upstream-Token", Token: "secret"}, wantHeader: "X-Upstream-Token", wantValue: "secret"},
		{name: "none", auth: config.OutboundAuthConfig{Type: "none", Token: "secret"}, wantHeader: "Authorization", wantValue: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get(tt.wantHeader); got != tt.wantValue {
					t.Fatalf("header %s = %q, want %q", tt.wantHeader, got, tt.wantValue)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"in_count":1,"out_count":2,"total_count":3}`))
			}))
			defer server.Close()

			adapter := NewHTTPAdapter(server.URL, time.Second, "/access", "/fire", zap.NewNop(), tt.auth)
			if _, err := adapter.GetAccessSummary(context.Background()); err != nil {
				t.Fatalf("GetAccessSummary returned error: %v", err)
			}
		})
	}
}
