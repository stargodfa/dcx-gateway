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

var (
	ErrInvalidReportScriptsPatch = errors.New("invalid report_scripts patch")
	ErrReportScriptsConflict     = errors.New("report_scripts conflict")
)

type ReportScriptPatch struct {
	ScriptID    string
	NewScriptID string
	Label       string
	LabelSet    bool
}

type ReportChapterPatch struct {
	ScriptID     string
	ChapterID    string
	NewChapterID string
	Label        string
	LabelSet     bool
}

type ReportScenePatch struct {
	ScriptID    string
	ChapterID   string
	SceneID     string
	NewSceneID  string
	Label       string
	LabelSet    bool
	Camera      *ue.Camera
	CameraSet   bool
	Bindings    []ue.ReportModelBinding
	BindingsSet bool
	Warmup      ue.ReportWarmup
	WarmupSet   bool
}

type ReportBindingPatch struct {
	ScriptID  string
	ChapterID string
	SceneID   string
	Bindings  []ue.ReportModelBinding
}

type ReportBindingDeletePatch struct {
	ScriptID    string
	ChapterID   string
	SceneID     string
	InstanceIDs []string
}

type ReportWarmupPatch struct {
	ScriptID  string
	ChapterID string
	SceneID   string
	Warmup    ue.ReportWarmup
}

func (s *ueConfigService) CreateReportScript(ctx context.Context, patch ReportScriptPatch) (json.RawMessage, error) {
	return s.updateReportScripts(ctx, func(doc *ue.ReportScripts) error {
		id, err := requiredID(patch.ScriptID, "script_id")
		if err != nil {
			return err
		}
		label := defaultLabel(patch.Label, id)
		if idx := findScriptIndex(doc, id); idx >= 0 {
			if strings.TrimSpace(doc.Scripts[idx].Label) == label {
				return nil
			}
			return reportScriptsConflict("script_id already exists with different label")
		}
		doc.Scripts = append(doc.Scripts, ue.ReportScript{ScriptID: id, Label: label, Chapters: []ue.ReportChapter{}})
		return nil
	})
}

func (s *ueConfigService) UpdateReportScript(ctx context.Context, patch ReportScriptPatch) (json.RawMessage, error) {
	return s.updateReportScripts(ctx, func(doc *ue.ReportScripts) error {
		id, err := requiredID(patch.ScriptID, "script_id")
		if err != nil {
			return err
		}
		newID := strings.TrimSpace(patch.NewScriptID)
		idx := findScriptIndex(doc, id)
		if idx < 0 {
			if newID != "" {
				if target := findScriptIndex(doc, newID); target >= 0 && reportScriptMatchesPatch(doc.Scripts[target], patch) {
					return nil
				}
			}
			return repository.ErrNotFound
		}
		if newID != "" && newID != id {
			if findScriptIndex(doc, newID) >= 0 {
				return reportScriptsConflict("new_script_id already exists")
			}
			doc.Scripts[idx].ScriptID = newID
		}
		if patch.LabelSet {
			nextID := doc.Scripts[idx].ScriptID
			doc.Scripts[idx].Label = defaultLabel(patch.Label, nextID)
		}
		normalizeReportScripts(doc)
		return nil
	})
}

func (s *ueConfigService) DeleteReportScript(ctx context.Context, scriptID string) (json.RawMessage, error) {
	return s.updateReportScripts(ctx, func(doc *ue.ReportScripts) error {
		id, err := requiredID(scriptID, "script_id")
		if err != nil {
			return err
		}
		if idx := findScriptIndex(doc, id); idx >= 0 {
			doc.Scripts = append(doc.Scripts[:idx], doc.Scripts[idx+1:]...)
		}
		return nil
	})
}

func (s *ueConfigService) CreateReportChapter(ctx context.Context, patch ReportChapterPatch) (json.RawMessage, error) {
	return s.updateReportScripts(ctx, func(doc *ue.ReportScripts) error {
		script, err := requireScript(doc, patch.ScriptID)
		if err != nil {
			return err
		}
		chapterID, err := requiredID(patch.ChapterID, "chapter_id")
		if err != nil {
			return err
		}
		label := defaultLabel(patch.Label, chapterID)
		if idx := findChapterIndex(script, chapterID); idx >= 0 {
			if strings.TrimSpace(script.Chapters[idx].Label) == label {
				return nil
			}
			return reportScriptsConflict("chapter_id already exists with different label")
		}
		script.Chapters = append(script.Chapters, ue.ReportChapter{ChapterID: chapterID, Label: label, Scenes: []ue.ReportScene{}})
		return nil
	})
}

func (s *ueConfigService) UpdateReportChapter(ctx context.Context, patch ReportChapterPatch) (json.RawMessage, error) {
	return s.updateReportScripts(ctx, func(doc *ue.ReportScripts) error {
		script, err := requireScript(doc, patch.ScriptID)
		if err != nil {
			return err
		}
		chapterID, err := requiredID(patch.ChapterID, "chapter_id")
		if err != nil {
			return err
		}
		newID := strings.TrimSpace(patch.NewChapterID)
		idx := findChapterIndex(script, chapterID)
		if idx < 0 {
			if newID != "" {
				if target := findChapterIndex(script, newID); target >= 0 && reportChapterMatchesPatch(script.Chapters[target], patch) {
					return nil
				}
			}
			return repository.ErrNotFound
		}
		if newID != "" && newID != chapterID {
			if findChapterIndex(script, newID) >= 0 {
				return reportScriptsConflict("new_chapter_id already exists")
			}
			script.Chapters[idx].ChapterID = newID
		}
		if patch.LabelSet {
			nextID := script.Chapters[idx].ChapterID
			script.Chapters[idx].Label = defaultLabel(patch.Label, nextID)
		}
		normalizeReportScripts(doc)
		return nil
	})
}

func (s *ueConfigService) DeleteReportChapter(ctx context.Context, patch ReportChapterPatch) (json.RawMessage, error) {
	return s.updateReportScripts(ctx, func(doc *ue.ReportScripts) error {
		scriptID, err := requiredID(patch.ScriptID, "script_id")
		if err != nil {
			return err
		}
		chapterID, err := requiredID(patch.ChapterID, "chapter_id")
		if err != nil {
			return err
		}
		if sidx := findScriptIndex(doc, scriptID); sidx >= 0 {
			if cidx := findChapterIndex(&doc.Scripts[sidx], chapterID); cidx >= 0 {
				chapters := doc.Scripts[sidx].Chapters
				doc.Scripts[sidx].Chapters = append(chapters[:cidx], chapters[cidx+1:]...)
			}
		}
		return nil
	})
}

func (s *ueConfigService) CreateReportScene(ctx context.Context, patch ReportScenePatch) (json.RawMessage, error) {
	return s.updateReportScripts(ctx, func(doc *ue.ReportScripts) error {
		chapter, err := requireChapter(doc, patch.ScriptID, patch.ChapterID)
		if err != nil {
			return err
		}
		sceneID, err := requiredID(patch.SceneID, "scene_id")
		if err != nil {
			return err
		}
		label := defaultLabel(patch.Label, sceneID)
		bindings := patch.Bindings
		if !patch.BindingsSet {
			bindings = []ue.ReportModelBinding{}
		} else {
			var err error
			bindings, err = normalizeBindingsAllowEmpty(bindings)
			if err != nil {
				return err
			}
		}
		warmup := patch.Warmup
		if !patch.WarmupSet {
			warmup = emptyReportWarmup()
		}
		scene := ue.ReportScene{SceneID: sceneID, Label: label, Camera: patch.Camera, Bindings: bindings, Warmup: normalizeWarmup(warmup)}
		if idx := findSceneIndex(chapter, sceneID); idx >= 0 {
			existing := chapter.Scenes[idx]
			if strings.TrimSpace(existing.Label) == label && camerasEqual(existing.Camera, patch.Camera) {
				return nil
			}
			return reportScriptsConflict("scene_id already exists with different content")
		}
		chapter.Scenes = append(chapter.Scenes, scene)
		return nil
	})
}

func (s *ueConfigService) UpdateReportScene(ctx context.Context, patch ReportScenePatch) (json.RawMessage, error) {
	return s.updateReportScripts(ctx, func(doc *ue.ReportScripts) error {
		chapter, err := requireChapter(doc, patch.ScriptID, patch.ChapterID)
		if err != nil {
			return err
		}
		sceneID, err := requiredID(patch.SceneID, "scene_id")
		if err != nil {
			return err
		}
		newID := strings.TrimSpace(patch.NewSceneID)
		idx := findSceneIndex(chapter, sceneID)
		if idx < 0 {
			if newID != "" {
				if target := findSceneIndex(chapter, newID); target >= 0 && reportSceneMatchesPatch(chapter.Scenes[target], patch) {
					return nil
				}
			}
			return repository.ErrNotFound
		}
		if newID != "" && newID != sceneID {
			if findSceneIndex(chapter, newID) >= 0 {
				return reportScriptsConflict("new_scene_id already exists")
			}
			chapter.Scenes[idx].SceneID = newID
		}
		if patch.LabelSet {
			nextID := chapter.Scenes[idx].SceneID
			chapter.Scenes[idx].Label = defaultLabel(patch.Label, nextID)
		}
		normalizeReportScripts(doc)
		return nil
	})
}

func (s *ueConfigService) DeleteReportScene(ctx context.Context, patch ReportScenePatch) (json.RawMessage, error) {
	return s.updateReportScripts(ctx, func(doc *ue.ReportScripts) error {
		if _, err := requiredID(patch.ScriptID, "script_id"); err != nil {
			return err
		}
		if _, err := requiredID(patch.ChapterID, "chapter_id"); err != nil {
			return err
		}
		sceneID, err := requiredID(patch.SceneID, "scene_id")
		if err != nil {
			return err
		}
		chapter, err := optionalChapter(doc, patch.ScriptID, patch.ChapterID)
		if err != nil || chapter == nil {
			return nil
		}
		if idx := findSceneIndex(chapter, sceneID); idx >= 0 {
			scenes := chapter.Scenes
			chapter.Scenes = append(scenes[:idx], scenes[idx+1:]...)
		}
		return nil
	})
}

func (s *ueConfigService) UpdateReportSceneCamera(ctx context.Context, patch ReportScenePatch) (json.RawMessage, error) {
	return s.updateReportScripts(ctx, func(doc *ue.ReportScripts) error {
		scene, err := requireScene(doc, patch.ScriptID, patch.ChapterID, patch.SceneID)
		if err != nil {
			return err
		}
		if patch.Camera == nil {
			return reportScriptsInvalid("camera is required")
		}
		scene.Camera = patch.Camera
		normalizeReportScripts(doc)
		return nil
	})
}

func (s *ueConfigService) CreateReportSceneBindings(ctx context.Context, patch ReportBindingPatch) (json.RawMessage, error) {
	return s.updateReportScripts(ctx, func(doc *ue.ReportScripts) error {
		scene, err := requireScene(doc, patch.ScriptID, patch.ChapterID, patch.SceneID)
		if err != nil {
			return err
		}
		bindings, err := normalizeBindings(patch.Bindings)
		if err != nil {
			return err
		}
		existing := make(map[string]ue.ReportModelBinding, len(scene.Bindings))
		for _, binding := range scene.Bindings {
			existing[binding.InstanceID] = binding
		}
		for _, binding := range bindings {
			if current, ok := existing[binding.InstanceID]; ok {
				if !bindingsEqual(current, binding) {
					return reportScriptsConflict("binding already exists with different content")
				}
				continue
			}
			scene.Bindings = append(scene.Bindings, binding)
			existing[binding.InstanceID] = binding
		}
		normalizeReportScripts(doc)
		return nil
	})
}

func (s *ueConfigService) UpdateReportSceneBindings(ctx context.Context, patch ReportBindingPatch) (json.RawMessage, error) {
	return s.updateReportScripts(ctx, func(doc *ue.ReportScripts) error {
		scene, err := requireScene(doc, patch.ScriptID, patch.ChapterID, patch.SceneID)
		if err != nil {
			return err
		}
		bindings, err := normalizeBindings(patch.Bindings)
		if err != nil {
			return err
		}
		index := make(map[string]int, len(scene.Bindings))
		for i, binding := range scene.Bindings {
			index[binding.InstanceID] = i
		}
		for _, binding := range bindings {
			i, ok := index[binding.InstanceID]
			if !ok {
				return repository.ErrNotFound
			}
			scene.Bindings[i] = binding
		}
		normalizeReportScripts(doc)
		return nil
	})
}

func (s *ueConfigService) DeleteReportSceneBindings(ctx context.Context, patch ReportBindingDeletePatch) (json.RawMessage, error) {
	return s.updateReportScripts(ctx, func(doc *ue.ReportScripts) error {
		scene, err := requireScene(doc, patch.ScriptID, patch.ChapterID, patch.SceneID)
		if err != nil {
			return err
		}
		ids, err := normalizeIDList(patch.InstanceIDs, "instance_ids")
		if err != nil {
			return err
		}
		remove := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			remove[id] = struct{}{}
		}
		next := make([]ue.ReportModelBinding, 0, len(scene.Bindings))
		for _, binding := range scene.Bindings {
			if _, ok := remove[binding.InstanceID]; !ok {
				next = append(next, binding)
			}
		}
		scene.Bindings = next
		normalizeReportScripts(doc)
		return nil
	})
}

func (s *ueConfigService) UpdateReportSceneWarmup(ctx context.Context, patch ReportWarmupPatch) (json.RawMessage, error) {
	return s.updateReportScripts(ctx, func(doc *ue.ReportScripts) error {
		scene, err := requireScene(doc, patch.ScriptID, patch.ChapterID, patch.SceneID)
		if err != nil {
			return err
		}
		scene.Warmup = normalizeWarmup(patch.Warmup)
		normalizeReportScripts(doc)
		return nil
	})
}

func (s *ueConfigService) updateReportScripts(ctx context.Context, mutate func(doc *ue.ReportScripts) error) (json.RawMessage, error) {
	next, err := s.rawConfigRepo.UpdateContentJSONAtomic(ctx, "report_scripts", func(raw []byte) ([]byte, error) {
		var doc ue.ReportScripts
		if err := json.Unmarshal(raw, &doc); err != nil {
			return nil, err
		}
		normalizeReportScripts(&doc)
		if err := mutate(&doc); err != nil {
			return nil, err
		}
		normalizeReportScripts(&doc)
		out, err := json.Marshal(&doc)
		if err != nil {
			return nil, err
		}
		return out, nil
	})
	if err != nil {
		return nil, err
	}
	return json.RawMessage(next), nil
}

func normalizeReportScripts(doc *ue.ReportScripts) {
	if doc.Version == 0 {
		doc.Version = 1
	}
	if doc.Scripts == nil {
		doc.Scripts = []ue.ReportScript{}
	}
	for si := range doc.Scripts {
		if doc.Scripts[si].Chapters == nil {
			doc.Scripts[si].Chapters = []ue.ReportChapter{}
		}
		for ci := range doc.Scripts[si].Chapters {
			if doc.Scripts[si].Chapters[ci].Scenes == nil {
				doc.Scripts[si].Chapters[ci].Scenes = []ue.ReportScene{}
			}
			for sc := range doc.Scripts[si].Chapters[ci].Scenes {
				scene := &doc.Scripts[si].Chapters[ci].Scenes[sc]
				if scene.Bindings == nil {
					scene.Bindings = []ue.ReportModelBinding{}
				}
				scene.Warmup = normalizeWarmup(scene.Warmup)
			}
		}
	}
}

func normalizeBindings(bindings []ue.ReportModelBinding) ([]ue.ReportModelBinding, error) {
	if len(bindings) == 0 {
		return nil, reportScriptsInvalid("bindings must not be empty")
	}
	return normalizeBindingsAllowEmpty(bindings)
}

func normalizeBindingsAllowEmpty(bindings []ue.ReportModelBinding) ([]ue.ReportModelBinding, error) {
	if len(bindings) == 0 {
		return []ue.ReportModelBinding{}, nil
	}
	seen := make(map[string]struct{}, len(bindings))
	out := make([]ue.ReportModelBinding, 0, len(bindings))
	for _, binding := range bindings {
		instanceID, err := requiredID(binding.InstanceID, "instance_id")
		if err != nil {
			return nil, err
		}
		modelID, err := requiredID(binding.ModelID, "model_id")
		if err != nil {
			return nil, err
		}
		if _, ok := seen[instanceID]; ok {
			return nil, reportScriptsInvalid("duplicate instance_id in bindings")
		}
		seen[instanceID] = struct{}{}
		out = append(out, ue.ReportModelBinding{
			InstanceID: instanceID,
			ModelID:    modelID,
			Visible:    binding.Visible,
			Transform:  binding.Transform,
		})
	}
	return out, nil
}

func normalizeWarmup(w ue.ReportWarmup) ue.ReportWarmup {
	return ue.ReportWarmup{
		Data:   normalizeStringList(w.Data),
		Videos: normalizeStringList(w.Videos),
		Charts: normalizeStringList(w.Charts),
	}
}

func emptyReportWarmup() ue.ReportWarmup {
	return ue.ReportWarmup{Data: []string{}, Videos: []string{}, Charts: []string{}}
}

func normalizeStringList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if out == nil {
		return []string{}
	}
	return out
}

func normalizeIDList(values []string, field string) ([]string, error) {
	ids := normalizeStringList(values)
	if len(ids) == 0 {
		return nil, reportScriptsInvalid(field + " must not be empty")
	}
	return ids, nil
}

func requiredID(value string, field string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", reportScriptsInvalid(field + " is required")
	}
	return value, nil
}

func defaultLabel(label string, id string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return id
	}
	return label
}

func findScriptIndex(doc *ue.ReportScripts, id string) int {
	id = strings.TrimSpace(id)
	for i, script := range doc.Scripts {
		if script.ScriptID == id {
			return i
		}
	}
	return -1
}

func findChapterIndex(script *ue.ReportScript, id string) int {
	id = strings.TrimSpace(id)
	for i, chapter := range script.Chapters {
		if chapter.ChapterID == id {
			return i
		}
	}
	return -1
}

func findSceneIndex(chapter *ue.ReportChapter, id string) int {
	id = strings.TrimSpace(id)
	for i, scene := range chapter.Scenes {
		if scene.SceneID == id {
			return i
		}
	}
	return -1
}

func requireScript(doc *ue.ReportScripts, scriptID string) (*ue.ReportScript, error) {
	id, err := requiredID(scriptID, "script_id")
	if err != nil {
		return nil, err
	}
	idx := findScriptIndex(doc, id)
	if idx < 0 {
		return nil, repository.ErrNotFound
	}
	return &doc.Scripts[idx], nil
}

func optionalChapter(doc *ue.ReportScripts, scriptID string, chapterID string) (*ue.ReportChapter, error) {
	scriptID, err := requiredID(scriptID, "script_id")
	if err != nil {
		return nil, err
	}
	chapterID, err = requiredID(chapterID, "chapter_id")
	if err != nil {
		return nil, err
	}
	sidx := findScriptIndex(doc, scriptID)
	if sidx < 0 {
		return nil, nil
	}
	cidx := findChapterIndex(&doc.Scripts[sidx], chapterID)
	if cidx < 0 {
		return nil, nil
	}
	return &doc.Scripts[sidx].Chapters[cidx], nil
}

func requireChapter(doc *ue.ReportScripts, scriptID string, chapterID string) (*ue.ReportChapter, error) {
	script, err := requireScript(doc, scriptID)
	if err != nil {
		return nil, err
	}
	id, err := requiredID(chapterID, "chapter_id")
	if err != nil {
		return nil, err
	}
	idx := findChapterIndex(script, id)
	if idx < 0 {
		return nil, repository.ErrNotFound
	}
	return &script.Chapters[idx], nil
}

func requireScene(doc *ue.ReportScripts, scriptID string, chapterID string, sceneID string) (*ue.ReportScene, error) {
	chapter, err := requireChapter(doc, scriptID, chapterID)
	if err != nil {
		return nil, err
	}
	id, err := requiredID(sceneID, "scene_id")
	if err != nil {
		return nil, err
	}
	idx := findSceneIndex(chapter, id)
	if idx < 0 {
		return nil, repository.ErrNotFound
	}
	return &chapter.Scenes[idx], nil
}

func reportScriptMatchesPatch(script ue.ReportScript, patch ReportScriptPatch) bool {
	if patch.LabelSet && strings.TrimSpace(script.Label) != defaultLabel(patch.Label, script.ScriptID) {
		return false
	}
	return true
}

func reportChapterMatchesPatch(chapter ue.ReportChapter, patch ReportChapterPatch) bool {
	if patch.LabelSet && strings.TrimSpace(chapter.Label) != defaultLabel(patch.Label, chapter.ChapterID) {
		return false
	}
	return true
}

func reportSceneMatchesPatch(scene ue.ReportScene, patch ReportScenePatch) bool {
	if patch.LabelSet && strings.TrimSpace(scene.Label) != defaultLabel(patch.Label, scene.SceneID) {
		return false
	}
	return true
}

func camerasEqual(a *ue.Camera, b *ue.Camera) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func bindingsEqual(a ue.ReportModelBinding, b ue.ReportModelBinding) bool {
	if a.InstanceID != b.InstanceID || a.ModelID != b.ModelID || a.Visible != b.Visible {
		return false
	}
	aj, _ := json.Marshal(a.Transform)
	bj, _ := json.Marshal(b.Transform)
	return bytes.Equal(aj, bj)
}

func reportScriptsInvalid(message string) error {
	return fmt.Errorf("%w: %s", ErrInvalidReportScriptsPatch, message)
}

func reportScriptsConflict(message string) error {
	return fmt.Errorf("%w: %s", ErrReportScriptsConflict, message)
}
