package dto

// PreloadRequest 用于 POST /api/scripts/:id/preload。

type PreloadRequest struct {
	Force bool `json:"force"`
}

type PreloadResponse struct {
	TaskID   string `json:"task_id"`
	ScriptID string `json:"script_id"`
	Status   string `json:"status"`
}
