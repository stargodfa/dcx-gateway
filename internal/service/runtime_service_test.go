package service

import (
	"context"
	"errors"
	"testing"

	"dcx-gateway/internal/model/entity"
	gateway "dcx-gateway/internal/model/protocol/gateway"
)

func TestRuntimeServiceBuildWarmupOptions(t *testing.T) {
	modelRepos := newMemoryModelRepos()
	modelRepos.instances["test_model_8"] = entity.UEModelInstance{
		InstanceID:  "test_model_8",
		ModelID:     "test_model",
		DisplayName: "测试模型 8",
		Enabled:     true,
	}
	modelRepos.instances["disabled_model"] = entity.UEModelInstance{
		InstanceID: "disabled_model",
		ModelID:    "test_model",
		Enabled:    false,
	}
	svc := NewRuntimeService(
		fakeScriptRepo{scripts: []entity.Script{{ScriptID: "security_main", Name: "安防汇报脚本", SceneID: "campus_security", Status: "enabled"}}},
		fakeScriptModuleRepo{modules: []entity.ScriptModule{
			{ModuleID: "today_access_card", ScriptID: "security_main", Title: "今日出入人数", DataKey: "security.access.today", TargetKind: "chart", TargetID: "chart_h", Enabled: true},
			{ModuleID: "gate_video_panel", ScriptID: "security_main", Title: "校门口监控", DataKey: "security.video.gate_01", TargetKind: "video", TargetID: "vs_cam_65", Enabled: true},
		}},
		fakeDataBindingRepo{bindings: []entity.DataBinding{{DataKey: "security.access.today", BindingID: "sim_3", ChartID: "chart_h", ModuleID: "today_access_card", Enabled: true}}},
		fakeVideoSourceRepo{videos: []entity.VideoSource{{VideoID: "vs_cam_65", Name: "校门口监控", PlayURL: "http://example.test/cam_65", Enabled: true}}},
		&memoryModelInstanceRepo{store: modelRepos},
		&memoryRawDocumentRepo{docs: map[string]entity.UEConfigDocument{}},
		&fakeRuntimeScriptService{},
		nil,
	)

	options, err := svc.BuildWarmupOptions(context.Background())
	if err != nil {
		t.Fatalf("BuildWarmupOptions returned error: %v", err)
	}
	if options.Version != 1 {
		t.Fatalf("version = %d, want 1", options.Version)
	}
	if len(options.DataScripts) != 1 || options.DataScripts[0].DataScriptID != "security_main" || options.DataScripts[0].DataSceneID != "campus_security" {
		t.Fatalf("data_scripts = %#v", options.DataScripts)
	}
	if len(options.DataScripts[0].DataKeys) != 2 || options.DataScripts[0].DataKeys[0] != "security.access.today" {
		t.Fatalf("data keys should be sorted: %#v", options.DataScripts[0].DataKeys)
	}
	if len(options.Charts) != 1 || options.Charts[0].BindingID != "sim_3" || options.Charts[0].ChartID != "chart_h" {
		t.Fatalf("charts = %#v", options.Charts)
	}
	if len(options.Videos) != 1 || options.Videos[0].VideoID != "vs_cam_65" || options.Videos[0].DataScriptID != "security_main" {
		t.Fatalf("videos = %#v", options.Videos)
	}
	if len(options.Instances) != 1 || options.Instances[0].InstanceID != "test_model_8" {
		t.Fatalf("instances = %#v", options.Instances)
	}
}

func TestRuntimeServiceBuildWarmupOptionsEmpty(t *testing.T) {
	modelRepos := newMemoryModelRepos()
	svc := NewRuntimeService(
		fakeScriptRepo{},
		fakeScriptModuleRepo{},
		fakeDataBindingRepo{},
		fakeVideoSourceRepo{},
		&memoryModelInstanceRepo{store: modelRepos},
		&memoryRawDocumentRepo{docs: map[string]entity.UEConfigDocument{}},
		&fakeRuntimeScriptService{},
		nil,
	)
	options, err := svc.BuildWarmupOptions(context.Background())
	if err != nil {
		t.Fatalf("BuildWarmupOptions returned error: %v", err)
	}
	if len(options.DataScripts) != 0 || len(options.Charts) != 0 || len(options.Videos) != 0 || len(options.Instances) != 0 {
		t.Fatalf("options should be empty arrays: %#v", options)
	}
}

func TestRuntimeServicePreloadReportScene(t *testing.T) {
	preloader := &fakeRuntimeScriptService{}
	svc := newReportSceneRuntimeService(preloader, `{
		"version": 1,
		"scripts": [{
			"script_id": "report_main",
			"label": "report",
			"chapters": [{
				"chapter_id": "chapter_a",
				"label": "chapter",
				"scenes": [{
					"scene_id": "scene_1",
					"label": "scene",
					"camera": null,
					"bindings": [],
					"warmup": {
						"data": ["security_main", "energy_main"],
						"videos": ["vs_cam_65"],
						"charts": ["sim_3"]
					}
				}]
			}]
		}]
	}`)

	resp, err := svc.PreloadReportScene(context.Background(), ReportScenePreloadRequest{
		ScriptID:  "report_main",
		ChapterID: "chapter_a",
		SceneID:   "scene_1",
		Force:     true,
		PushWS:    false,
		Source:    "test",
		AuthType:  "static",
		Username:  "operator",
	})
	if err != nil {
		t.Fatalf("PreloadReportScene returned error: %v", err)
	}
	if len(preloader.calls) != 2 {
		t.Fatalf("preload calls = %#v", preloader.calls)
	}
	if preloader.calls[0].scriptID != "security_main" || preloader.calls[0].opts.PushWS || !preloader.calls[0].opts.Force {
		t.Fatalf("first call = %#v", preloader.calls[0])
	}
	if len(resp.Results) != 2 || !resp.Results[0].Success || resp.Results[0].ChartsRequested[0] != "sim_3" || resp.Results[0].VideosRequested[0] != "vs_cam_65" {
		t.Fatalf("response = %#v", resp)
	}
}

func TestRuntimeServicePreloadReportSceneEmptyWarmup(t *testing.T) {
	preloader := &fakeRuntimeScriptService{}
	svc := newReportSceneRuntimeService(preloader, `{"version":1,"scripts":[{"script_id":"report_main","chapters":[{"chapter_id":"chapter_a","scenes":[{"scene_id":"scene_1","label":"scene","camera":null,"bindings":[],"warmup":{"data":[],"videos":[],"charts":[]}}]}]}]}`)
	resp, err := svc.PreloadReportScene(context.Background(), ReportScenePreloadRequest{ScriptID: "report_main", ChapterID: "chapter_a", SceneID: "scene_1", Force: true, PushWS: true})
	if err != nil {
		t.Fatalf("PreloadReportScene returned error: %v", err)
	}
	if len(resp.Results) != 0 || len(preloader.calls) != 0 {
		t.Fatalf("empty warmup should not preload: response=%#v calls=%#v", resp, preloader.calls)
	}
}

func TestRuntimeServicePreloadReportSceneNotFound(t *testing.T) {
	svc := newReportSceneRuntimeService(&fakeRuntimeScriptService{}, `{"version":1,"scripts":[]}`)
	_, err := svc.PreloadReportScene(context.Background(), ReportScenePreloadRequest{ScriptID: "missing", ChapterID: "chapter", SceneID: "scene", Force: true, PushWS: true})
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestRuntimeServicePreloadReportSceneCorruptedJSON(t *testing.T) {
	svc := newReportSceneRuntimeService(&fakeRuntimeScriptService{}, `{bad json`)
	_, err := svc.PreloadReportScene(context.Background(), ReportScenePreloadRequest{ScriptID: "report_main", ChapterID: "chapter", SceneID: "scene", Force: true, PushWS: true})
	if err == nil {
		t.Fatal("expected corrupted JSON error")
	}
}

func TestRuntimeServicePreloadReportScenePartialFailure(t *testing.T) {
	preloader := &fakeRuntimeScriptService{fail: map[string]error{"security_main": errors.New("adapter down")}}
	svc := newReportSceneRuntimeService(preloader, `{"version":1,"scripts":[{"script_id":"report_main","chapters":[{"chapter_id":"chapter_a","scenes":[{"scene_id":"scene_1","warmup":{"data":["security_main","energy_main"],"videos":[],"charts":[]}}]}]}]}`)
	resp, err := svc.PreloadReportScene(context.Background(), ReportScenePreloadRequest{ScriptID: "report_main", ChapterID: "chapter_a", SceneID: "scene_1", Force: true, PushWS: true})
	if err != nil {
		t.Fatalf("PreloadReportScene should return per-item failure, got: %v", err)
	}
	if len(resp.Results) != 2 || resp.Results[0].Success || !resp.Results[1].Success {
		t.Fatalf("partial failure response = %#v", resp.Results)
	}
}

func newReportSceneRuntimeService(preloader ScriptService, raw string) RuntimeService {
	modelRepos := newMemoryModelRepos()
	return NewRuntimeService(
		fakeScriptRepo{},
		fakeScriptModuleRepo{},
		fakeDataBindingRepo{},
		fakeVideoSourceRepo{},
		&memoryModelInstanceRepo{store: modelRepos},
		&memoryRawDocumentRepo{docs: map[string]entity.UEConfigDocument{
			"report_scripts": {ConfigKey: "report_scripts", ContentJSON: raw, Enabled: true},
		}},
		preloader,
		nil,
	)
}

type fakeRuntimeScriptService struct {
	calls []fakeRuntimePreloadCall
	fail  map[string]error
}

type fakeRuntimePreloadCall struct {
	scriptID string
	opts     PreloadOptions
}

func (s *fakeRuntimeScriptService) BuildDataBundle(ctx context.Context, scriptID string, useCache bool) (*gateway.DataBundle, error) {
	return &gateway.DataBundle{ScriptID: scriptID, DataScriptID: scriptID}, nil
}

func (s *fakeRuntimeScriptService) Preload(ctx context.Context, scriptID string, force bool) (*gateway.DataBundle, error) {
	return s.PreloadWithOptions(ctx, scriptID, PreloadOptions{Force: force, PushWS: true})
}

func (s *fakeRuntimeScriptService) PreloadWithOptions(ctx context.Context, scriptID string, opts PreloadOptions) (*gateway.DataBundle, error) {
	s.calls = append(s.calls, fakeRuntimePreloadCall{scriptID: scriptID, opts: opts})
	if s.fail != nil {
		if err := s.fail[scriptID]; err != nil {
			return nil, err
		}
	}
	return &gateway.DataBundle{BundleID: "bundle_" + scriptID, ScriptID: scriptID, DataScriptID: scriptID}, nil
}
