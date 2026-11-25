package dto

import "encoding/json"

type SceneDesiredModelsRequest struct {
	Models []SceneDesiredModelRequest `json:"models"`
}

func (r *SceneDesiredModelsRequest) UnmarshalJSON(data []byte) error {
	var batch struct {
		Models []SceneDesiredModelRequest `json:"models"`
	}
	if err := json.Unmarshal(data, &batch); err != nil {
		return err
	}
	if batch.Models != nil {
		r.Models = batch.Models
		return nil
	}

	var single SceneDesiredModelRequest
	if err := json.Unmarshal(data, &single); err != nil {
		return err
	}
	r.Models = []SceneDesiredModelRequest{single}
	return nil
}

type SceneDesiredModelRequest struct {
	InstanceID  string            `json:"instance_id"`
	ModelID     *string           `json:"model_id,omitempty"`
	URI         *string           `json:"uri,omitempty"`
	Format      *string           `json:"format,omitempty"`
	SourcePath  *string           `json:"source_path,omitempty"`
	DisplayName *string           `json:"display_name,omitempty"`
	BoundsSize  *Vector3Request   `json:"bounds_size,omitempty"`
	Visible     *bool             `json:"visible,omitempty"`
	Transform   *TransformRequest `json:"transform,omitempty"`
	Position    *Vector3Request   `json:"position,omitempty"`
	Rotation    *RotationRequest  `json:"rotation,omitempty"`
	Scale       *Vector3Request   `json:"scale,omitempty"`
	ConfigJSON  map[string]any    `json:"config_json,omitempty"`
	rawFields   map[string]json.RawMessage
}

func (r *SceneDesiredModelRequest) UnmarshalJSON(data []byte) error {
	type alias SceneDesiredModelRequest
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	a.rawFields = fields
	*r = SceneDesiredModelRequest(a)
	return nil
}

func (r SceneDesiredModelRequest) HasField(name string) bool {
	_, ok := r.rawFields[name]
	return ok
}

type TransformRequest struct {
	Position *Vector3Request  `json:"position,omitempty"`
	Rotation *RotationRequest `json:"rotation,omitempty"`
	Scale    *Vector3Request  `json:"scale,omitempty"`
}
