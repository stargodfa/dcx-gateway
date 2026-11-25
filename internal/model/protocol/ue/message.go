package ue

// WSMessage 是发给 FastAPI /ws_data 的通用消息外壳。
// 如果 FastAPI 现有协议有 seq/request_id 字段，可继续在这里扩展。
type WSMessage struct {
	Type    string `json:"type"`
	Seq     int64  `json:"seq,omitempty"`
	Payload any    `json:"payload"`
}
