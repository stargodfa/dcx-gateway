package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dcx-gateway/internal/model/dto"
	"dcx-gateway/internal/model/protocol/ue"
	"dcx-gateway/internal/repository"
	"dcx-gateway/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type fakeUEConfigService struct {
	raw map[string]json.RawMessage
}

func (s fakeUEConfigService) GetRawConfig(ctx context.Context, key string) (json.RawMessage, error) {
	if raw, ok := s.raw[key]; ok {
		return raw, nil
	}
	return nil, repository.ErrNotFound
}

func (s fakeUEConfigService) ReplaceRawConfig(ctx context.Context, routeKey string, raw []byte) (json.RawMessage, error) {
	var object map[string]json.RawMessage
	if !json.Valid(raw) || json.Unmarshal(raw, &object) != nil || object == nil {
		return nil, service.ErrInvalidModelConfigPatch
	}
	return json.RawMessage(raw), nil
}

func (s fakeUEConfigService) ListRawConfigs(ctx context.Context) (*ue.ConfigIndex, error) {
	return &ue.ConfigIndex{Configs: []ue.ConfigMeta{
		{ConfigKey: "chart_config", FileName: "chart_config.json", Version: "v1", Enabled: true},
	}}, nil
}

func (s fakeUEConfigService) BuildBootstrap(ctx context.Context) (*ue.BootstrapConfig, error) {
	return &ue.BootstrapConfig{
		Version:      1,
		InitConfig:   json.RawMessage(`{"version":1}`),
		PathConfig:   json.RawMessage(`{"version":1}`),
		ChartConfig:  json.RawMessage(`{"version":1}`),
		ChartBinding: json.RawMessage(`{"version":1}`),
		DroneConfig:  json.RawMessage(`{"version":1}`),
		ModelTable:   &ue.ModelTable{Version: 1},
		ModelConfig:  &ue.ModelConfig{Version: 1},
		SceneDesired: &ue.SceneDesiredConfig{Version: 1},
	}, nil
}

func (s fakeUEConfigService) BuildChartBinding(ctx context.Context) (*ue.ChartBinding, error) {
	return &ue.ChartBinding{
		Version: 1,
		Bindings: []ue.ChartBindingItem{
			{BindingID: "sim_3", ChartID: "chart_h"},
		},
	}, nil
}

func (s fakeUEConfigService) BuildVideoStreams(ctx context.Context) (*ue.VideoStreams, error) {
	return &ue.VideoStreams{Version: 1}, nil
}

func (s fakeUEConfigService) BuildChartConfig(ctx context.Context) (*ue.ChartConfig, error) {
	return &ue.ChartConfig{Version: 1}, nil
}

func (s fakeUEConfigService) BuildStructuredChartConfig(ctx context.Context) (*ue.ChartConfig, error) {
	return &ue.ChartConfig{
		Version:                 1,
		OverlayHUDMaxPanelWidth: 920,
		Charts: []ue.ChartPanel{
			{ChartID: "chart_h", ChartName: "今日出入人数", Visible: true, CellValues: [][]string{{"指标", "数值"}}},
		},
	}, nil
}

func (s fakeUEConfigService) BuildRuntimeChartConfig(ctx context.Context) (*ue.ChartConfig, error) {
	return &ue.ChartConfig{
		Version:                 1,
		OverlayHUDMaxPanelWidth: 920,
		Charts: []ue.ChartPanel{
			{ChartID: "chart_h", ChartName: "今日出入人数", Visible: true, CellValues: [][]string{{"指标", "数值"}, {"今日进入", "1800"}}},
		},
	}, nil
}

func (s fakeUEConfigService) BuildReportScripts(ctx context.Context) (*ue.LightReportScripts, error) {
	return &ue.LightReportScripts{
		Version: 1,
		Scripts: []ue.LightReportScript{
			{
				ScriptID: "security_main",
				Label:    "security",
				Chapters: []ue.LightReportChapter{
					{
						ChapterID: "security_main",
						Label:     "security",
						Scenes: []ue.LightReportScene{
							{
								SceneID: "campus_security",
								Label:   "campus_security",
								Warmup: ue.ReportWarmup{
									Data:   []string{"security.access.today"},
									Videos: []string{"vs_cam_65"},
									Charts: []string{"chart_h"},
								},
							},
						},
					},
				},
			},
		},
	}, nil
}

func (s fakeUEConfigService) CreateReportScript(ctx context.Context, patch service.ReportScriptPatch) (json.RawMessage, error) {
	if patch.ScriptID == "" {
		return nil, service.ErrInvalidReportScriptsPatch
	}
	return json.RawMessage(`{"version":1,"scripts":[{"script_id":"` + patch.ScriptID + `","label":"` + patch.ScriptID + `","chapters":[]}]}`), nil
}

func (s fakeUEConfigService) UpdateReportScript(ctx context.Context, patch service.ReportScriptPatch) (json.RawMessage, error) {
	if patch.ScriptID == "missing" {
		return nil, repository.ErrNotFound
	}
	if patch.NewScriptID == "conflict" {
		return nil, service.ErrReportScriptsConflict
	}
	return json.RawMessage(`{"version":1,"scripts":[{"script_id":"` + patch.ScriptID + `","label":"ok","chapters":[]}]}`), nil
}

func (s fakeUEConfigService) DeleteReportScript(ctx context.Context, scriptID string) (json.RawMessage, error) {
	return json.RawMessage(`{"version":1,"scripts":[]}`), nil
}

func (s fakeUEConfigService) CreateReportChapter(ctx context.Context, patch service.ReportChapterPatch) (json.RawMessage, error) {
	return json.RawMessage(`{"version":1,"scripts":[{"script_id":"` + patch.ScriptID + `","label":"s","chapters":[{"chapter_id":"` + patch.ChapterID + `","label":"c","scenes":[]}]}]}`), nil
}

func (s fakeUEConfigService) UpdateReportChapter(ctx context.Context, patch service.ReportChapterPatch) (json.RawMessage, error) {
	return s.CreateReportChapter(ctx, patch)
}

func (s fakeUEConfigService) DeleteReportChapter(ctx context.Context, patch service.ReportChapterPatch) (json.RawMessage, error) {
	return json.RawMessage(`{"version":1,"scripts":[]}`), nil
}

func (s fakeUEConfigService) CreateReportScene(ctx context.Context, patch service.ReportScenePatch) (json.RawMessage, error) {
	return json.RawMessage(`{"version":1,"scripts":[{"script_id":"` + patch.ScriptID + `","label":"s","chapters":[{"chapter_id":"` + patch.ChapterID + `","label":"c","scenes":[{"scene_id":"` + patch.SceneID + `","label":"scene","camera":null,"bindings":[],"warmup":{"data":[],"videos":[],"charts":[]}}]}]}]}`), nil
}

func (s fakeUEConfigService) UpdateReportScene(ctx context.Context, patch service.ReportScenePatch) (json.RawMessage, error) {
	return s.CreateReportScene(ctx, patch)
}

func (s fakeUEConfigService) DeleteReportScene(ctx context.Context, patch service.ReportScenePatch) (json.RawMessage, error) {
	return json.RawMessage(`{"version":1,"scripts":[]}`), nil
}

func (s fakeUEConfigService) UpdateReportSceneCamera(ctx context.Context, patch service.ReportScenePatch) (json.RawMessage, error) {
	return json.RawMessage(`{"version":1,"scripts":[{"script_id":"` + patch.ScriptID + `","label":"s","chapters":[]}]}`), nil
}

func (s fakeUEConfigService) CreateReportSceneBindings(ctx context.Context, patch service.ReportBindingPatch) (json.RawMessage, error) {
	return json.RawMessage(`{"version":1,"scripts":[]}`), nil
}

func (s fakeUEConfigService) UpdateReportSceneBindings(ctx context.Context, patch service.ReportBindingPatch) (json.RawMessage, error) {
	return json.RawMessage(`{"version":1,"scripts":[]}`), nil
}

func (s fakeUEConfigService) DeleteReportSceneBindings(ctx context.Context, patch service.ReportBindingDeletePatch) (json.RawMessage, error) {
	return json.RawMessage(`{"version":1,"scripts":[]}`), nil
}

func (s fakeUEConfigService) UpdateReportSceneWarmup(ctx context.Context, patch service.ReportWarmupPatch) (json.RawMessage, error) {
	return json.RawMessage(`{"version":1,"scripts":[]}`), nil
}

func (s fakeUEConfigService) BuildModelTypes(ctx context.Context) (*ue.ModelTypesConfig, error) {
	return &ue.ModelTypesConfig{
		Version: 1,
		ModelTypes: []ue.ModelType{
			{
				ModelID:        "building",
				URI:            "file://building",
				Format:         "udatasmith",
				SourcePath:     "./building",
				BaseBoundsSize: ue.BoundsSize{X: 1, Y: 2, Z: 3},
				Enabled:        true,
			},
		},
	}, nil
}

func (s fakeUEConfigService) CreateModelTypes(ctx context.Context, patches []service.ModelTypePatch) (*ue.ModelTypesConfig, error) {
	return s.BuildModelTypes(ctx)
}

func (s fakeUEConfigService) UpdateModelTypes(ctx context.Context, patches []service.ModelTypePatch) (*ue.ModelTypesConfig, error) {
	return s.BuildModelTypes(ctx)
}

func (s fakeUEConfigService) DeleteModelTypes(ctx context.Context, ids []string) (*ue.ModelTypesConfig, error) {
	return s.BuildModelTypes(ctx)
}

func (s fakeUEConfigService) BuildModelTable(ctx context.Context) (*ue.ModelTable, error) {
	return &ue.ModelTable{Version: 1, Entries: []ue.ModelTableEntry{
		{InstanceID: "building_001", ModelID: "building", DisplayName: "building"},
	}}, nil
}

func (s fakeUEConfigService) CreateModelTableEntries(ctx context.Context, patches []service.ModelInstancePatch) (*ue.ModelTable, error) {
	return s.BuildModelTable(ctx)
}

func (s fakeUEConfigService) UpdateModelTableEntries(ctx context.Context, patches []service.ModelInstancePatch) (*ue.ModelTable, error) {
	return s.BuildModelTable(ctx)
}

func (s fakeUEConfigService) DeleteModelTableEntries(ctx context.Context, ids []string) (*ue.ModelTable, error) {
	return s.BuildModelTable(ctx)
}

func (s fakeUEConfigService) BuildStructuredModelConfig(ctx context.Context) (*ue.StructuredModelConfig, error) {
	return &ue.StructuredModelConfig{
		Version: 1,
		Placements: map[string]ue.ModelPlacementState{
			"building_001": {Visible: true, Transform: &ue.Transform{Scale: &ue.Vector3{X: 1, Y: 1, Z: 1}}},
		},
	}, nil
}

func (s fakeUEConfigService) BuildRuntimeModelConfig(ctx context.Context) (*ue.ModelConfig, error) {
	return &ue.ModelConfig{
		Version: 1,
		LastLoadByInstanceID: map[string]ue.ModelInstanceState{
			"building_001": {ModelID: "building", Visible: true, BoundsSize: &ue.BoundsSize{X: 1, Y: 2, Z: 3}},
		},
	}, nil
}

func (s fakeUEConfigService) UpdateModelPlacements(ctx context.Context, patches []service.ModelPlacementPatch) (*ue.StructuredModelConfig, error) {
	return s.BuildStructuredModelConfig(ctx)
}

func (s fakeUEConfigService) DeleteModelPlacements(ctx context.Context, ids []string) (*ue.StructuredModelConfig, error) {
	return s.BuildStructuredModelConfig(ctx)
}

func (s fakeUEConfigService) BuildSceneDesiredConfig(ctx context.Context) (*ue.SceneDesiredConfig, error) {
	return &ue.SceneDesiredConfig{Version: 1, SceneModelsDesired: []ue.SceneModelDesired{
		{InstanceID: "building_001", ModelID: "building", URI: "file://building", Format: "udatasmith", SourcePath: "./building", DisplayName: "building", Visible: true},
	}}, nil
}

func (s fakeUEConfigService) UpdateInitConfig(ctx context.Context, patch service.InitConfigUpdate) (json.RawMessage, error) {
	cameraID := "camera_main"
	if patch.CameraID != nil {
		cameraID = *patch.CameraID
	}
	return json.RawMessage(`{"version":1,"camera":{"camera_id":"` + cameraID + `","position":{"x":1,"y":2,"z":3},"rotation":{"yaw":10,"pitch":0,"roll":0}},"auto_load_on_startup":false,"instance_ids":[]}`), nil
}

func (s fakeUEConfigService) AddInitInstanceIDs(ctx context.Context, ids []string) (json.RawMessage, error) {
	return json.RawMessage(`{"version":1,"camera":{"camera_id":"camera_main"},"auto_load_on_startup":false,"instance_ids":["building_001","building_002"]}`), nil
}

func (s fakeUEConfigService) DeleteInitInstanceIDs(ctx context.Context, ids []string) (json.RawMessage, error) {
	return json.RawMessage(`{"version":1,"camera":{"camera_id":"camera_main"},"auto_load_on_startup":false,"instance_ids":[]}`), nil
}

func (s fakeUEConfigService) PatchSceneDesiredModels(ctx context.Context, patches []service.SceneDesiredModelPatch) (*ue.SceneDesiredConfig, error) {
	return s.BuildSceneDesiredConfig(ctx)
}

func (s fakeUEConfigService) UpsertSceneDesiredModels(ctx context.Context, patches []service.SceneDesiredModelPatch) (*ue.SceneDesiredConfig, error) {
	return s.BuildSceneDesiredConfig(ctx)
}

func (s fakeUEConfigService) DeleteSceneDesiredModels(ctx context.Context, ids []string) (*ue.SceneDesiredConfig, error) {
	return s.BuildSceneDesiredConfig(ctx)
}

func newUEConfigHandlerTestRouter(opts ...UEConfigHandlerOption) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewUEConfigHandler(fakeUEConfigService{raw: map[string]json.RawMessage{
		"chart_config":  json.RawMessage(`{"version":1,"charts":[]}`),
		"chart_binding": json.RawMessage(`{"version":1,"bindings":[{"binding_id":"sim_3","chart_id":"chart_h"}]}`),
	}}, zap.NewNop(), opts...)

	r := gin.New()
	r.GET("/api/ue/config/raw/:key", h.GetRawConfig)
	r.GET("/api/ue/config/index", h.GetConfigIndex)
	r.GET("/api/ue/config/bootstrap", h.GetBootstrap)
	r.GET("/api/ue/config/chart-config", h.GetChartConfig)
	r.PUT("/api/ue/config/chart-config", h.ReplaceChartConfig)
	r.GET("/api/ue/config/chart-binding", h.GetChartBinding)
	r.PUT("/api/ue/config/chart-binding", h.ReplaceChartBinding)
	r.GET("/api/ue/config/report-scripts", h.GetReportScripts)
	r.POST("/api/ue/config/report-scripts/scripts/create", h.CreateReportScript)
	r.PUT("/api/ue/config/report-scripts/scripts", h.UpdateReportScript)
	r.DELETE("/api/ue/config/report-scripts/scripts/delete", h.DeleteReportScript)
	r.POST("/api/ue/config/report-scripts/scenes/create", h.CreateReportScene)
	r.PUT("/api/ue/config/report-scripts/scenes", h.UpdateReportScene)
	r.PUT("/api/ue/config/report-scripts/scenes/camera", h.UpdateReportSceneCamera)
	r.POST("/api/ue/config/report-scripts/scenes/bindings/create", h.CreateReportSceneBindings)
	r.PUT("/api/ue/config/init-config", h.UpdateInitConfig)
	r.POST("/api/ue/config/init-config/instance_ids/create", h.CreateInitInstanceIDs)
	r.GET("/api/ue/config/scene-desired", h.GetSceneDesired)
	r.GET("/api/ue/config/model-types", h.GetModelTypes)
	r.GET("/api/ue/config/model-table", h.GetModelTable)
	r.GET("/api/ue/config/model-config", h.GetModelConfig)
	r.PUT("/api/ue/config/model-config/placements", h.UpdateModelPlacements)
	r.PUT("/api/ue/config/model-config/placements/:instance_id", h.UpdateModelPlacement)
	r.DELETE("/api/ue/config/model-config/placements/delete", h.DeleteModelPlacements)
	return r
}

func TestUEConfigHandlerRawConfigReturnsJSONBody(t *testing.T) {
	r := newUEConfigHandlerTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/ue/config/raw/chart_config", nil)

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), `"code"`) || strings.Contains(w.Body.String(), `"data"`) {
		t.Fatalf("raw response should not be wrapped: %s", w.Body.String())
	}
	if strings.TrimSpace(w.Body.String()) != `{"version":1,"charts":[]}` {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestUEConfigHandlerIndex(t *testing.T) {
	r := newUEConfigHandlerTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/ue/config/index", nil)

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"configs"`) || !strings.Contains(w.Body.String(), `"chart_config"`) {
		t.Fatalf("index body = %s", w.Body.String())
	}
}

func TestUEConfigHandlerChartBindingModes(t *testing.T) {
	r := newUEConfigHandlerTestRouter()

	full := httptest.NewRecorder()
	r.ServeHTTP(full, httptest.NewRequest(http.MethodGet, "/api/ue/config/chart-binding?mode=full", nil))
	if full.Code != http.StatusOK {
		t.Fatalf("full status = %d, body=%s", full.Code, full.Body.String())
	}
	if !strings.Contains(full.Body.String(), `"binding_id":"sim_3"`) {
		t.Fatalf("full body = %s", full.Body.String())
	}

	generated := httptest.NewRecorder()
	r.ServeHTTP(generated, httptest.NewRequest(http.MethodGet, "/api/ue/config/chart-binding?mode=generated", nil))
	if generated.Code != http.StatusOK {
		t.Fatalf("generated status = %d, body=%s", generated.Code, generated.Body.String())
	}
	if !strings.Contains(generated.Body.String(), `"binding_id":"sim_3"`) {
		t.Fatalf("generated body = %s", generated.Body.String())
	}
}

func TestUEConfigHandlerChartConfigModes(t *testing.T) {
	r := newUEConfigHandlerTestRouter()

	raw := httptest.NewRecorder()
	r.ServeHTTP(raw, httptest.NewRequest(http.MethodGet, "/api/ue/config/chart-config?mode=full", nil))
	if raw.Code != http.StatusOK {
		t.Fatalf("raw status = %d, body=%s", raw.Code, raw.Body.String())
	}
	if strings.TrimSpace(raw.Body.String()) != `{"version":1,"charts":[]}` {
		t.Fatalf("raw body = %s", raw.Body.String())
	}

	structured := httptest.NewRecorder()
	r.ServeHTTP(structured, httptest.NewRequest(http.MethodGet, "/api/ue/config/chart-config?mode=structured", nil))
	if structured.Code != http.StatusOK {
		t.Fatalf("structured status = %d, body=%s", structured.Code, structured.Body.String())
	}
	if !strings.Contains(structured.Body.String(), `"chart_id":"chart_h"`) {
		t.Fatalf("structured body = %s", structured.Body.String())
	}

	runtime := httptest.NewRecorder()
	r.ServeHTTP(runtime, httptest.NewRequest(http.MethodGet, "/api/ue/config/chart-config?mode=runtime", nil))
	if runtime.Code != http.StatusOK {
		t.Fatalf("runtime status = %d, body=%s", runtime.Code, runtime.Body.String())
	}
	if !strings.Contains(runtime.Body.String(), `"1800"`) {
		t.Fatalf("runtime body = %s", runtime.Body.String())
	}
}

func TestUEConfigHandlerChartConfigLightAliasesStructured(t *testing.T) {
	r := newUEConfigHandlerTestRouter()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/ue/config/chart-config?mode=light", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"chart_id":"chart_h"`) {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestUEConfigHandlerReportScriptsLightHasNoZeroCamera(t *testing.T) {
	r := newUEConfigHandlerTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/ue/config/report-scripts?mode=light", nil)

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), `"camera"`) {
		t.Fatalf("light report should not include zero-value camera: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"chapters"`) {
		t.Fatalf("light report should keep report_scripts shape: %s", w.Body.String())
	}
}

func TestUEConfigHandlerReportScriptsCreateReturnsRawBody(t *testing.T) {
	r := newUEConfigHandlerTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/ue/config/report-scripts/scripts/create", strings.NewReader(`{"script_id":"new_script"}`))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), `"code"`) || strings.Contains(w.Body.String(), `"data"`) {
		t.Fatalf("report_scripts response should not be wrapped: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"script_id":"new_script"`) {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestUEConfigHandlerReportScriptsErrors(t *testing.T) {
	r := newUEConfigHandlerTestRouter()

	bad := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/ue/config/report-scripts/scripts/create", strings.NewReader(`{`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(bad, req)
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("bad status = %d, want 400, body=%s", bad.Code, bad.Body.String())
	}

	conflict := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/ue/config/report-scripts/scripts", strings.NewReader(`{"script_id":"s","new_script_id":"conflict"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(conflict, req)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("conflict status = %d, want 409, body=%s", conflict.Code, conflict.Body.String())
	}

	missing := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/ue/config/report-scripts/scripts", strings.NewReader(`{"script_id":"missing"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(missing, req)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, want 404, body=%s", missing.Code, missing.Body.String())
	}
}

func TestUEConfigHandlerReportScriptsCameraValidation(t *testing.T) {
	r := newUEConfigHandlerTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/ue/config/report-scripts/scenes/camera", strings.NewReader(`{
		"script_id":"s",
		"chapter_id":"c",
		"scene_id":"scene",
		"camera":{"camera_id":"camera_main","position":{"x":0,"y":0,"z":0}}
	}`))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "rotation") {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestUEConfigHandlerReportScriptsSceneUpdateRejectsNestedFields(t *testing.T) {
	r := newUEConfigHandlerTestRouter()
	for _, body := range []string{
		`{"script_id":"s","chapter_id":"c","scene_id":"scene","camera":null}`,
		`{"script_id":"s","chapter_id":"c","scene_id":"scene","bindings":[]}`,
		`{"script_id":"s","chapter_id":"c","scene_id":"scene","warmup":{"data":[],"videos":[],"charts":[]}}`,
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/api/ue/config/report-scripts/scenes", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("body %s status = %d, want 400, response=%s", body, w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "/scenes/camera") || !strings.Contains(w.Body.String(), "/scenes/bindings") || !strings.Contains(w.Body.String(), "/scenes/warmup") {
			t.Fatalf("response should mention dedicated endpoints: %s", w.Body.String())
		}
	}
}

func TestUEConfigHandlerReportScriptsBodyLimit(t *testing.T) {
	r := newUEConfigHandlerTestRouter(WithMaxRawConfigBodyBytes(24))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/ue/config/report-scripts/scripts/create", strings.NewReader(`{"script_id":"this_body_is_too_large"}`))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "request body too large") {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestUEConfigHandlerUpdateInitConfigReturnsRawJSON(t *testing.T) {
	r := newUEConfigHandlerTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPut,
		"/api/ue/config/init-config",
		strings.NewReader(`{"camera_id":"default","position":{"x":1,"y":2,"z":3},"rotation":{"yaw":10,"pitch":0,"roll":0}}`),
	)
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), `"code"`) || strings.Contains(w.Body.String(), `"data"`) {
		t.Fatalf("init config update response should not be wrapped: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"camera_id":"default"`) {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestUEConfigHandlerAddInitInstanceIDsReturnsRawJSON(t *testing.T) {
	r := newUEConfigHandlerTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/ue/config/init-config/instance_ids/create",
		strings.NewReader(`{"instance_ids":["building_001","building_002"]}`),
	)
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), `"code"`) || strings.Contains(w.Body.String(), `"data"`) {
		t.Fatalf("init config update response should not be wrapped: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"building_002"`) {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestUEConfigHandlerUpdateInitConfigRejectsInstanceIDs(t *testing.T) {
	r := newUEConfigHandlerTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPut,
		"/api/ue/config/init-config",
		strings.NewReader(`{"instance_ids":["building_001"]}`),
	)
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "instance_ids/create") {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestUEConfigHandlerGetSceneDesiredReturnsRawJSONStyle(t *testing.T) {
	r := newUEConfigHandlerTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/ue/config/scene-desired", nil)

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), `"code"`) || strings.Contains(w.Body.String(), `"data"`) {
		t.Fatalf("scene desired response should not be wrapped: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"scene_models_desired"`) {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestUEConfigHandlerUpdateModelPlacementsRejectsBoundsSize(t *testing.T) {
	r := newUEConfigHandlerTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPut,
		"/api/ue/config/model-config/placements",
		strings.NewReader(`{"instance_id":"building_001","bounds_size":{"x":1,"y":2,"z":3}}`),
	)
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "bounds_size") {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestUEConfigHandlerUpdateModelPlacementsAcceptsTransformAndRejectsMixedFlat(t *testing.T) {
	r := newUEConfigHandlerTestRouter()

	ok := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPut,
		"/api/ue/config/model-config/placements/building_001",
		strings.NewReader(`{"visible":true,"transform":{"position":{"x":1,"y":2,"z":3},"rotation":{"yaw":0,"pitch":0,"roll":0},"scale":{"x":1,"y":1,"z":1}}}`),
	)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(ok, req)
	if ok.Code != http.StatusOK {
		t.Fatalf("transform status = %d, want 200, body=%s", ok.Code, ok.Body.String())
	}

	mixed := httptest.NewRecorder()
	req = httptest.NewRequest(
		http.MethodPut,
		"/api/ue/config/model-config/placements",
		strings.NewReader(`{"instance_id":"building_001","transform":{"position":{"x":1,"y":2,"z":3}},"position":{"x":4,"y":5,"z":6}}`),
	)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(mixed, req)
	if mixed.Code != http.StatusBadRequest {
		t.Fatalf("mixed status = %d, want 400, body=%s", mixed.Code, mixed.Body.String())
	}
	if !strings.Contains(mixed.Body.String(), "do not mix transform") {
		t.Fatalf("mixed body = %s", mixed.Body.String())
	}
}

func TestModelPlacementPatchesFromRequestPreservesFalseAndZeroValues(t *testing.T) {
	req := decodeModelPlacementRequest(t, `{
		"visible": false,
		"transform": {
			"position": {"x": 0, "y": 0, "z": 0},
			"rotation": {"yaw": 0, "pitch": 0, "roll": 0},
			"scale": {"x": 0, "y": 0, "z": 0}
		}
	}`)
	patches, err := modelPlacementPatchesFromRequest([]dto.ModelPlacementRequest{req}, "building_001")
	if err != nil {
		t.Fatalf("modelPlacementPatchesFromRequest returned error: %v", err)
	}
	patch := patches[0]
	if patch.InstanceID != "building_001" {
		t.Fatalf("instance_id = %q, want path id", patch.InstanceID)
	}
	if patch.Visible == nil || *patch.Visible != false {
		t.Fatalf("patch visible = %#v, want explicit false", patch.Visible)
	}
	if patch.Position == nil || patch.Position.X != 0 || patch.Position.Y != 0 || patch.Position.Z != 0 {
		t.Fatalf("patch position = %#v, want zero vector", patch.Position)
	}
	if patch.Rotation == nil || patch.Rotation.Yaw != 0 || patch.Rotation.Pitch != 0 || patch.Rotation.Roll != 0 {
		t.Fatalf("patch rotation = %#v, want zero rotation", patch.Rotation)
	}
	if patch.Scale == nil || patch.Scale.X != 0 || patch.Scale.Y != 0 || patch.Scale.Z != 0 {
		t.Fatalf("patch scale = %#v, want zero vector", patch.Scale)
	}
}

func TestModelPlacementPatchesFromRequestAcceptsFlatZeroValues(t *testing.T) {
	req := decodeModelPlacementRequest(t, `{
		"instance_id": "building_001",
		"position": {"x": 0, "y": 0, "z": 0},
		"rotation": {"yaw": 0, "pitch": 0, "roll": 0},
		"scale": {"x": 0, "y": 0, "z": 0}
	}`)
	patches, err := modelPlacementPatchesFromRequest([]dto.ModelPlacementRequest{req}, "")
	if err != nil {
		t.Fatalf("modelPlacementPatchesFromRequest returned error: %v", err)
	}
	patch := patches[0]
	if patch.Position == nil || patch.Position.X != 0 || patch.Rotation == nil || patch.Rotation.Yaw != 0 || patch.Scale == nil || patch.Scale.X != 0 {
		t.Fatalf("flat zero patch = %#v, want zero position/rotation/scale", patch)
	}
}

func decodeModelPlacementRequest(t *testing.T, raw string) dto.ModelPlacementRequest {
	t.Helper()
	var req dto.ModelPlacementRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	return req
}

func TestUEConfigHandlerDeleteModelPlacementsAndBootstrap(t *testing.T) {
	r := newUEConfigHandlerTestRouter()

	del := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/ue/config/model-config/placements/delete", strings.NewReader(`{"instance_ids":["building_001","missing"]}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(del, req)
	if del.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want 200, body=%s", del.Code, del.Body.String())
	}
	if !strings.Contains(del.Body.String(), `"placements"`) {
		t.Fatalf("delete body = %s", del.Body.String())
	}

	boot := httptest.NewRecorder()
	r.ServeHTTP(boot, httptest.NewRequest(http.MethodGet, "/api/ue/config/bootstrap", nil))
	if boot.Code != http.StatusOK {
		t.Fatalf("bootstrap status = %d, want 200, body=%s", boot.Code, boot.Body.String())
	}
	for _, field := range []string{"init_config", "path_config", "chart_config", "chart_binding", "drone_config", "model_table", "model_config", "scene_desired"} {
		if !strings.Contains(boot.Body.String(), `"`+field+`"`) {
			t.Fatalf("bootstrap missing %s: %s", field, boot.Body.String())
		}
	}
}

func TestUEConfigHandlerReplaceRawConfigReturnsBody(t *testing.T) {
	r := newUEConfigHandlerTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/ue/config/chart-config", strings.NewReader(`{"version":2,"charts":[{"chart_id":"raw"}]}`))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	if strings.TrimSpace(w.Body.String()) != `{"version":2,"charts":[{"chart_id":"raw"}]}` {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestUEConfigHandlerReplaceRawConfigBodyLimit(t *testing.T) {
	r := newUEConfigHandlerTestRouter(WithMaxRawConfigBodyBytes(16))

	large := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/ue/config/chart-config", strings.NewReader(`{"version":123456789}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(large, req)
	if large.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("large body status = %d, want 413, body=%s", large.Code, large.Body.String())
	}
	if !strings.Contains(large.Body.String(), "request body too large") {
		t.Fatalf("large body response = %s", large.Body.String())
	}

	small := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/ue/config/chart-config", strings.NewReader(`{"v":1}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(small, req)
	if small.Code != http.StatusOK {
		t.Fatalf("small body status = %d, want 200, body=%s", small.Code, small.Body.String())
	}

	invalid := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/ue/config/chart-config", strings.NewReader(`{`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(invalid, req)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid body status = %d, want 400, body=%s", invalid.Code, invalid.Body.String())
	}
}
