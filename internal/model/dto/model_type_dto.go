package dto

import "encoding/json"

type ModelTypesRequest struct {
	ModelTypes []ModelTypeRequest `json:"model_types"`
}

func (r *ModelTypesRequest) UnmarshalJSON(data []byte) error {
	var batch struct {
		ModelTypes []ModelTypeRequest `json:"model_types"`
	}
	if err := json.Unmarshal(data, &batch); err != nil {
		return err
	}
	if batch.ModelTypes != nil {
		r.ModelTypes = batch.ModelTypes
		return nil
	}

	var single ModelTypeRequest
	if err := json.Unmarshal(data, &single); err != nil {
		return err
	}
	r.ModelTypes = []ModelTypeRequest{single}
	return nil
}

type ModelTypeRequest struct {
	ModelID                  string          `json:"model_id"`
	URI                      *string         `json:"uri,omitempty"`
	Format                   *string         `json:"format,omitempty"`
	SourcePath               *string         `json:"source_path,omitempty"`
	BaseBoundsSize           *Vector3Request `json:"base_bounds_size,omitempty"`
	BoundsNegativeExtentAxis *int            `json:"bounds_negative_extent_axis,omitempty"`
	Enabled                  *bool           `json:"enabled,omitempty"`
	rawFields                map[string]json.RawMessage
}

func (r *ModelTypeRequest) UnmarshalJSON(data []byte) error {
	type alias ModelTypeRequest
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	a.rawFields = fields
	*r = ModelTypeRequest(a)
	return nil
}

func (r ModelTypeRequest) HasField(name string) bool {
	_, ok := r.rawFields[name]
	return ok
}

type ModelTypeIDsRequest struct {
	ModelID  string   `json:"model_id,omitempty"`
	ModelIDs []string `json:"model_ids,omitempty"`
}
