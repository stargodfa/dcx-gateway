package education

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"dcx-gateway/internal/config"

	"go.uber.org/zap"
)

func TestHTTPAdapterGetClassroomUsageSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"classrooms":[{"room_id":"A101","room_name":"A101","status":"occupied","course_name":"高等数学","teacher_name":"张老师","timestamp":"2026-06-09T10:00:00Z"},{"room_id":"A102","room_name":"A102","status":"free"}],"timestamp":"2026-06-09T10:00:00Z"}`))
	}))
	defer server.Close()

	adapter := NewHTTPAdapter(server.URL, time.Second, "/usage", zap.NewNop())
	snapshot, err := adapter.GetClassroomUsage(context.Background())
	if err != nil {
		t.Fatalf("GetClassroomUsage returned error: %v", err)
	}
	if len(snapshot.Classrooms) != 2 || snapshot.Classrooms[0].ClassroomID != "A101" || snapshot.Classrooms[0].Status != "occupied" || snapshot.Classrooms[1].Status != "free" {
		t.Fatalf("snapshot = %#v", snapshot)
	}
}

func TestHTTPAdapterEducationNon2xxReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusInternalServerError)
	}))
	defer server.Close()

	adapter := NewHTTPAdapter(server.URL, time.Second, "/usage", zap.NewNop())
	if _, err := adapter.GetClassroomUsage(context.Background()); err == nil {
		t.Fatal("expected non-2xx error")
	}
}

func TestHTTPAdapterEducationInvalidJSONReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{`))
	}))
	defer server.Close()

	adapter := NewHTTPAdapter(server.URL, time.Second, "/usage", zap.NewNop())
	if _, err := adapter.GetClassroomUsage(context.Background()); err == nil {
		t.Fatal("expected invalid json error")
	}
}

func TestHTTPAdapterEducationMissingClassroomsReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"timestamp":"2026-06-09T10:00:00Z"}`))
	}))
	defer server.Close()

	adapter := NewHTTPAdapter(server.URL, time.Second, "/usage", zap.NewNop())
	if _, err := adapter.GetClassroomUsage(context.Background()); err == nil {
		t.Fatal("expected missing classrooms error")
	}
}

func TestHTTPAdapterEducationContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"classrooms":[]}`))
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	adapter := NewHTTPAdapter(server.URL, time.Second, "/usage", zap.NewNop())
	if _, err := adapter.GetClassroomUsage(ctx); err == nil {
		t.Fatal("expected context canceled error")
	}
}

func TestHTTPAdapterEducationAuthHeaders(t *testing.T) {
	tests := []struct {
		name       string
		auth       config.OutboundAuthConfig
		wantHeader string
		wantValue  string
	}{
		{name: "bearer", auth: config.OutboundAuthConfig{Type: "bearer", Token: "secret"}, wantHeader: "Authorization", wantValue: "Bearer secret"},
		{name: "api key", auth: config.OutboundAuthConfig{Type: "api_key", HeaderName: "X-Education-Key", Token: "secret"}, wantHeader: "X-Education-Key", wantValue: "secret"},
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
				_, _ = w.Write([]byte(`{"classrooms":[]}`))
			}))
			defer server.Close()

			adapter := NewHTTPAdapter(server.URL, time.Second, "/usage", zap.NewNop(), tt.auth)
			if _, err := adapter.GetClassroomUsage(context.Background()); err != nil {
				t.Fatalf("GetClassroomUsage returned error: %v", err)
			}
		})
	}
}
