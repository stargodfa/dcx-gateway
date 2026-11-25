package service

import (
	"context"
	"encoding/json"
	"fmt"

	"dcx-gateway/internal/cache"
	"dcx-gateway/internal/model/protocol/ue"
	"dcx-gateway/internal/repository"
	"dcx-gateway/internal/service/mapper"
)

type UEConfigService interface {
	GetRawConfig(ctx context.Context, key string) (json.RawMessage, error)
	ReplaceRawConfig(ctx context.Context, routeKey string, raw []byte) (json.RawMessage, error)
	ListRawConfigs(ctx context.Context) (*ue.ConfigIndex, error)
	BuildBootstrap(ctx context.Context) (*ue.BootstrapConfig, error)
	BuildChartBinding(ctx context.Context) (*ue.ChartBinding, error)
	BuildVideoStreams(ctx context.Context) (*ue.VideoStreams, error)
	BuildChartConfig(ctx context.Context) (*ue.ChartConfig, error)
	BuildStructuredChartConfig(ctx context.Context) (*ue.ChartConfig, error)
	BuildRuntimeChartConfig(ctx context.Context) (*ue.ChartConfig, error)
	BuildReportScripts(ctx context.Context) (*ue.LightReportScripts, error)
	CreateReportScript(ctx context.Context, patch ReportScriptPatch) (json.RawMessage, error)
	UpdateReportScript(ctx context.Context, patch ReportScriptPatch) (json.RawMessage, error)
	DeleteReportScript(ctx context.Context, scriptID string) (json.RawMessage, error)
	CreateReportChapter(ctx context.Context, patch ReportChapterPatch) (json.RawMessage, error)
	UpdateReportChapter(ctx context.Context, patch ReportChapterPatch) (json.RawMessage, error)
	DeleteReportChapter(ctx context.Context, patch ReportChapterPatch) (json.RawMessage, error)
	CreateReportScene(ctx context.Context, patch ReportScenePatch) (json.RawMessage, error)
	UpdateReportScene(ctx context.Context, patch ReportScenePatch) (json.RawMessage, error)
	DeleteReportScene(ctx context.Context, patch ReportScenePatch) (json.RawMessage, error)
	UpdateReportSceneCamera(ctx context.Context, patch ReportScenePatch) (json.RawMessage, error)
	CreateReportSceneBindings(ctx context.Context, patch ReportBindingPatch) (json.RawMessage, error)
	UpdateReportSceneBindings(ctx context.Context, patch ReportBindingPatch) (json.RawMessage, error)
	DeleteReportSceneBindings(ctx context.Context, patch ReportBindingDeletePatch) (json.RawMessage, error)
	UpdateReportSceneWarmup(ctx context.Context, patch ReportWarmupPatch) (json.RawMessage, error)
	BuildModelTypes(ctx context.Context) (*ue.ModelTypesConfig, error)
	CreateModelTypes(ctx context.Context, patches []ModelTypePatch) (*ue.ModelTypesConfig, error)
	UpdateModelTypes(ctx context.Context, patches []ModelTypePatch) (*ue.ModelTypesConfig, error)
	DeleteModelTypes(ctx context.Context, ids []string) (*ue.ModelTypesConfig, error)
	BuildModelTable(ctx context.Context) (*ue.ModelTable, error)
	CreateModelTableEntries(ctx context.Context, patches []ModelInstancePatch) (*ue.ModelTable, error)
	UpdateModelTableEntries(ctx context.Context, patches []ModelInstancePatch) (*ue.ModelTable, error)
	DeleteModelTableEntries(ctx context.Context, ids []string) (*ue.ModelTable, error)
	BuildStructuredModelConfig(ctx context.Context) (*ue.StructuredModelConfig, error)
	BuildRuntimeModelConfig(ctx context.Context) (*ue.ModelConfig, error)
	UpdateModelPlacements(ctx context.Context, patches []ModelPlacementPatch) (*ue.StructuredModelConfig, error)
	DeleteModelPlacements(ctx context.Context, ids []string) (*ue.StructuredModelConfig, error)
	BuildSceneDesiredConfig(ctx context.Context) (*ue.SceneDesiredConfig, error)
	UpdateInitConfig(ctx context.Context, patch InitConfigUpdate) (json.RawMessage, error)
	AddInitInstanceIDs(ctx context.Context, ids []string) (json.RawMessage, error)
	DeleteInitInstanceIDs(ctx context.Context, ids []string) (json.RawMessage, error)
	PatchSceneDesiredModels(ctx context.Context, patches []SceneDesiredModelPatch) (*ue.SceneDesiredConfig, error)
	UpsertSceneDesiredModels(ctx context.Context, patches []SceneDesiredModelPatch) (*ue.SceneDesiredConfig, error)
	DeleteSceneDesiredModels(ctx context.Context, ids []string) (*ue.SceneDesiredConfig, error)
}

type ueConfigService struct {
	scriptRepo        repository.ScriptRepository
	scriptModuleRepo  repository.ScriptModuleRepository
	dataBindingRepo   repository.DataBindingRepository
	videoSourceRepo   repository.VideoSourceRepository
	rawConfigRepo     repository.UEConfigDocumentRepository
	chartConfigRepo   repository.UEChartConfigRepository
	sceneDesiredRepo  repository.UESceneDesiredModelRepository
	modelTypeRepo     repository.UEModelTypeRepository
	modelInstanceRepo repository.UEModelInstanceRepository
	placementRepo     repository.UEModelInstancePlacementRepository
	chartRuntimeCache cache.ChartRuntimeCache
}

type UEModelConfigRepositories struct {
	ModelType     repository.UEModelTypeRepository
	ModelInstance repository.UEModelInstanceRepository
	Placement     repository.UEModelInstancePlacementRepository
}

func NewUEConfigService(
	scriptRepo repository.ScriptRepository,
	scriptModuleRepo repository.ScriptModuleRepository,
	dataBindingRepo repository.DataBindingRepository,
	videoSourceRepo repository.VideoSourceRepository,
	rawConfigRepo repository.UEConfigDocumentRepository,
	chartConfigRepo repository.UEChartConfigRepository,
	sceneDesiredRepo repository.UESceneDesiredModelRepository,
	chartRuntimeCache cache.ChartRuntimeCache,
	modelRepos ...UEModelConfigRepositories,
) UEConfigService {
	svc := &ueConfigService{
		scriptRepo:        scriptRepo,
		scriptModuleRepo:  scriptModuleRepo,
		dataBindingRepo:   dataBindingRepo,
		videoSourceRepo:   videoSourceRepo,
		rawConfigRepo:     rawConfigRepo,
		chartConfigRepo:   chartConfigRepo,
		sceneDesiredRepo:  sceneDesiredRepo,
		chartRuntimeCache: chartRuntimeCache,
	}
	if len(modelRepos) > 0 {
		svc.modelTypeRepo = modelRepos[0].ModelType
		svc.modelInstanceRepo = modelRepos[0].ModelInstance
		svc.placementRepo = modelRepos[0].Placement
	}
	return svc
}

func (s *ueConfigService) GetRawConfig(ctx context.Context, key string) (json.RawMessage, error) {
	doc, err := s.rawConfigRepo.FindByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	raw := json.RawMessage(doc.ContentJSON)
	if !json.Valid(raw) {
		return nil, fmt.Errorf("ue config document contains invalid json: %s", key)
	}
	return raw, nil
}

func (s *ueConfigService) ListRawConfigs(ctx context.Context) (*ue.ConfigIndex, error) {
	docs, err := s.rawConfigRepo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	configs := make([]ue.ConfigMeta, 0, len(docs))
	for _, doc := range docs {
		configs = append(configs, ue.ConfigMeta{
			ConfigKey: doc.ConfigKey,
			FileName:  doc.FileName,
			Version:   doc.Version,
			Enabled:   doc.Enabled,
			Source:    doc.Source,
		})
	}
	return &ue.ConfigIndex{Configs: configs}, nil
}

func (s *ueConfigService) BuildChartBinding(ctx context.Context) (*ue.ChartBinding, error) {
	bindings, err := s.dataBindingRepo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	return mapper.DataBindingsToChartBindingConfig(bindings), nil
}

func (s *ueConfigService) BuildVideoStreams(ctx context.Context) (*ue.VideoStreams, error) {
	videos, err := s.videoSourceRepo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	return mapper.VideoSourcesToVideoStreamsConfig(videos), nil
}

func (s *ueConfigService) BuildChartConfig(ctx context.Context) (*ue.ChartConfig, error) {
	return s.BuildStructuredChartConfig(ctx)
}

func (s *ueConfigService) BuildStructuredChartConfig(ctx context.Context) (*ue.ChartConfig, error) {
	if s.chartConfigRepo == nil {
		return s.buildLegacyLightChartConfig(ctx)
	}
	settings, err := s.chartConfigRepo.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	charts, err := s.chartConfigRepo.ListEnabledCharts(ctx)
	if err != nil {
		return nil, err
	}
	return mapper.UEChartConfigsToChartConfig(settings, charts, nil), nil
}

func (s *ueConfigService) BuildRuntimeChartConfig(ctx context.Context) (*ue.ChartConfig, error) {
	if s.chartConfigRepo == nil {
		return s.buildLegacyLightChartConfig(ctx)
	}
	settings, err := s.chartConfigRepo.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	charts, err := s.chartConfigRepo.ListEnabledCharts(ctx)
	if err != nil {
		return nil, err
	}
	runtimeCellValues := make(map[string][][]string)
	if s.chartRuntimeCache != nil {
		for _, chart := range charts {
			runtime, err := s.chartRuntimeCache.GetByChartID(ctx, chart.ChartID)
			if err != nil {
				continue
			}
			if runtime != nil && len(runtime.CellValues) > 0 {
				runtimeCellValues[chart.ChartID] = runtime.CellValues
			}
		}
	}
	return mapper.UEChartConfigsToChartConfig(settings, charts, runtimeCellValues), nil
}

func (s *ueConfigService) buildLegacyLightChartConfig(ctx context.Context) (*ue.ChartConfig, error) {
	bindings, err := s.dataBindingRepo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	modules, err := s.scriptModuleRepo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	return mapper.DataBindingsToChartConfig(bindings, modules), nil
}

func (s *ueConfigService) BuildReportScripts(ctx context.Context) (*ue.LightReportScripts, error) {
	scripts, err := s.scriptRepo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	modules, err := s.scriptModuleRepo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	bindings, err := s.dataBindingRepo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	videos, err := s.videoSourceRepo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	return mapper.BuildLightReportScripts(scripts, modules, bindings, videos), nil
}
