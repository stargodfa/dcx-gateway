package main

import (
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"dcx-gateway/internal/model/protocol/ue"

	"github.com/gorilla/websocket"
)

type demoReceiver struct {
	url         string
	messages    atomic.Int64
	mutex       sync.Mutex
	connections map[*websocket.Conn]struct{}
	stopHTTP    func() error
}

// The receiver observes real chart_data_push messages but does not emulate an
// ACK protocol. Gateway delivery remains best-effort, as in the real service.
func startDemoReceiver(token string) (*demoReceiver, error) {
	receiver := &demoReceiver{connections: map[*websocket.Conn]struct{}{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/ws_data", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		upgrader := websocket.Upgrader{}
		connection, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		receiver.mutex.Lock()
		receiver.connections[connection] = struct{}{}
		receiver.mutex.Unlock()
		defer func() {
			receiver.mutex.Lock()
			delete(receiver.connections, connection)
			receiver.mutex.Unlock()
			_ = connection.Close()
		}()
		for {
			var message ue.WSMessage
			if err := connection.ReadJSON(&message); err != nil {
				return
			}
			if message.Type == "chart_data_push" {
				receiver.messages.Add(1)
			}
		}
	})
	baseURL, stop, err := serveLocal(mux)
	if err != nil {
		return nil, err
	}
	receiver.url = "ws" + strings.TrimPrefix(baseURL, "http") + "/ws_data"
	receiver.stopHTTP = stop
	return receiver, nil
}

func (r *demoReceiver) close() error {
	r.mutex.Lock()
	connections := make([]*websocket.Conn, 0, len(r.connections))
	for connection := range r.connections {
		connections = append(connections, connection)
	}
	r.mutex.Unlock()
	for _, connection := range connections {
		_ = connection.Close()
	}
	return r.stopHTTP()
}
