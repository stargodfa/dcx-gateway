package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"dcx-gateway/internal/model/entity"
	"dcx-gateway/internal/model/protocol/ue"
	"dcx-gateway/internal/repository"
)

const initConfigRaw = `{
  "version": 1,
  "camera": {
    "camera_id": "camera_main",
    "position": {"x": 1, "y": 2, "z": 3},
    "rotation": {"yaw": 10, "pitch": 20, "roll": 30},
    "unknown_camera": "keep"
  },
  "auto_load_on_startup": false,
  "instance_ids": ["a", "b"],
  "unknown_top": {"keep": true}
}`

func TestUpdateInitConfigJSONCameraWholePreservesUnknownFields(t *testing.T) {
	raw, err := updateInitConfigJSON([]byte(initConfigRaw), initConfigPatch{
		cameraIDSet: true,
		cameraID:    "default",
		positionSet: true,
		position:    ue.Vector3{X: -1, Y: 0, Z: 3},
		rotationSet: true,
		rotation:    ue.Rotation{Yaw: 90, Pitch: 0, Roll: 0},
	})
	if err != nil {
		t.Fatalf("updateInitConfigJSON returned error: %v", err)
	}
	root := decodeJSONForTest(t, raw)
	camera := root["camera"].(map[string]any)
	if camera["camera_id"] != "default" {
		t.Fatalf("camera_id = %v", camera["camera_id"])
	}
	if camera["unknown_camera"] != "keep" {
		t.Fatalf("unknown camera field lost: %#v", camera)
	}
	if _, ok := root["unknown_top"]; !ok {
		t.Fatalf("unknown top field lost: %#v", root)
	}
	if root["auto_load_on_startup"] != false {
		t.Fatalf("auto_load_on_startup changed: %#v", root["auto_load_on_startup"])
	}
}

func TestUpdateInitConfigJSONCameraIDOnly(t *testing.T) {
	raw, err := updateInitConfigJSON([]byte(initConfigRaw), initConfigPatch{
		cameraIDSet: true,
		cameraID:    "new_camera",
	})
	if err != nil {
		t.Fatalf("updateInitConfigJSON returned error: %v", err)
	}
	camera := decodeJSONForTest(t, raw)["camera"].(map[string]any)
	if camera["camera_id"] != "new_camera" {
		t.Fatalf("camera_id = %v", camera["camera_id"])
	}
	if camera["position"].(map[string]any)["x"] != float64(1) {
		t.Fatalf("position changed: %#v", camera["position"])
	}
	if camera["rotation"].(map[string]any)["yaw"] != float64(10) {
		t.Fatalf("rotation changed: %#v", camera["rotation"])
	}
}

func TestUpdateInitConfigJSONPositionOnlyCreatesCamera(t *testing.T) {
	raw, err := updateInitConfigJSON([]byte(`{"version":1,"keep":true}`), initConfigPatch{
		positionSet: true,
		position:    ue.Vector3{X: 100, Y: 200, Z: 300},
	})
	if err != nil {
		t.Fatalf("updateInitConfigJSON returned error: %v", err)
	}
	root := decodeJSONForTest(t, raw)
	camera := root["camera"].(map[string]any)
	position := camera["position"].(map[string]any)
	if position["x"] != float64(100) || position["y"] != float64(200) || position["z"] != float64(300) {
		t.Fatalf("position = %#v", position)
	}
	if root["keep"] != true {
		t.Fatalf("top unknown field changed: %#v", root)
	}
}

func TestUpdateInitConfigJSONRotationOnly(t *testing.T) {
	raw, err := updateInitConfigJSON([]byte(initConfigRaw), initConfigPatch{
		rotationSet: true,
		rotation:    ue.Rotation{Yaw: -90, Pitch: 0, Roll: 1},
	})
	if err != nil {
		t.Fatalf("updateInitConfigJSON returned error: %v", err)
	}
	camera := decodeJSONForTest(t, raw)["camera"].(map[string]any)
	rotation := camera["rotation"].(map[string]any)
	if rotation["yaw"] != float64(-90) || rotation["roll"] != float64(1) {
		t.Fatalf("rotation = %#v", rotation)
	}
	if camera["camera_id"] != "camera_main" {
		t.Fatalf("camera_id changed: %#v", camera)
	}
}

func TestUpdateInitConfigJSONAutoLoad(t *testing.T) {
	raw, err := updateInitConfigJSON([]byte(initConfigRaw), initConfigPatch{
		autoLoadSet: true,
		autoLoad:    true,
	})
	if err != nil {
		t.Fatalf("updateInitConfigJSON returned error: %v", err)
	}
	root := decodeJSONForTest(t, raw)
	if root["auto_load_on_startup"] != true {
		t.Fatalf("auto_load_on_startup = %#v", root["auto_load_on_startup"])
	}
	if root["camera"].(map[string]any)["camera_id"] != "camera_main" {
		t.Fatalf("camera changed: %#v", root["camera"])
	}
}

func TestUpdateInitConfigJSONInstanceIDsAddIsIdempotent(t *testing.T) {
	raw, err := updateInitConfigJSON([]byte(initConfigRaw), initConfigPatch{
		instanceMode: initInstanceModeAdd,
		instanceIDs:  []string{"b", " c ", "c", "d"},
	})
	if err != nil {
		t.Fatalf("updateInitConfigJSON returned error: %v", err)
	}
	ids := stringSliceFromAny(decodeJSONForTest(t, raw)["instance_ids"])
	want := []string{"a", "b", "c", "d"}
	assertStringSlice(t, ids, want)

	rawAgain, err := updateInitConfigJSON(raw, initConfigPatch{
		instanceMode: initInstanceModeAdd,
		instanceIDs:  []string{"c", "d"},
	})
	if err != nil {
		t.Fatalf("second update returned error: %v", err)
	}
	assertStringSlice(t, stringSliceFromAny(decodeJSONForTest(t, rawAgain)["instance_ids"]), want)
}

func TestUpdateInitConfigJSONInstanceIDsReplace(t *testing.T) {
	raw, err := updateInitConfigJSON([]byte(initConfigRaw), initConfigPatch{
		instanceMode: initInstanceModeReplace,
		instanceIDs:  []string{" x ", "y", "x", ""},
	})
	if err != nil {
		t.Fatalf("updateInitConfigJSON returned error: %v", err)
	}
	assertStringSlice(t, stringSliceFromAny(decodeJSONForTest(t, raw)["instance_ids"]), []string{"x", "y"})
}

func TestUpdateInitConfigJSONInstanceIDsDeleteIsIdempotent(t *testing.T) {
	raw, err := updateInitConfigJSON([]byte(initConfigRaw), initConfigPatch{
		instanceMode: initInstanceModeDelete,
		instanceIDs:  []string{"b", "missing"},
	})
	if err != nil {
		t.Fatalf("updateInitConfigJSON returned error: %v", err)
	}
	assertStringSlice(t, stringSliceFromAny(decodeJSONForTest(t, raw)["instance_ids"]), []string{"a"})

	rawAgain, err := updateInitConfigJSON(raw, initConfigPatch{
		instanceMode: initInstanceModeDelete,
		instanceIDs:  []string{"b", "missing"},
	})
	if err != nil {
		t.Fatalf("second delete returned error: %v", err)
	}
	assertStringSlice(t, stringSliceFromAny(decodeJSONForTest(t, rawAgain)["instance_ids"]), []string{"a"})
}

func TestUpdateInitConfigJSONDeleteCreatesStableEmptyInstanceIDs(t *testing.T) {
	raw, err := updateInitConfigJSON([]byte(`{"version":1}`), initConfigPatch{
		instanceMode: initInstanceModeDelete,
		instanceIDs:  []string{"missing"},
	})
	if err != nil {
		t.Fatalf("updateInitConfigJSON returned error: %v", err)
	}
	assertStringSlice(t, stringSliceFromAny(decodeJSONForTest(t, raw)["instance_ids"]), []string{})
}

func TestUpdateInitConfigJSONInvalidRequests(t *testing.T) {
	tests := []struct {
		name  string
		patch initConfigPatch
	}{
		{name: "empty camera id", patch: initConfigPatch{cameraIDSet: true, cameraID: " "}},
		{name: "empty instance ids", patch: initConfigPatch{instanceMode: initInstanceModeAdd, instanceIDs: []string{" "}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := updateInitConfigJSON([]byte(initConfigRaw), tt.patch)
			if !errors.Is(err, ErrInvalidInitConfigPatch) {
				t.Fatalf("err = %v, want ErrInvalidInitConfigPatch", err)
			}
		})
	}
}

func TestUEConfigServiceUpdateInitConfigUsesAtomicRepository(t *testing.T) {
	repo := &memoryAtomicRawConfigRepo{raw: []byte(initConfigRaw)}
	svc := NewUEConfigService(
		fakeScriptRepo{},
		fakeScriptModuleRepo{},
		fakeDataBindingRepo{},
		fakeVideoSourceRepo{},
		repo,
		nil,
		nil,
		nil,
	)

	cameraID := "new_camera"
	raw, err := svc.UpdateInitConfig(context.Background(), InitConfigUpdate{CameraID: &cameraID})
	if err != nil {
		t.Fatalf("UpdateInitConfig returned error: %v", err)
	}
	camera := decodeJSONForTest(t, raw)["camera"].(map[string]any)
	if camera["camera_id"] != "new_camera" {
		t.Fatalf("camera_id = %v", camera["camera_id"])
	}
	if repo.updateCalls != 1 {
		t.Fatalf("updateCalls = %d, want 1", repo.updateCalls)
	}
}

func TestUEConfigServiceAddInitInstanceIDsDoesNotValidateSceneDesired(t *testing.T) {
	repo := &memoryAtomicRawConfigRepo{raw: []byte(initConfigRaw)}
	svc := NewUEConfigService(
		fakeScriptRepo{},
		fakeScriptModuleRepo{},
		fakeDataBindingRepo{},
		fakeVideoSourceRepo{},
		repo,
		nil,
		nil,
		nil,
	)

	raw, err := svc.AddInitInstanceIDs(context.Background(), []string{"missing", "missing"})
	if err != nil {
		t.Fatalf("AddInitInstanceIDs returned error: %v", err)
	}
	assertStringSlice(t, stringSliceFromAny(decodeJSONForTest(t, raw)["instance_ids"]), []string{"a", "b", "missing"})
}

func TestUEConfigServiceConcurrentAtomicUpdatesDoNotOverwrite(t *testing.T) {
	repo := &memoryAtomicRawConfigRepo{raw: []byte(initConfigRaw)}
	svc := NewUEConfigService(
		fakeScriptRepo{},
		fakeScriptModuleRepo{},
		fakeDataBindingRepo{},
		fakeVideoSourceRepo{},
		repo,
		nil,
		nil,
		nil,
	)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		position := ue.Vector3{X: 100, Y: 200, Z: 300}
		_, _ = svc.UpdateInitConfig(context.Background(), InitConfigUpdate{Position: &position})
	}()
	go func() {
		defer wg.Done()
		autoLoad := true
		_, _ = svc.UpdateInitConfig(context.Background(), InitConfigUpdate{AutoLoad: &autoLoad})
	}()
	wg.Wait()

	root := decodeJSONForTest(t, repo.raw)
	position := root["camera"].(map[string]any)["position"].(map[string]any)
	if position["x"] != float64(100) || root["auto_load_on_startup"] != true {
		t.Fatalf("concurrent updates not preserved: %s", string(repo.raw))
	}
}

type memoryAtomicRawConfigRepo struct {
	mu          sync.Mutex
	raw         []byte
	updateCalls int
}

func (r *memoryAtomicRawConfigRepo) FindByKey(ctx context.Context, key string) (*entity.UEConfigDocument, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if key != "init_config" {
		return nil, repository.ErrNotFound
	}
	return &entity.UEConfigDocument{ConfigKey: key, ContentJSON: string(r.raw), Enabled: true}, nil
}

func (r *memoryAtomicRawConfigRepo) ListEnabled(ctx context.Context) ([]entity.UEConfigDocument, error) {
	return []entity.UEConfigDocument{{ConfigKey: "init_config", ContentJSON: string(r.raw), Enabled: true}}, nil
}

func (r *memoryAtomicRawConfigRepo) UpdateContentJSONAtomic(ctx context.Context, key string, mutate func(raw []byte) ([]byte, error)) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if key != "init_config" {
		return nil, repository.ErrNotFound
	}
	next, err := mutate(r.raw)
	if err != nil {
		return nil, err
	}
	r.raw = append([]byte(nil), next...)
	r.updateCalls++
	return next, nil
}

func (r *memoryAtomicRawConfigRepo) ReplaceContentJSONAtomic(ctx context.Context, key string, fileName string, raw []byte) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.raw = append([]byte(nil), raw...)
	r.updateCalls++
	return raw, nil
}

type fakeSceneDesiredRepo struct {
	models []entity.UESceneDesiredModel
}

func (r fakeSceneDesiredRepo) ListEnabled(ctx context.Context) ([]entity.UESceneDesiredModel, error) {
	out := make([]entity.UESceneDesiredModel, 0)
	for _, model := range r.models {
		if model.Enabled {
			out = append(out, model)
		}
	}
	return out, nil
}

func (r fakeSceneDesiredRepo) ListByInstanceIDs(ctx context.Context, ids []string) ([]entity.UESceneDesiredModel, error) {
	allowed := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		allowed[id] = struct{}{}
	}
	out := make([]entity.UESceneDesiredModel, 0)
	for _, model := range r.models {
		if _, ok := allowed[model.InstanceID]; ok {
			out = append(out, model)
		}
	}
	return out, nil
}

func (r fakeSceneDesiredRepo) ExistsEnabledByInstanceIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	out := make(map[string]bool, len(ids))
	for _, model := range r.models {
		if model.Enabled {
			out[model.InstanceID] = true
		}
	}
	return out, nil
}

func (r fakeSceneDesiredRepo) MaxSortOrder(ctx context.Context) (int, error) {
	maxSort := 0
	for _, model := range r.models {
		if model.SortOrder > maxSort {
			maxSort = model.SortOrder
		}
	}
	return maxSort, nil
}

func (r fakeSceneDesiredRepo) UpsertMany(ctx context.Context, models []entity.UESceneDesiredModel) error {
	return nil
}

func (r fakeSceneDesiredRepo) SaveManyAtomic(ctx context.Context, models []entity.UESceneDesiredModel) error {
	return nil
}

func (r fakeSceneDesiredRepo) UpdateManyAtomic(ctx context.Context, ids []string, mutate func(models map[string]*entity.UESceneDesiredModel) ([]entity.UESceneDesiredModel, error)) error {
	models := make(map[string]*entity.UESceneDesiredModel, len(r.models))
	for i := range r.models {
		model := r.models[i]
		models[model.InstanceID] = &model
	}
	_, err := mutate(models)
	return err
}

func (r fakeSceneDesiredRepo) SoftDeleteByInstanceIDs(ctx context.Context, ids []string) error {
	return nil
}

func decodeJSONForTest(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("json.Unmarshal failed: %v\n%s", err, string(raw))
	}
	return out
}

func stringSliceFromAny(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func assertStringSlice(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d; got=%#v", len(got), len(want), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q; got=%#v", i, got[i], want[i], got)
		}
	}
}
