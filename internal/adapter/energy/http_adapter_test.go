package energy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"dcx-gateway/internal/config"

	"go.uber.org/zap"
)

func TestHTTPAdapterGetEnergySnapshotSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"buildings":[{"building_id":"library","building_name":"图书馆","value":1280.5,"unit":"kWh","timestamp":"2026-06-09T10:00:00Z"}],"timestamp":"2026-06-09T10:00:00Z"}`))
	}))
	defer server.Close()

	adapter := NewHTTPAdapter(server.URL, time.Second, "/snapshot", zap.NewNop())
	snapshot, err := adapter.GetEnergySnapshot(context.Background())
	if err != nil {
		t.Fatalf("GetEnergySnapshot returned error: %v", err)
	}
	if len(snapshot.Buildings) != 1 || snapshot.Buildings[0].BuildingID != "library" || snapshot.Buildings[0].Name != "图书馆" || snapshot.Buildings[0].Value != 1280.5 {
		t.Fatalf("snapshot = %#v", snapshot)
	}
}

func TestHTTPAdapterEnergyNon2xxReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusInternalServerError)
	}))
	defer server.Close()

	adapter := NewHTTPAdapter(server.URL, time.Second, "/snapshot", zap.NewNop())
	if _, err := adapter.GetEnergySnapshot(context.Background()); err == nil {
		t.Fatal("expected non-2xx error")
	}
}

func TestHTTPAdapterEnergyInvalidJSONReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{`))
	}))
	defer server.Close()

	adapter := NewHTTPAdapter(server.URL, time.Second, "/snapshot", zap.NewNop())
	if _, err := adapter.GetEnergySnapshot(context.Background()); err == nil {
		t.Fatal("expected invalid json error")
	}
}

func TestHTTPAdapterEnergyMissingBuildingsReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"timestamp":"2026-06-09T10:00:00Z"}`))
	}))
	defer server.Close()

	adapter := NewHTTPAdapter(server.URL, time.Second, "/snapshot", zap.NewNop())
	if _, err := adapter.GetEnergySnapshot(context.Background()); err == nil {
		t.Fatal("expected missing buildings error")
	}
}

func TestHTTPAdapterEnergyContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"buildings":[]}`))
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	adapter := NewHTTPAdapter(server.URL, time.Second, "/snapshot", zap.NewNop())
	if _, err := adapter.GetEnergySnapshot(ctx); err == nil {
		t.Fatal("expected context canceled error")
	}
}

func TestHTTPAdapterEnergyAuthHeaders(t *testing.T) {
	tests := []struct {
		name       string
		auth       config.OutboundAuthConfig
		wantHeader string
		wantValue  string
	}{
		{name: "bearer", auth: config.OutboundAuthConfig{Type: "bearer", Token: "secret"}, wantHeader: "Authorization", wantValue: "Bearer secret"},
		{name: "api key", auth: config.OutboundAuthConfig{Type: "api_key", HeaderName: "X-Energy-Key", Token: "secret"}, wantHeader: "X-Energy-Key", wantValue: "secret"},
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
				_, _ = w.Write([]byte(`{"buildings":[]}`))
			}))
			defer server.Close()

			adapter := NewHTTPAdapter(server.URL, time.Second, "/snapshot", zap.NewNop(), tt.auth)
			if _, err := adapter.GetEnergySnapshot(context.Background()); err != nil {
				t.Fatalf("GetEnergySnapshot returned error: %v", err)
			}
		})
	}
}
