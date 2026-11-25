package ue

import "encoding/json"

// 可操作模型实例清单，主要给前端展示和汇报脚本选择用
// 对应model_table.json
type ModelTable struct {
	Version int               `json:"version"`
	Entries []ModelTableEntry `json:"entries"`
}

type ModelTableEntry struct {
	InstanceID  string `json:"instance_id"`
	ModelID     string `json:"model_id"`
	DisplayName string `json:"display_name"`
}

// 记录一键加载/上次加载的模型示例状态
// 对应model_config.go
type ModelConfig struct {
	Version              int                           `json:"version"`
	LastLoadByInstanceID map[string]ModelInstanceState `json:"last_load_by_instance_id"`
}

type ModelInstanceState struct {
	ModelID    string      `json:"model_id"`
	Visible    bool        `json:"visible"`
	Transform  *Transform  `json:"transform,omitempty"`
	BoundsSize *BoundsSize `json:"bounds_size,omitempty"`
}

// 生产级desired状态，server希望UE当前应加载哪些模型，Server重启，UE重连时他恢复
type SceneDesiredConfig struct {
	Version            int                 `json:"version"`
	SceneModelsDesired []SceneModelDesired `json:"scene_models_desired"`
}

type SceneModelDesired struct {
	InstanceID  string         `json:"instance_id"`
	ModelID     string         `json:"model_id"`
	URI         string         `json:"uri"`
	Format      string         `json:"format"`
	SourcePath  string         `json:"source_path"`
	DisplayName string         `json:"display_name"`
	BoundsSize  *BoundsSize    `json:"bounds_size,omitempty"`
	Visible     bool           `json:"visible"`
	Transform   *Transform     `json:"transform,omitempty"`
	Extra       map[string]any `json:"-"`
}

func (m SceneModelDesired) MarshalJSON() ([]byte, error) {
	type alias SceneModelDesired
	base, err := json.Marshal(alias(m))
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(base, &out); err != nil {
		return nil, err
	}
	for key, value := range m.Extra {
		if _, exists := out[key]; exists {
			continue
		}
		out[key] = value
	}
	return json.Marshal(out)
}
