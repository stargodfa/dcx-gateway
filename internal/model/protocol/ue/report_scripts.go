package ue

// 汇报脚本编排:脚本 -> 章节 -> 场景。用于场景切换、相机跳转、模型显隐和预热资源。
// report_scripts.json
type ReportScripts struct {
	Version int            `json:"version"`
	Scripts []ReportScript `json:"scripts"`
}

type ReportScript struct {
	ScriptID string          `json:"script_id"`
	Label    string          `json:"label"`
	Chapters []ReportChapter `json:"chapters"`
}

type ReportChapter struct {
	ChapterID string        `json:"chapter_id"`
	Label     string        `json:"label"`
	Scenes    []ReportScene `json:"scenes"`
}

type ReportScene struct {
	SceneID  string               `json:"scene_id"`
	Label    string               `json:"label"`
	Camera   *Camera              `json:"camera"`
	Bindings []ReportModelBinding `json:"bindings"`
	Warmup   ReportWarmup         `json:"warmup"`
}

type ReportModelBinding struct {
	InstanceID string     `json:"instance_id"`
	ModelID    string     `json:"model_id"`
	Visible    bool       `json:"visible"`
	Transform  *Transform `json:"transform"`
}

type ReportWarmup struct {
	// 这里先用 []string，表示数据源 ID / binding ID。
	// 当前样例是空数组，但后面大概率会放数据绑定 ID。
	Data   []string `json:"data"`
	Videos []string `json:"videos"`
	Charts []string `json:"charts"`
}
