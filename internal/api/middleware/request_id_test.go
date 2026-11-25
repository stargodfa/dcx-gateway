package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestRequestIDUsesRequestHeader(t *testing.T) {
	router := requestIDRouter()
	resp := performRequestIDRequest(router, map[string]string{HeaderRequestID: "rid-1"})
	if got := resp.Header().Get(HeaderRequestID); got != "rid-1" {
		t.Fatalf("response request id = %q, want rid-1", got)
	}
	if body := resp.Body.String(); body != "rid-1" {
		t.Fatalf("context request id = %q, want rid-1", body)
	}
}

func TestRequestIDUsesTraceHeader(t *testing.T) {
	router := requestIDRouter()
	resp := performRequestIDRequest(router, map[string]string{HeaderTraceID: "trace-1"})
	if got := resp.Header().Get(HeaderRequestID); got != "trace-1" {
		t.Fatalf("response request id = %q, want trace-1", got)
	}
}

func TestRequestIDGeneratesWhenMissing(t *testing.T) {
	router := requestIDRouter()
	resp := performRequestIDRequest(router, nil)
	if got := resp.Header().Get(HeaderRequestID); got == "" {
		t.Fatalf("generated request id is empty")
	}
}

func TestRequestIDTruncatesLongRequestHeader(t *testing.T) {
	router := requestIDRouter()
	longID := strings.Repeat("r", maxRequestIDLength+20)
	resp := performRequestIDRequest(router, map[string]string{HeaderRequestID: longID})
	gotHeader := resp.Header().Get(HeaderRequestID)
	if len(gotHeader) != maxRequestIDLength {
		t.Fatalf("response request id length = %d, want %d", len(gotHeader), maxRequestIDLength)
	}
	if gotBody := resp.Body.String(); gotBody != gotHeader {
		t.Fatalf("context request id = %q, want response header %q", gotBody, gotHeader)
	}
}

func TestRequestIDTruncatesLongTraceHeader(t *testing.T) {
	router := requestIDRouter()
	longID := strings.Repeat("t", maxRequestIDLength+20)
	resp := performRequestIDRequest(router, map[string]string{HeaderTraceID: longID})
	gotHeader := resp.Header().Get(HeaderRequestID)
	if len(gotHeader) != maxRequestIDLength {
		t.Fatalf("response request id length = %d, want %d", len(gotHeader), maxRequestIDLength)
	}
	if gotBody := resp.Body.String(); gotBody != gotHeader {
		t.Fatalf("context request id = %q, want response header %q", gotBody, gotHeader)
	}
}

func requestIDRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID(), RequestLogger(zap.NewNop()))
	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, RequestIDFromContext(c))
	})
	return router
}

func performRequestIDRequest(router *gin.Engine, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}
