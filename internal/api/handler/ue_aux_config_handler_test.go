package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dcx-gateway/internal/model/domain"
	"dcx-gateway/internal/repository"
	"dcx-gateway/internal/service"

	"github.com/gin-gonic/gin"
)

type stubAuxConfigService struct {
	service.UEAuxConfigService
	err   error
	calls int
}

func (s *stubAuxConfigService) Replace(context.Context, domain.AuxConfigResource, map[string]any) (map[string]any, error) {
	s.calls++
	return map[string]any{"version": 1}, s.err
}

func TestAuxConfigHandlerRejectsMalformedAndOversizedBodies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tt := range []struct {
		body   string
		status int
	}{
		{`null`, 400}, {`[]`, 400}, {`{`, 400}, {`{} {}`, 400}, {`{} garbage`, 400},
		{`{"devices":[]}` + strings.Repeat(" ", 100), 413},
		{`{"devices":[]}`, 200},
	} {
		svc := &stubAuxConfigService{}
		h := NewUEAuxConfigHandler(svc, nil, 32)
		router := gin.New()
		router.PUT("/config", h.Replace(domain.SensorDevices))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/config", strings.NewReader(tt.body)))
		if w.Code != tt.status {
			t.Fatalf("body %q: status=%d response=%s", tt.body, w.Code, w.Body)
		}
		if tt.status != 200 && svc.calls != 0 {
			t.Fatal("invalid request reached service")
		}
	}
}

func TestAuxConfigHandlerMapsErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tt := range []struct {
		err    error
		status int
	}{
		{service.ErrInvalidAuxConfig, 400},
		{repository.ErrAuxConfigConflict, 409},
		{errors.New("private database details"), 500},
	} {
		h := NewUEAuxConfigHandler(&stubAuxConfigService{err: tt.err}, nil, 0)
		router := gin.New()
		router.PUT("/config", h.Replace(domain.SensorDevices))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/config", strings.NewReader(`{"devices":[]}`)))
		if w.Code != tt.status || strings.Contains(w.Body.String(), "private database details") {
			t.Fatalf("status=%d response=%s", w.Code, w.Body)
		}
	}
}
