package mapper

import (
	"encoding/json"
	"math"
	"sort"

	"dcx-gateway/internal/model/entity"
	"dcx-gateway/internal/model/protocol/ue"
)

func DataBindingsToChartBindingConfig(bindings []entity.DataBinding) *ue.ChartBinding {
	items := make([]ue.ChartBindingItem, 0, len(bindings))
	for _, binding := range bindings {
		if binding.BindingID == "" || binding.ChartID == "" {
			continue
		}
		items = append(items, ue.ChartBindingItem{
			BindingID: binding.BindingID,
			ChartID:   binding.ChartID,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].BindingID == items[j].BindingID {
			return items[i].ChartID < items[j].ChartID
		}
		return items[i].BindingID < items[j].BindingID
	})

	return &ue.ChartBinding{
		Version:  1,
		Bindings: items,
	}
}

func VideoSourcesToVideoStreamsConfig(videos []entity.VideoSource) *ue.VideoStreams {
	streams := make([]ue.VideoStream, 0, len(videos))
	for _, video := range videos {
		if video.VideoID == "" {
			continue
		}
		streams = append(streams, ue.VideoStream{
			StreamID:        video.VideoID,
			Name:            video.Name,
			WebRTCPlayerURL: video.PlayURL,
		})
	}

	sort.Slice(streams, func(i, j int) bool {
		return streams[i].StreamID < streams[j].StreamID
	})

	return &ue.VideoStreams{
		Version: 1,
		Streams: streams,
	}
}

func DataBindingsToChartConfig(bindings []entity.DataBinding, modules []entity.ScriptModule) *ue.ChartConfig {
	moduleByID := make(map[string]entity.ScriptModule, len(modules))
	moduleByDataKey := make(map[string]entity.ScriptModule, len(modules))
	for _, module := range modules {
		if module.ModuleID != "" {
			moduleByID[module.ModuleID] = module
		}
		if module.DataKey != "" {
			moduleByDataKey[module.DataKey] = module
		}
	}

	seen := make(map[string]struct{})
	charts := make([]ue.ChartPanel, 0, len(bindings))
	for _, binding := range bindings {
		if binding.ChartID == "" {
			continue
		}
		if _, ok := seen[binding.ChartID]; ok {
			continue
		}
		seen[binding.ChartID] = struct{}{}

		title := binding.ChartID
		if module, ok := moduleByID[binding.ModuleID]; ok && module.Title != "" {
			title = module.Title
		} else if module, ok := moduleByDataKey[binding.DataKey]; ok && module.Title != "" {
			title = module.Title
		}

		charts = append(charts, ue.ChartPanel{
			ChartID:   binding.ChartID,
			ChartName: title,
			Visible:   true,
		})
	}

	sort.Slice(charts, func(i, j int) bool {
		return charts[i].ChartID < charts[j].ChartID
	})

	return &ue.ChartConfig{
		Version: 1,
		Charts:  charts,
	}
}

func UEChartConfigsToChartConfig(
	settings *entity.UEChartConfigSettings,
	charts []entity.UEChartConfig,
	runtimeCellValues map[string][][]string,
) *ue.ChartConfig {
	version := 1
	width := 480
	if settings != nil {
		if settings.Version > 0 {
			version = settings.Version
		}
		if settings.OverlayHUDMaxPanelWidth > 0 {
			width = settings.OverlayHUDMaxPanelWidth
		}
	}

	panels := make([]ue.ChartPanel, 0, len(charts))
	for _, chart := range charts {
		panel := ue.ChartPanel{
			ChartID:   chart.ChartID,
			ChartName: chart.ChartName,
			Visible:   chart.Visible,
			Applied:   chart.Applied,
		}
		if chart.PanelKey != nil {
			panel.PanelKey = *chart.PanelKey
		}
		if chart.LayoutMode != nil {
			panel.LayoutMode = *chart.LayoutMode
		}
		if chart.RowsCount != nil {
			panel.Rows = *chart.RowsCount
		}
		if chart.ColsCount != nil {
			panel.Cols = *chart.ColsCount
		}
		panel.ColFields = decodeStringArray(chart.ColFieldsJSON)
		panel.RowFields = decodeStringArray(chart.RowFieldsJSON)
		panel.PairLabelCols = decodeIntArray(chart.PairLabelColsJSON)
		panel.PairLabelRows = decodeIntArray(chart.PairLabelRowsJSON)

		cellValues := decodeStringMatrix(chart.DefaultCellValuesJSON)
		if runtimeCellValues != nil {
			if runtime, ok := runtimeCellValues[chart.ChartID]; ok && len(runtime) > 0 {
				cellValues = runtime
			}
		}
		if len(cellValues) == 0 && panel.Rows > 0 && panel.Cols > 0 {
			cellValues = emptyStringMatrix(panel.Rows, panel.Cols)
		}
		panel.CellValues = cellValues

		panels = append(panels, panel)
	}

	return &ue.ChartConfig{
		Version:                 version,
		OverlayHUDMaxPanelWidth: width,
		Charts:                  panels,
	}
}

func BuildLightReportScripts(
	scripts []entity.Script,
	modules []entity.ScriptModule,
	bindings []entity.DataBinding,
	videos []entity.VideoSource,
) *ue.LightReportScripts {
	modulesByScriptID := make(map[string][]entity.ScriptModule)
	for _, module := range modules {
		modulesByScriptID[module.ScriptID] = append(modulesByScriptID[module.ScriptID], module)
	}

	bindingByDataKey := make(map[string]entity.DataBinding, len(bindings))
	for _, binding := range bindings {
		if binding.DataKey != "" {
			bindingByDataKey[binding.DataKey] = binding
		}
	}

	enabledVideoIDs := make(map[string]struct{}, len(videos))
	for _, video := range videos {
		if video.VideoID != "" {
			enabledVideoIDs[video.VideoID] = struct{}{}
		}
	}

	out := make([]ue.LightReportScript, 0, len(scripts))
	for _, script := range scripts {
		warmup := ue.ReportWarmup{
			Data:   make([]string, 0),
			Videos: make([]string, 0),
			Charts: make([]string, 0),
		}
		seenData := make(map[string]struct{})
		seenVideos := make(map[string]struct{})
		seenCharts := make(map[string]struct{})
		addUnique(script.ScriptID, seenData, &warmup.Data)

		for _, module := range modulesByScriptID[script.ScriptID] {
			if module.TargetKind == "video" && module.TargetID != "" {
				if _, ok := enabledVideoIDs[module.TargetID]; ok {
					addUnique(module.TargetID, seenVideos, &warmup.Videos)
				}
			}
			if binding, ok := bindingByDataKey[module.DataKey]; ok && binding.BindingID != "" {
				addUnique(binding.BindingID, seenCharts, &warmup.Charts)
			}
		}

		sort.Strings(warmup.Data)
		sort.Strings(warmup.Videos)
		sort.Strings(warmup.Charts)

		out = append(out, ue.LightReportScript{
			ScriptID: script.ScriptID,
			Label:    script.Name,
			Chapters: []ue.LightReportChapter{
				{
					ChapterID: script.ScriptID,
					Label:     script.Name,
					Scenes: []ue.LightReportScene{
						{
							SceneID: script.SceneID,
							Label:   script.SceneID,
							Warmup:  warmup,
						},
					},
				},
			},
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].ScriptID < out[j].ScriptID
	})

	return &ue.LightReportScripts{
		Version: 1,
		Scripts: out,
	}
}

func UESceneDesiredModelsToConfig(models []entity.UESceneDesiredModel) *ue.SceneDesiredConfig {
	items := make([]ue.SceneModelDesired, 0, len(models))
	for _, model := range models {
		item := ue.SceneModelDesired{
			InstanceID:  model.InstanceID,
			ModelID:     model.ModelID,
			URI:         model.URI,
			Format:      model.Format,
			SourcePath:  model.SourcePath,
			DisplayName: model.DisplayName,
			Visible:     model.Visible,
			Transform: &ue.Transform{
				Position: &ue.Vector3{X: model.PositionX, Y: model.PositionY, Z: model.PositionZ},
				Rotation: &ue.Rotation{Yaw: model.RotationYaw, Pitch: model.RotationPitch, Roll: model.RotationRoll},
				Scale:    &ue.Vector3{X: model.ScaleX, Y: model.ScaleY, Z: model.ScaleZ},
			},
			Extra: decodeObject(model.ConfigJSON),
		}
		if model.BoundsX != nil && model.BoundsY != nil && model.BoundsZ != nil {
			item.BoundsSize = &ue.BoundsSize{X: *model.BoundsX, Y: *model.BoundsY, Z: *model.BoundsZ}
		}
		items = append(items, item)
	}
	return &ue.SceneDesiredConfig{
		Version:            1,
		SceneModelsDesired: items,
	}
}

func UEModelTypesToConfig(models []entity.UEModelType) *ue.ModelTypesConfig {
	items := make([]ue.ModelType, 0, len(models))
	for _, model := range models {
		items = append(items, ue.ModelType{
			ModelID:    model.ModelID,
			URI:        model.URI,
			Format:     model.Format,
			SourcePath: model.SourcePath,
			BaseBoundsSize: ue.BoundsSize{
				X: model.BaseBoundsX,
				Y: model.BaseBoundsY,
				Z: model.BaseBoundsZ,
			},
			BoundsNegativeExtentAxis: model.BoundsNegativeExtentAxis,
			Enabled:                  model.Enabled,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ModelID < items[j].ModelID
	})
	return &ue.ModelTypesConfig{Version: 1, ModelTypes: items}
}

func UEModelInstancesToTable(models []entity.UEModelInstance) *ue.ModelTable {
	items := make([]ue.ModelTableEntry, 0, len(models))
	for _, model := range models {
		items = append(items, ue.ModelTableEntry{
			InstanceID:  model.InstanceID,
			ModelID:     model.ModelID,
			DisplayName: model.DisplayName,
		})
	}
	return &ue.ModelTable{Version: 1, Entries: items}
}

func UEPlacementsToStructuredModelConfig(
	instances []entity.UEModelInstance,
	placements []entity.UEModelInstancePlacement,
) *ue.StructuredModelConfig {
	placementByID := placementByInstanceID(placements)
	out := make(map[string]ue.ModelPlacementState, len(instances))
	for _, instance := range instances {
		placement := placementForInstance(instance.InstanceID, placementByID)
		out[instance.InstanceID] = ue.ModelPlacementState{
			Visible:   placement.Visible,
			Transform: transformFromPlacement(placement),
		}
	}
	return &ue.StructuredModelConfig{Version: 1, Placements: out}
}

func UEModelsToRuntimeModelConfig(
	types []entity.UEModelType,
	instances []entity.UEModelInstance,
	placements []entity.UEModelInstancePlacement,
) *ue.ModelConfig {
	typeByID := modelTypeByID(types)
	placementByID := placementByInstanceID(placements)
	out := make(map[string]ue.ModelInstanceState, len(instances))
	for _, instance := range instances {
		modelType, ok := typeByID[instance.ModelID]
		if !ok || !modelType.Enabled {
			continue
		}
		placement := placementForInstance(instance.InstanceID, placementByID)
		out[instance.InstanceID] = ue.ModelInstanceState{
			ModelID:    instance.ModelID,
			Visible:    placement.Visible,
			Transform:  transformFromPlacement(placement),
			BoundsSize: boundsFromTypeAndPlacement(modelType, placement),
		}
	}
	return &ue.ModelConfig{Version: 1, LastLoadByInstanceID: out}
}

func UEModelsToSceneDesiredConfig(
	types []entity.UEModelType,
	instances []entity.UEModelInstance,
	placements []entity.UEModelInstancePlacement,
) *ue.SceneDesiredConfig {
	typeByID := modelTypeByID(types)
	placementByID := placementByInstanceID(placements)
	items := make([]ue.SceneModelDesired, 0, len(instances))
	for _, instance := range instances {
		modelType, ok := typeByID[instance.ModelID]
		if !ok || !modelType.Enabled {
			continue
		}
		placement := placementForInstance(instance.InstanceID, placementByID)
		items = append(items, ue.SceneModelDesired{
			InstanceID:  instance.InstanceID,
			ModelID:     instance.ModelID,
			URI:         modelType.URI,
			Format:      modelType.Format,
			SourcePath:  modelType.SourcePath,
			DisplayName: instance.DisplayName,
			BoundsSize:  boundsFromTypeAndPlacement(modelType, placement),
			Visible:     placement.Visible,
			Transform:   transformFromPlacement(placement),
		})
	}
	return &ue.SceneDesiredConfig{Version: 1, SceneModelsDesired: items}
}

func modelTypeByID(models []entity.UEModelType) map[string]entity.UEModelType {
	out := make(map[string]entity.UEModelType, len(models))
	for _, model := range models {
		out[model.ModelID] = model
	}
	return out
}

func placementByInstanceID(models []entity.UEModelInstancePlacement) map[string]entity.UEModelInstancePlacement {
	out := make(map[string]entity.UEModelInstancePlacement, len(models))
	for _, model := range models {
		out[model.InstanceID] = model
	}
	return out
}

func placementForInstance(instanceID string, placements map[string]entity.UEModelInstancePlacement) entity.UEModelInstancePlacement {
	if placement, ok := placements[instanceID]; ok {
		return placement
	}
	return entity.UEModelInstancePlacement{
		InstanceID: instanceID,
		Visible:    true,
		ScaleX:     1,
		ScaleY:     1,
		ScaleZ:     1,
	}
}

func transformFromPlacement(placement entity.UEModelInstancePlacement) *ue.Transform {
	return &ue.Transform{
		Position: &ue.Vector3{X: placement.PositionX, Y: placement.PositionY, Z: placement.PositionZ},
		Rotation: &ue.Rotation{Yaw: placement.RotationYaw, Pitch: placement.RotationPitch, Roll: placement.RotationRoll},
		Scale:    &ue.Vector3{X: placement.ScaleX, Y: placement.ScaleY, Z: placement.ScaleZ},
	}
}

func boundsFromTypeAndPlacement(modelType entity.UEModelType, placement entity.UEModelInstancePlacement) *ue.BoundsSize {
	return &ue.BoundsSize{
		X: modelType.BaseBoundsX * math.Abs(placement.ScaleX),
		Y: modelType.BaseBoundsY * math.Abs(placement.ScaleY),
		Z: modelType.BaseBoundsZ * math.Abs(placement.ScaleZ),
	}
}

func addUnique(value string, seen map[string]struct{}, target *[]string) {
	if value == "" {
		return
	}
	if _, ok := seen[value]; ok {
		return
	}
	seen[value] = struct{}{}
	*target = append(*target, value)
}

func decodeStringArray(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func decodeIntArray(raw string) []int {
	if raw == "" {
		return nil
	}
	var out []int
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func decodeStringMatrix(raw string) [][]string {
	if raw == "" {
		return nil
	}
	var out [][]string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func decodeObject(raw string) map[string]any {
	if raw == "" {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func emptyStringMatrix(rows, cols int) [][]string {
	out := make([][]string, rows)
	for i := range out {
		out[i] = make([]string, cols)
	}
	return out
}
