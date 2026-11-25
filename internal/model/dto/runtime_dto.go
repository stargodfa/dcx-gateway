package dto

type ReportScenePreloadRequest struct {
	ScriptID  string `json:"script_id" binding:"required"`
	ChapterID string `json:"chapter_id" binding:"required"`
	SceneID   string `json:"scene_id" binding:"required"`
	Force     *bool  `json:"force,omitempty"`
	PushWS    *bool  `json:"push_ws,omitempty"`
}
