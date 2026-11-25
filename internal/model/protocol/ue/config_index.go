package ue

type ConfigIndex struct {
	Configs []ConfigMeta `json:"configs"`
}

type ConfigMeta struct {
	ConfigKey string `json:"config_key"`
	FileName  string `json:"file_name"`
	Version   string `json:"version"`
	Enabled   bool   `json:"enabled"`
	Source    string `json:"source,omitempty"`
}
