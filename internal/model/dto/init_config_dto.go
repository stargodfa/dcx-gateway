package dto

import "encoding/json"

type UpdateInitConfigRequest struct {
	Camera            *InitCameraPatchRequest `json:"camera,omitempty"`
	CameraID          *string                 `json:"camera_id,omitempty"`
	Position          *Vector3Request         `json:"position,omitempty"`
	Rotation          *RotationRequest        `json:"rotation,omitempty"`
	AutoLoadOnStartup *bool                   `json:"auto_load_on_startup,omitempty"`

	hasInstanceField bool
}

func (r *UpdateInitConfigRequest) UnmarshalJSON(data []byte) error {
	type alias UpdateInitConfigRequest
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	_, hasInstanceID := fields["instance_id"]
	_, hasInstanceIDs := fields["instance_ids"]
	_, hasInstanceIDsAdd := fields["instance_ids_add"]
	_, hasInstanceIDsDelete := fields["instance_ids_delete"]
	a.hasInstanceField = hasInstanceID || hasInstanceIDs || hasInstanceIDsAdd || hasInstanceIDsDelete

	*r = UpdateInitConfigRequest(a)
	return nil
}

func (r UpdateInitConfigRequest) HasInstanceField() bool {
	return r.hasInstanceField
}

type InitCameraPatchRequest struct {
	CameraID *string          `json:"camera_id,omitempty"`
	Position *Vector3Request  `json:"position,omitempty"`
	Rotation *RotationRequest `json:"rotation,omitempty"`
}

type Vector3Request struct {
	X *float64 `json:"x"`
	Y *float64 `json:"y"`
	Z *float64 `json:"z"`
}

type RotationRequest struct {
	Yaw   *float64 `json:"yaw"`
	Pitch *float64 `json:"pitch"`
	Roll  *float64 `json:"roll"`
}

type UpdateAutoLoadRequest struct {
	AutoLoadOnStartup *bool `json:"auto_load_on_startup"`
}

type UpdateInstanceIDsRequest struct {
	InstanceID  string   `json:"instance_id,omitempty"`
	InstanceIDs []string `json:"instance_ids,omitempty"`
}
