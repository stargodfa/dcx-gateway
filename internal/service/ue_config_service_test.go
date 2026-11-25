package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"dcx-gateway/internal/cache"
	"dcx-gateway/internal/model/entity"
	"dcx-gateway/internal/model/protocol/ue"
	"dcx-gateway/internal/repository"
)

type fakeScriptRepo struct {
	scripts []entity.Script
	err     error
}

func (r fakeScriptRepo) FindByScriptID(ctx context.Context, scriptID string) (*entity.Script, error) {
	if r.err != nil {
		return nil, r.err
	}
	for _, script := range r.scripts {
		if script.ScriptID == scriptID {
			return &script, nil
		}
	}
	return nil, errors.New("not found")
}

func (r fakeScriptRepo) ListEnabled(ctx context.Context) ([]entity.Script, error) {
	return r.scripts, r.err
}

type fakeScriptModuleRepo struct {
	modules []entity.ScriptModule
	err     error
}

func (r fakeScriptModuleRepo) ListByScriptID(ctx context.Context, scriptID string) ([]entity.ScriptModule, error) {
	if r.err != nil {
		return nil, r.err
	}
	out := make([]entity.ScriptModule, 0)
	for _, module := range r.modules {
		if module.ScriptID == scriptID {
			out = append(out, module)
		}
	}
	return out, nil
}

func (r fakeScriptModuleRepo) ListEnabled(ctx context.Context) ([]entity.ScriptModule, error) {
	return r.modules, r.err
}

type fakeDataBindingRepo struct {
	bindings []entity.DataBinding
	err      error
}

func (r fakeDataBindingRepo) FindByDataKey(ctx context.Context, dataKey string) (*entity.DataBinding, error) {
	if r.err != nil {
		return nil, r.err
	}
	for _, binding := range r.bindings {
		if binding.DataKey == dataKey {
			return &binding, nil
		}
	}
	return nil, errors.New("not found")
}

func (r fakeDataBindingRepo) ListByDataKeys(ctx context.Context, dataKeys []string) ([]entity.DataBinding, error) {
	if r.err != nil {
		return nil, r.err
	}
	allowed := make(map[string]struct{}, len(dataKeys))
	for _, key := range dataKeys {
		allowed[key] = struct{}{}
	}
	out := make([]entity.DataBinding, 0)
	for _, binding := range r.bindings {
		if _, ok := allowed[binding.DataKey]; ok {
			out = append(out, binding)
		}
	}
	return out, nil
}

func (r fakeDataBindingRepo) ListEnabled(ctx context.Context) ([]entity.DataBinding, error) {
	return r.bindings, r.err
}

type fakeVideoSourceRepo struct {
	videos []entity.VideoSource
	err    error
}

func (r fakeVideoSourceRepo) FindByVideoID(ctx context.Context, videoID string) (*entity.VideoSource, error) {
	if r.err != nil {
		return nil, r.err
	}
	for _, video := range r.videos {
		if video.VideoID == videoID {
			return &video, nil
		}
	}
	return nil, errors.New("not found")
}

func (r fakeVideoSourceRepo) ListEnabled(ctx context.Context) ([]entity.VideoSource, error) {
	return r.videos, r.err
}

func (r fakeVideoSourceRepo) ListByVideoIDs(ctx context.Context, videoIDs []string) ([]entity.VideoSource, error) {
	if r.err != nil {
		return nil, r.err
	}
	allowed := make(map[string]struct{}, len(videoIDs))
	for _, id := range videoIDs {
		allowed[id] = struct{}{}
	}
	out := make([]entity.VideoSource, 0)
	for _, video := range r.videos {
		if _, ok := allowed[video.VideoID]; ok {
			out = append(out, video)
		}
	}
	return out, nil
}

type fakeRawConfigRepo struct {
	docs []entity.UEConfigDocument
	err  error
}

type memoryRawDocumentRepo struct {
	docs map[string]entity.UEConfigDocument
}

type fakeChartConfigRepo struct {
	settings *entity.UEChartConfigSettings
	charts   []entity.UEChartConfig
	err      error
}

func (r fakeChartConfigRepo) GetSettings(ctx context.Context) (*entity.UEChartConfigSettings, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.settings == nil {
		return nil, repository.ErrNotFound
	}
	return r.settings, nil
}

func (r fakeChartConfigRepo) ListEnabledCharts(ctx context.Context) ([]entity.UEChartConfig, error) {
	return r.charts, r.err
}

func (r fakeChartConfigRepo) FindByChartID(ctx context.Context, chartID string) (*entity.UEChartConfig, error) {
	if r.err != nil {
		return nil, r.err
	}
	for _, chart := range r.charts {
		if chart.ChartID == chartID {
			return &chart, nil
		}
	}
	return nil, repository.ErrNotFound
}

type fakeChartRuntimeCache struct {
	byBinding map[string]cache.ChartRuntimeData
	byChart   map[string]cache.ChartRuntimeData
	err       error
}

func (c *fakeChartRuntimeCache) SetChartRuntime(ctx context.Context, data cache.ChartRuntimeData, ttl time.Duration) error {
	if c.err != nil {
		return c.err
	}
	if c.byBinding == nil {
		c.byBinding = make(map[string]cache.ChartRuntimeData)
	}
	if c.byChart == nil {
		c.byChart = make(map[string]cache.ChartRuntimeData)
	}
	c.byBinding[data.BindingID] = data
	c.byChart[data.ChartID] = data
	return nil
}

func (c *fakeChartRuntimeCache) GetByBindingID(ctx context.Context, bindingID string) (*cache.ChartRuntimeData, error) {
	if c.err != nil {
		return nil, c.err
	}
	if data, ok := c.byBinding[bindingID]; ok {
		return &data, nil
	}
	return nil, nil
}

func (c *fakeChartRuntimeCache) GetByChartID(ctx context.Context, chartID string) (*cache.ChartRuntimeData, error) {
	if c.err != nil {
		return nil, c.err
	}
	if data, ok := c.byChart[chartID]; ok {
		return &data, nil
	}
	return nil, nil
}

func (r fakeRawConfigRepo) FindByKey(ctx context.Context, key string) (*entity.UEConfigDocument, error) {
	if r.err != nil {
		return nil, r.err
	}
	for _, doc := range r.docs {
		if doc.ConfigKey == key && doc.Enabled {
			return &doc, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r fakeRawConfigRepo) ListEnabled(ctx context.Context) ([]entity.UEConfigDocument, error) {
	if r.err != nil {
		return nil, r.err
	}
	out := make([]entity.UEConfigDocument, 0)
	for _, doc := range r.docs {
		if doc.Enabled {
			out = append(out, doc)
		}
	}
	return out, nil
}

func (r fakeRawConfigRepo) UpdateContentJSONAtomic(ctx context.Context, key string, mutate func(raw []byte) ([]byte, error)) ([]byte, error) {
	if r.err != nil {
		return nil, r.err
	}
	for _, doc := range r.docs {
		if doc.ConfigKey == key && doc.Enabled {
			return mutate([]byte(doc.ContentJSON))
		}
	}
	return nil, repository.ErrNotFound
}

func (r fakeRawConfigRepo) ReplaceContentJSONAtomic(ctx context.Context, key string, fileName string, raw []byte) ([]byte, error) {
	if r.err != nil {
		return nil, r.err
	}
	return raw, nil
}

func (r *memoryRawDocumentRepo) FindByKey(ctx context.Context, key string) (*entity.UEConfigDocument, error) {
	if doc, ok := r.docs[key]; ok && doc.Enabled {
		return &doc, nil
	}
	return nil, repository.ErrNotFound
}

func (r *memoryRawDocumentRepo) ListEnabled(ctx context.Context) ([]entity.UEConfigDocument, error) {
	out := make([]entity.UEConfigDocument, 0, len(r.docs))
	for _, doc := range r.docs {
		if doc.Enabled {
			out = append(out, doc)
		}
	}
	return out, nil
}

func (r *memoryRawDocumentRepo) UpdateContentJSONAtomic(ctx context.Context, key string, mutate func(raw []byte) ([]byte, error)) ([]byte, error) {
	doc, ok := r.docs[key]
	if !ok || !doc.Enabled {
		return nil, repository.ErrNotFound
	}
	next, err := mutate([]byte(doc.ContentJSON))
	if err != nil {
		return nil, err
	}
	doc.ContentJSON = string(next)
	r.docs[key] = doc
	return next, nil
}

func (r *memoryRawDocumentRepo) ReplaceContentJSONAtomic(ctx context.Context, key string, fileName string, raw []byte) ([]byte, error) {
	if r.docs == nil {
		r.docs = make(map[string]entity.UEConfigDocument)
	}
	r.docs[key] = entity.UEConfigDocument{
		ConfigKey:   key,
		FileName:    fileName,
		ContentJSON: string(raw),
		Source:      "gateway_preset",
		Enabled:     true,
		Version:     "v1",
	}
	return raw, nil
}

func TestUEConfigServiceBuildChartBinding(t *testing.T) {
	svc := NewUEConfigService(
		fakeScriptRepo{},
		fakeScriptModuleRepo{},
		fakeDataBindingRepo{bindings: []entity.DataBinding{
			{DataKey: "security.access.today", BindingID: "sim_3", ChartID: "chart_h", Enabled: true},
			{DataKey: "security.fire.status", BindingID: "sim_4", ChartID: "chart_l", Enabled: true},
			{DataKey: "invalid.empty.chart", BindingID: "invalid.empty.chart", Enabled: true},
		}},
		fakeVideoSourceRepo{},
		fakeRawConfigRepo{},
		nil,
		nil,
		nil,
	)

	cfg, err := svc.BuildChartBinding(context.Background())
	if err != nil {
		t.Fatalf("BuildChartBinding returned error: %v", err)
	}
	if cfg.Version != 1 {
		t.Fatalf("Version = %d, want 1", cfg.Version)
	}
	if len(cfg.Bindings) != 2 {
		t.Fatalf("len(Bindings) = %d, want 2", len(cfg.Bindings))
	}
	if cfg.Bindings[0].BindingID != "sim_3" || cfg.Bindings[0].ChartID != "chart_h" {
		t.Fatalf("first binding = %#v, want sim_3 -> chart_h", cfg.Bindings[0])
	}
	if cfg.Bindings[1].BindingID != "sim_4" || cfg.Bindings[1].ChartID != "chart_l" {
		t.Fatalf("second binding = %#v, want sim_4 -> chart_l", cfg.Bindings[1])
	}
}

func TestUEConfigServiceBuildVideoStreams(t *testing.T) {
	svc := NewUEConfigService(
		fakeScriptRepo{},
		fakeScriptModuleRepo{},
		fakeDataBindingRepo{},
		fakeVideoSourceRepo{videos: []entity.VideoSource{
			{
				VideoID: "vs_cam_65",
				Name:    "gate camera",
				PlayURL: "http://example.test/stream.html?src=cam_65",
				Enabled: true,
			},
		}},
		fakeRawConfigRepo{},
		nil,
		nil,
		nil,
	)

	cfg, err := svc.BuildVideoStreams(context.Background())
	if err != nil {
		t.Fatalf("BuildVideoStreams returned error: %v", err)
	}
	if cfg.Version != 1 {
		t.Fatalf("Version = %d, want 1", cfg.Version)
	}
	if len(cfg.Streams) != 1 {
		t.Fatalf("len(Streams) = %d, want 1", len(cfg.Streams))
	}
	stream := cfg.Streams[0]
	if stream.StreamID != "vs_cam_65" {
		t.Fatalf("StreamID = %q, want vs_cam_65", stream.StreamID)
	}
	if stream.Name != "gate camera" {
		t.Fatalf("Name = %q, want gate camera", stream.Name)
	}
	if stream.WebRTCPlayerURL != "http://example.test/stream.html?src=cam_65" {
		t.Fatalf("WebRTCPlayerURL = %q", stream.WebRTCPlayerURL)
	}
}

func TestUEConfigServiceBuildReportScriptsLightWarmupUsesUnifiedIDs(t *testing.T) {
	svc := NewUEConfigService(
		fakeScriptRepo{scripts: []entity.Script{
			{ScriptID: "security_main", Name: "安防汇报脚本", SceneID: "campus_security", Status: "enabled"},
			{ScriptID: "energy_main", Name: "后勤能耗汇报脚本", SceneID: "campus_energy", Status: "enabled"},
			{ScriptID: "education_main", Name: "教学汇报脚本", SceneID: "campus_teaching", Status: "enabled"},
		}},
		fakeScriptModuleRepo{modules: []entity.ScriptModule{
			{ModuleID: "today_access_card", ScriptID: "security_main", DataKey: "security.access.today", TargetKind: "chart", TargetID: "chart_h", Enabled: true},
			{ModuleID: "fire_alert_card", ScriptID: "security_main", DataKey: "security.fire.status", TargetKind: "chart", TargetID: "chart_l", Enabled: true},
			{ModuleID: "gate_video_panel", ScriptID: "security_main", DataKey: "security.video.gate_01", TargetKind: "video", TargetID: "vs_cam_65", Enabled: true},
			{ModuleID: "energy_total_card", ScriptID: "energy_main", DataKey: "energy.total.today", TargetKind: "chart", TargetID: "chart_energy_total", Enabled: true},
			{ModuleID: "energy_building_table", ScriptID: "energy_main", DataKey: "energy.building.snapshot", TargetKind: "chart", TargetID: "chart_energy_building", Enabled: true},
			{ModuleID: "classroom_usage_card", ScriptID: "education_main", DataKey: "education.classroom.usage", TargetKind: "chart", TargetID: "chart_education_usage", Enabled: true},
			{ModuleID: "classroom_schedule_table", ScriptID: "education_main", DataKey: "education.classroom.schedule", TargetKind: "chart", TargetID: "chart_education_schedule", Enabled: true},
			{ModuleID: "unbound_chart", ScriptID: "security_main", DataKey: "security.unbound", TargetKind: "chart", TargetID: "chart_unbound", Enabled: true},
		}},
		fakeDataBindingRepo{bindings: []entity.DataBinding{
			{DataKey: "security.access.today", BindingID: "sim_3", ChartID: "chart_h", ModuleID: "today_access_card", Enabled: true},
			{DataKey: "security.fire.status", BindingID: "sim_4", ChartID: "chart_l", ModuleID: "fire_alert_card", Enabled: true},
			{DataKey: "energy.total.today", BindingID: "energy.total.today", ChartID: "chart_energy_total", ModuleID: "energy_total_card", Enabled: true},
			{DataKey: "energy.building.snapshot", BindingID: "energy.building.snapshot", ChartID: "chart_energy_building", ModuleID: "energy_building_table", Enabled: true},
			{DataKey: "education.classroom.usage", BindingID: "education.classroom.usage", ChartID: "chart_education_usage", ModuleID: "classroom_usage_card", Enabled: true},
			{DataKey: "education.classroom.schedule", BindingID: "education.classroom.schedule", ChartID: "chart_education_schedule", ModuleID: "classroom_schedule_table", Enabled: true},
		}},
		fakeVideoSourceRepo{videos: []entity.VideoSource{{VideoID: "vs_cam_65", Name: "gate camera", Enabled: true}}},
		fakeRawConfigRepo{},
		nil,
		nil,
		nil,
	)

	cfg, err := svc.BuildReportScripts(context.Background())
	if err != nil {
		t.Fatalf("BuildReportScripts returned error: %v", err)
	}

	security := findLightReportScriptForTest(t, cfg, "security_main")
	if len(security.Chapters) != 1 || len(security.Chapters[0].Scenes) != 1 {
		t.Fatalf("security light shape = %#v", security)
	}
	securityWarmup := security.Chapters[0].Scenes[0].Warmup
	assertStringSlice(t, securityWarmup.Data, []string{"security_main"})
	assertStringSlice(t, securityWarmup.Charts, []string{"sim_3", "sim_4"})
	assertStringSlice(t, securityWarmup.Videos, []string{"vs_cam_65"})
	for _, forbidden := range []string{"chart_h", "chart_l", "chart_unbound"} {
		if sliceContains(securityWarmup.Charts, forbidden) {
			t.Fatalf("security warmup charts = %#v, should not contain %s", securityWarmup.Charts, forbidden)
		}
	}

	energyWarmup := findLightReportScriptForTest(t, cfg, "energy_main").Chapters[0].Scenes[0].Warmup
	assertStringSlice(t, energyWarmup.Data, []string{"energy_main"})
	assertStringSlice(t, energyWarmup.Charts, []string{"energy.building.snapshot", "energy.total.today"})

	educationWarmup := findLightReportScriptForTest(t, cfg, "education_main").Chapters[0].Scenes[0].Warmup
	assertStringSlice(t, educationWarmup.Data, []string{"education_main"})
	assertStringSlice(t, educationWarmup.Charts, []string{"education.classroom.schedule", "education.classroom.usage"})
}

func TestUEConfigServiceGetRawConfig(t *testing.T) {
	svc := NewUEConfigService(
		fakeScriptRepo{},
		fakeScriptModuleRepo{},
		fakeDataBindingRepo{},
		fakeVideoSourceRepo{},
		fakeRawConfigRepo{docs: []entity.UEConfigDocument{
			{
				ConfigKey:   "chart_config",
				FileName:    "chart_config.json",
				ContentJSON: `{"version":1,"charts":[]}`,
				Enabled:     true,
				Version:     "v1",
			},
			{
				ConfigKey:   "disabled_config",
				FileName:    "disabled.json",
				ContentJSON: `{"version":1}`,
				Enabled:     false,
				Version:     "v1",
			},
		}},
		nil,
		nil,
		nil,
	)

	raw, err := svc.GetRawConfig(context.Background(), "chart_config")
	if err != nil {
		t.Fatalf("GetRawConfig returned error: %v", err)
	}
	if string(raw) != `{"version":1,"charts":[]}` {
		t.Fatalf("raw = %s", raw)
	}

	if _, err := svc.GetRawConfig(context.Background(), "missing"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("missing err = %v, want ErrNotFound", err)
	}
	if _, err := svc.GetRawConfig(context.Background(), "disabled_config"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("disabled err = %v, want ErrNotFound", err)
	}
}

func TestUEConfigServiceListRawConfigs(t *testing.T) {
	svc := NewUEConfigService(
		fakeScriptRepo{},
		fakeScriptModuleRepo{},
		fakeDataBindingRepo{},
		fakeVideoSourceRepo{},
		fakeRawConfigRepo{docs: []entity.UEConfigDocument{
			{
				ConfigKey: "chart_config",
				FileName:  "chart_config.json",
				Enabled:   true,
				Version:   "v1",
				Source:    "ue_config",
			},
			{
				ConfigKey: "disabled_config",
				FileName:  "disabled.json",
				Enabled:   false,
				Version:   "v1",
			},
		}},
		nil,
		nil,
		nil,
	)

	index, err := svc.ListRawConfigs(context.Background())
	if err != nil {
		t.Fatalf("ListRawConfigs returned error: %v", err)
	}
	if len(index.Configs) != 1 {
		t.Fatalf("len(Configs) = %d, want 1", len(index.Configs))
	}
	if index.Configs[0].ConfigKey != "chart_config" || index.Configs[0].FileName != "chart_config.json" {
		t.Fatalf("index config = %#v", index.Configs[0])
	}
}

func TestUEConfigServiceReplaceRawConfigWholeDocumentAndCreatesMissing(t *testing.T) {
	rawRepo := &memoryRawDocumentRepo{docs: map[string]entity.UEConfigDocument{}}
	svc := NewUEConfigService(
		fakeScriptRepo{},
		fakeScriptModuleRepo{},
		fakeDataBindingRepo{},
		fakeVideoSourceRepo{},
		rawRepo,
		nil,
		nil,
		nil,
	)

	body := []byte(`{"version":2,"plans_by_id":{"p1":{"path_id":"p1"}}}`)
	raw, err := svc.ReplaceRawConfig(context.Background(), "path-config", body)
	if err != nil {
		t.Fatalf("ReplaceRawConfig returned error: %v", err)
	}
	if string(raw) != string(body) {
		t.Fatalf("raw = %s", raw)
	}
	got, err := svc.GetRawConfig(context.Background(), "path_config")
	if err != nil {
		t.Fatalf("GetRawConfig returned error: %v", err)
	}
	if string(got) != string(body) {
		t.Fatalf("stored raw = %s", got)
	}
	if rawRepo.docs["path_config"].FileName != "path_config.json" {
		t.Fatalf("file name = %q", rawRepo.docs["path_config"].FileName)
	}

	if _, err := svc.ReplaceRawConfig(context.Background(), "raw/any", []byte(`{"version":1}`)); !errors.Is(err, ErrInvalidModelConfigPatch) {
		t.Fatalf("arbitrary key err = %v, want invalid patch", err)
	}
	if _, err := svc.ReplaceRawConfig(context.Background(), "path-config", []byte(`[]`)); !errors.Is(err, ErrInvalidModelConfigPatch) {
		t.Fatalf("array body err = %v, want invalid patch", err)
	}
	if _, err := svc.ReplaceRawConfig(context.Background(), "path-config", []byte(`{`)); !errors.Is(err, ErrInvalidModelConfigPatch) {
		t.Fatalf("invalid json err = %v, want invalid patch", err)
	}
}

func TestUEConfigServiceReplaceChartRawDoesNotAffectStructuredRuntimeOrGeneratedBinding(t *testing.T) {
	rawRepo := &memoryRawDocumentRepo{docs: map[string]entity.UEConfigDocument{
		"chart_config":  {ConfigKey: "chart_config", FileName: "chart_config.json", ContentJSON: `{"version":1,"charts":[]}`, Enabled: true},
		"chart_binding": {ConfigKey: "chart_binding", FileName: "chart_binding.json", ContentJSON: `{"version":1,"bindings":[]}`, Enabled: true},
	}}
	chartRepo := fakeChartConfigRepo{
		settings: &entity.UEChartConfigSettings{Version: 1, OverlayHUDMaxPanelWidth: 920, Enabled: true},
		charts: []entity.UEChartConfig{
			{ChartID: "chart_structured", ChartName: "structured", Visible: true, Applied: true},
		},
	}
	svc := NewUEConfigService(
		fakeScriptRepo{},
		fakeScriptModuleRepo{},
		fakeDataBindingRepo{bindings: []entity.DataBinding{{BindingID: "binding_generated", ChartID: "chart_generated", Enabled: true}}},
		fakeVideoSourceRepo{},
		rawRepo,
		chartRepo,
		nil,
		nil,
	)

	if _, err := svc.ReplaceRawConfig(context.Background(), "chart-config", []byte(`{"version":9,"charts":[{"chart_id":"raw_only"}]}`)); err != nil {
		t.Fatalf("ReplaceRawConfig chart-config returned error: %v", err)
	}
	raw, err := svc.GetRawConfig(context.Background(), "chart_config")
	if err != nil {
		t.Fatalf("GetRawConfig chart_config returned error: %v", err)
	}
	if !json.Valid(raw) || !containsString(string(raw), "raw_only") {
		t.Fatalf("raw chart_config = %s", raw)
	}
	structured, err := svc.BuildStructuredChartConfig(context.Background())
	if err != nil {
		t.Fatalf("BuildStructuredChartConfig returned error: %v", err)
	}
	if structured.Charts[0].ChartID != "chart_structured" {
		t.Fatalf("structured chart was polluted by raw PUT: %#v", structured.Charts)
	}
	runtime, err := svc.BuildRuntimeChartConfig(context.Background())
	if err != nil {
		t.Fatalf("BuildRuntimeChartConfig returned error: %v", err)
	}
	if runtime.Charts[0].ChartID != "chart_structured" {
		t.Fatalf("runtime chart was polluted by raw PUT: %#v", runtime.Charts)
	}

	if _, err := svc.ReplaceRawConfig(context.Background(), "chart-binding", []byte(`{"version":9,"bindings":[{"binding_id":"raw_only","chart_id":"raw_chart"}]}`)); err != nil {
		t.Fatalf("ReplaceRawConfig chart-binding returned error: %v", err)
	}
	generated, err := svc.BuildChartBinding(context.Background())
	if err != nil {
		t.Fatalf("BuildChartBinding returned error: %v", err)
	}
	if generated.Bindings[0].BindingID != "binding_generated" {
		t.Fatalf("generated chart binding was polluted by raw PUT: %#v", generated.Bindings)
	}
}

func TestUEConfigServiceReplaceRawConfigAllPresetKeys(t *testing.T) {
	rawRepo := &memoryRawDocumentRepo{docs: map[string]entity.UEConfigDocument{}}
	svc := NewUEConfigService(
		fakeScriptRepo{},
		fakeScriptModuleRepo{},
		fakeDataBindingRepo{},
		fakeVideoSourceRepo{},
		rawRepo,
		nil,
		nil,
		nil,
	)

	for _, routeKey := range []string{"path-config", "chart-config", "chart-binding", "drone-config", "render-config", "video-streams", "report-scripts"} {
		if _, err := svc.ReplaceRawConfig(context.Background(), routeKey, []byte(`{"version":1}`)); err != nil {
			t.Fatalf("ReplaceRawConfig(%s) returned error: %v", routeKey, err)
		}
	}
}

func TestUEConfigServiceBuildBootstrapUsesRawAndGeneratedSources(t *testing.T) {
	rawRepo := &memoryRawDocumentRepo{docs: map[string]entity.UEConfigDocument{
		"init_config":  {ConfigKey: "init_config", ContentJSON: `{"version":1,"instance_ids":[]}`, Enabled: true},
		"path_config":  {ConfigKey: "path_config", ContentJSON: `{"version":1,"plans_by_id":{}}`, Enabled: true},
		"chart_config": {ConfigKey: "chart_config", ContentJSON: `{"version":1,"charts":[]}`, Enabled: true},
		"drone_config": {ConfigKey: "drone_config", ContentJSON: `{"version":1}`, Enabled: true},
	}}
	repos := newMemoryModelRepos()
	repos.types["room"] = entity.UEModelType{ModelID: "room", URI: "file://room", Format: "udatasmith", SourcePath: "./room", BaseBoundsX: 1, BaseBoundsY: 2, BaseBoundsZ: 3, Enabled: true}
	repos.instances["room_1"] = entity.UEModelInstance{InstanceID: "room_1", ModelID: "room", DisplayName: "room", Enabled: true}
	repos.placements["room_1"] = entity.UEModelInstancePlacement{InstanceID: "room_1", Visible: true, ScaleX: 1, ScaleY: 1, ScaleZ: 1}
	svc := NewUEConfigService(
		fakeScriptRepo{},
		fakeScriptModuleRepo{},
		fakeDataBindingRepo{bindings: []entity.DataBinding{{BindingID: "binding_generated", ChartID: "chart_generated", Enabled: true}}},
		fakeVideoSourceRepo{},
		rawRepo,
		nil,
		nil,
		nil,
		UEModelConfigRepositories{
			ModelType:     &memoryModelTypeRepo{store: repos},
			ModelInstance: &memoryModelInstanceRepo{store: repos},
			Placement:     &memoryModelPlacementRepo{store: repos},
		},
	)

	bootstrap, err := svc.BuildBootstrap(context.Background())
	if err != nil {
		t.Fatalf("BuildBootstrap returned error: %v", err)
	}
	if bootstrap.Version != 1 || bootstrap.ModelTable == nil || bootstrap.ModelConfig == nil || bootstrap.SceneDesired == nil {
		t.Fatalf("bootstrap missing generated fields: %#v", bootstrap)
	}
	if _, ok := bootstrap.ChartBinding.(*ue.ChartBinding); !ok {
		t.Fatalf("chart_binding should fallback to generated binding when raw is missing: %#v", bootstrap.ChartBinding)
	}
	encoded, err := json.Marshal(bootstrap)
	if err != nil {
		t.Fatalf("marshal bootstrap returned error: %v", err)
	}
	for _, field := range []string{"init_config", "path_config", "chart_config", "chart_binding", "drone_config", "model_table", "model_config", "scene_desired"} {
		if !containsString(string(encoded), `"`+field+`"`) {
			t.Fatalf("bootstrap missing snake_case field %s: %s", field, encoded)
		}
	}

	delete(rawRepo.docs, "path_config")
	if _, err := svc.BuildBootstrap(context.Background()); err == nil {
		t.Fatalf("BuildBootstrap should fail when required raw config is missing")
	}
}

func TestUEConfigServiceBuildStructuredChartConfig(t *testing.T) {
	svc := NewUEConfigService(
		fakeScriptRepo{},
		fakeScriptModuleRepo{},
		fakeDataBindingRepo{},
		fakeVideoSourceRepo{},
		fakeRawConfigRepo{},
		fakeChartConfigRepo{
			settings: &entity.UEChartConfigSettings{Version: 1, OverlayHUDMaxPanelWidth: 920, Enabled: true},
			charts: []entity.UEChartConfig{
				{
					ChartID:               "chart_h",
					ChartName:             "今日出入人数",
					Visible:               true,
					Applied:               true,
					LayoutMode:            stringPtr("column_pairs"),
					RowsCount:             intPtr(4),
					ColsCount:             intPtr(3),
					ColFieldsJSON:         `["指标","数值","单位"]`,
					PairLabelColsJSON:     `[0]`,
					DefaultCellValuesJSON: `[["指标","数值","单位"],["今日进入","","人次"]]`,
				},
				{
					ChartID:   "chart_l",
					ChartName: "消防告警状态",
					Visible:   true,
					Applied:   true,
				},
			},
		},
		nil,
		nil,
	)

	cfg, err := svc.BuildStructuredChartConfig(context.Background())
	if err != nil {
		t.Fatalf("BuildStructuredChartConfig returned error: %v", err)
	}
	if cfg.OverlayHUDMaxPanelWidth != 920 {
		t.Fatalf("OverlayHUDMaxPanelWidth = %d, want 920", cfg.OverlayHUDMaxPanelWidth)
	}
	if len(cfg.Charts) != 2 {
		t.Fatalf("len(Charts) = %d, want 2", len(cfg.Charts))
	}
	if cfg.Charts[0].ChartID != "chart_h" || cfg.Charts[1].ChartID != "chart_l" {
		t.Fatalf("chart ids = %#v", cfg.Charts)
	}
	if cfg.Charts[0].CellValues[1][0] != "今日进入" {
		t.Fatalf("default cell values not applied: %#v", cfg.Charts[0].CellValues)
	}
}

func containsString(s string, substr string) bool {
	return strings.Contains(s, substr)
}

func findLightReportScriptForTest(t *testing.T, cfg *ue.LightReportScripts, scriptID string) ue.LightReportScript {
	t.Helper()
	if cfg == nil {
		t.Fatal("light report scripts is nil")
	}
	for _, script := range cfg.Scripts {
		if script.ScriptID == scriptID {
			return script
		}
	}
	t.Fatalf("script %s not found in %#v", scriptID, cfg.Scripts)
	return ue.LightReportScript{}
}

func sliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestUEConfigServiceBuildRuntimeChartConfigOverridesDefaults(t *testing.T) {
	runtime := &fakeChartRuntimeCache{
		byChart: map[string]cache.ChartRuntimeData{
			"chart_h": {
				ChartID:    "chart_h",
				BindingID:  "sim_3",
				CellValues: [][]string{{"指标", "数值"}, {"今日进入", "1800"}},
			},
		},
	}
	svc := NewUEConfigService(
		fakeScriptRepo{},
		fakeScriptModuleRepo{},
		fakeDataBindingRepo{},
		fakeVideoSourceRepo{},
		fakeRawConfigRepo{},
		fakeChartConfigRepo{
			settings: &entity.UEChartConfigSettings{Version: 1, OverlayHUDMaxPanelWidth: 920, Enabled: true},
			charts: []entity.UEChartConfig{
				{
					ChartID:               "chart_h",
					ChartName:             "今日出入人数",
					Visible:               true,
					Applied:               true,
					RowsCount:             intPtr(2),
					ColsCount:             intPtr(2),
					DefaultCellValuesJSON: `[["指标","数值"],["今日进入",""]]`,
				},
				{
					ChartID:               "chart_l",
					ChartName:             "消防告警状态",
					Visible:               true,
					Applied:               true,
					DefaultCellValuesJSON: `[["等级","位置","信息"],["normal","","无告警"]]`,
				},
				{
					ChartID:   "chart_empty",
					ChartName: "空表",
					Visible:   true,
					Applied:   true,
					RowsCount: intPtr(2),
					ColsCount: intPtr(3),
				},
			},
		},
		nil,
		runtime,
	)

	cfg, err := svc.BuildRuntimeChartConfig(context.Background())
	if err != nil {
		t.Fatalf("BuildRuntimeChartConfig returned error: %v", err)
	}
	if cfg.Charts[0].CellValues[1][1] != "1800" {
		t.Fatalf("runtime cell values did not override default: %#v", cfg.Charts[0].CellValues)
	}
	if cfg.Charts[1].CellValues[1][2] != "无告警" {
		t.Fatalf("default cell values should remain without runtime: %#v", cfg.Charts[1].CellValues)
	}
	if len(cfg.Charts[2].CellValues) != 2 || len(cfg.Charts[2].CellValues[0]) != 3 {
		t.Fatalf("empty matrix = %#v, want 2x3", cfg.Charts[2].CellValues)
	}
}

func stringPtr(v string) *string {
	return &v
}

func intPtr(v int) *int {
	return &v
}
