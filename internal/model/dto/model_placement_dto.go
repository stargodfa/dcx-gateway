package dto

import "encoding/json"

type ModelPlacementsRequest struct {
	Placements []ModelPlacementRequest `json:"placements"`
}

func (r *ModelPlacementsRequest) UnmarshalJSON(data []byte) error {
	var batch struct {
		Placements []ModelPlacementRequest `json:"placements"`
	}
	if err := json.Unmarshal(data, &batch); err != nil {
		return err
	}
	if batch.Placements != nil {
		r.Placements = batch.Placements
		return nil
	}

	var single ModelPlacementRequest
	if err := json.Unmarshal(data, &single); err != nil {
		return err
	}
	r.Placements = []ModelPlacementRequest{single}
	return nil
}

type ModelPlacementRequest struct {
	InstanceID string            `json:"instance_id,omitempty"`
	Visible    *bool             `json:"visible,omitempty"`
	Transform  *TransformRequest `json:"transform,omitempty"`
	Position   *Vector3Request   `json:"position,omitempty"`
	Rotation   *RotationRequest  `json:"rotation,omitempty"`
	Scale      *Vector3Request   `json:"scale,omitempty"`
	rawFields  map[string]json.RawMessage
}

func (r *ModelPlacementRequest) UnmarshalJSON(data []byte) error {
	type alias ModelPlacementRequest
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	a.rawFields = fields
	*r = ModelPlacementRequest(a)
	return nil
}

func (r ModelPlacementRequest) HasField(name string) bool {
	_, ok := r.rawFields[name]
	return ok
}
