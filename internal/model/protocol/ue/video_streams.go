package ue

// 视频源内容
// VideoStreams 对应 UE/FastAPI 现有 video_streams.json。
type VideoStreams struct {
	Version int           `json:"version"`
	Streams []VideoStream `json:"streams"`
}

type VideoStream struct {
	StreamID        string `json:"stream_id"`
	Name            string `json:"name"`
	WebRTCPlayerURL string `json:"webrtc_player_url"`
}
