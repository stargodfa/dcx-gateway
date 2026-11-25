package ue

// InitConfig 对应 init_config.json
// 用于server启动初始化
type InitConfig struct {
	Version           int      `json:"version"`
	Camera            Camera   `json:"camera"`
	AutoLoadOnStartup bool     `json:"auto_load_on_startup"`
	InstanceIDs       []string `json:"instance_ids"`
}
