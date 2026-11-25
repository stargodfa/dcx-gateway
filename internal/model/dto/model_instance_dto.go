package dto

import "encoding/json"

type ModelTableEntriesRequest struct {
	Entries []ModelTableEntryRequest `json:"entries"`
}

func (r *ModelTableEntriesRequest) UnmarshalJSON(data []byte) error {
	var batch struct {
		Entries []ModelTableEntryRequest `json:"entries"`
	}
	if err := json.Unmarshal(data, &batch); err != nil {
		return err
	}
	if batch.Entries != nil {
		r.Entries = batch.Entries
		return nil
	}

	var single ModelTableEntryRequest
	if err := json.Unmarshal(data, &single); err != nil {
		return err
	}
	r.Entries = []ModelTableEntryRequest{single}
	return nil
}

type ModelTableEntryRequest struct {
	InstanceID  string  `json:"instance_id"`
	ModelID     *string `json:"model_id,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	SortOrder   *int    `json:"sort_order,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
	rawFields   map[string]json.RawMessage
}

func (r *ModelTableEntryRequest) UnmarshalJSON(data []byte) error {
	type alias ModelTableEntryRequest
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	a.rawFields = fields
	*r = ModelTableEntryRequest(a)
	return nil
}

func (r ModelTableEntryRequest) HasField(name string) bool {
	_, ok := r.rawFields[name]
	return ok
}
