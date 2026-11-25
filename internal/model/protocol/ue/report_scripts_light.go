package ue

type LightReportScripts struct {
	Version int                 `json:"version"`
	Scripts []LightReportScript `json:"scripts"`
}

type LightReportScript struct {
	ScriptID string               `json:"script_id"`
	Label    string               `json:"label"`
	Chapters []LightReportChapter `json:"chapters"`
}

type LightReportChapter struct {
	ChapterID string             `json:"chapter_id"`
	Label     string             `json:"label"`
	Scenes    []LightReportScene `json:"scenes"`
}

type LightReportScene struct {
	SceneID string       `json:"scene_id"`
	Label   string       `json:"label"`
	Warmup  ReportWarmup `json:"warmup,omitempty"`
}
