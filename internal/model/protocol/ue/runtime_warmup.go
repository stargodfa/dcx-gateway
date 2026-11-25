package ue

// WarmupOptions is the editable resource catalog for UE report scene warmup.
// Data scripts are Gateway DataBundle scripts; report_scripts are a separate UE
// report choreography document.
type WarmupOptions struct {
	Version     int                      `json:"version"`
	DataScripts []WarmupDataScriptOption `json:"data_scripts"`
	Charts      []WarmupChartOption      `json:"charts"`
	Videos      []WarmupVideoOption      `json:"videos"`
	Instances   []WarmupInstanceOption   `json:"instances"`
}

type WarmupDataScriptOption struct {
	DataScriptID string   `json:"data_script_id"`
	Label        string   `json:"label"`
	DataSceneID  string   `json:"data_scene_id"`
	DataKeys     []string `json:"data_keys"`
}

type WarmupChartOption struct {
	BindingID    string `json:"binding_id"`
	ChartID      string `json:"chart_id,omitempty"`
	Title        string `json:"title"`
	DataKey      string `json:"data_key"`
	DataScriptID string `json:"data_script_id,omitempty"`
	ModuleID     string `json:"module_id,omitempty"`
}

type WarmupVideoOption struct {
	VideoID      string `json:"video_id"`
	Title        string `json:"title"`
	PlayURL      string `json:"play_url"`
	DataScriptID string `json:"data_script_id,omitempty"`
	ModuleID     string `json:"module_id,omitempty"`
}

type WarmupInstanceOption struct {
	InstanceID  string `json:"instance_id"`
	ModelID     string `json:"model_id"`
	DisplayName string `json:"display_name"`
	Enabled     bool   `json:"enabled"`
}

type ReportScenePreloadResponse struct {
	Version   int                        `json:"version"`
	ScriptID  string                     `json:"script_id"`
	ChapterID string                     `json:"chapter_id"`
	SceneID   string                     `json:"scene_id"`
	Warmup    ReportWarmup               `json:"warmup"`
	Results   []ReportScenePreloadResult `json:"results"`
}

type ReportScenePreloadResult struct {
	DataScriptID    string   `json:"data_script_id"`
	Force           bool     `json:"force"`
	PushWS          bool     `json:"push_ws"`
	Success         bool     `json:"success"`
	BundleID        string   `json:"bundle_id,omitempty"`
	Error           string   `json:"error,omitempty"`
	ChartsRequested []string `json:"charts_requested"`
	VideosRequested []string `json:"videos_requested"`
}
