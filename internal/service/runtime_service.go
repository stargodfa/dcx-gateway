package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"dcx-gateway/internal/model/entity"
	"dcx-gateway/internal/model/protocol/ue"
	"dcx-gateway/internal/repository"

	"go.uber.org/zap"
)

var ErrInvalidRuntimeRequest = errors.New("invalid runtime request")

type RuntimeService interface {
	BuildWarmupOptions(ctx context.Context) (*ue.WarmupOptions, error)
	PreloadReportScene(ctx context.Context, req ReportScenePreloadRequest) (*ue.ReportScenePreloadResponse, error)
}

type ReportScenePreloadRequest struct {
	ScriptID  string
	ChapterID string
	SceneID   string
	Force     bool
	PushWS    bool
	Source    string
	AuthType  string
	Username  string
}

type runtimeService struct {
	scriptRepo        repository.ScriptRepository
	scriptModuleRepo  repository.ScriptModuleRepository
	dataBindingRepo   repository.DataBindingRepository
	videoSourceRepo   repository.VideoSourceRepository
	modelInstanceRepo repository.UEModelInstanceRepository
	rawConfigRepo     repository.UEConfigDocumentRepository
	scriptService     ScriptService
	log               *zap.Logger
}

func NewRuntimeService(
	scriptRepo repository.ScriptRepository,
	scriptModuleRepo repository.ScriptModuleRepository,
	dataBindingRepo repository.DataBindingRepository,
	videoSourceRepo repository.VideoSourceRepository,
	modelInstanceRepo repository.UEModelInstanceRepository,
	rawConfigRepo repository.UEConfigDocumentRepository,
	scriptService ScriptService,
	log *zap.Logger,
) RuntimeService {
	return &runtimeService{
		scriptRepo:        scriptRepo,
		scriptModuleRepo:  scriptModuleRepo,
		dataBindingRepo:   dataBindingRepo,
		videoSourceRepo:   videoSourceRepo,
		modelInstanceRepo: modelInstanceRepo,
		rawConfigRepo:     rawConfigRepo,
		scriptService:     scriptService,
		log:               log,
	}
}

func (s *runtimeService) BuildWarmupOptions(ctx context.Context) (*ue.WarmupOptions, error) {
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
	instances, err := s.modelInstanceRepo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}

	return &ue.WarmupOptions{
		Version:     1,
		DataScripts: buildDataScriptOptions(scripts, modules),
		Charts:      buildChartOptions(bindings, modules),
		Videos:      buildVideoOptions(videos, modules),
		Instances:   buildInstanceOptions(instances),
	}, nil
}

func (s *runtimeService) PreloadReportScene(ctx context.Context, req ReportScenePreloadRequest) (*ue.ReportScenePreloadResponse, error) {
	scriptID, err := requiredRuntimeID(req.ScriptID, "script_id")
	if err != nil {
		return nil, err
	}
	chapterID, err := requiredRuntimeID(req.ChapterID, "chapter_id")
	if err != nil {
		return nil, err
	}
	sceneID, err := requiredRuntimeID(req.SceneID, "scene_id")
	if err != nil {
		return nil, err
	}
	if s.rawConfigRepo == nil {
		return nil, repository.ErrNotFound
	}
	doc, err := s.rawConfigRepo.FindByKey(ctx, "report_scripts")
	if err != nil {
		return nil, err
	}
	var report ue.ReportScripts
	if err := json.Unmarshal([]byte(doc.ContentJSON), &report); err != nil {
		return nil, fmt.Errorf("decode report_scripts: %w", err)
	}
	normalizeReportScripts(&report)
	scene, err := requireScene(&report, scriptID, chapterID, sceneID)
	if err != nil {
		return nil, err
	}
	warmup := normalizeWarmup(scene.Warmup)
	dataScripts := normalizeStringList(warmup.Data)
	results := make([]ue.ReportScenePreloadResult, 0, len(dataScripts))
	for _, dataScriptID := range dataScripts {
		result := ue.ReportScenePreloadResult{
			DataScriptID:    dataScriptID,
			Force:           req.Force,
			PushWS:          req.PushWS,
			ChartsRequested: append([]string{}, warmup.Charts...),
			VideosRequested: append([]string{}, warmup.Videos...),
		}
		bundle, err := s.scriptService.PreloadWithOptions(ctx, dataScriptID, PreloadOptions{
			Force:    req.Force,
			PushWS:   req.PushWS,
			Source:   defaultString(req.Source, "report_scene_preload"),
			AuthType: defaultString(req.AuthType, "unknown"),
			Username: req.Username,
		})
		if err != nil {
			result.Success = false
			result.Error = err.Error()
			if s.log != nil {
				s.log.Warn("report scene warmup preload failed",
					zap.String("report_script_id", scriptID),
					zap.String("report_chapter_id", chapterID),
					zap.String("report_scene_id", sceneID),
					zap.String("data_script_id", dataScriptID),
					zap.Bool("push_ws", req.PushWS),
					zap.Error(err),
				)
			}
		} else {
			result.Success = true
			if bundle != nil {
				result.BundleID = bundle.BundleID
			}
		}
		results = append(results, result)
	}
	return &ue.ReportScenePreloadResponse{
		Version:   1,
		ScriptID:  scriptID,
		ChapterID: chapterID,
		SceneID:   sceneID,
		Warmup:    warmup,
		Results:   results,
	}, nil
}

func buildDataScriptOptions(scripts []entity.Script, modules []entity.ScriptModule) []ue.WarmupDataScriptOption {
	dataKeysByScript := make(map[string][]string)
	seenByScript := make(map[string]map[string]struct{})
	for _, module := range modules {
		if strings.TrimSpace(module.DataKey) == "" {
			continue
		}
		if seenByScript[module.ScriptID] == nil {
			seenByScript[module.ScriptID] = map[string]struct{}{}
		}
		if _, ok := seenByScript[module.ScriptID][module.DataKey]; ok {
			continue
		}
		seenByScript[module.ScriptID][module.DataKey] = struct{}{}
		dataKeysByScript[module.ScriptID] = append(dataKeysByScript[module.ScriptID], module.DataKey)
	}
	out := make([]ue.WarmupDataScriptOption, 0, len(scripts))
	for _, script := range scripts {
		keys := append([]string{}, dataKeysByScript[script.ScriptID]...)
		sort.Strings(keys)
		label := strings.TrimSpace(script.Name)
		if label == "" {
			label = script.ScriptID
		}
		out = append(out, ue.WarmupDataScriptOption{
			DataScriptID: script.ScriptID,
			Label:        label,
			DataSceneID:  script.SceneID,
			DataKeys:     keys,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DataScriptID < out[j].DataScriptID })
	return out
}

func buildChartOptions(bindings []entity.DataBinding, modules []entity.ScriptModule) []ue.WarmupChartOption {
	moduleByID := make(map[string]entity.ScriptModule)
	modulesByDataKey := make(map[string][]entity.ScriptModule)
	for _, module := range modules {
		moduleByID[module.ModuleID] = module
		if module.DataKey != "" {
			modulesByDataKey[module.DataKey] = append(modulesByDataKey[module.DataKey], module)
		}
	}
	out := make([]ue.WarmupChartOption, 0, len(bindings))
	for _, binding := range bindings {
		candidates := []entity.ScriptModule{}
		if binding.ModuleID != "" {
			if module, ok := moduleByID[binding.ModuleID]; ok {
				candidates = append(candidates, module)
			}
		}
		if len(candidates) == 0 {
			candidates = append(candidates, modulesByDataKey[binding.DataKey]...)
		}
		if len(candidates) == 0 {
			out = append(out, warmupChartOption(binding, entity.ScriptModule{}))
			continue
		}
		for _, module := range candidates {
			out = append(out, warmupChartOption(binding, module))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].BindingID != out[j].BindingID {
			return out[i].BindingID < out[j].BindingID
		}
		if out[i].DataScriptID != out[j].DataScriptID {
			return out[i].DataScriptID < out[j].DataScriptID
		}
		return out[i].ModuleID < out[j].ModuleID
	})
	return out
}

func warmupChartOption(binding entity.DataBinding, module entity.ScriptModule) ue.WarmupChartOption {
	title := strings.TrimSpace(module.Title)
	if title == "" {
		title = binding.DataKey
	}
	return ue.WarmupChartOption{
		BindingID:    binding.BindingID,
		ChartID:      binding.ChartID,
		Title:        title,
		DataKey:      binding.DataKey,
		DataScriptID: module.ScriptID,
		ModuleID:     module.ModuleID,
	}
}

func buildVideoOptions(videos []entity.VideoSource, modules []entity.ScriptModule) []ue.WarmupVideoOption {
	modulesByVideoID := make(map[string][]entity.ScriptModule)
	for _, module := range modules {
		if module.TargetKind == "video" && module.TargetID != "" {
			modulesByVideoID[module.TargetID] = append(modulesByVideoID[module.TargetID], module)
		}
	}
	out := make([]ue.WarmupVideoOption, 0, len(videos))
	for _, video := range videos {
		candidates := modulesByVideoID[video.VideoID]
		if len(candidates) == 0 {
			out = append(out, warmupVideoOption(video, entity.ScriptModule{}))
			continue
		}
		for _, module := range candidates {
			out = append(out, warmupVideoOption(video, module))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].VideoID != out[j].VideoID {
			return out[i].VideoID < out[j].VideoID
		}
		if out[i].DataScriptID != out[j].DataScriptID {
			return out[i].DataScriptID < out[j].DataScriptID
		}
		return out[i].ModuleID < out[j].ModuleID
	})
	return out
}

func warmupVideoOption(video entity.VideoSource, module entity.ScriptModule) ue.WarmupVideoOption {
	title := strings.TrimSpace(video.Name)
	if title == "" {
		title = video.VideoID
	}
	return ue.WarmupVideoOption{
		VideoID:      video.VideoID,
		Title:        title,
		PlayURL:      video.PlayURL,
		DataScriptID: module.ScriptID,
		ModuleID:     module.ModuleID,
	}
}

func buildInstanceOptions(instances []entity.UEModelInstance) []ue.WarmupInstanceOption {
	out := make([]ue.WarmupInstanceOption, 0, len(instances))
	for _, instance := range instances {
		out = append(out, ue.WarmupInstanceOption{
			InstanceID:  instance.InstanceID,
			ModelID:     instance.ModelID,
			DisplayName: instance.DisplayName,
			Enabled:     instance.Enabled,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].InstanceID < out[j].InstanceID })
	return out
}

func requiredRuntimeID(value string, field string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%w: %s is required", ErrInvalidRuntimeRequest, field)
	}
	return value, nil
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
