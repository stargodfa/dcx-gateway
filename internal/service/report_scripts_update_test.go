package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"dcx-gateway/internal/model/entity"
	"dcx-gateway/internal/model/protocol/ue"
	"dcx-gateway/internal/repository"
)

const reportScriptsRaw = `{
  "version": 1,
  "scripts": [
    {
      "script_id": "script_a",
      "label": "Script A",
      "chapters": [
        {
          "chapter_id": "chapter_a",
          "label": "Chapter A",
          "scenes": [
            {
              "scene_id": "scene_a",
              "label": "Scene A",
              "camera": null,
              "bindings": [
                {"instance_id":"model_1","model_id":"model","visible":true,"transform":null},
                {"instance_id":"model_2","model_id":"model","visible":true,"transform":null}
              ],
              "warmup": {"data":["d1"],"videos":["v1"],"charts":["chart_1"]}
            }
          ]
        }
      ]
    }
  ]
}`

type countingReportRawRepo struct {
	docs        map[string]entity.UEConfigDocument
	updateCalls int
}

func newReportScriptsService(raw string) (UEConfigService, *countingReportRawRepo) {
	repo := &countingReportRawRepo{docs: map[string]entity.UEConfigDocument{
		"report_scripts": {ConfigKey: "report_scripts", FileName: "report_scripts.json", ContentJSON: raw, Enabled: true},
	}}
	return NewUEConfigService(fakeScriptRepo{}, fakeScriptModuleRepo{}, fakeDataBindingRepo{}, fakeVideoSourceRepo{}, repo, nil, nil, nil), repo
}

func (r *countingReportRawRepo) FindByKey(ctx context.Context, key string) (*entity.UEConfigDocument, error) {
	if doc, ok := r.docs[key]; ok && doc.Enabled {
		return &doc, nil
	}
	return nil, repository.ErrNotFound
}

func (r *countingReportRawRepo) ListEnabled(ctx context.Context) ([]entity.UEConfigDocument, error) {
	return nil, nil
}

func (r *countingReportRawRepo) UpdateContentJSONAtomic(ctx context.Context, key string, mutate func(raw []byte) ([]byte, error)) ([]byte, error) {
	doc, ok := r.docs[key]
	if !ok || !doc.Enabled {
		return nil, repository.ErrNotFound
	}
	r.updateCalls++
	next, err := mutate([]byte(doc.ContentJSON))
	if err != nil {
		return nil, err
	}
	doc.ContentJSON = string(next)
	r.docs[key] = doc
	return next, nil
}

func (r *countingReportRawRepo) ReplaceContentJSONAtomic(ctx context.Context, key string, fileName string, raw []byte) ([]byte, error) {
	r.docs[key] = entity.UEConfigDocument{ConfigKey: key, FileName: fileName, ContentJSON: string(raw), Enabled: true}
	return raw, nil
}

func TestReportScriptsScriptCreateIdempotentAndConflict(t *testing.T) {
	svc, repo := newReportScriptsService(`{"version":1,"scripts":[]}`)
	patch := ReportScriptPatch{ScriptID: " script_new ", Label: ""}

	raw, err := svc.CreateReportScript(context.Background(), patch)
	if err != nil {
		t.Fatalf("CreateReportScript returned error: %v", err)
	}
	doc := decodeReportScriptsForTest(t, raw)
	if len(doc.Scripts) != 1 || doc.Scripts[0].ScriptID != "script_new" || doc.Scripts[0].Label != "script_new" || len(doc.Scripts[0].Chapters) != 0 {
		t.Fatalf("created script = %#v", doc.Scripts)
	}
	if repo.updateCalls != 1 {
		t.Fatalf("updateCalls = %d, want 1", repo.updateCalls)
	}
	if _, err := svc.CreateReportScript(context.Background(), patch); err != nil {
		t.Fatalf("same create should be idempotent: %v", err)
	}
	if _, err := svc.CreateReportScript(context.Background(), ReportScriptPatch{ScriptID: "script_new", Label: "changed"}); !errors.Is(err, ErrReportScriptsConflict) {
		t.Fatalf("changed create err = %v, want conflict", err)
	}
}

func TestReportScriptsScriptUpdateRenameIdempotentAndDeleteMissing(t *testing.T) {
	svc, _ := newReportScriptsService(reportScriptsRaw)
	patch := ReportScriptPatch{ScriptID: "script_a", NewScriptID: "script_b", Label: "Script B", LabelSet: true}
	if _, err := svc.UpdateReportScript(context.Background(), patch); err != nil {
		t.Fatalf("UpdateReportScript returned error: %v", err)
	}
	if _, err := svc.UpdateReportScript(context.Background(), patch); err != nil {
		t.Fatalf("rename retry should be idempotent: %v", err)
	}
	raw, err := svc.DeleteReportScript(context.Background(), "missing")
	if err != nil {
		t.Fatalf("DeleteReportScript missing returned error: %v", err)
	}
	if len(decodeReportScriptsForTest(t, raw).Scripts) != 1 {
		t.Fatalf("missing delete should not remove existing script: %s", raw)
	}
}

func TestReportScriptsChapterCreateUpdateDelete(t *testing.T) {
	svc, _ := newReportScriptsService(reportScriptsRaw)
	create := ReportChapterPatch{ScriptID: "script_a", ChapterID: "chapter_b", Label: ""}
	if _, err := svc.CreateReportChapter(context.Background(), create); err != nil {
		t.Fatalf("CreateReportChapter returned error: %v", err)
	}
	if _, err := svc.CreateReportChapter(context.Background(), create); err != nil {
		t.Fatalf("same chapter create should be idempotent: %v", err)
	}
	update := ReportChapterPatch{ScriptID: "script_a", ChapterID: "chapter_b", NewChapterID: "chapter_c", Label: "Chapter C", LabelSet: true}
	if _, err := svc.UpdateReportChapter(context.Background(), update); err != nil {
		t.Fatalf("UpdateReportChapter returned error: %v", err)
	}
	if _, err := svc.UpdateReportChapter(context.Background(), update); err != nil {
		t.Fatalf("chapter rename retry should be idempotent: %v", err)
	}
	if _, err := svc.DeleteReportChapter(context.Background(), ReportChapterPatch{ScriptID: "missing", ChapterID: "missing"}); err != nil {
		t.Fatalf("DeleteReportChapter missing should be idempotent: %v", err)
	}
}

func TestReportScriptsSceneCreateUpdateDelete(t *testing.T) {
	svc, _ := newReportScriptsService(reportScriptsRaw)
	raw, err := svc.CreateReportScene(context.Background(), ReportScenePatch{
		ScriptID:  "script_a",
		ChapterID: "chapter_a",
		SceneID:   "scene_empty",
		Label:     "",
	})
	if err != nil {
		t.Fatalf("CreateReportScene without camera returned error: %v", err)
	}
	scene := findReportSceneForTest(t, decodeReportScriptsForTest(t, raw), "scene_empty")
	if scene.Camera != nil || len(scene.Bindings) != 0 || len(scene.Warmup.Data) != 0 || len(scene.Warmup.Videos) != 0 || len(scene.Warmup.Charts) != 0 {
		t.Fatalf("empty scene defaults = %#v", scene)
	}
	if !strings.Contains(string(raw), `"camera":null`) {
		t.Fatalf("empty scene should marshal camera:null: %s", raw)
	}

	camera := &ue.Camera{CameraID: "camera_main", Position: ue.Vector3{X: 0, Y: -2, Z: 3}, Rotation: ue.Rotation{Yaw: 0, Pitch: 1, Roll: 0}}
	raw, err = svc.CreateReportScene(context.Background(), ReportScenePatch{
		ScriptID:  "script_a",
		ChapterID: "chapter_a",
		SceneID:   "scene_camera",
		Label:     "Scene Camera",
		Camera:    camera,
		CameraSet: true,
	})
	if err != nil {
		t.Fatalf("CreateReportScene with camera returned error: %v", err)
	}
	if got := findReportSceneForTest(t, decodeReportScriptsForTest(t, raw), "scene_camera").Camera; got == nil || got.Position.Y != -2 || got.Rotation.Yaw != 0 {
		t.Fatalf("camera = %#v, want zero/negative values preserved", got)
	}

	raw, err = svc.UpdateReportScene(context.Background(), ReportScenePatch{
		ScriptID:   "script_a",
		ChapterID:  "chapter_a",
		SceneID:    "scene_camera",
		NewSceneID: "scene_renamed",
		Label:      "Renamed",
		LabelSet:   true,
	})
	if err != nil {
		t.Fatalf("UpdateReportScene returned error: %v", err)
	}
	renamed := findReportSceneForTest(t, decodeReportScriptsForTest(t, raw), "scene_renamed")
	if renamed.Camera == nil || renamed.Camera.Position.Y != -2 {
		t.Fatalf("scene update should preserve camera: %#v", renamed)
	}
	if _, err := svc.DeleteReportScene(context.Background(), ReportScenePatch{ScriptID: "missing", ChapterID: "missing", SceneID: "missing"}); err != nil {
		t.Fatalf("DeleteReportScene missing should be idempotent: %v", err)
	}
}

func TestReportScriptsCameraUpdateFromNilAndPreservesZero(t *testing.T) {
	svc, _ := newReportScriptsService(reportScriptsRaw)
	camera := &ue.Camera{CameraID: "camera_main", Position: ue.Vector3{X: 0, Y: 0, Z: 0}, Rotation: ue.Rotation{Yaw: 0, Pitch: 0, Roll: 0}}
	raw, err := svc.UpdateReportSceneCamera(context.Background(), ReportScenePatch{
		ScriptID:  "script_a",
		ChapterID: "chapter_a",
		SceneID:   "scene_a",
		Camera:    camera,
	})
	if err != nil {
		t.Fatalf("UpdateReportSceneCamera returned error: %v", err)
	}
	got := findReportSceneForTest(t, decodeReportScriptsForTest(t, raw), "scene_a").Camera
	if got == nil || got.Position.X != 0 || got.Rotation.Yaw != 0 {
		t.Fatalf("camera zero values not preserved: %#v", got)
	}
}

func TestReportScriptsBindingsCreateUpdateDelete(t *testing.T) {
	svc, _ := newReportScriptsService(reportScriptsRaw)
	falseValue := false
	binding := ue.ReportModelBinding{InstanceID: "model_3", ModelID: "model", Visible: falseValue, Transform: nil}
	if _, err := svc.CreateReportSceneBindings(context.Background(), ReportBindingPatch{ScriptID: "script_a", ChapterID: "chapter_a", SceneID: "scene_a", Bindings: []ue.ReportModelBinding{binding}}); err != nil {
		t.Fatalf("CreateReportSceneBindings returned error: %v", err)
	}
	if _, err := svc.CreateReportSceneBindings(context.Background(), ReportBindingPatch{ScriptID: "script_a", ChapterID: "chapter_a", SceneID: "scene_a", Bindings: []ue.ReportModelBinding{binding}}); err != nil {
		t.Fatalf("same binding create should be idempotent: %v", err)
	}
	changed := binding
	changed.Visible = true
	if _, err := svc.CreateReportSceneBindings(context.Background(), ReportBindingPatch{ScriptID: "script_a", ChapterID: "chapter_a", SceneID: "scene_a", Bindings: []ue.ReportModelBinding{changed}}); !errors.Is(err, ErrReportScriptsConflict) {
		t.Fatalf("changed binding create err = %v, want conflict", err)
	}
	if _, err := svc.CreateReportSceneBindings(context.Background(), ReportBindingPatch{ScriptID: "script_a", ChapterID: "chapter_a", SceneID: "scene_a", Bindings: []ue.ReportModelBinding{binding, binding}}); !errors.Is(err, ErrInvalidReportScriptsPatch) {
		t.Fatalf("duplicate binding err = %v, want invalid", err)
	}

	raw, err := svc.UpdateReportSceneBindings(context.Background(), ReportBindingPatch{ScriptID: "script_a", ChapterID: "chapter_a", SceneID: "scene_a", Bindings: []ue.ReportModelBinding{binding}})
	if err != nil {
		t.Fatalf("UpdateReportSceneBindings returned error: %v", err)
	}
	scene := findReportSceneForTest(t, decodeReportScriptsForTest(t, raw), "scene_a")
	if len(scene.Bindings) != 3 || scene.Bindings[0].InstanceID != "model_1" || scene.Bindings[2].InstanceID != "model_3" || scene.Bindings[2].Visible {
		t.Fatalf("binding update should only replace target and keep order: %#v", scene.Bindings)
	}
	if !strings.Contains(string(raw), `"transform":null`) {
		t.Fatalf("binding transform=null should be preserved: %s", raw)
	}
	if _, err := svc.UpdateReportSceneBindings(context.Background(), ReportBindingPatch{ScriptID: "script_a", ChapterID: "chapter_a", SceneID: "scene_a", Bindings: []ue.ReportModelBinding{{InstanceID: "missing", ModelID: "model", Visible: true}}}); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("missing binding update err = %v, want not found", err)
	}
	raw, err = svc.DeleteReportSceneBindings(context.Background(), ReportBindingDeletePatch{ScriptID: "script_a", ChapterID: "chapter_a", SceneID: "scene_a", InstanceIDs: []string{"model_3", "missing"}})
	if err != nil {
		t.Fatalf("DeleteReportSceneBindings returned error: %v", err)
	}
	scene = findReportSceneForTest(t, decodeReportScriptsForTest(t, raw), "scene_a")
	if len(scene.Bindings) != 2 || scene.Bindings[0].InstanceID != "model_1" || scene.Bindings[1].InstanceID != "model_2" {
		t.Fatalf("delete should preserve remaining order: %#v", scene.Bindings)
	}
}

func TestReportScriptsWarmupReplaceAndErrorPaths(t *testing.T) {
	svc, _ := newReportScriptsService(reportScriptsRaw)
	raw, err := svc.UpdateReportSceneWarmup(context.Background(), ReportWarmupPatch{
		ScriptID:  "script_a",
		ChapterID: "chapter_a",
		SceneID:   "scene_a",
		Warmup: ue.ReportWarmup{
			Data:   []string{" d1 ", "", "d1", "d2"},
			Videos: []string{"v1", "v1"},
			Charts: []string{"chart_1", "plain", "chart_1"},
		},
	})
	if err != nil {
		t.Fatalf("UpdateReportSceneWarmup returned error: %v", err)
	}
	warmup := findReportSceneForTest(t, decodeReportScriptsForTest(t, raw), "scene_a").Warmup
	assertStringSlice(t, warmup.Data, []string{"d1", "d2"})
	assertStringSlice(t, warmup.Videos, []string{"v1"})
	assertStringSlice(t, warmup.Charts, []string{"chart_1", "plain"})

	badSvc, _ := newReportScriptsService(`{`)
	if _, err := badSvc.CreateReportScript(context.Background(), ReportScriptPatch{ScriptID: "x"}); err == nil {
		t.Fatalf("corrupted stored json should fail")
	}
	missingSvc, _ := newReportScriptsService(reportScriptsRaw)
	delete(missingSvc.(*ueConfigService).rawConfigRepo.(*countingReportRawRepo).docs, "report_scripts")
	if _, err := missingSvc.CreateReportScript(context.Background(), ReportScriptPatch{ScriptID: "x"}); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("missing report_scripts err = %v, want not found", err)
	}
}

func decodeReportScriptsForTest(t *testing.T, raw []byte) ue.ReportScripts {
	t.Helper()
	var doc ue.ReportScripts
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("json.Unmarshal report_scripts returned error: %v\nraw=%s", err, raw)
	}
	return doc
}

func findReportSceneForTest(t *testing.T, doc ue.ReportScripts, sceneID string) ue.ReportScene {
	t.Helper()
	for _, script := range doc.Scripts {
		for _, chapter := range script.Chapters {
			for _, scene := range chapter.Scenes {
				if scene.SceneID == sceneID {
					return scene
				}
			}
		}
	}
	t.Fatalf("scene %s not found in %#v", sceneID, doc)
	return ue.ReportScene{}
}
