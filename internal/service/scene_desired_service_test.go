package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	"dcx-gateway/internal/model/entity"
	"dcx-gateway/internal/model/protocol/ue"
)

func TestPatchSceneDesiredModelsAppliesPatchInsideAtomicUpdate(t *testing.T) {
	repo := newMemorySceneDesiredRepo([]entity.UESceneDesiredModel{
		{
			ID:          1,
			InstanceID:  "discussion_room_1",
			ModelID:     "discussion_room",
			URI:         "file://model",
			Format:      "udatasmith",
			SourcePath:  "./model",
			DisplayName: "discussion",
			Visible:     true,
			PositionX:   1,
			PositionY:   2,
			PositionZ:   3,
			ScaleX:      1,
			ScaleY:      1,
			ScaleZ:      1,
			Enabled:     true,
			SortOrder:   10,
		},
	})
	svc := newUEConfigServiceForSceneDesired(repo)

	visible := false
	position := ue.Vector3{X: 100, Y: 200, Z: 300}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = svc.PatchSceneDesiredModels(context.Background(), []SceneDesiredModelPatch{
			{InstanceID: "discussion_room_1", Visible: &visible},
		})
	}()
	go func() {
		defer wg.Done()
		_, _ = svc.PatchSceneDesiredModels(context.Background(), []SceneDesiredModelPatch{
			{InstanceID: "discussion_room_1", Position: &position},
		})
	}()
	wg.Wait()

	model := repo.model("discussion_room_1")
	if model.Visible != false {
		t.Fatalf("visible = %v, want false", model.Visible)
	}
	if model.PositionX != 100 || model.PositionY != 200 || model.PositionZ != 300 {
		t.Fatalf("position = %v,%v,%v; want 100,200,300", model.PositionX, model.PositionY, model.PositionZ)
	}
	if repo.listByInstanceIDsCalls != 0 {
		t.Fatalf("PatchSceneDesiredModels must not read stale rows outside atomic update, ListByInstanceIDs calls=%d", repo.listByInstanceIDsCalls)
	}
	if repo.updateManyAtomicCalls != 2 {
		t.Fatalf("UpdateManyAtomic calls=%d, want 2", repo.updateManyAtomicCalls)
	}
}

func TestPatchSceneDesiredModelsReportsMissingIDs(t *testing.T) {
	repo := newMemorySceneDesiredRepo([]entity.UESceneDesiredModel{
		{
			ID:          1,
			InstanceID:  "discussion_room_1",
			ModelID:     "discussion_room",
			URI:         "file://model",
			Format:      "udatasmith",
			SourcePath:  "./model",
			DisplayName: "discussion",
			Visible:     true,
			ScaleX:      1,
			ScaleY:      1,
			ScaleZ:      1,
			Enabled:     true,
		},
	})
	svc := newUEConfigServiceForSceneDesired(repo)

	visible := false
	_, err := svc.PatchSceneDesiredModels(context.Background(), []SceneDesiredModelPatch{
		{InstanceID: "discussion_room_1", Visible: &visible},
		{InstanceID: "missing_room", Visible: &visible},
	})
	var missing *MissingSceneDesiredModelsError
	if !errors.As(err, &missing) {
		t.Fatalf("err = %v, want MissingSceneDesiredModelsError", err)
	}
	if len(missing.InstanceIDs) != 1 || missing.InstanceIDs[0] != "missing_room" {
		t.Fatalf("missing ids = %#v", missing.InstanceIDs)
	}
}

func TestUpsertSceneDesiredModelsUsesNextSortOrderForNewRows(t *testing.T) {
	repo := newMemorySceneDesiredRepo([]entity.UESceneDesiredModel{
		{ID: 1, InstanceID: "existing", SortOrder: 10, ModelID: "m", URI: "file://m", Format: "udatasmith", SourcePath: "./m", DisplayName: "m", Visible: true, Enabled: true},
	})
	svc := newUEConfigServiceForSceneDesired(repo)

	modelID := "new_model"
	uri := "file://new"
	format := "udatasmith"
	sourcePath := "./new"
	displayName := "new"
	visible := true
	_, err := svc.UpsertSceneDesiredModels(context.Background(), []SceneDesiredModelPatch{
		{InstanceID: "new_1", ModelID: &modelID, URI: &uri, Format: &format, SourcePath: &sourcePath, DisplayName: &displayName, Visible: &visible},
		{InstanceID: "new_2", ModelID: &modelID, URI: &uri, Format: &format, SourcePath: &sourcePath, DisplayName: &displayName, Visible: &visible},
	})
	if err != nil {
		t.Fatalf("UpsertSceneDesiredModels returned error: %v", err)
	}
	if repo.model("new_1").SortOrder != 11 || repo.model("new_2").SortOrder != 12 {
		t.Fatalf("sort orders: new_1=%d new_2=%d", repo.model("new_1").SortOrder, repo.model("new_2").SortOrder)
	}
}

func newUEConfigServiceForSceneDesired(repo *memorySceneDesiredRepo) UEConfigService {
	return NewUEConfigService(
		fakeScriptRepo{},
		fakeScriptModuleRepo{},
		fakeDataBindingRepo{},
		fakeVideoSourceRepo{},
		fakeRawConfigRepo{},
		nil,
		repo,
		nil,
	)
}

type memorySceneDesiredRepo struct {
	mu                     sync.Mutex
	models                 map[string]entity.UESceneDesiredModel
	listByInstanceIDsCalls int
	updateManyAtomicCalls  int
	nextID                 uint64
}

func newMemorySceneDesiredRepo(models []entity.UESceneDesiredModel) *memorySceneDesiredRepo {
	out := &memorySceneDesiredRepo{models: make(map[string]entity.UESceneDesiredModel), nextID: 100}
	for _, model := range models {
		out.models[model.InstanceID] = model
	}
	return out
}

func (r *memorySceneDesiredRepo) model(id string) entity.UESceneDesiredModel {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.models[id]
}

func (r *memorySceneDesiredRepo) ListEnabled(ctx context.Context) ([]entity.UESceneDesiredModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]entity.UESceneDesiredModel, 0, len(r.models))
	for _, model := range r.models {
		if model.Enabled {
			out = append(out, model)
		}
	}
	return out, nil
}

func (r *memorySceneDesiredRepo) ListByInstanceIDs(ctx context.Context, ids []string) ([]entity.UESceneDesiredModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.listByInstanceIDsCalls++
	out := make([]entity.UESceneDesiredModel, 0, len(ids))
	for _, id := range ids {
		if model, ok := r.models[id]; ok {
			out = append(out, model)
		}
	}
	return out, nil
}

func (r *memorySceneDesiredRepo) ExistsEnabledByInstanceIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[string]bool, len(ids))
	for _, id := range ids {
		if model, ok := r.models[id]; ok && model.Enabled {
			out[id] = true
		}
	}
	return out, nil
}

func (r *memorySceneDesiredRepo) MaxSortOrder(ctx context.Context) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	maxSort := 0
	for _, model := range r.models {
		if model.SortOrder > maxSort {
			maxSort = model.SortOrder
		}
	}
	return maxSort, nil
}

func (r *memorySceneDesiredRepo) UpsertMany(ctx context.Context, models []entity.UESceneDesiredModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, model := range models {
		if model.ID == 0 {
			r.nextID++
			model.ID = r.nextID
		}
		r.models[model.InstanceID] = model
	}
	return nil
}

func (r *memorySceneDesiredRepo) SaveManyAtomic(ctx context.Context, models []entity.UESceneDesiredModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, model := range models {
		r.models[model.InstanceID] = model
	}
	return nil
}

func (r *memorySceneDesiredRepo) UpdateManyAtomic(ctx context.Context, ids []string, mutate func(models map[string]*entity.UESceneDesiredModel) ([]entity.UESceneDesiredModel, error)) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.updateManyAtomicCalls++
	locked := make(map[string]*entity.UESceneDesiredModel, len(ids))
	for _, id := range ids {
		if model, ok := r.models[id]; ok {
			copy := model
			locked[id] = &copy
		}
	}
	next, err := mutate(locked)
	if err != nil {
		return err
	}
	for _, model := range next {
		r.models[model.InstanceID] = model
	}
	return nil
}

func (r *memorySceneDesiredRepo) SoftDeleteByInstanceIDs(ctx context.Context, ids []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, id := range ids {
		model, ok := r.models[id]
		if !ok {
			continue
		}
		model.Enabled = false
		r.models[id] = model
	}
	return nil
}
