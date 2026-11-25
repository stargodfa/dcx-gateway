package ue

import "encoding/json"

type BootstrapConfig struct {
	Version      int                 `json:"version"`
	InitConfig   json.RawMessage     `json:"init_config"`
	PathConfig   json.RawMessage     `json:"path_config"`
	ChartConfig  json.RawMessage     `json:"chart_config"`
	ChartBinding any                 `json:"chart_binding"`
	DroneConfig  json.RawMessage     `json:"drone_config"`
	ModelTable   *ModelTable         `json:"model_table"`
	ModelConfig  *ModelConfig        `json:"model_config"`
	SceneDesired *SceneDesiredConfig `json:"scene_desired"`
}
