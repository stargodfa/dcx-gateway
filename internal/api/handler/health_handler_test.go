package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthHandlerLight(t *testing.T) {
	router := healthRouter(NewHealthHandlerWithPings(okPing, okPing, "ws://127.0.0.1:8000/ws_data"))
	resp := performHealthRequest(router, "/api/health")
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.Code)
	}
	if strings.TrimSpace(resp.Body.String()) != `{"status":"ok"}` {
		t.Fatalf("body = %s", resp.Body.String())
	}
}

func TestHealthHandlerDeepOK(t *testing.T) {
	router := healthRouter(NewHealthHandlerWithPings(okPing, okPing, "ws://127.0.0.1:8000/ws_data"))
	resp := performHealthRequest(router, "/api/health?deep=true")
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", resp.Code, resp.Body.String())
	}
	for _, want := range []string{`"status":"ok"`, `"mysql"`, `"redis"`, `"fastapi_ws"`, `"configured":true`} {
		if !strings.Contains(resp.Body.String(), want) {
			t.Fatalf("body %s does not contain %s", resp.Body.String(), want)
		}
	}
}

func TestHealthHandlerDeepMySQLErrorDoesNotExposeRawError(t *testing.T) {
	router := healthRouter(NewHealthHandlerWithPings(func(ctx context.Context) error {
		return errors.New("dial tcp 127.0.0.1:3306: connect: connection refused")
	}, okPing, ""))
	resp := performHealthRequest(router, "/api/health?deep=true")
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503, body=%s", resp.Code, resp.Body.String())
	}
	for _, want := range []string{`"status":"degraded"`, `"mysql"`, `"fastapi_ws"`, `"configured":false`} {
		if !strings.Contains(resp.Body.String(), want) {
			t.Fatalf("body %s does not contain %s", resp.Body.String(), want)
		}
	}
	for _, leak := range []string{"dial tcp", "127.0.0.1", "3306", "connection refused"} {
		if strings.Contains(resp.Body.String(), leak) {
			t.Fatalf("body leaked raw error %q: %s", leak, resp.Body.String())
		}
	}
}

func TestHealthHandlerDeepRedisErrorDoesNotExposeRawError(t *testing.T) {
	router := healthRouter(NewHealthHandlerWithPings(okPing, func(ctx context.Context) error {
		return errors.New("redis 127.0.0.1:6379 authentication failed")
	}, "ws://127.0.0.1:8000/ws_data"))
	resp := performHealthRequest(router, "/api/health?deep=true")
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503, body=%s", resp.Code, resp.Body.String())
	}
	for _, want := range []string{`"status":"degraded"`, `"redis"`, `"status":"error"`} {
		if !strings.Contains(resp.Body.String(), want) {
			t.Fatalf("body %s does not contain %s", resp.Body.String(), want)
		}
	}
	for _, leak := range []string{"127.0.0.1", "6379", "authentication failed"} {
		if strings.Contains(resp.Body.String(), leak) {
			t.Fatalf("body leaked raw error %q: %s", leak, resp.Body.String())
		}
	}
}

func healthRouter(h *HealthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/health", h.Get)
	return router
}

func performHealthRequest(router *gin.Engine, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func okPing(ctx context.Context) error {
	return nil
}
