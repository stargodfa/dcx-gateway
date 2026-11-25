package domain

// VideoStatus 是视频资源运行时状态
type VideoStatus struct {
	VideoID   string `json:"video_id"`
	Status    string `json:"status"` //online / offline/unknown
	LatencyMS int64  `json:"latency_ms,omitempty"`
	Timestamp string `json:"timestamp"`
}
