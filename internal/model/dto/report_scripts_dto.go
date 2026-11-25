package dto

import (
	"encoding/json"

	"dcx-gateway/internal/model/protocol/ue"
)

type ReportScriptRequest struct {
	ScriptID    string `json:"script_id"`
	NewScriptID string `json:"new_script_id,omitempty"`
	Label       string `json:"label,omitempty"`
	rawFields   map[string]json.RawMessage
}

func (r *ReportScriptRequest) UnmarshalJSON(data []byte) error {
	type alias ReportScriptRequest
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	a.rawFields = fields
	*r = ReportScriptRequest(a)
	return nil
}

func (r ReportScriptRequest) HasField(name string) bool {
	_, ok := r.rawFields[name]
	return ok
}

type ReportChapterRequest struct {
	ScriptID     string `json:"script_id"`
	ChapterID    string `json:"chapter_id"`
	NewChapterID string `json:"new_chapter_id,omitempty"`
	Label        string `json:"label,omitempty"`
	rawFields    map[string]json.RawMessage
}

func (r *ReportChapterRequest) UnmarshalJSON(data []byte) error {
	type alias ReportChapterRequest
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	a.rawFields = fields
	*r = ReportChapterRequest(a)
	return nil
}

func (r ReportChapterRequest) HasField(name string) bool {
	_, ok := r.rawFields[name]
	return ok
}

type ReportSceneRequest struct {
	ScriptID   string               `json:"script_id"`
	ChapterID  string               `json:"chapter_id"`
	SceneID    string               `json:"scene_id"`
	NewSceneID string               `json:"new_scene_id,omitempty"`
	Label      string               `json:"label,omitempty"`
	Camera     *ReportCameraRequest `json:"camera,omitempty"`
	Bindings   []ReportBindingDTO   `json:"bindings,omitempty"`
	Warmup     *ReportWarmupRequest `json:"warmup,omitempty"`
	rawFields  map[string]json.RawMessage
}

func (r *ReportSceneRequest) UnmarshalJSON(data []byte) error {
	type alias ReportSceneRequest
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	a.rawFields = fields
	*r = ReportSceneRequest(a)
	return nil
}

func (r ReportSceneRequest) HasField(name string) bool {
	_, ok := r.rawFields[name]
	return ok
}

type ReportSceneCameraRequest struct {
	ScriptID  string               `json:"script_id"`
	ChapterID string               `json:"chapter_id"`
	SceneID   string               `json:"scene_id"`
	Camera    *ReportCameraRequest `json:"camera"`
}

type ReportCameraRequest struct {
	CameraID string           `json:"camera_id"`
	Position *Vector3Request  `json:"position"`
	Rotation *RotationRequest `json:"rotation"`
}

type ReportBindingsRequest struct {
	ScriptID    string             `json:"script_id"`
	ChapterID   string             `json:"chapter_id"`
	SceneID     string             `json:"scene_id"`
	Bindings    []ReportBindingDTO `json:"bindings,omitempty"`
	InstanceID  string             `json:"instance_id,omitempty"`
	InstanceIDs []string           `json:"instance_ids,omitempty"`
}

type ReportBindingDTO struct {
	InstanceID string        `json:"instance_id"`
	ModelID    string        `json:"model_id"`
	Visible    *bool         `json:"visible,omitempty"`
	Transform  *ue.Transform `json:"transform"`
}

type ReportSceneWarmupRequest struct {
	ScriptID  string               `json:"script_id"`
	ChapterID string               `json:"chapter_id"`
	SceneID   string               `json:"scene_id"`
	Warmup    *ReportWarmupRequest `json:"warmup"`
}

type ReportWarmupRequest struct {
	Data   []string `json:"data,omitempty"`
	Videos []string `json:"videos,omitempty"`
	Charts []string `json:"charts,omitempty"`
}
