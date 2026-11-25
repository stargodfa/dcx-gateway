package service

import (
	"context"
	"errors"
	"testing"

	"dcx-gateway/internal/model/entity"
	"dcx-gateway/internal/model/protocol/ue"
	"dcx-gateway/internal/repository"
)

func TestModelTypesCreateIsIdempotentAndConflictsOnDifferentContent(t *testing.T) {
	repos := newMemoryModelRepos()
	svc := newUEConfigServiceForModels(repos)
	patch := ModelTypePatch{
		ModelID:        "room",
		URI:            stringPtr("file://room"),
		Format:         stringPtr("udatasmith"),
		SourcePath:     stringPtr("./room"),
		BaseBoundsSize: &ue.Vector3{X: 10, Y: 20, Z: 30},
	}

	if _, err := svc.CreateModelTypes(context.Background(), []ModelTypePatch{patch}); err != nil {
		t.Fatalf("CreateModelTypes returned error: %v", err)
	}
	if _, err := svc.CreateModelTypes(context.Background(), []ModelTypePatch{patch}); err != nil {
		t.Fatalf("same CreateModelTypes should be idempotent: %v", err)
	}

	changed := patch
	changed.URI = stringPtr("file://changed")
	if _, err := svc.CreateModelTypes(context.Background(), []ModelTypePatch{changed}); !errors.Is(err, ErrModelConfigConflict) {
		t.Fatalf("changed create err = %v, want conflict", err)
	}
}

func TestModelTypesDeleteReferencedEnabledInstanceConflicts(t *testing.T) {
	repos := newMemoryModelRepos()
	repos.types["room"] = entity.UEModelType{ModelID: "room", URI: "file://room", Format: "udatasmith", SourcePath: "./room", BaseBoundsX: 1, BaseBoundsY: 2, BaseBoundsZ: 3, Enabled: true}
	repos.instances["room_1"] = entity.UEModelInstance{InstanceID: "room_1", ModelID: "room", Enabled: true}
	svc := newUEConfigServiceForModels(repos)

	if _, err := svc.DeleteModelTypes(context.Background(), []string{"room"}); !errors.Is(err, ErrModelConfigConflict) {
		t.Fatalf("DeleteModelTypes err = %v, want conflict", err)
	}
}

func TestModelTableCreateRequiresModelTypeAndCreatesDefaultPlacement(t *testing.T) {
	repos := newMemoryModelRepos()
	repos.types["room"] = entity.UEModelType{ModelID: "room", URI: "file://room", Format: "udatasmith", SourcePath: "./room", BaseBoundsX: 1, BaseBoundsY: 2, BaseBoundsZ: 3, Enabled: true}
	svc := newUEConfigServiceForModels(repos)

	if _, err := svc.CreateModelTableEntries(context.Background(), []ModelInstancePatch{
		{InstanceID: "room_1", ModelID: stringPtr("room"), DisplayName: stringPtr("room 1")},
	}); err != nil {
		t.Fatalf("CreateModelTableEntries returned error: %v", err)
	}
	if _, ok := repos.placements["room_1"]; !ok {
		t.Fatalf("default placement was not created")
	}
	if _, err := svc.CreateModelTableEntries(context.Background(), []ModelInstancePatch{
		{InstanceID: "missing_1", ModelID: stringPtr("missing")},
	}); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("missing model_id err = %v, want not found", err)
	}
}

func TestModelTableDeleteDoesNotModifyInitConfig(t *testing.T) {
	repos := newMemoryModelRepos()
	repos.instances["room_1"] = entity.UEModelInstance{InstanceID: "room_1", ModelID: "room", Enabled: true}
	rawRepo := &memoryAtomicRawConfigRepo{raw: []byte(initConfigRaw)}
	svc := newUEConfigServiceForModelsWithRaw(repos, rawRepo)

	if _, err := svc.DeleteModelTableEntries(context.Background(), []string{"room_1"}); err != nil {
		t.Fatalf("DeleteModelTableEntries returned error: %v", err)
	}
	if rawRepo.updateCalls != 0 {
		t.Fatalf("init_config should not be modified, updateCalls=%d", rawRepo.updateCalls)
	}
}

func TestModelTableUpdateDisplayNamePreservesOtherFields(t *testing.T) {
	repos := newMemoryModelRepos()
	repos.types["room"] = entity.UEModelType{ModelID: "room", URI: "file://room", Format: "udatasmith", SourcePath: "./room", BaseBoundsX: 1, BaseBoundsY: 2, BaseBoundsZ: 3, Enabled: true}
	repos.instances["room_1"] = entity.UEModelInstance{InstanceID: "room_1", ModelID: "room", DisplayName: "old", SortOrder: 7, Enabled: true}
	svc := newUEConfigServiceForModels(repos)

	if _, err := svc.UpdateModelTableEntries(context.Background(), []ModelInstancePatch{
		{InstanceID: "room_1", DisplayName: stringPtr("new name")},
	}); err != nil {
		t.Fatalf("UpdateModelTableEntries returned error: %v", err)
	}
	got := repos.instances["room_1"]
	if got.DisplayName != "new name" {
		t.Fatalf("display_name = %q, want new name", got.DisplayName)
	}
	if got.ModelID != "room" || got.SortOrder != 7 || !got.Enabled {
		t.Fatalf("unexpected field mutation: %#v", got)
	}

	if _, err := svc.UpdateModelTableEntries(context.Background(), []ModelInstancePatch{
		{InstanceID: "missing", DisplayName: stringPtr("x")},
	}); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("missing instance err = %v, want not found", err)
	}
}

func TestRuntimeModelConfigComputesBoundsAndFiltersDisabledInstances(t *testing.T) {
	repos := newMemoryModelRepos()
	repos.types["test_model"] = entity.UEModelType{ModelID: "test_model", URI: "file://test", Format: "udatasmith", SourcePath: "./test", BaseBoundsX: 770, BaseBoundsY: 1130, BaseBoundsZ: 600, Enabled: true}
	repos.instances["test_model_1"] = entity.UEModelInstance{InstanceID: "test_model_1", ModelID: "test_model", Enabled: true}
	repos.instances["test_model_disabled"] = entity.UEModelInstance{InstanceID: "test_model_disabled", ModelID: "test_model", Enabled: false}
	repos.placements["test_model_1"] = entity.UEModelInstancePlacement{InstanceID: "test_model_1", Visible: true, ScaleX: 0.1, ScaleY: -0.1, ScaleZ: 0.1}
	svc := newUEConfigServiceForModels(repos)

	cfg, err := svc.BuildRuntimeModelConfig(context.Background())
	if err != nil {
		t.Fatalf("BuildRuntimeModelConfig returned error: %v", err)
	}
	state, ok := cfg.LastLoadByInstanceID["test_model_1"]
	if !ok {
		t.Fatalf("test_model_1 missing from runtime config")
	}
	if state.BoundsSize.X != 77 || state.BoundsSize.Y != 113 || state.BoundsSize.Z != 60 {
		t.Fatalf("bounds_size = %#v, want 77/113/60", state.BoundsSize)
	}
	if _, ok := cfg.LastLoadByInstanceID["test_model_disabled"]; ok {
		t.Fatalf("disabled instance should not be output")
	}
}

func TestSceneDesiredGeneratedUsesNormalizedTablesNotLegacyRepo(t *testing.T) {
	repos := newMemoryModelRepos()
	repos.types["room"] = entity.UEModelType{ModelID: "room", URI: "file://room", Format: "udatasmith", SourcePath: "./room", BaseBoundsX: 10, BaseBoundsY: 20, BaseBoundsZ: 30, Enabled: true}
	repos.instances["room_1"] = entity.UEModelInstance{InstanceID: "room_1", ModelID: "room", DisplayName: "room 1", Enabled: true}
	repos.placements["room_1"] = entity.UEModelInstancePlacement{InstanceID: "room_1", Visible: true, ScaleX: 2, ScaleY: 1, ScaleZ: 1}
	legacy := &countingSceneDesiredRepo{}
	svc := newUEConfigServiceForModelsAndLegacy(repos, legacy)

	cfg, err := svc.BuildSceneDesiredConfig(context.Background())
	if err != nil {
		t.Fatalf("BuildSceneDesiredConfig returned error: %v", err)
	}
	if legacy.listEnabledCalls != 0 {
		t.Fatalf("scene_desired generated should not read legacy repo")
	}
	if len(cfg.SceneModelsDesired) != 1 {
		t.Fatalf("len(scene_models_desired) = %d, want 1", len(cfg.SceneModelsDesired))
	}
	item := cfg.SceneModelsDesired[0]
	if item.URI != "file://room" || item.BoundsSize.X != 20 {
		t.Fatalf("generated item = %#v", item)
	}
}

func TestUpdateModelPlacementsRejectsEmptyPatchAndCreatesMissingPlacement(t *testing.T) {
	repos := newMemoryModelRepos()
	repos.instances["room_1"] = entity.UEModelInstance{InstanceID: "room_1", ModelID: "room", Enabled: true}
	svc := newUEConfigServiceForModels(repos)
	visible := false

	cfg, err := svc.UpdateModelPlacements(context.Background(), []ModelPlacementPatch{
		{InstanceID: "room_1", Visible: &visible},
	})
	if err != nil {
		t.Fatalf("UpdateModelPlacements returned error: %v", err)
	}
	if cfg.Placements["room_1"].Visible != false {
		t.Fatalf("visible = true, want false")
	}
	if _, ok := repos.placements["room_1"]; !ok {
		t.Fatalf("missing placement should be created")
	}
	state := cfg.Placements["room_1"]
	if state.Transform.Position.X != 0 || state.Transform.Position.Y != 0 || state.Transform.Position.Z != 0 {
		t.Fatalf("missing placement position = %#v, want zero", state.Transform.Position)
	}
	if state.Transform.Rotation.Yaw != 0 || state.Transform.Rotation.Pitch != 0 || state.Transform.Rotation.Roll != 0 {
		t.Fatalf("missing placement rotation = %#v, want zero", state.Transform.Rotation)
	}
	if state.Transform.Scale.X != 1 || state.Transform.Scale.Y != 1 || state.Transform.Scale.Z != 1 {
		t.Fatalf("missing placement scale = %#v, want default 1", state.Transform.Scale)
	}
}

func TestUpdateModelPlacementsPreservesLatestFieldsAcrossPartialPatches(t *testing.T) {
	repos := newMemoryModelRepos()
	repos.instances["discussion_room_1"] = entity.UEModelInstance{
		InstanceID: "discussion_room_1",
		ModelID:    "discussion_room",
		Enabled:    true,
	}
	repos.placements["discussion_room_1"] = entity.UEModelInstancePlacement{
		InstanceID:    "discussion_room_1",
		Visible:       true,
		PositionX:     1,
		PositionY:     2,
		PositionZ:     3,
		RotationYaw:   10,
		RotationPitch: 20,
		RotationRoll:  30,
		ScaleX:        1,
		ScaleY:        2,
		ScaleZ:        3,
	}
	svc := newUEConfigServiceForModels(repos)

	visible := false
	if _, err := svc.UpdateModelPlacements(context.Background(), []ModelPlacementPatch{
		{InstanceID: "discussion_room_1", Visible: &visible},
	}); err != nil {
		t.Fatalf("UpdateModelPlacements visible patch returned error: %v", err)
	}

	if _, err := svc.UpdateModelPlacements(context.Background(), []ModelPlacementPatch{
		{InstanceID: "discussion_room_1", Position: &ue.Vector3{X: 100, Y: 200, Z: 300}},
	}); err != nil {
		t.Fatalf("UpdateModelPlacements position patch returned error: %v", err)
	}

	got := repos.placements["discussion_room_1"]
	if got.Visible != false {
		t.Fatalf("visible = true, want false")
	}
	if got.PositionX != 100 || got.PositionY != 200 || got.PositionZ != 300 {
		t.Fatalf("position = %v/%v/%v, want 100/200/300", got.PositionX, got.PositionY, got.PositionZ)
	}
	if got.RotationYaw != 10 || got.RotationPitch != 20 || got.RotationRoll != 30 {
		t.Fatalf("rotation = %v/%v/%v, want 10/20/30", got.RotationYaw, got.RotationPitch, got.RotationRoll)
	}
	if got.ScaleX != 1 || got.ScaleY != 2 || got.ScaleZ != 3 {
		t.Fatalf("scale = %v/%v/%v, want 1/2/3", got.ScaleX, got.ScaleY, got.ScaleZ)
	}
}

func TestUpdateModelPlacementsMissingPlacementPreservesPriorPatch(t *testing.T) {
	repos := newMemoryModelRepos()
	repos.instances["discussion_room_1"] = entity.UEModelInstance{
		InstanceID: "discussion_room_1",
		ModelID:    "discussion_room",
		Enabled:    true,
	}
	svc := newUEConfigServiceForModels(repos)

	visible := false
	if _, err := svc.UpdateModelPlacements(context.Background(), []ModelPlacementPatch{
		{InstanceID: "discussion_room_1", Visible: &visible},
	}); err != nil {
		t.Fatalf("UpdateModelPlacements visible patch returned error: %v", err)
	}

	cfg, err := svc.UpdateModelPlacements(context.Background(), []ModelPlacementPatch{
		{InstanceID: "discussion_room_1", Position: &ue.Vector3{X: 100, Y: 200, Z: 300}},
	})
	if err != nil {
		t.Fatalf("UpdateModelPlacements position patch returned error: %v", err)
	}

	got := repos.placements["discussion_room_1"]
	if got.Visible {
		t.Fatalf("stored visible = true, want false from prior patch")
	}
	if got.PositionX != 100 || got.PositionY != 200 || got.PositionZ != 300 {
		t.Fatalf("stored position = %v/%v/%v, want 100/200/300", got.PositionX, got.PositionY, got.PositionZ)
	}
	state := cfg.Placements["discussion_room_1"]
	if state.Visible {
		t.Fatalf("structured visible = true, want false from prior patch")
	}
	if state.Transform.Position.X != 100 || state.Transform.Position.Y != 200 || state.Transform.Position.Z != 300 {
		t.Fatalf("structured position = %#v, want 100/200/300", state.Transform.Position)
	}
}

func TestUpdateModelPlacementsWritesFalseAndZeroValuesAcrossOutputs(t *testing.T) {
	repos := newMemoryModelRepos()
	repos.types["discussion_room"] = entity.UEModelType{
		ModelID:     "discussion_room",
		URI:         "file://discussion_room",
		Format:      "udatasmith",
		SourcePath:  "./discussion_room",
		BaseBoundsX: 10,
		BaseBoundsY: 20,
		BaseBoundsZ: 30,
		Enabled:     true,
	}
	repos.instances["discussion_room_1"] = entity.UEModelInstance{
		InstanceID:  "discussion_room_1",
		ModelID:     "discussion_room",
		DisplayName: "discussion room 1",
		Enabled:     true,
	}
	repos.placements["discussion_room_1"] = entity.UEModelInstancePlacement{
		InstanceID:    "discussion_room_1",
		Visible:       true,
		PositionX:     1,
		PositionY:     2,
		PositionZ:     3,
		RotationYaw:   10,
		RotationPitch: 20,
		RotationRoll:  30,
		ScaleX:        1,
		ScaleY:        1,
		ScaleZ:        1,
	}
	svc := newUEConfigServiceForModels(repos)

	visible := false
	cfg, err := svc.UpdateModelPlacements(context.Background(), []ModelPlacementPatch{
		{InstanceID: "discussion_room_1", Visible: &visible},
	})
	if err != nil {
		t.Fatalf("UpdateModelPlacements visible patch returned error: %v", err)
	}
	state := cfg.Placements["discussion_room_1"]
	if state.Visible {
		t.Fatalf("structured visible = true, want false")
	}
	if state.Transform.Position.X != 1 || state.Transform.Position.Y != 2 || state.Transform.Position.Z != 3 {
		t.Fatalf("position = %#v, want preserved 1/2/3", state.Transform.Position)
	}
	if state.Transform.Rotation.Yaw != 10 || state.Transform.Rotation.Pitch != 20 || state.Transform.Rotation.Roll != 30 {
		t.Fatalf("rotation = %#v, want preserved 10/20/30", state.Transform.Rotation)
	}
	if state.Transform.Scale.X != 1 || state.Transform.Scale.Y != 1 || state.Transform.Scale.Z != 1 {
		t.Fatalf("scale = %#v, want preserved 1/1/1", state.Transform.Scale)
	}
	runtime, err := svc.BuildRuntimeModelConfig(context.Background())
	if err != nil {
		t.Fatalf("BuildRuntimeModelConfig returned error: %v", err)
	}
	if runtime.LastLoadByInstanceID["discussion_room_1"].Visible {
		t.Fatalf("runtime visible = true, want false")
	}
	scene, err := svc.BuildSceneDesiredConfig(context.Background())
	if err != nil {
		t.Fatalf("BuildSceneDesiredConfig returned error: %v", err)
	}
	if len(scene.SceneModelsDesired) != 1 || scene.SceneModelsDesired[0].Visible {
		t.Fatalf("scene visible = %#v, want false", scene.SceneModelsDesired)
	}

	zeroVector := &ue.Vector3{}
	zeroRotation := &ue.Rotation{}
	cfg, err = svc.UpdateModelPlacements(context.Background(), []ModelPlacementPatch{
		{
			InstanceID: "discussion_room_1",
			Position:   zeroVector,
			Rotation:   zeroRotation,
			Scale:      zeroVector,
		},
	})
	if err != nil {
		t.Fatalf("UpdateModelPlacements zero transform patch returned error: %v", err)
	}
	state = cfg.Placements["discussion_room_1"]
	if state.Transform.Position.X != 0 || state.Transform.Position.Y != 0 || state.Transform.Position.Z != 0 {
		t.Fatalf("zero position = %#v, want 0/0/0", state.Transform.Position)
	}
	if state.Transform.Rotation.Yaw != 0 || state.Transform.Rotation.Pitch != 0 || state.Transform.Rotation.Roll != 0 {
		t.Fatalf("zero rotation = %#v, want 0/0/0", state.Transform.Rotation)
	}
	if state.Transform.Scale.X != 0 || state.Transform.Scale.Y != 0 || state.Transform.Scale.Z != 0 {
		t.Fatalf("zero scale = %#v, want 0/0/0", state.Transform.Scale)
	}
	runtime, err = svc.BuildRuntimeModelConfig(context.Background())
	if err != nil {
		t.Fatalf("BuildRuntimeModelConfig after zero patch returned error: %v", err)
	}
	if runtime.LastLoadByInstanceID["discussion_room_1"].BoundsSize.X != 0 ||
		runtime.LastLoadByInstanceID["discussion_room_1"].BoundsSize.Y != 0 ||
		runtime.LastLoadByInstanceID["discussion_room_1"].BoundsSize.Z != 0 {
		t.Fatalf("runtime bounds_size = %#v, want zero", runtime.LastLoadByInstanceID["discussion_room_1"].BoundsSize)
	}
}

func TestDeleteModelPlacementsIsIdempotentAndFallsBackToDefaultPlacement(t *testing.T) {
	repos := newMemoryModelRepos()
	repos.types["room"] = entity.UEModelType{ModelID: "room", URI: "file://room", Format: "udatasmith", SourcePath: "./room", BaseBoundsX: 10, BaseBoundsY: 20, BaseBoundsZ: 30, Enabled: true}
	repos.instances["room_1"] = entity.UEModelInstance{InstanceID: "room_1", ModelID: "room", DisplayName: "room", Enabled: true}
	repos.instances["room_2"] = entity.UEModelInstance{InstanceID: "room_2", ModelID: "room", DisplayName: "room 2", Enabled: true}
	repos.placements["room_1"] = entity.UEModelInstancePlacement{InstanceID: "room_1", Visible: false, PositionX: 1, PositionY: 2, PositionZ: 3, ScaleX: 2, ScaleY: 2, ScaleZ: 2}
	repos.placements["room_2"] = entity.UEModelInstancePlacement{InstanceID: "room_2", Visible: false, PositionX: 4, PositionY: 5, PositionZ: 6, ScaleX: 3, ScaleY: 3, ScaleZ: 3}
	rawRepo := &memoryAtomicRawConfigRepo{raw: []byte(initConfigRaw)}
	svc := newUEConfigServiceForModelsWithRaw(repos, rawRepo)

	cfg, err := svc.DeleteModelPlacements(context.Background(), []string{"room_1", "missing"})
	if err != nil {
		t.Fatalf("DeleteModelPlacements returned error: %v", err)
	}
	if _, ok := repos.placements["room_1"]; ok {
		t.Fatalf("room_1 placement should be deleted")
	}
	state := cfg.Placements["room_1"]
	if !state.Visible || state.Transform.Position.X != 0 || state.Transform.Scale.X != 1 {
		t.Fatalf("deleted placement should use default state: %#v", state)
	}
	if _, ok := repos.instances["room_1"]; !ok {
		t.Fatalf("model-table entry should not be deleted")
	}
	if rawRepo.updateCalls != 0 {
		t.Fatalf("init_config should not be modified, updateCalls=%d", rawRepo.updateCalls)
	}

	if _, err := svc.DeleteModelPlacements(context.Background(), []string{"room_1", "room_2"}); err != nil {
		t.Fatalf("second DeleteModelPlacements returned error: %v", err)
	}
	runtime, err := svc.BuildRuntimeModelConfig(context.Background())
	if err != nil {
		t.Fatalf("BuildRuntimeModelConfig returned error: %v", err)
	}
	if runtime.LastLoadByInstanceID["room_1"].BoundsSize.X != 10 {
		t.Fatalf("runtime should use default scale after placement delete: %#v", runtime.LastLoadByInstanceID["room_1"])
	}
	scene, err := svc.BuildSceneDesiredConfig(context.Background())
	if err != nil {
		t.Fatalf("BuildSceneDesiredConfig returned error: %v", err)
	}
	if len(scene.SceneModelsDesired) != 2 {
		t.Fatalf("scene desired len = %d, want 2", len(scene.SceneModelsDesired))
	}
}

func newUEConfigServiceForModels(repos *memoryModelRepos) UEConfigService {
	return newUEConfigServiceForModelsWithRaw(repos, fakeRawConfigRepo{})
}

func newUEConfigServiceForModelsWithRaw(repos *memoryModelRepos, raw repository.UEConfigDocumentRepository) UEConfigService {
	return newUEConfigServiceForModelsAndLegacyWithRaw(repos, &countingSceneDesiredRepo{}, raw)
}

func newUEConfigServiceForModelsAndLegacy(repos *memoryModelRepos, legacy repository.UESceneDesiredModelRepository) UEConfigService {
	return newUEConfigServiceForModelsAndLegacyWithRaw(repos, legacy, fakeRawConfigRepo{})
}

func newUEConfigServiceForModelsAndLegacyWithRaw(repos *memoryModelRepos, legacy repository.UESceneDesiredModelRepository, raw repository.UEConfigDocumentRepository) UEConfigService {
	return NewUEConfigService(
		fakeScriptRepo{},
		fakeScriptModuleRepo{},
		fakeDataBindingRepo{},
		fakeVideoSourceRepo{},
		raw,
		nil,
		legacy,
		nil,
		UEModelConfigRepositories{
			ModelType:     &memoryModelTypeRepo{store: repos},
			ModelInstance: &memoryModelInstanceRepo{store: repos},
			Placement:     &memoryModelPlacementRepo{store: repos},
		},
	)
}

type memoryModelRepos struct {
	types      map[string]entity.UEModelType
	instances  map[string]entity.UEModelInstance
	placements map[string]entity.UEModelInstancePlacement
}

func newMemoryModelRepos() *memoryModelRepos {
	return &memoryModelRepos{
		types:      make(map[string]entity.UEModelType),
		instances:  make(map[string]entity.UEModelInstance),
		placements: make(map[string]entity.UEModelInstancePlacement),
	}
}

type memoryModelTypeRepo struct {
	store *memoryModelRepos
}

func (r *memoryModelTypeRepo) ListEnabled(ctx context.Context) ([]entity.UEModelType, error) {
	store := r.store
	out := make([]entity.UEModelType, 0, len(store.types))
	for _, model := range store.types {
		if model.Enabled {
			out = append(out, model)
		}
	}
	return out, nil
}

func (r *memoryModelTypeRepo) ListByModelIDs(ctx context.Context, ids []string) ([]entity.UEModelType, error) {
	out := make([]entity.UEModelType, 0, len(ids))
	for _, id := range ids {
		if model, ok := r.store.types[id]; ok {
			out = append(out, model)
		}
	}
	return out, nil
}

func (r *memoryModelTypeRepo) CreateManyAtomic(ctx context.Context, ids []string, mutate func(existing map[string]*entity.UEModelType) ([]entity.UEModelType, error)) error {
	existing := make(map[string]*entity.UEModelType, len(ids))
	for _, id := range ids {
		if model, ok := r.store.types[id]; ok {
			copy := model
			existing[id] = &copy
		}
	}
	next, err := mutate(existing)
	if err != nil {
		return err
	}
	for _, model := range next {
		r.store.types[model.ModelID] = model
	}
	return nil
}

func (r *memoryModelTypeRepo) UpdateManyAtomic(ctx context.Context, ids []string, mutate func(existing map[string]*entity.UEModelType) ([]entity.UEModelType, error)) error {
	existing := make(map[string]*entity.UEModelType, len(ids))
	for _, id := range ids {
		if model, ok := r.store.types[id]; ok {
			copy := model
			existing[id] = &copy
		}
	}
	next, err := mutate(existing)
	if err != nil {
		return err
	}
	for _, model := range next {
		r.store.types[model.ModelID] = model
	}
	return nil
}

type memoryModelInstanceRepo struct {
	store *memoryModelRepos
}

func (r *memoryModelInstanceRepo) ListEnabled(ctx context.Context) ([]entity.UEModelInstance, error) {
	out := make([]entity.UEModelInstance, 0, len(r.store.instances))
	for _, model := range r.store.instances {
		if model.Enabled {
			out = append(out, model)
		}
	}
	return out, nil
}

func (r *memoryModelInstanceRepo) ListByInstanceIDs(ctx context.Context, ids []string) ([]entity.UEModelInstance, error) {
	out := make([]entity.UEModelInstance, 0, len(ids))
	for _, id := range ids {
		if model, ok := r.store.instances[id]; ok {
			out = append(out, model)
		}
	}
	return out, nil
}

func (r *memoryModelInstanceRepo) CountEnabledByModelIDs(ctx context.Context, modelIDs []string) (map[string]int64, error) {
	out := make(map[string]int64, len(modelIDs))
	allowed := make(map[string]struct{}, len(modelIDs))
	for _, id := range modelIDs {
		allowed[id] = struct{}{}
	}
	for _, instance := range r.store.instances {
		if !instance.Enabled {
			continue
		}
		if _, ok := allowed[instance.ModelID]; ok {
			out[instance.ModelID]++
		}
	}
	return out, nil
}

func (r *memoryModelInstanceRepo) CreateManyAtomic(ctx context.Context, ids []string, mutate func(existing map[string]*entity.UEModelInstance) ([]entity.UEModelInstance, []entity.UEModelInstancePlacement, error)) error {
	existing := make(map[string]*entity.UEModelInstance, len(ids))
	for _, id := range ids {
		if model, ok := r.store.instances[id]; ok {
			copy := model
			existing[id] = &copy
		}
	}
	instances, placements, err := mutate(existing)
	if err != nil {
		return err
	}
	for _, instance := range instances {
		r.store.instances[instance.InstanceID] = instance
	}
	for _, placement := range placements {
		if _, ok := r.store.placements[placement.InstanceID]; !ok {
			r.store.placements[placement.InstanceID] = placement
		}
	}
	return nil
}

func (r *memoryModelInstanceRepo) UpdateManyAtomic(ctx context.Context, ids []string, mutate func(existing map[string]*entity.UEModelInstance) ([]entity.UEModelInstance, error)) error {
	existing := make(map[string]*entity.UEModelInstance, len(ids))
	for _, id := range ids {
		if model, ok := r.store.instances[id]; ok {
			copy := model
			existing[id] = &copy
		}
	}
	next, err := mutate(existing)
	if err != nil {
		return err
	}
	for _, model := range next {
		r.store.instances[model.InstanceID] = model
	}
	return nil
}

type memoryModelPlacementRepo struct {
	store *memoryModelRepos
}

func (r *memoryModelPlacementRepo) ListByInstanceIDs(ctx context.Context, ids []string) ([]entity.UEModelInstancePlacement, error) {
	out := make([]entity.UEModelInstancePlacement, 0, len(ids))
	for _, id := range ids {
		if model, ok := r.store.placements[id]; ok {
			out = append(out, model)
		}
	}
	return out, nil
}

func (r *memoryModelPlacementRepo) UpdateManyAtomic(ctx context.Context, ids []string, mutate func(existing map[string]*entity.UEModelInstancePlacement) ([]entity.UEModelInstancePlacement, error)) error {
	existing := make(map[string]*entity.UEModelInstancePlacement, len(ids))
	for _, id := range ids {
		if model, ok := r.store.placements[id]; ok {
			copy := model
			existing[id] = &copy
		}
	}
	next, err := mutate(existing)
	if err != nil {
		return err
	}
	for _, model := range next {
		r.store.placements[model.InstanceID] = model
	}
	return nil
}

func (r *memoryModelPlacementRepo) DeleteByInstanceIDsAtomic(ctx context.Context, ids []string) error {
	for _, id := range ids {
		delete(r.store.placements, id)
	}
	return nil
}

type countingSceneDesiredRepo struct {
	fakeSceneDesiredRepo
	listEnabledCalls int
}

func (r *countingSceneDesiredRepo) ListEnabled(ctx context.Context) ([]entity.UESceneDesiredModel, error) {
	r.listEnabledCalls++
	return nil, nil
}

var _ repository.UEModelTypeRepository = (*memoryModelTypeRepo)(nil)
var _ repository.UEModelInstanceRepository = (*memoryModelInstanceRepo)(nil)
var _ repository.UEModelInstancePlacementRepository = (*memoryModelPlacementRepo)(nil)
