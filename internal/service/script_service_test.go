package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"dcx-gateway/internal/cache"
	"dcx-gateway/internal/model/domain"
	"dcx-gateway/internal/model/entity"
	gateway "dcx-gateway/internal/model/protocol/gateway"
	"dcx-gateway/internal/model/protocol/ue"
	"dcx-gateway/internal/websocket"

	"go.uber.org/zap"
)

type fakeSecurityAdapter struct{}

func (a fakeSecurityAdapter) GetAccessSummary(ctx context.Context) (*domain.AccessSummary, error) {
	return &domain.AccessSummary{
		InCount:    1800,
		OutCount:   1480,
		TotalCount: 3280,
		Timestamp:  "2026-06-02T00:00:00Z",
	}, nil
}

func (a fakeSecurityAdapter) GetFireAlerts(ctx context.Context) ([]domain.FireAlert, error) {
	return []domain.FireAlert{
		{
			AlertID:   "fire_mock_001",
			Level:     "normal",
			Location:  "校门口",
			Message:   "消防状态正常",
			Timestamp: time.Now().Format(time.RFC3339),
		},
	}, nil
}

type fakeEnergyAdapter struct {
	snapshot *domain.EnergySnapshot
	err      error
	calls    *int
}

func (a fakeEnergyAdapter) GetEnergySnapshot(ctx context.Context) (*domain.EnergySnapshot, error) {
	if a.calls != nil {
		*a.calls = *a.calls + 1
	}
	if a.err != nil {
		return nil, a.err
	}
	if a.snapshot != nil {
		return a.snapshot, nil
	}
	return &domain.EnergySnapshot{
		Buildings: []domain.BuildingEnergyMetric{
			{BuildingID: "library", Name: "图书馆", Value: 1280.5, Unit: "kWh", Timestamp: "2026-06-02T00:00:00Z"},
			{BuildingID: "teaching_a", Name: "教学楼A", Value: 2460.0, Unit: "kWh", Timestamp: "2026-06-02T00:00:00Z"},
			{BuildingID: "canteen", Name: "食堂", Value: 980.8, Unit: "kWh", Timestamp: "2026-06-02T00:00:00Z"},
			{BuildingID: "dormitory", Name: "宿舍区", Value: 3120.2, Unit: "kWh", Timestamp: "2026-06-02T00:00:00Z"},
		},
	}, nil
}

type fakeEducationAdapter struct {
	snapshot *domain.EducationSnapshot
	err      error
	calls    *int
}

func (a fakeEducationAdapter) GetClassroomUsage(ctx context.Context) (*domain.EducationSnapshot, error) {
	if a.calls != nil {
		*a.calls = *a.calls + 1
	}
	if a.err != nil {
		return nil, a.err
	}
	if a.snapshot != nil {
		return a.snapshot, nil
	}
	return &domain.EducationSnapshot{
		Classrooms: []domain.ClassroomUsage{
			{ClassroomID: "A101", Name: "A101", Status: "occupied", CourseName: "高等数学", TeacherName: "张老师", Timestamp: "2026-06-02T00:00:00Z"},
			{ClassroomID: "A102", Name: "A102", Status: "free", Timestamp: "2026-06-02T00:00:00Z"},
			{ClassroomID: "B201", Name: "B201", Status: "occupied", CourseName: "大学英语", TeacherName: "李老师", Timestamp: "2026-06-02T00:00:00Z"},
			{ClassroomID: "B305", Name: "B305", Status: "unknown", Timestamp: "2026-06-02T00:00:00Z"},
		},
	}, nil
}

func TestScriptServiceBuildDataBundleWritesChartRuntimeCache(t *testing.T) {
	runtime := &fakeChartRuntimeCache{}
	svc := NewScriptService(
		fakeScriptRepo{scripts: []entity.Script{{ScriptID: "security_main", SceneID: "campus_security", Status: "enabled"}}},
		fakeScriptModuleRepo{modules: []entity.ScriptModule{
			{
				ModuleID: "today_access_card",
				ScriptID: "security_main",
				Type:     "stat_card",
				Title:    "今日出入人数",
				DataKey:  "security.access.today",
				Template: "stat_card_basic",
				Enabled:  true,
			},
		}},
		fakeDataBindingRepo{bindings: []entity.DataBinding{
			{DataKey: "security.access.today", BindingID: "sim_3", ChartID: "chart_h", ModuleID: "today_access_card", Enabled: true},
		}},
		fakeVideoSourceRepo{},
		fakeSecurityAdapter{},
		fakeEnergyAdapter{},
		fakeEducationAdapter{},
		nil,
		nil,
		nil,
		runtime,
		nil,
		false,
		zap.NewNop(),
	)

	bundle, err := svc.BuildDataBundle(context.Background(), "security_main", false)
	if err != nil {
		t.Fatalf("BuildDataBundle returned error: %v", err)
	}
	if len(bundle.Charts) != 1 {
		t.Fatalf("len(Charts) = %d, want 1", len(bundle.Charts))
	}
	cached, err := runtime.GetByChartID(context.Background(), "chart_h")
	if err != nil {
		t.Fatalf("GetByChartID returned error: %v", err)
	}
	if cached == nil {
		t.Fatal("runtime cache missing chart_h")
	}
	if cached.BindingID != "sim_3" || cached.DataKey != "security.access.today" {
		t.Fatalf("cached metadata = %#v", cached)
	}
	if cached.CellValues[1][1] != "1800" {
		t.Fatalf("cached cell value = %#v", cached.CellValues)
	}
}

func TestScriptServiceBuildDataBundleIgnoresChartRuntimeCacheFailure(t *testing.T) {
	runtime := &fakeChartRuntimeCache{err: errors.New("redis down")}
	svc := NewScriptService(
		fakeScriptRepo{scripts: []entity.Script{{ScriptID: "security_main", SceneID: "campus_security", Status: "enabled"}}},
		fakeScriptModuleRepo{modules: []entity.ScriptModule{
			{
				ModuleID: "today_access_card",
				ScriptID: "security_main",
				Type:     "stat_card",
				Title:    "今日出入人数",
				DataKey:  "security.access.today",
				Template: "stat_card_basic",
				Enabled:  true,
			},
		}},
		fakeDataBindingRepo{bindings: []entity.DataBinding{
			{DataKey: "security.access.today", BindingID: "sim_3", ChartID: "chart_h", ModuleID: "today_access_card", Enabled: true},
		}},
		fakeVideoSourceRepo{},
		fakeSecurityAdapter{},
		fakeEnergyAdapter{},
		fakeEducationAdapter{},
		nil,
		nil,
		nil,
		runtime,
		nil,
		false,
		zap.NewNop(),
	)

	bundle, err := svc.BuildDataBundle(context.Background(), "security_main", false)
	if err != nil {
		t.Fatalf("BuildDataBundle should ignore runtime cache failure: %v", err)
	}
	if len(bundle.Charts) != 1 {
		t.Fatalf("len(Charts) = %d, want 1", len(bundle.Charts))
	}
}

func TestScriptServiceBuildSecurityDataBundle(t *testing.T) {
	runtime := &fakeChartRuntimeCache{}
	svc := NewScriptService(
		fakeScriptRepo{scripts: []entity.Script{{ScriptID: "security_main", SceneID: "campus_security", Status: "enabled"}}},
		fakeScriptModuleRepo{modules: []entity.ScriptModule{
			{
				ModuleID:   "today_access_card",
				ScriptID:   "security_main",
				Type:       "stat_card",
				Title:      "今日出入人数",
				DataKey:    "security.access.today",
				Template:   "stat_card_basic",
				TargetKind: "chart",
				TargetID:   "chart_h",
				Enabled:    true,
			},
			{
				ModuleID:   "fire_alert_card",
				ScriptID:   "security_main",
				Type:       "alert_status",
				Title:      "消防告警状态",
				DataKey:    "security.fire.status",
				Template:   "status_basic",
				TargetKind: "chart",
				TargetID:   "chart_l",
				Enabled:    true,
			},
			{
				ModuleID:   "gate_video_panel",
				ScriptID:   "security_main",
				Type:       "video",
				Title:      "校门口监控",
				DataKey:    "security.video.gate_01",
				Template:   "video_basic",
				TargetKind: "video",
				TargetID:   "vs_cam_65",
				Enabled:    true,
			},
		}},
		fakeDataBindingRepo{bindings: []entity.DataBinding{
			{DataKey: "security.access.today", BindingID: "sim_3", ChartID: "chart_h", ModuleID: "today_access_card", Enabled: true},
			{DataKey: "security.fire.status", BindingID: "sim_4", ChartID: "chart_l", ModuleID: "fire_alert_card", Enabled: true},
		}},
		fakeVideoSourceRepo{videos: []entity.VideoSource{
			{VideoID: "vs_cam_65", Name: "校门口监控", PlayURL: "http://example.test/cam_65", SourceType: "mock", BindModelID: "gate_model_01", Enabled: true},
		}},
		fakeSecurityAdapter{},
		fakeEnergyAdapter{},
		fakeEducationAdapter{},
		nil,
		nil,
		nil,
		runtime,
		nil,
		false,
		zap.NewNop(),
	)

	bundle, err := svc.BuildDataBundle(context.Background(), "security_main", false)
	if err != nil {
		t.Fatalf("BuildDataBundle returned error: %v", err)
	}
	if bundle.ScriptID != "security_main" || bundle.SceneID != "campus_security" {
		t.Fatalf("bundle identity = %#v", bundle)
	}
	if bundle.DataScriptID != "security_main" || bundle.DataSceneID != "campus_security" {
		t.Fatalf("bundle data aliases = %#v", bundle)
	}
	if len(bundle.Modules) != 3 || len(bundle.Charts) != 2 || len(bundle.Videos) != 1 {
		t.Fatalf("bundle sizes: modules=%d charts=%d videos=%d", len(bundle.Modules), len(bundle.Charts), len(bundle.Videos))
	}

	access := findModuleData(t, bundle.Modules, "security.access.today")
	if fmt.Sprint(access.Value) != "3280" || access.Unit != "人次" {
		t.Fatalf("access module = %#v", access)
	}
	accessChart := findChartData(t, bundle.Charts, "security.access.today")
	if accessChart.BindingID != "sim_3" || accessChart.ChartID != "chart_h" || accessChart.Rows[1][1] != "1800" {
		t.Fatalf("access chart = %#v", accessChart)
	}

	fire := findModuleData(t, bundle.Modules, "security.fire.status")
	if fire.Value != "normal" || fire.Payload == nil {
		t.Fatalf("fire module = %#v", fire)
	}
	fireChart := findChartData(t, bundle.Charts, "security.fire.status")
	if fireChart.BindingID != "sim_4" || fireChart.ChartID != "chart_l" || fireChart.Rows[1][0] != "normal" {
		t.Fatalf("fire chart = %#v", fireChart)
	}

	videoModule := findModuleData(t, bundle.Modules, "security.video.gate_01")
	if videoModule.TargetKind != "video" || videoModule.Value != "online" {
		t.Fatalf("video module = %#v", videoModule)
	}
	video := bundle.Videos[0]
	if video.VideoID != "vs_cam_65" || video.PlayURL == "" {
		t.Fatalf("video source = %#v", video)
	}
}

func TestScriptServiceBuildEnergyDataBundle(t *testing.T) {
	runtime := &fakeChartRuntimeCache{}
	energyCalls := 0
	svc := NewScriptService(
		fakeScriptRepo{scripts: []entity.Script{{ScriptID: "energy_main", SceneID: "campus_energy", Status: "enabled"}}},
		fakeScriptModuleRepo{modules: []entity.ScriptModule{
			{
				ModuleID:   "energy_total_card",
				ScriptID:   "energy_main",
				Type:       "stat_card",
				Title:      "今日总能耗",
				DataKey:    "energy.total.today",
				Template:   "stat_card_basic",
				TargetKind: "chart",
				TargetID:   "chart_energy_total",
				Enabled:    true,
			},
			{
				ModuleID:   "energy_building_table",
				ScriptID:   "energy_main",
				Type:       "table",
				Title:      "楼栋能耗排行",
				DataKey:    "energy.building.snapshot",
				Template:   "table_basic",
				TargetKind: "chart",
				TargetID:   "chart_energy_building",
				Enabled:    true,
			},
		}},
		fakeDataBindingRepo{bindings: []entity.DataBinding{
			{DataKey: "energy.total.today", BindingID: "energy.total.today", ChartID: "chart_energy_total", ModuleID: "energy_total_card", Enabled: true},
			{DataKey: "energy.building.snapshot", BindingID: "energy.building.snapshot", ChartID: "chart_energy_building", ModuleID: "energy_building_table", Enabled: true},
		}},
		fakeVideoSourceRepo{},
		fakeSecurityAdapter{},
		fakeEnergyAdapter{calls: &energyCalls},
		fakeEducationAdapter{},
		nil,
		nil,
		nil,
		runtime,
		nil,
		false,
		zap.NewNop(),
	)

	bundle, err := svc.BuildDataBundle(context.Background(), "energy_main", false)
	if err != nil {
		t.Fatalf("BuildDataBundle returned error: %v", err)
	}
	if bundle.ScriptID != "energy_main" || bundle.SceneID != "campus_energy" {
		t.Fatalf("bundle identity = %#v", bundle)
	}
	if bundle.DataScriptID != "energy_main" || bundle.DataSceneID != "campus_energy" {
		t.Fatalf("bundle data aliases = %#v", bundle)
	}
	if len(bundle.Modules) != 2 || len(bundle.Charts) != 2 {
		t.Fatalf("bundle sizes: modules=%d charts=%d", len(bundle.Modules), len(bundle.Charts))
	}
	if energyCalls != 1 {
		t.Fatalf("energy adapter calls = %d, want 1", energyCalls)
	}

	total := findModuleData(t, bundle.Modules, "energy.total.today")
	if total.Value != 7841.5 || total.Unit != "kWh" {
		t.Fatalf("energy total module = %#v", total)
	}
	totalChart := findChartData(t, bundle.Charts, "energy.total.today")
	if totalChart.BindingID != "energy.total.today" || totalChart.ChartID != "chart_energy_total" {
		t.Fatalf("energy total chart metadata = %#v", totalChart)
	}
	if totalChart.Rows[1][1] != "7841.5" || totalChart.Rows[2][1] != "4" {
		t.Fatalf("energy total rows = %#v", totalChart.Rows)
	}

	buildingChart := findChartData(t, bundle.Charts, "energy.building.snapshot")
	if buildingChart.BindingID != "energy.building.snapshot" || buildingChart.ChartID != "chart_energy_building" {
		t.Fatalf("building chart metadata = %#v", buildingChart)
	}
	if buildingChart.Rows[1][0] != "图书馆" || buildingChart.Rows[1][1] != "1280.5" {
		t.Fatalf("building rows = %#v", buildingChart.Rows)
	}
	if buildingChart.Rows[2][0] != "教学楼A" || buildingChart.Rows[2][1] != "2460.0" {
		t.Fatalf("building rows should keep one decimal place: %#v", buildingChart.Rows)
	}
	cached, err := runtime.GetByChartID(context.Background(), "chart_energy_total")
	if err != nil {
		t.Fatalf("GetByChartID returned error: %v", err)
	}
	if cached == nil || cached.CellValues[1][1] != "7841.5" {
		t.Fatalf("runtime cache missing energy total: %#v", cached)
	}
}

func TestScriptServiceBuildEducationDataBundle(t *testing.T) {
	runtime := &fakeChartRuntimeCache{}
	educationCalls := 0
	svc := NewScriptService(
		fakeScriptRepo{scripts: []entity.Script{{ScriptID: "education_main", SceneID: "campus_teaching", Status: "enabled"}}},
		fakeScriptModuleRepo{modules: []entity.ScriptModule{
			{
				ModuleID:   "classroom_usage_card",
				ScriptID:   "education_main",
				Type:       "stat_card",
				Title:      "教室使用概况",
				DataKey:    "education.classroom.usage",
				Template:   "stat_card_basic",
				TargetKind: "chart",
				TargetID:   "chart_education_usage",
				Enabled:    true,
			},
			{
				ModuleID:   "classroom_schedule_table",
				ScriptID:   "education_main",
				Type:       "table",
				Title:      "今日课堂状态",
				DataKey:    "education.classroom.schedule",
				Template:   "table_basic",
				TargetKind: "chart",
				TargetID:   "chart_education_schedule",
				Enabled:    true,
			},
		}},
		fakeDataBindingRepo{bindings: []entity.DataBinding{
			{DataKey: "education.classroom.usage", BindingID: "education.classroom.usage", ChartID: "chart_education_usage", ModuleID: "classroom_usage_card", Enabled: true},
			{DataKey: "education.classroom.schedule", BindingID: "education.classroom.schedule", ChartID: "chart_education_schedule", ModuleID: "classroom_schedule_table", Enabled: true},
		}},
		fakeVideoSourceRepo{},
		fakeSecurityAdapter{},
		fakeEnergyAdapter{},
		fakeEducationAdapter{calls: &educationCalls},
		nil,
		nil,
		nil,
		runtime,
		nil,
		false,
		zap.NewNop(),
	)

	bundle, err := svc.BuildDataBundle(context.Background(), "education_main", false)
	if err != nil {
		t.Fatalf("BuildDataBundle returned error: %v", err)
	}
	if bundle.ScriptID != "education_main" || bundle.SceneID != "campus_teaching" {
		t.Fatalf("bundle identity = %#v", bundle)
	}
	if bundle.DataScriptID != "education_main" || bundle.DataSceneID != "campus_teaching" {
		t.Fatalf("bundle data aliases = %#v", bundle)
	}
	if len(bundle.Modules) != 2 || len(bundle.Charts) != 2 {
		t.Fatalf("bundle sizes: modules=%d charts=%d", len(bundle.Modules), len(bundle.Charts))
	}
	if educationCalls != 1 {
		t.Fatalf("education adapter calls = %d, want 1", educationCalls)
	}

	usage := findModuleData(t, bundle.Modules, "education.classroom.usage")
	if usage.Value != 2 || usage.Unit != "间" {
		t.Fatalf("education usage module = %#v", usage)
	}
	usageChart := findChartData(t, bundle.Charts, "education.classroom.usage")
	if usageChart.Rows[1][1] != "2" || usageChart.Rows[2][1] != "1" || usageChart.Rows[3][1] != "1" {
		t.Fatalf("usage rows = %#v", usageChart.Rows)
	}
	scheduleChart := findChartData(t, bundle.Charts, "education.classroom.schedule")
	if scheduleChart.BindingID != "education.classroom.schedule" || scheduleChart.ChartID != "chart_education_schedule" {
		t.Fatalf("schedule chart metadata = %#v", scheduleChart)
	}
	if scheduleChart.Rows[1][0] != "A101" || scheduleChart.Rows[1][2] != "高等数学" || scheduleChart.Rows[1][3] != "张老师" {
		t.Fatalf("schedule rows = %#v", scheduleChart.Rows)
	}
	cached, err := runtime.GetByChartID(context.Background(), "chart_education_usage")
	if err != nil {
		t.Fatalf("GetByChartID returned error: %v", err)
	}
	if cached == nil || cached.CellValues[1][1] != "2" {
		t.Fatalf("runtime cache missing education usage: %#v", cached)
	}
}

func TestScriptServicePreloadWithOptionsPushWSFalseWritesCacheOnly(t *testing.T) {
	scriptCache := &fakeScriptCache{}
	ws := &fakeWSClient{}
	svc := newPreloadTestService(scriptCache, ws)

	bundle, err := svc.PreloadWithOptions(context.Background(), "security_main", PreloadOptions{
		Force:  true,
		PushWS: false,
		Source: "scheduler",
	})
	if err != nil {
		t.Fatalf("PreloadWithOptions returned error: %v", err)
	}
	if bundle == nil || bundle.ScriptID != "security_main" {
		t.Fatalf("bundle = %#v", bundle)
	}
	if scriptCache.setCount != 1 {
		t.Fatalf("script cache set count = %d, want 1", scriptCache.setCount)
	}
	if ws.pushCount != 0 {
		t.Fatalf("ws push count = %d, want 0", ws.pushCount)
	}
}

func TestScriptServiceBuildDataBundleCacheHitBackfillsDataAliases(t *testing.T) {
	oldBundle := &gateway.DataBundle{
		BundleID: "bundle_old",
		ScriptID: "security_main",
		SceneID:  "campus_security",
	}
	scriptCache := &fakeScriptCache{bundles: map[string]*gateway.DataBundle{"security_main": oldBundle}}
	svc := newPreloadTestService(scriptCache, &fakeWSClient{})

	bundle, err := svc.BuildDataBundle(context.Background(), "security_main", true)
	if err != nil {
		t.Fatalf("BuildDataBundle returned error: %v", err)
	}
	if bundle.ScriptID != "security_main" || bundle.SceneID != "campus_security" {
		t.Fatalf("legacy identity changed: %#v", bundle)
	}
	if bundle.DataScriptID != "security_main" || bundle.DataSceneID != "campus_security" {
		t.Fatalf("data aliases were not backfilled: %#v", bundle)
	}
}

func TestScriptServicePreloadWithOptionsCacheHitBackfillsDataAliases(t *testing.T) {
	oldBundle := &gateway.DataBundle{
		BundleID: "bundle_old",
		ScriptID: "security_main",
		SceneID:  "campus_security",
	}
	scriptCache := &fakeScriptCache{bundles: map[string]*gateway.DataBundle{"security_main": oldBundle}}
	ws := &fakeWSClient{}
	svc := newPreloadTestService(scriptCache, ws)

	bundle, err := svc.PreloadWithOptions(context.Background(), "security_main", PreloadOptions{
		Force:  false,
		PushWS: true,
		Source: "http_preload",
	})
	if err != nil {
		t.Fatalf("PreloadWithOptions returned error: %v", err)
	}
	if bundle.ScriptID != "security_main" || bundle.SceneID != "campus_security" {
		t.Fatalf("legacy identity changed: %#v", bundle)
	}
	if bundle.DataScriptID != "security_main" || bundle.DataSceneID != "campus_security" {
		t.Fatalf("data aliases were not backfilled: %#v", bundle)
	}
	if ws.pushCount != 0 {
		t.Fatalf("cache hit should keep existing preload behavior and not push ws, got %d", ws.pushCount)
	}
}

func TestScriptServicePreloadWithOptionsPushWSTruePushesMessages(t *testing.T) {
	scriptCache := &fakeScriptCache{}
	ws := &fakeWSClient{}
	svc := newPreloadTestService(scriptCache, ws)

	_, err := svc.PreloadWithOptions(context.Background(), "security_main", PreloadOptions{
		Force:  true,
		PushWS: true,
		Source: "http_preload",
	})
	if err != nil {
		t.Fatalf("PreloadWithOptions returned error: %v", err)
	}
	if scriptCache.setCount != 1 {
		t.Fatalf("script cache set count = %d, want 1", scriptCache.setCount)
	}
	if ws.pushCount != 1 || len(ws.messages) != 1 || ws.messages[0].Type != "chart_data_push" {
		t.Fatalf("ws pushes = count %d messages %#v", ws.pushCount, ws.messages)
	}
}

func TestScriptServicePreloadLegacyMethodStillPushesWS(t *testing.T) {
	scriptCache := &fakeScriptCache{}
	ws := &fakeWSClient{}
	svc := newPreloadTestService(scriptCache, ws)

	if _, err := svc.Preload(context.Background(), "security_main", true); err != nil {
		t.Fatalf("Preload returned error: %v", err)
	}
	if ws.pushCount != 1 {
		t.Fatalf("ws push count = %d, want 1", ws.pushCount)
	}
}

func newPreloadTestService(scriptCache *fakeScriptCache, ws *fakeWSClient) ScriptService {
	return NewScriptService(
		fakeScriptRepo{scripts: []entity.Script{{ScriptID: "security_main", SceneID: "campus_security", Status: "enabled"}}},
		fakeScriptModuleRepo{modules: []entity.ScriptModule{
			{
				ModuleID:   "today_access_card",
				ScriptID:   "security_main",
				Type:       "stat_card",
				Title:      "access",
				DataKey:    "security.access.today",
				Template:   "stat_card_basic",
				TargetKind: "chart",
				TargetID:   "chart_h",
				Enabled:    true,
			},
		}},
		fakeDataBindingRepo{bindings: []entity.DataBinding{
			{DataKey: "security.access.today", BindingID: "sim_3", ChartID: "chart_h", ModuleID: "today_access_card", Enabled: true},
		}},
		fakeVideoSourceRepo{},
		fakeSecurityAdapter{},
		fakeEnergyAdapter{},
		fakeEducationAdapter{},
		scriptCache,
		nil,
		nil,
		&fakeChartRuntimeCache{},
		ws,
		false,
		zap.NewNop(),
	)
}

type fakeScriptCache struct {
	bundles  map[string]*gateway.DataBundle
	setCount int
}

func (c *fakeScriptCache) GetPreloadBundle(ctx context.Context, scriptID string) (*gateway.DataBundle, bool, error) {
	if c.bundles == nil {
		return nil, false, nil
	}
	bundle, ok := c.bundles[scriptID]
	return bundle, ok, nil
}

func (c *fakeScriptCache) SetPreloadBundle(ctx context.Context, bundle *gateway.DataBundle, ttl time.Duration) error {
	if c.bundles == nil {
		c.bundles = map[string]*gateway.DataBundle{}
	}
	c.setCount++
	c.bundles[bundle.ScriptID] = bundle
	return nil
}

func (c *fakeScriptCache) SetCurrentScript(ctx context.Context, scriptID string, ttl time.Duration) error {
	return nil
}

func (c *fakeScriptCache) GetCurrentScript(ctx context.Context) (string, bool, error) {
	return "", false, nil
}

type fakeWSClient struct {
	pushCount int
	messages  []ue.WSMessage
}

func (c *fakeWSClient) PushMessage(ctx context.Context, msg ue.WSMessage) error {
	c.pushCount++
	c.messages = append(c.messages, msg)
	return nil
}

func (c *fakeWSClient) PushMessages(ctx context.Context, messages []ue.WSMessage) error {
	c.pushCount++
	c.messages = append(c.messages, messages...)
	return nil
}

func (c *fakeWSClient) Stats() websocket.FastAPIClientStats {
	return websocket.FastAPIClientStats{Enabled: true, MessageCountByType: map[string]int64{}}
}

func (c *fakeWSClient) Close() error {
	return nil
}

func TestChartRuntimeDataRoundTripInFakeCache(t *testing.T) {
	runtime := &fakeChartRuntimeCache{}
	data := cache.ChartRuntimeData{
		DataKey:   "security.access.today",
		BindingID: "sim_3",
		ChartID:   "chart_h",
		CellValues: [][]string{
			{"指标", "数值"},
			{"今日进入", "1800"},
		},
		UpdatedAt: "2026-06-02T00:00:00Z",
	}
	if err := runtime.SetChartRuntime(context.Background(), data, cache.DefaultChartRuntimeTTL); err != nil {
		t.Fatalf("SetChartRuntime returned error: %v", err)
	}
	byBinding, err := runtime.GetByBindingID(context.Background(), "sim_3")
	if err != nil {
		t.Fatalf("GetByBindingID returned error: %v", err)
	}
	byChart, err := runtime.GetByChartID(context.Background(), "chart_h")
	if err != nil {
		t.Fatalf("GetByChartID returned error: %v", err)
	}
	if byBinding == nil || byChart == nil {
		t.Fatalf("runtime cache lookup missing: binding=%#v chart=%#v", byBinding, byChart)
	}
	if byBinding.CellValues[1][1] != "1800" || byChart.CellValues[1][1] != "1800" {
		t.Fatalf("runtime values mismatch: binding=%#v chart=%#v", byBinding, byChart)
	}
}

func findModuleData(t *testing.T, modules []gateway.ModuleData, dataKey string) gateway.ModuleData {
	t.Helper()
	for _, module := range modules {
		if module.DataKey == dataKey {
			return module
		}
	}
	t.Fatalf("module %s not found in %#v", dataKey, modules)
	return gateway.ModuleData{}
}

func findChartData(t *testing.T, charts []gateway.ChartData, dataKey string) gateway.ChartData {
	t.Helper()
	for _, chart := range charts {
		if chart.DataKey == dataKey {
			return chart
		}
	}
	t.Fatalf("chart %s not found in %#v", dataKey, charts)
	return gateway.ChartData{}
}
