package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apimiddleware "dcx-gateway/internal/api/middleware"
	"dcx-gateway/internal/config"
	"dcx-gateway/internal/model/protocol/ue"
	"dcx-gateway/internal/service"
	"dcx-gateway/internal/websocket"

	"github.com/gin-gonic/gin"
)

type fakeRuntimeWSClient struct {
	stats websocket.FastAPIClientStats
}

func (c fakeRuntimeWSClient) PushMessage(ctx context.Context, msg ue.WSMessage) error {
	return nil
}

func (c fakeRuntimeWSClient) PushMessages(ctx context.Context, messages []ue.WSMessage) error {
	return nil
}

func (c fakeRuntimeWSClient) Stats() websocket.FastAPIClientStats {
	return c.stats
}

func (c fakeRuntimeWSClient) Close() error {
	return nil
}

type fakeRuntimeService struct {
	options *ue.WarmupOptions
	resp    *ue.ReportScenePreloadResponse
	err     error
	req     service.ReportScenePreloadRequest
}

func (s *fakeRuntimeService) BuildWarmupOptions(ctx context.Context) (*ue.WarmupOptions, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.options != nil {
		return s.options, nil
	}
	return &ue.WarmupOptions{Version: 1, DataScripts: []ue.WarmupDataScriptOption{}, Charts: []ue.WarmupChartOption{}, Videos: []ue.WarmupVideoOption{}, Instances: []ue.WarmupInstanceOption{}}, nil
}

func (s *fakeRuntimeService) PreloadReportScene(ctx context.Context, req service.ReportScenePreloadRequest) (*ue.ReportScenePreloadResponse, error) {
	s.req = req
	if s.err != nil {
		return nil, s.err
	}
	if s.resp != nil {
		return s.resp, nil
	}
	return &ue.ReportScenePreloadResponse{Version: 1, ScriptID: req.ScriptID, ChapterID: req.ChapterID, SceneID: req.SceneID, Results: []ue.ReportScenePreloadResult{}}, nil
}

func TestRuntimeHandlerWSDataStatusDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewRuntimeHandler(nil, nil)
	r := gin.New()
	r.GET("/status", h.GetWSDataStatus)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if body := w.Body.String(); !strings.Contains(body, `"enabled":false`) {
		t.Fatalf("body = %s", body)
	}
}

func TestRuntimeHandlerWSDataStatusStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewRuntimeHandler(fakeRuntimeWSClient{stats: websocket.FastAPIClientStats{
		Enabled:            true,
		Connected:          true,
		MessageCountByType: map[string]int64{"chart_data_push": 2},
	}}, nil)
	r := gin.New()
	r.GET("/status", h.GetWSDataStatus)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if body := w.Body.String(); !strings.Contains(body, `"chart_data_push":2`) {
		t.Fatalf("body = %s", body)
	}
}

func TestRuntimeHandlerWarmupOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewRuntimeHandler(nil, &fakeRuntimeService{options: &ue.WarmupOptions{
		Version:     1,
		DataScripts: []ue.WarmupDataScriptOption{{DataScriptID: "security_main", DataSceneID: "campus_security"}},
		Charts:      []ue.WarmupChartOption{},
		Videos:      []ue.WarmupVideoOption{},
		Instances:   []ue.WarmupInstanceOption{},
	}})
	r := gin.New()
	r.GET("/warmup-options", h.GetWarmupOptions)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/warmup-options", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"data_script_id":"security_main"`) {
		t.Fatalf("body = %s", body)
	}
}

func TestRuntimeHandlerPreloadReportScene(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fake := &fakeRuntimeService{}
	h := NewRuntimeHandler(nil, fake)
	r := gin.New()
	r.POST("/report-scenes/preload", h.PreloadReportScene)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/report-scenes/preload", strings.NewReader(`{"script_id":"report_main","chapter_id":"chapter_a","scene_id":"scene_1","force":false,"push_ws":false}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	if fake.req.Force || fake.req.PushWS {
		t.Fatalf("force/push_ws defaults were not bound from request: %#v", fake.req)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := body["code"]; ok {
		t.Fatalf("runtime response should not be code/data wrapped: %#v", body)
	}
	if body["version"] == nil {
		t.Fatalf("runtime response should contain JSON body directly: %#v", body)
	}
}

func TestRuntimeReportScenePreloadRequiresAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewRuntimeHandler(nil, &fakeRuntimeService{})
	r := gin.New()
	api := r.Group("/api", apimiddleware.Auth(config.SecurityConfig{Enabled: true}, nil))
	api.POST("/ue/runtime/report-scenes/preload", h.PreloadReportScene)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/ue/runtime/report-scenes/preload", strings.NewReader(`{"script_id":"report_main","chapter_id":"chapter_a","scene_id":"scene_1"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
}

func TestRuntimeWarmupOptionsProtectReadFalseAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewRuntimeHandler(nil, &fakeRuntimeService{})
	r := gin.New()
	api := r.Group("/api", apimiddleware.Auth(config.SecurityConfig{Enabled: true, ProtectRead: false}, nil))
	api.GET("/ue/runtime/warmup-options", h.GetWarmupOptions)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/ue/runtime/warmup-options", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
}
