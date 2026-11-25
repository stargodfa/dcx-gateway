package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"dcx-gateway/internal/model/protocol/ue"
	"dcx-gateway/internal/repository"
)

type rawPresetConfig struct {
	ConfigKey string
	FileName  string
}

var rawPresetAllowlist = map[string]rawPresetConfig{
	"path-config":    {ConfigKey: "path_config", FileName: "path_config.json"},
	"chart-config":   {ConfigKey: "chart_config", FileName: "chart_config.json"},
	"chart-binding":  {ConfigKey: "chart_binding", FileName: "chart_binding.json"},
	"drone-config":   {ConfigKey: "drone_config", FileName: "drone_config.json"},
	"render-config":  {ConfigKey: "render_config", FileName: "render_config.json"},
	"video-streams":  {ConfigKey: "video_streams", FileName: "video_streams.json"},
	"report-scripts": {ConfigKey: "report_scripts", FileName: "report_scripts.json"},
}

func (s *ueConfigService) ReplaceRawConfig(ctx context.Context, routeKey string, raw []byte) (json.RawMessage, error) {
	preset, ok := rawPresetAllowlist[strings.TrimSpace(routeKey)]
	if !ok {
		return nil, fmt.Errorf("%w: raw config key is not writable: %s", ErrInvalidModelConfigPatch, routeKey)
	}
	raw = bytes.TrimSpace(raw)
	if !json.Valid(raw) {
		return nil, fmt.Errorf("%w: request body must be valid json", ErrInvalidModelConfigPatch)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return nil, fmt.Errorf("%w: request body must be a json object", ErrInvalidModelConfigPatch)
	}
	next, err := s.rawConfigRepo.ReplaceContentJSONAtomic(ctx, preset.ConfigKey, preset.FileName, raw)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(next), nil
}

func (s *ueConfigService) BuildBootstrap(ctx context.Context) (*ue.BootstrapConfig, error) {
	initConfig, err := s.GetRawConfig(ctx, "init_config")
	if err != nil {
		return nil, fmt.Errorf("read init_config: %w", err)
	}
	pathConfig, err := s.GetRawConfig(ctx, "path_config")
	if err != nil {
		return nil, fmt.Errorf("read path_config: %w", err)
	}
	chartConfig, err := s.GetRawConfig(ctx, "chart_config")
	if err != nil {
		return nil, fmt.Errorf("read chart_config: %w", err)
	}
	chartBinding, err := s.bootstrapChartBinding(ctx)
	if err != nil {
		return nil, err
	}
	droneConfig, err := s.GetRawConfig(ctx, "drone_config")
	if err != nil {
		return nil, fmt.Errorf("read drone_config: %w", err)
	}
	modelTable, err := s.BuildModelTable(ctx)
	if err != nil {
		return nil, err
	}
	modelConfig, err := s.BuildRuntimeModelConfig(ctx)
	if err != nil {
		return nil, err
	}
	sceneDesired, err := s.BuildSceneDesiredConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &ue.BootstrapConfig{
		Version:      1,
		InitConfig:   initConfig,
		PathConfig:   pathConfig,
		ChartConfig:  chartConfig,
		ChartBinding: chartBinding,
		DroneConfig:  droneConfig,
		ModelTable:   modelTable,
		ModelConfig:  modelConfig,
		SceneDesired: sceneDesired,
	}, nil
}

func (s *ueConfigService) bootstrapChartBinding(ctx context.Context) (any, error) {
	raw, err := s.GetRawConfig(ctx, "chart_binding")
	if err == nil {
		return raw, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("read chart_binding: %w", err)
	}
	return s.BuildChartBinding(ctx)
}
