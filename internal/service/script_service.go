package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"dcx-gateway/internal/adapter/education"
	"dcx-gateway/internal/adapter/energy"
	"dcx-gateway/internal/adapter/security"
	"dcx-gateway/internal/cache"
	"dcx-gateway/internal/model/domain"
	"dcx-gateway/internal/model/entity"
	gateway "dcx-gateway/internal/model/protocol/gateway"
	"dcx-gateway/internal/model/protocol/ue"
	"dcx-gateway/internal/repository"
	"dcx-gateway/internal/service/mapper"
	"dcx-gateway/internal/websocket"

	"go.uber.org/zap"
)

//查脚本
//查模块
//查 data_binding
//调用 security adapter
//组装 gateway.DataBundle

type ScriptService interface {
	BuildDataBundle(ctx context.Context, scriptID string, useCache bool) (*gateway.DataBundle, error)
	Preload(ctx context.Context, scriptID string, force bool) (*gateway.DataBundle, error)
	PreloadWithOptions(ctx context.Context, scriptID string, opts PreloadOptions) (*gateway.DataBundle, error)
}

type PreloadOptions struct {
	Force    bool
	PushWS   bool
	Source   string
	AuthType string
	Username string
}

type scriptService struct {
	scriptRepo       repository.ScriptRepository
	scriptModuleRepo repository.ScriptModuleRepository
	dataBindingRepo  repository.DataBindingRepository
	videoSourceRepo  repository.VideoSourceRepository

	securityAdapter   security.Adapter
	energyAdapter     energy.Adapter
	educationAdapter  education.Adapter
	scriptCache       cache.ScriptCache
	moduleCache       cache.ModuleCache
	videoCache        cache.VideoCache
	chartRuntimeCache cache.ChartRuntimeCache
	fastAPIClient     websocket.FastAPIClient
	pushVideoStreams  bool
	log               *zap.Logger
}

func NewScriptService(
	scriptRepo repository.ScriptRepository,
	scriptModuleRepo repository.ScriptModuleRepository,
	dataBindingRepo repository.DataBindingRepository,
	videoSourceRepo repository.VideoSourceRepository,

	securityAdapter security.Adapter,
	energyAdapter energy.Adapter,
	educationAdapter education.Adapter,
	scriptCache cache.ScriptCache,
	moduleCache cache.ModuleCache,
	videoCache cache.VideoCache,
	chartRuntimeCache cache.ChartRuntimeCache,
	fastAPIClient websocket.FastAPIClient,
	pushVideoStreams bool,
	log *zap.Logger,
) ScriptService {
	return &scriptService{
		scriptRepo:        scriptRepo,
		scriptModuleRepo:  scriptModuleRepo,
		dataBindingRepo:   dataBindingRepo,
		videoSourceRepo:   videoSourceRepo,
		securityAdapter:   securityAdapter,
		energyAdapter:     energyAdapter,
		educationAdapter:  educationAdapter,
		scriptCache:       scriptCache,
		moduleCache:       moduleCache,
		videoCache:        videoCache,
		chartRuntimeCache: chartRuntimeCache,
		fastAPIClient:     fastAPIClient,
		pushVideoStreams:  pushVideoStreams,
		log:               log,
	}
}

func (s *scriptService) BuildDataBundle(ctx context.Context, scriptID string, useCache bool) (*gateway.DataBundle, error) {
	if useCache && s.scriptCache != nil {
		if bundle, ok, err := s.scriptCache.GetPreloadBundle(ctx, scriptID); err != nil {
			s.log.Warn("get preload bundle cache failed", zap.String("script_id", scriptID), zap.Error(err))
		} else if ok {
			normalizeDataBundleAliases(bundle)
			s.writeChartRuntimeCache(ctx, bundle)
			return bundle, nil
		}
	}

	script, err := s.scriptRepo.FindByScriptID(ctx, scriptID)
	if err != nil {
		return nil, err
	}

	if script.Status != "enabled" {
		return nil, fmt.Errorf("script disabled: %s", scriptID)
	}
	modules, err := s.scriptModuleRepo.ListByScriptID(ctx, scriptID)
	if err != nil {
		return nil, err
	}
	dataKeys := collectDataKeys(modules)
	bindings, err := s.dataBindingRepo.ListByDataKeys(ctx, dataKeys)
	if err != nil {
		return nil, err
	}
	bindingMap := make(map[string]entity.DataBinding)
	for _, b := range bindings {
		bindingMap[b.DataKey] = b
	}

	now := time.Now().Format(time.RFC3339)

	bundle := &gateway.DataBundle{
		BundleID:     fmt.Sprintf("bundle_%d", time.Now().UnixNano()),
		ScriptID:     script.ScriptID,
		SceneID:      script.SceneID,
		DataScriptID: script.ScriptID,
		DataSceneID:  script.SceneID,
		Version:      "v1",
		GeneratedAt:  now,
		TTLSeconds:   30,

		Modules:       make([]gateway.ModuleData, 0),
		Charts:        make([]gateway.ChartData, 0),
		Videos:        make([]gateway.VideoSource, 0),
		ModelBindings: make([]gateway.ModelBinding, 0),
	}

	buildCtx := &buildDataContext{}
	for _, module := range modules {
		moduleData, chartData, err := s.buildModuleData(ctx, module, bindingMap, buildCtx)
		if err != nil {
			s.log.Warn("build module data failed",
				zap.String("module_id", module.ModuleID),
				zap.String("data_key", module.DataKey),
				zap.Error(err),
			)
			continue
		}
		if moduleData != nil {
			bundle.Modules = append(bundle.Modules, *moduleData)

		}
		if chartData != nil {
			bundle.Charts = append(bundle.Charts, *chartData)
		}

		if s.moduleCache != nil && moduleData != nil && module.DataKey != "" {
			if err := s.moduleCache.SetModuleData(ctx, module.DataKey, *moduleData, cache.DefaultModuleTTL); err != nil {
				s.log.Warn("set module cache failed", zap.String("data_key", module.DataKey), zap.Error(err))
			}
		}

		if module.TargetKind == "video" && module.TargetID != "" {
			video, err := s.videoSourceRepo.FindByVideoID(ctx, module.TargetID)
			if err != nil {
				s.log.Warn("video source not found",
					zap.String("video_id", module.TargetID),
					zap.Error(err),
				)
				continue
			}

			status := "online"
			if s.videoCache != nil {
				if cachedStatus, ok, err := s.videoCache.GetVideoStatus(ctx, video.VideoID); err != nil {
					s.log.Warn("get video status cache failed", zap.String("video_id", video.VideoID), zap.Error(err))
				} else if ok {
					status = cachedStatus.Status
				} else {
					_ = s.videoCache.SetVideoStatus(ctx, domain.VideoStatus{
						VideoID:   video.VideoID,
						Status:    "online",
						Timestamp: time.Now().Format(time.RFC3339),
					}, cache.DefaultVideoTTL)
				}
			}

			bundle.Videos = append(bundle.Videos, gateway.VideoSource{
				VideoID:     video.VideoID,
				Title:       video.Name,
				PlayURL:     video.PlayURL,
				Status:      status,
				BindModelID: video.BindModelID,
			})
		}
	}
	s.writeChartRuntimeCache(ctx, bundle)
	return bundle, nil
}

type buildDataContext struct {
	energyLoaded      bool
	energySnapshot    *domain.EnergySnapshot
	energyErr         error
	educationLoaded   bool
	educationSnapshot *domain.EducationSnapshot
	educationErr      error
}

func (c *buildDataContext) getEnergySnapshot(ctx context.Context, adapter energy.Adapter) (*domain.EnergySnapshot, error) {
	if c.energyLoaded {
		return c.energySnapshot, c.energyErr
	}
	c.energyLoaded = true
	if adapter == nil {
		c.energyErr = fmt.Errorf("energy adapter is nil")
		return nil, c.energyErr
	}
	c.energySnapshot, c.energyErr = adapter.GetEnergySnapshot(ctx)
	return c.energySnapshot, c.energyErr
}

func (c *buildDataContext) getEducationSnapshot(ctx context.Context, adapter education.Adapter) (*domain.EducationSnapshot, error) {
	if c.educationLoaded {
		return c.educationSnapshot, c.educationErr
	}
	c.educationLoaded = true
	if adapter == nil {
		c.educationErr = fmt.Errorf("education adapter is nil")
		return nil, c.educationErr
	}
	c.educationSnapshot, c.educationErr = adapter.GetClassroomUsage(ctx)
	return c.educationSnapshot, c.educationErr
}

func (s *scriptService) Preload(ctx context.Context, scriptID string, force bool) (*gateway.DataBundle, error) {
	return s.PreloadWithOptions(ctx, scriptID, PreloadOptions{
		Force:  force,
		PushWS: true,
		Source: "http_preload",
	})
}

func (s *scriptService) PreloadWithOptions(ctx context.Context, scriptID string, opts PreloadOptions) (*gateway.DataBundle, error) {
	source := opts.Source
	if source == "" {
		source = "unknown"
	}
	authType := opts.AuthType
	if authType == "" {
		authType = "unknown"
	}

	if !opts.Force && s.scriptCache != nil {
		if bundle, ok, err := s.scriptCache.GetPreloadBundle(ctx, scriptID); err != nil {
			s.log.Warn("get preload bundle cache failed", zap.String("script_id", scriptID), zap.Error(err))
		} else if ok {
			normalizeDataBundleAliases(bundle)
			s.writeChartRuntimeCache(ctx, bundle)
			s.log.Info("preload cache hit",
				zap.String("source", source),
				zap.String("auth_type", authType),
				zap.String("username", opts.Username),
				zap.String("script_id", scriptID),
				zap.Bool("force", opts.Force),
				zap.Bool("push_ws", opts.PushWS),
			)
			return bundle, nil
		}
	}

	bundle, err := s.BuildDataBundle(ctx, scriptID, false)
	if err != nil {
		return nil, err
	}
	if s.scriptCache != nil {
		if err := s.scriptCache.SetPreloadBundle(ctx, bundle, cache.DefaultScriptTTL); err != nil {
			return nil, err
		}
	}
	if opts.PushWS && s.fastAPIClient != nil {
		messages := mapper.BundleToWSMessages(bundle, 0, s.pushVideoStreams)
		if len(messages) > 0 {
			start := time.Now()
			if err := s.fastAPIClient.PushMessages(ctx, messages); err != nil {
				s.log.Warn("push data bundle to fastapi ws_data failed",
					zap.String("source", source),
					zap.String("auth_type", authType),
					zap.String("username", opts.Username),
					zap.String("script_id", scriptID),
					zap.Bool("force", opts.Force),
					zap.Bool("push_ws", opts.PushWS),
					zap.Int("message_count", len(messages)),
					zap.Strings("message_types", wsMessageTypes(messages)),
					zap.Duration("duration", time.Since(start)),
					zap.Error(err),
				)
			} else {
				s.log.Info("push data bundle to fastapi ws_data success",
					zap.String("source", source),
					zap.String("auth_type", authType),
					zap.String("username", opts.Username),
					zap.String("script_id", scriptID),
					zap.Bool("force", opts.Force),
					zap.Bool("push_ws", opts.PushWS),
					zap.Int("message_count", len(messages)),
					zap.Strings("message_types", wsMessageTypes(messages)),
					zap.Duration("duration", time.Since(start)),
				)
			}
		}
	}
	return bundle, nil
}

func normalizeDataBundleAliases(bundle *gateway.DataBundle) {
	if bundle == nil {
		return
	}
	if bundle.DataScriptID == "" {
		bundle.DataScriptID = bundle.ScriptID
	}
	if bundle.DataSceneID == "" {
		bundle.DataSceneID = bundle.SceneID
	}
}

func wsMessageTypes(messages []ue.WSMessage) []string {
	types := make([]string, 0, len(messages))
	for _, msg := range messages {
		types = append(types, msg.Type)
	}
	return types
}

func (s *scriptService) writeChartRuntimeCache(ctx context.Context, bundle *gateway.DataBundle) {
	if s.chartRuntimeCache == nil || bundle == nil {
		return
	}
	for _, chart := range bundle.Charts {
		if chart.BindingID == "" || chart.ChartID == "" || len(chart.Rows) == 0 {
			continue
		}
		updatedAt := bundle.GeneratedAt
		if updatedAt == "" {
			updatedAt = time.Now().Format(time.RFC3339)
		}
		err := s.chartRuntimeCache.SetChartRuntime(ctx, cache.ChartRuntimeData{
			DataKey:    chart.DataKey,
			BindingID:  chart.BindingID,
			ChartID:    chart.ChartID,
			CellValues: chart.Rows,
			UpdatedAt:  updatedAt,
		}, cache.DefaultChartRuntimeTTL)
		if err != nil {
			s.log.Warn("set chart runtime cache failed",
				zap.String("data_key", chart.DataKey),
				zap.String("binding_id", chart.BindingID),
				zap.String("chart_id", chart.ChartID),
				zap.Error(err),
			)
		}
	}
}

func (s *scriptService) buildModuleData(ctx context.Context, module entity.ScriptModule, bindingMap map[string]entity.DataBinding, buildCtx *buildDataContext) (*gateway.ModuleData, *gateway.ChartData, error) {
	binding, hasBinding := bindingMap[module.DataKey]

	moduleData := &gateway.ModuleData{
		ModuleID:   module.ModuleID,
		Type:       module.Type,
		Title:      module.Title,
		DataKey:    module.DataKey,
		TargetKind: module.TargetKind,
		TargetID:   module.TargetID,
		Timestamp:  time.Now().Format(time.RFC3339),
	}

	var chartData *gateway.ChartData

	switch module.DataKey {
	case "security.access.today":
		access, err := s.securityAdapter.GetAccessSummary(ctx)
		if err != nil {
			return nil, nil, err
		}
		moduleData.Value = access.TotalCount
		moduleData.Unit = "人次"
		moduleData.Timestamp = access.Timestamp

		chartData = &gateway.ChartData{
			ModuleID: module.ModuleID,
			DataKey:  module.DataKey,
			Template: module.Template,
			Title:    module.Title,
			Rows: [][]string{
				{"指标", "数值", "单位"},
				{"今日进入", fmt.Sprint(access.InCount), "人次"},
				{"今日离开", fmt.Sprint(access.OutCount), "人次"},
				{"今日总数", fmt.Sprint(access.TotalCount), "人次"},
			},
		}
	case "security.fire.status":
		alerts, err := s.securityAdapter.GetFireAlerts(ctx)
		if err != nil {
			return nil, nil, err
		}
		status := "normal"
		if len(alerts) > 0 && alerts[0].Level != "normal" {
			status = alerts[0].Level
		}
		moduleData.Value = status
		moduleData.Payload = alerts
		rows := [][]string{
			{"等级", "位置", "信息"},
		}
		for _, alert := range alerts {
			rows = append(rows, []string{
				alert.Level,
				alert.Location,
				alert.Message,
			})
		}
		chartData = &gateway.ChartData{
			ModuleID: module.ModuleID,
			DataKey:  module.DataKey,
			Template: module.Template,
			Title:    module.Title,
			Rows:     rows,
		}
	case "energy.total.today":
		snapshot, err := buildCtx.getEnergySnapshot(ctx, s.energyAdapter)
		if err != nil {
			return nil, nil, err
		}
		total := 0.0
		if snapshot != nil {
			for _, building := range snapshot.Buildings {
				total += building.Value
			}
		}
		moduleData.Value = total
		moduleData.Unit = "kWh"
		moduleData.Timestamp = energySnapshotTimestamp(snapshot)
		chartData = &gateway.ChartData{
			ModuleID: module.ModuleID,
			DataKey:  module.DataKey,
			Template: module.Template,
			Title:    module.Title,
			Rows:     buildEnergyTotalRows(snapshot, total),
		}
	case "energy.building.snapshot":
		snapshot, err := buildCtx.getEnergySnapshot(ctx, s.energyAdapter)
		if err != nil {
			return nil, nil, err
		}
		buildingCount := 0
		if snapshot != nil {
			buildingCount = len(snapshot.Buildings)
			moduleData.Payload = snapshot.Buildings
		}
		moduleData.Value = buildingCount
		moduleData.Unit = "栋"
		moduleData.Timestamp = energySnapshotTimestamp(snapshot)
		chartData = &gateway.ChartData{
			ModuleID: module.ModuleID,
			DataKey:  module.DataKey,
			Template: module.Template,
			Title:    module.Title,
			Rows:     buildEnergyBuildingRows(snapshot),
		}
	case "education.classroom.usage":
		snapshot, err := buildCtx.getEducationSnapshot(ctx, s.educationAdapter)
		if err != nil {
			return nil, nil, err
		}
		occupied, free, unknown := countClassroomStatuses(snapshot)
		moduleData.Value = occupied
		moduleData.Unit = "间"
		moduleData.Timestamp = educationSnapshotTimestamp(snapshot)
		chartData = &gateway.ChartData{
			ModuleID: module.ModuleID,
			DataKey:  module.DataKey,
			Template: module.Template,
			Title:    module.Title,
			Rows:     buildEducationUsageRows(occupied, free, unknown),
		}
	case "education.classroom.schedule":
		snapshot, err := buildCtx.getEducationSnapshot(ctx, s.educationAdapter)
		if err != nil {
			return nil, nil, err
		}
		classroomCount := 0
		if snapshot != nil {
			classroomCount = len(snapshot.Classrooms)
			moduleData.Payload = snapshot.Classrooms
		}
		moduleData.Value = classroomCount
		moduleData.Unit = "间"
		moduleData.Timestamp = educationSnapshotTimestamp(snapshot)
		chartData = &gateway.ChartData{
			ModuleID: module.ModuleID,
			DataKey:  module.DataKey,
			Template: module.Template,
			Title:    module.Title,
			Rows:     buildEducationScheduleRows(snapshot),
		}
	default:
		if module.TargetKind == "video" {
			moduleData.Value = "online"
		}
	}
	if chartData != nil && hasBinding {
		chartData.BindingID = binding.BindingID
		chartData.ChartID = binding.ChartID
	}

	return moduleData, chartData, nil

}

func buildEnergyTotalRows(snapshot *domain.EnergySnapshot, total float64) [][]string {
	buildingCount := 0
	if snapshot != nil {
		buildingCount = len(snapshot.Buildings)
	}
	return [][]string{
		{"指标", "数值", "单位"},
		{"今日总能耗", formatEnergyValue(total), "kWh"},
		{"覆盖楼栋", strconv.Itoa(buildingCount), "栋"},
	}
}

func buildEnergyBuildingRows(snapshot *domain.EnergySnapshot) [][]string {
	rows := [][]string{{"楼栋", "能耗", "单位"}}
	if snapshot == nil {
		return rows
	}
	for _, building := range snapshot.Buildings {
		rows = append(rows, []string{
			building.Name,
			formatEnergyValue(building.Value),
			building.Unit,
		})
	}
	return rows
}

func countClassroomStatuses(snapshot *domain.EducationSnapshot) (occupied int, free int, unknown int) {
	if snapshot == nil {
		return 0, 0, 0
	}
	for _, classroom := range snapshot.Classrooms {
		switch classroom.Status {
		case "occupied":
			occupied++
		case "free":
			free++
		default:
			unknown++
		}
	}
	return occupied, free, unknown
}

func buildEducationUsageRows(occupied int, free int, unknown int) [][]string {
	return [][]string{
		{"状态", "数量", "单位"},
		{"上课中", strconv.Itoa(occupied), "间"},
		{"空闲", strconv.Itoa(free), "间"},
		{"未知", strconv.Itoa(unknown), "间"},
	}
}

func buildEducationScheduleRows(snapshot *domain.EducationSnapshot) [][]string {
	rows := [][]string{{"教室", "状态", "课程", "教师"}}
	if snapshot == nil {
		return rows
	}
	for _, classroom := range snapshot.Classrooms {
		rows = append(rows, []string{
			classroom.Name,
			classroom.Status,
			classroom.CourseName,
			classroom.TeacherName,
		})
	}
	return rows
}

func energySnapshotTimestamp(snapshot *domain.EnergySnapshot) string {
	if snapshot != nil {
		for _, building := range snapshot.Buildings {
			if building.Timestamp != "" {
				return building.Timestamp
			}
		}
	}
	return time.Now().Format(time.RFC3339)
}

func educationSnapshotTimestamp(snapshot *domain.EducationSnapshot) string {
	if snapshot != nil {
		for _, classroom := range snapshot.Classrooms {
			if classroom.Timestamp != "" {
				return classroom.Timestamp
			}
		}
	}
	return time.Now().Format(time.RFC3339)
}

func formatEnergyValue(v float64) string {
	return fmt.Sprintf("%.1f", v)
}

func collectDataKeys(modules []entity.ScriptModule) []string {
	result := make([]string, 0)
	seen := make(map[string]struct{})
	for _, module := range modules {
		if module.DataKey == "" {
			continue
		}
		if _, ok := seen[module.DataKey]; ok {
			continue
		}
		seen[module.DataKey] = struct{}{}
		result = append(result, module.DataKey)
	}
	return result
}
