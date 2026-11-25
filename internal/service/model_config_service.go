package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"dcx-gateway/internal/model/entity"
	"dcx-gateway/internal/model/protocol/ue"
	"dcx-gateway/internal/repository"
	"dcx-gateway/internal/service/mapper"
)

var (
	ErrInvalidModelConfigPatch = errors.New("invalid model config patch")
	ErrModelConfigConflict     = errors.New("model config conflict")
)

type ModelTypePatch struct {
	ModelID string

	URI                      *string
	Format                   *string
	SourcePath               *string
	BaseBoundsSize           *ue.Vector3
	BoundsNegativeExtentAxis *int
	BoundsAxisSet            bool
	Enabled                  *bool
}

type ModelInstancePatch struct {
	InstanceID string

	ModelID     *string
	DisplayName *string
	SortOrder   *int
	Enabled     *bool
}

type ModelPlacementPatch struct {
	InstanceID string

	Visible  *bool
	Position *ue.Vector3
	Rotation *ue.Rotation
	Scale    *ue.Vector3
}

func (s *ueConfigService) BuildModelTypes(ctx context.Context) (*ue.ModelTypesConfig, error) {
	models, err := s.modelTypeRepo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	return mapper.UEModelTypesToConfig(models), nil
}

func (s *ueConfigService) CreateModelTypes(ctx context.Context, patches []ModelTypePatch) (*ue.ModelTypesConfig, error) {
	patches, err := normalizeModelTypePatches(patches, true)
	if err != nil {
		return nil, err
	}
	ids := modelTypePatchIDs(patches)
	if err := s.modelTypeRepo.CreateManyAtomic(ctx, ids, func(existing map[string]*entity.UEModelType) ([]entity.UEModelType, error) {
		out := make([]entity.UEModelType, 0, len(patches))
		for _, patch := range patches {
			next := modelTypeFromCreatePatch(patch)
			if current, ok := existing[patch.ModelID]; ok {
				if !sameModelTypeForCreate(*current, next) {
					return nil, conflict("model_id already exists with different content: " + patch.ModelID)
				}
				continue
			}
			out = append(out, next)
		}
		return out, nil
	}); err != nil {
		return nil, err
	}
	return s.BuildModelTypes(ctx)
}

func (s *ueConfigService) UpdateModelTypes(ctx context.Context, patches []ModelTypePatch) (*ue.ModelTypesConfig, error) {
	patches, err := normalizeModelTypePatches(patches, false)
	if err != nil {
		return nil, err
	}
	ids := modelTypePatchIDs(patches)
	if err := s.modelTypeRepo.UpdateManyAtomic(ctx, ids, func(existing map[string]*entity.UEModelType) ([]entity.UEModelType, error) {
		out := make([]entity.UEModelType, 0, len(patches))
		for _, patch := range patches {
			current, ok := existing[patch.ModelID]
			if !ok {
				return nil, repository.ErrNotFound
			}
			applyModelTypePatch(current, patch)
			out = append(out, *current)
		}
		return out, nil
	}); err != nil {
		return nil, err
	}
	return s.BuildModelTypes(ctx)
}

func (s *ueConfigService) DeleteModelTypes(ctx context.Context, ids []string) (*ue.ModelTypesConfig, error) {
	normalized, err := normalizeIDs(ids, "model_ids")
	if err != nil {
		return nil, err
	}
	refs, err := s.modelInstanceRepo.CountEnabledByModelIDs(ctx, normalized)
	if err != nil {
		return nil, err
	}
	for _, id := range normalized {
		if refs[id] > 0 {
			return nil, conflict("model_id is still referenced by enabled instances: " + id)
		}
	}
	if err := s.modelTypeRepo.UpdateManyAtomic(ctx, normalized, func(existing map[string]*entity.UEModelType) ([]entity.UEModelType, error) {
		out := make([]entity.UEModelType, 0, len(existing))
		for _, id := range normalized {
			current, ok := existing[id]
			if !ok {
				continue
			}
			current.Enabled = false
			out = append(out, *current)
		}
		return out, nil
	}); err != nil {
		return nil, err
	}
	return s.BuildModelTypes(ctx)
}

func (s *ueConfigService) BuildModelTable(ctx context.Context) (*ue.ModelTable, error) {
	instances, err := s.modelInstanceRepo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	return mapper.UEModelInstancesToTable(instances), nil
}

func (s *ueConfigService) CreateModelTableEntries(ctx context.Context, patches []ModelInstancePatch) (*ue.ModelTable, error) {
	patches, err := normalizeModelInstancePatches(patches, true)
	if err != nil {
		return nil, err
	}
	if err := s.ensureEnabledModelTypes(ctx, modelIDsFromInstancePatches(patches)); err != nil {
		return nil, err
	}
	ids := modelInstancePatchIDs(patches)
	if err := s.modelInstanceRepo.CreateManyAtomic(ctx, ids, func(existing map[string]*entity.UEModelInstance) ([]entity.UEModelInstance, []entity.UEModelInstancePlacement, error) {
		instances := make([]entity.UEModelInstance, 0, len(patches))
		placements := make([]entity.UEModelInstancePlacement, 0, len(patches))
		for _, patch := range patches {
			next := modelInstanceFromCreatePatch(patch)
			if current, ok := existing[patch.InstanceID]; ok {
				if !sameModelInstanceForCreate(*current, next) {
					return nil, nil, conflict("instance_id already exists with different content: " + patch.InstanceID)
				}
				continue
			}
			instances = append(instances, next)
			placements = append(placements, defaultPlacement(patch.InstanceID))
		}
		return instances, placements, nil
	}); err != nil {
		return nil, err
	}
	return s.BuildModelTable(ctx)
}

func (s *ueConfigService) UpdateModelTableEntries(ctx context.Context, patches []ModelInstancePatch) (*ue.ModelTable, error) {
	patches, err := normalizeModelInstancePatches(patches, false)
	if err != nil {
		return nil, err
	}
	if err := s.ensureEnabledModelTypes(ctx, modelIDsFromInstancePatches(patches)); err != nil {
		return nil, err
	}
	ids := modelInstancePatchIDs(patches)
	if err := s.modelInstanceRepo.UpdateManyAtomic(ctx, ids, func(existing map[string]*entity.UEModelInstance) ([]entity.UEModelInstance, error) {
		out := make([]entity.UEModelInstance, 0, len(patches))
		for _, patch := range patches {
			current, ok := existing[patch.InstanceID]
			if !ok {
				return nil, repository.ErrNotFound
			}
			applyModelInstancePatch(current, patch)
			out = append(out, *current)
		}
		return out, nil
	}); err != nil {
		return nil, err
	}
	return s.BuildModelTable(ctx)
}

func (s *ueConfigService) DeleteModelTableEntries(ctx context.Context, ids []string) (*ue.ModelTable, error) {
	normalized, err := normalizeIDs(ids, "instance_ids")
	if err != nil {
		return nil, err
	}
	if err := s.modelInstanceRepo.UpdateManyAtomic(ctx, normalized, func(existing map[string]*entity.UEModelInstance) ([]entity.UEModelInstance, error) {
		out := make([]entity.UEModelInstance, 0, len(existing))
		for _, id := range normalized {
			current, ok := existing[id]
			if !ok {
				continue
			}
			current.Enabled = false
			out = append(out, *current)
		}
		return out, nil
	}); err != nil {
		return nil, err
	}
	return s.BuildModelTable(ctx)
}

func (s *ueConfigService) BuildStructuredModelConfig(ctx context.Context) (*ue.StructuredModelConfig, error) {
	instances, placements, err := s.enabledInstancesAndPlacements(ctx)
	if err != nil {
		return nil, err
	}
	return mapper.UEPlacementsToStructuredModelConfig(instances, placements), nil
}

func (s *ueConfigService) BuildRuntimeModelConfig(ctx context.Context) (*ue.ModelConfig, error) {
	types, instances, placements, err := s.enabledModelSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	return mapper.UEModelsToRuntimeModelConfig(types, instances, placements), nil
}

func (s *ueConfigService) BuildSceneDesiredConfig(ctx context.Context) (*ue.SceneDesiredConfig, error) {
	types, instances, placements, err := s.enabledModelSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	return mapper.UEModelsToSceneDesiredConfig(types, instances, placements), nil
}

func (s *ueConfigService) UpdateModelPlacements(ctx context.Context, patches []ModelPlacementPatch) (*ue.StructuredModelConfig, error) {
	patches, err := normalizeModelPlacementPatches(patches)
	if err != nil {
		return nil, err
	}
	if err := s.ensureEnabledModelInstances(ctx, modelPlacementPatchIDs(patches)); err != nil {
		return nil, err
	}
	ids := modelPlacementPatchIDs(patches)
	if err := s.placementRepo.UpdateManyAtomic(ctx, ids, func(existing map[string]*entity.UEModelInstancePlacement) ([]entity.UEModelInstancePlacement, error) {
		out := make([]entity.UEModelInstancePlacement, 0, len(patches))
		for _, patch := range patches {
			current, ok := existing[patch.InstanceID]
			if !ok {
				value := defaultPlacement(patch.InstanceID)
				current = &value
			}
			applyModelPlacementPatch(current, patch)
			out = append(out, *current)
		}
		return out, nil
	}); err != nil {
		return nil, err
	}
	return s.BuildStructuredModelConfig(ctx)
}

func (s *ueConfigService) DeleteModelPlacements(ctx context.Context, ids []string) (*ue.StructuredModelConfig, error) {
	normalized, err := normalizeIDs(ids, "instance_ids")
	if err != nil {
		return nil, err
	}
	if err := s.placementRepo.DeleteByInstanceIDsAtomic(ctx, normalized); err != nil {
		return nil, err
	}
	return s.BuildStructuredModelConfig(ctx)
}

func (s *ueConfigService) enabledModelSnapshot(ctx context.Context) ([]entity.UEModelType, []entity.UEModelInstance, []entity.UEModelInstancePlacement, error) {
	types, err := s.modelTypeRepo.ListEnabled(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	instances, placements, err := s.enabledInstancesAndPlacements(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	return types, instances, placements, nil
}

func (s *ueConfigService) enabledInstancesAndPlacements(ctx context.Context) ([]entity.UEModelInstance, []entity.UEModelInstancePlacement, error) {
	instances, err := s.modelInstanceRepo.ListEnabled(ctx)
	if err != nil {
		return nil, nil, err
	}
	ids := make([]string, 0, len(instances))
	for _, instance := range instances {
		ids = append(ids, instance.InstanceID)
	}
	placements, err := s.placementRepo.ListByInstanceIDs(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	return instances, placements, nil
}

func (s *ueConfigService) ensureEnabledModelTypes(ctx context.Context, ids []string) error {
	ids, err := normalizeIDsAllowEmpty(ids)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	models, err := s.modelTypeRepo.ListByModelIDs(ctx, ids)
	if err != nil {
		return err
	}
	enabled := make(map[string]bool, len(models))
	for _, model := range models {
		enabled[model.ModelID] = model.Enabled
	}
	for _, id := range ids {
		if !enabled[id] {
			return repository.ErrNotFound
		}
	}
	return nil
}

func (s *ueConfigService) ensureEnabledModelInstances(ctx context.Context, ids []string) error {
	models, err := s.modelInstanceRepo.ListByInstanceIDs(ctx, ids)
	if err != nil {
		return err
	}
	enabled := make(map[string]bool, len(models))
	for _, model := range models {
		enabled[model.InstanceID] = model.Enabled
	}
	for _, id := range ids {
		if !enabled[id] {
			return repository.ErrNotFound
		}
	}
	return nil
}

func normalizeModelTypePatches(patches []ModelTypePatch, requireCreateFields bool) ([]ModelTypePatch, error) {
	out := make([]ModelTypePatch, 0, len(patches))
	seen := make(map[string]struct{}, len(patches))
	for _, patch := range patches {
		patch.ModelID = strings.TrimSpace(patch.ModelID)
		if patch.ModelID == "" {
			return nil, invalidModelPatch("model_id is required")
		}
		if _, ok := seen[patch.ModelID]; ok {
			continue
		}
		seen[patch.ModelID] = struct{}{}
		if requireCreateFields {
			if patch.URI == nil || strings.TrimSpace(*patch.URI) == "" ||
				patch.SourcePath == nil || strings.TrimSpace(*patch.SourcePath) == "" ||
				patch.BaseBoundsSize == nil {
				return nil, invalidModelPatch("model_id, uri, source_path and base_bounds_size are required")
			}
		}
		out = append(out, patch)
	}
	if len(out) == 0 {
		return nil, invalidModelPatch("model_types are required")
	}
	return out, nil
}

func normalizeModelInstancePatches(patches []ModelInstancePatch, requireCreateFields bool) ([]ModelInstancePatch, error) {
	out := make([]ModelInstancePatch, 0, len(patches))
	seen := make(map[string]struct{}, len(patches))
	for _, patch := range patches {
		patch.InstanceID = strings.TrimSpace(patch.InstanceID)
		if patch.InstanceID == "" {
			return nil, invalidModelPatch("instance_id is required")
		}
		if _, ok := seen[patch.InstanceID]; ok {
			continue
		}
		seen[patch.InstanceID] = struct{}{}
		if requireCreateFields && (patch.ModelID == nil || strings.TrimSpace(*patch.ModelID) == "") {
			return nil, invalidModelPatch("model_id is required")
		}
		out = append(out, patch)
	}
	if len(out) == 0 {
		return nil, invalidModelPatch("entries are required")
	}
	return out, nil
}

func normalizeModelPlacementPatches(patches []ModelPlacementPatch) ([]ModelPlacementPatch, error) {
	out := make([]ModelPlacementPatch, 0, len(patches))
	seen := make(map[string]struct{}, len(patches))
	for _, patch := range patches {
		patch.InstanceID = strings.TrimSpace(patch.InstanceID)
		if patch.InstanceID == "" {
			return nil, invalidModelPatch("instance_id is required")
		}
		if _, ok := seen[patch.InstanceID]; ok {
			continue
		}
		seen[patch.InstanceID] = struct{}{}
		if patch.Visible == nil && patch.Position == nil && patch.Rotation == nil && patch.Scale == nil {
			return nil, invalidModelPatch("placement patch is empty")
		}
		out = append(out, patch)
	}
	if len(out) == 0 {
		return nil, invalidModelPatch("placements are required")
	}
	return out, nil
}

func modelTypeFromCreatePatch(patch ModelTypePatch) entity.UEModelType {
	format := "udatasmith"
	if patch.Format != nil && strings.TrimSpace(*patch.Format) != "" {
		format = strings.TrimSpace(*patch.Format)
	}
	enabled := true
	if patch.Enabled != nil {
		enabled = *patch.Enabled
	}
	return entity.UEModelType{
		ModelID:                  patch.ModelID,
		URI:                      strings.TrimSpace(*patch.URI),
		Format:                   format,
		SourcePath:               strings.TrimSpace(*patch.SourcePath),
		BaseBoundsX:              patch.BaseBoundsSize.X,
		BaseBoundsY:              patch.BaseBoundsSize.Y,
		BaseBoundsZ:              patch.BaseBoundsSize.Z,
		BoundsNegativeExtentAxis: patch.BoundsNegativeExtentAxis,
		Enabled:                  enabled,
	}
}

func applyModelTypePatch(model *entity.UEModelType, patch ModelTypePatch) {
	if patch.URI != nil {
		model.URI = strings.TrimSpace(*patch.URI)
	}
	if patch.Format != nil {
		model.Format = strings.TrimSpace(*patch.Format)
	}
	if patch.SourcePath != nil {
		model.SourcePath = strings.TrimSpace(*patch.SourcePath)
	}
	if patch.BaseBoundsSize != nil {
		model.BaseBoundsX = patch.BaseBoundsSize.X
		model.BaseBoundsY = patch.BaseBoundsSize.Y
		model.BaseBoundsZ = patch.BaseBoundsSize.Z
	}
	if patch.BoundsAxisSet {
		model.BoundsNegativeExtentAxis = patch.BoundsNegativeExtentAxis
	}
	if patch.Enabled != nil {
		model.Enabled = *patch.Enabled
	}
}

func modelInstanceFromCreatePatch(patch ModelInstancePatch) entity.UEModelInstance {
	displayName := ""
	if patch.DisplayName != nil {
		displayName = strings.TrimSpace(*patch.DisplayName)
	}
	sortOrder := 0
	if patch.SortOrder != nil {
		sortOrder = *patch.SortOrder
	}
	enabled := true
	if patch.Enabled != nil {
		enabled = *patch.Enabled
	}
	return entity.UEModelInstance{
		InstanceID:  patch.InstanceID,
		ModelID:     strings.TrimSpace(*patch.ModelID),
		DisplayName: displayName,
		SortOrder:   sortOrder,
		Enabled:     enabled,
	}
}

func applyModelInstancePatch(model *entity.UEModelInstance, patch ModelInstancePatch) {
	if patch.ModelID != nil {
		model.ModelID = strings.TrimSpace(*patch.ModelID)
	}
	if patch.DisplayName != nil {
		model.DisplayName = strings.TrimSpace(*patch.DisplayName)
	}
	if patch.SortOrder != nil {
		model.SortOrder = *patch.SortOrder
	}
	if patch.Enabled != nil {
		model.Enabled = *patch.Enabled
	}
}

func applyModelPlacementPatch(model *entity.UEModelInstancePlacement, patch ModelPlacementPatch) {
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
}

func defaultPlacement(instanceID string) entity.UEModelInstancePlacement {
	return entity.UEModelInstancePlacement{
		InstanceID: instanceID,
		Visible:    true,
		ScaleX:     1,
		ScaleY:     1,
		ScaleZ:     1,
	}
}

func sameModelTypeForCreate(a, b entity.UEModelType) bool {
	return a.ModelID == b.ModelID &&
		a.URI == b.URI &&
		a.Format == b.Format &&
		a.SourcePath == b.SourcePath &&
		floatEqual(a.BaseBoundsX, b.BaseBoundsX) &&
		floatEqual(a.BaseBoundsY, b.BaseBoundsY) &&
		floatEqual(a.BaseBoundsZ, b.BaseBoundsZ) &&
		intPtrEqual(a.BoundsNegativeExtentAxis, b.BoundsNegativeExtentAxis) &&
		a.Enabled == b.Enabled
}

func sameModelInstanceForCreate(a, b entity.UEModelInstance) bool {
	return a.InstanceID == b.InstanceID &&
		a.ModelID == b.ModelID &&
		a.DisplayName == b.DisplayName &&
		a.SortOrder == b.SortOrder &&
		a.Enabled == b.Enabled
}

func modelTypePatchIDs(patches []ModelTypePatch) []string {
	ids := make([]string, 0, len(patches))
	for _, patch := range patches {
		ids = append(ids, patch.ModelID)
	}
	return ids
}

func modelInstancePatchIDs(patches []ModelInstancePatch) []string {
	ids := make([]string, 0, len(patches))
	for _, patch := range patches {
		ids = append(ids, patch.InstanceID)
	}
	return ids
}

func modelPlacementPatchIDs(patches []ModelPlacementPatch) []string {
	ids := make([]string, 0, len(patches))
	for _, patch := range patches {
		ids = append(ids, patch.InstanceID)
	}
	return ids
}

func modelIDsFromInstancePatches(patches []ModelInstancePatch) []string {
	ids := make([]string, 0, len(patches))
	for _, patch := range patches {
		if patch.ModelID != nil {
			ids = append(ids, *patch.ModelID)
		}
	}
	return ids
}

func normalizeIDs(values []string, field string) ([]string, error) {
	out, err := normalizeIDsAllowEmpty(values)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, invalidModelPatch(field + " is required")
	}
	return out, nil
}

func normalizeIDsAllowEmpty(values []string) ([]string, error) {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		id := strings.TrimSpace(value)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

func floatEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.000001
}

func intPtrEqual(a, b *int) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func invalidModelPatch(message string) error {
	return fmt.Errorf("%w: %s", ErrInvalidModelConfigPatch, message)
}

func conflict(message string) error {
	return fmt.Errorf("%w: %s", ErrModelConfigConflict, message)
}
