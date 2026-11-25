package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"dcx-gateway/internal/model/entity"
	"dcx-gateway/internal/model/protocol/ue"
	"dcx-gateway/internal/service/mapper"
)

var ErrInvalidSceneDesiredPatch = errors.New("invalid scene_desired patch")

type MissingSceneDesiredModelsError struct {
	InstanceIDs []string
}

func (e *MissingSceneDesiredModelsError) Error() string {
	return "scene desired model not found: " + strings.Join(e.InstanceIDs, ",")
}

type SceneDesiredModelPatch struct {
	InstanceID string

	ModelID     *string
	URI         *string
	Format      *string
	SourcePath  *string
	DisplayName *string
	BoundsSize  *ue.Vector3
	Visible     *bool
	Position    *ue.Vector3
	Rotation    *ue.Rotation
	Scale       *ue.Vector3
	ConfigJSON  map[string]any
}

func (s *ueConfigService) PatchSceneDesiredModels(ctx context.Context, patches []SceneDesiredModelPatch) (*ue.SceneDesiredConfig, error) {
	patches, err := normalizeSceneDesiredPatches(patches)
	if err != nil {
		return nil, err
	}
	ids := sceneDesiredPatchIDs(patches)
	if err := s.sceneDesiredRepo.UpdateManyAtomic(ctx, ids, func(locked map[string]*entity.UESceneDesiredModel) ([]entity.UESceneDesiredModel, error) {
		missing := make([]string, 0)
		models := make([]entity.UESceneDesiredModel, 0, len(patches))
		for _, patch := range patches {
			model, ok := locked[patch.InstanceID]
			if !ok || !model.Enabled {
				missing = append(missing, patch.InstanceID)
				continue
			}
			if err := applySceneDesiredPatch(model, patch, false); err != nil {
				return nil, err
			}
			models = append(models, *model)
		}
		if len(missing) > 0 {
			return nil, &MissingSceneDesiredModelsError{InstanceIDs: missing}
		}
		return models, nil
	}); err != nil {
		return nil, err
	}
	return s.buildLegacySceneDesiredConfig(ctx)
}

func (s *ueConfigService) UpsertSceneDesiredModels(ctx context.Context, patches []SceneDesiredModelPatch) (*ue.SceneDesiredConfig, error) {
	patches, err := normalizeSceneDesiredPatches(patches)
	if err != nil {
		return nil, err
	}
	ids := sceneDesiredPatchIDs(patches)
	existing, err := s.sceneDesiredRepo.ListByInstanceIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := sceneDesiredModelsByID(existing)
	maxSortOrder, err := s.sceneDesiredRepo.MaxSortOrder(ctx)
	if err != nil {
		return nil, err
	}
	models := make([]entity.UESceneDesiredModel, 0, len(patches))
	newIndex := 0
	for _, patch := range patches {
		model, exists := byID[patch.InstanceID]
		if !exists {
			model = entity.UESceneDesiredModel{
				InstanceID: patch.InstanceID,
				Format:     "udatasmith",
				Visible:    true,
				ScaleX:     1,
				ScaleY:     1,
				ScaleZ:     1,
				Enabled:    true,
				SortOrder:  maxSortOrder + 1 + newIndex,
			}
			newIndex++
		}
		if err := applySceneDesiredPatch(&model, patch, !exists); err != nil {
			return nil, err
		}
		model.Enabled = true
		models = append(models, model)
	}
	if err := s.sceneDesiredRepo.UpsertMany(ctx, models); err != nil {
		return nil, err
	}
	return s.buildLegacySceneDesiredConfig(ctx)
}

func (s *ueConfigService) DeleteSceneDesiredModels(ctx context.Context, ids []string) (*ue.SceneDesiredConfig, error) {
	normalized, err := normalizeInstanceIDs(ids)
	if err != nil {
		return nil, err
	}
	if err := s.sceneDesiredRepo.SoftDeleteByInstanceIDs(ctx, normalized); err != nil {
		return nil, err
	}
	return s.buildLegacySceneDesiredConfig(ctx)
}

func (s *ueConfigService) buildLegacySceneDesiredConfig(ctx context.Context) (*ue.SceneDesiredConfig, error) {
	models, err := s.sceneDesiredRepo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	return mapper.UESceneDesiredModelsToConfig(models), nil
}

func normalizeSceneDesiredPatches(patches []SceneDesiredModelPatch) ([]SceneDesiredModelPatch, error) {
	out := make([]SceneDesiredModelPatch, 0, len(patches))
	seen := make(map[string]struct{}, len(patches))
	for _, patch := range patches {
		patch.InstanceID = strings.TrimSpace(patch.InstanceID)
		if patch.InstanceID == "" {
			return nil, invalidScenePatch("instance_id is required")
		}
		if _, ok := seen[patch.InstanceID]; ok {
			continue
		}
		seen[patch.InstanceID] = struct{}{}
		out = append(out, patch)
	}
	if len(out) == 0 {
		return nil, invalidScenePatch("models are required")
	}
	return out, nil
}

func sceneDesiredPatchIDs(patches []SceneDesiredModelPatch) []string {
	ids := make([]string, 0, len(patches))
	for _, patch := range patches {
		ids = append(ids, patch.InstanceID)
	}
	return ids
}

func sceneDesiredModelsByID(models []entity.UESceneDesiredModel) map[string]entity.UESceneDesiredModel {
	out := make(map[string]entity.UESceneDesiredModel, len(models))
	for _, model := range models {
		out[model.InstanceID] = model
	}
	return out
}

func applySceneDesiredPatch(model *entity.UESceneDesiredModel, patch SceneDesiredModelPatch, requireComplete bool) error {
	if patch.ModelID != nil {
		model.ModelID = strings.TrimSpace(*patch.ModelID)
	}
	if patch.URI != nil {
		model.URI = strings.TrimSpace(*patch.URI)
	}
	if patch.Format != nil {
		model.Format = strings.TrimSpace(*patch.Format)
	}
	if model.Format == "" {
		model.Format = "udatasmith"
	}
	if patch.SourcePath != nil {
		model.SourcePath = strings.TrimSpace(*patch.SourcePath)
	}
	if patch.DisplayName != nil {
		model.DisplayName = strings.TrimSpace(*patch.DisplayName)
	}
	if patch.BoundsSize != nil {
		model.BoundsX = float64Ptr(patch.BoundsSize.X)
		model.BoundsY = float64Ptr(patch.BoundsSize.Y)
		model.BoundsZ = float64Ptr(patch.BoundsSize.Z)
	}
	if patch.Visible != nil {
		model.Visible = *patch.Visible
	}
	if patch.Position != nil {
		model.PositionX = patch.Position.X
		model.PositionY = patch.Position.Y
		model.PositionZ = patch.Position.Z
	}
	if patch.Rotation != nil {
		model.RotationYaw = patch.Rotation.Yaw
		model.RotationPitch = patch.Rotation.Pitch
		model.RotationRoll = patch.Rotation.Roll
	}
	if patch.Scale != nil {
		model.ScaleX = patch.Scale.X
		model.ScaleY = patch.Scale.Y
		model.ScaleZ = patch.Scale.Z
	}
	if patch.ConfigJSON != nil {
		raw, err := json.Marshal(patch.ConfigJSON)
		if err != nil {
			return err
		}
		model.ConfigJSON = string(raw)
	}

	if requireComplete {
		if model.ModelID == "" {
			return invalidScenePatch("model_id is required")
		}
		if model.URI == "" {
			return invalidScenePatch("uri is required")
		}
		if model.SourcePath == "" {
			return invalidScenePatch("source_path is required")
		}
		if model.DisplayName == "" {
			return invalidScenePatch("display_name is required")
		}
		if patch.Visible == nil {
			return invalidScenePatch("visible is required")
		}
	}
	if model.ModelID == "" || model.URI == "" || model.Format == "" || model.SourcePath == "" || model.DisplayName == "" {
		return invalidScenePatch("model_id, uri, format, source_path and display_name must be non-empty")
	}
	return nil
}

func float64Ptr(v float64) *float64 {
	return &v
}

func invalidScenePatch(message string) error {
	return fmt.Errorf("%w: %s", ErrInvalidSceneDesiredPatch, message)
}
