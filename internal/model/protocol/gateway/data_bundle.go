package gateway

// DataBundle 是 Go 数据网关输出给 FastAPI 的统一展示数据包。
// 它不等于 UE 原生协议，后续可通过 mapper 转成 protocol/ue。
type DataBundle struct {
	BundleID string `json:"bundle_id"`
	ScriptID string `json:"script_id"`
	SceneID  string `json:"scene_id"`
	// data_script_id/data_scene_id are aliases for the Gateway DataBundle
	// identity. They avoid ambiguity with UE report_scripts script_id/scene_id.
	DataScriptID string `json:"data_script_id,omitempty"`
	DataSceneID  string `json:"data_scene_id,omitempty"`
	Version      string `json:"version"`
	GeneratedAt  string `json:"generated_at"`
	TTLSeconds   int    `json:"ttl_seconds,omitempty"`

	Modules       []ModuleData   `json:"modules"`
	Charts        []ChartData    `json:"charts"`
	Videos        []VideoSource  `json:"videos"`
	ModelBindings []ModelBinding `json:"model_bindings"`
}

type ModuleData struct {
	ModuleID   string `json:"module_id"`
	Type       string `json:"type"`
	Title      string `json:"title"`
	DataKey    string `json:"data_key,omitempty"`
	TargetKind string `json:"target_kind,omitempty"` // chart / video / model / hud
	TargetID   string `json:"target_id,omitempty"`   // chart_id / stream_id / instance_id

	Value     any    `json:"value,omitempty"`
	Unit      string `json:"unit,omitempty"`
	Payload   any    `json:"payload,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

type ChartData struct {
	ModuleID  string        `json:"module_id"`
	DataKey   string        `json:"data_key,omitempty"`
	BindingID string        `json:"binding_id,omitempty"`
	ChartID   string        `json:"chart_id,omitempty"`
	Template  string        `json:"template"`
	Title     string        `json:"title"`
	XAxis     []string      `json:"x_axis,omitempty"`
	Series    []ChartSeries `json:"series,omitempty"`
	Rows      [][]string    `json:"rows,omitempty"`
}

type ChartSeries struct {
	Name string    `json:"name"`
	Unit string    `json:"unit"`
	Data []float64 `json:"data"`
}

type VideoSource struct {
	VideoID     string `json:"video_id"`
	Title       string `json:"title"`
	PlayURL     string `json:"play_url"`
	Status      string `json:"status"`
	BindModelID string `json:"bind_model_id,omitempty"`
}

type ModelBinding struct {
	BindModelID string   `json:"bind_model_id"`
	ObjectType  string   `json:"object_type"`
	ObjectID    string   `json:"object_id"`
	DataKeys    []string `json:"data_keys"`
}
