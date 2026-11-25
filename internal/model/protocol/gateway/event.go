package gateway

// Event 是 Go网关内部或对FastAPI推送的统一事件结构
type Event struct {
	EventID   string `json:"event_id"`
	Type      string `json:"type"`
	Source    string `json:"source"`
	Timestamp string `json:"timestamp"`
	Payload   any    `json:"payload"`
}
