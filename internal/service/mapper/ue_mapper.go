package mapper

import (
	gateway "dcx-gateway/internal/model/protocol/gateway"
	"dcx-gateway/internal/model/protocol/ue"
)

const (
	MsgTypeChartDataPush   = "chart_data_push"
	MsgTypeVideoStreamsSet = "video_streams_set"
	MsgTypeGatewayEvent    = "gateway_event"
)

// BundleToChartDataPushMessages 将网关 DataBundle 中的 Charts 转成 FastAPI /ws_data 可消费的运行时图表消息。
func BundleToChartDataPushMessages(bundle *gateway.DataBundle, seqStart int64) []ue.WSMessage {
	if bundle == nil {
		return nil
	}

	messages := make([]ue.WSMessage, 0, len(bundle.Charts))
	seq := seqStart
	for _, chart := range bundle.Charts {
		if chart.BindingID == "" {
			continue
		}

		payload := ue.ChartDataPush{
			BindingID:  chart.BindingID,
			Timestamp:  bundle.GeneratedAt,
			Rows:       chart.Rows,
			CellValues: chart.Rows,
		}

		if len(chart.Series) > 0 || len(chart.XAxis) > 0 {
			payload.Payload = chart
		}

		messages = append(messages, ue.WSMessage{
			Type:    MsgTypeChartDataPush,
			Seq:     seq,
			Payload: payload,
		})
		seq++
	}

	return messages
}

// BundleToVideoStreams 将网关视频源转成 UE/FastAPI 现有 video_streams 结构。
func BundleToVideoStreams(bundle *gateway.DataBundle) ue.VideoStreams {
	streams := make([]ue.VideoStream, 0)
	if bundle == nil {
		return ue.VideoStreams{Version: 1, Streams: streams}
	}

	for _, video := range bundle.Videos {
		streams = append(streams, ue.VideoStream{
			StreamID:        video.VideoID,
			Name:            video.Title,
			WebRTCPlayerURL: video.PlayURL,
		})
	}

	return ue.VideoStreams{Version: 1, Streams: streams}
}

// BundleToWSMessages 生成一组可推给 FastAPI /ws_data 的消息。
// 默认只推 chart_data_push；video_streams_set 仅在显式开启时保留兼容输出。
func BundleToWSMessages(bundle *gateway.DataBundle, seqStart int64, includeVideoStreams ...bool) []ue.WSMessage {
	messages := BundleToChartDataPushMessages(bundle, seqStart)
	pushVideos := len(includeVideoStreams) > 0 && includeVideoStreams[0]
	if pushVideos && bundle != nil && len(bundle.Videos) > 0 {
		messages = append(messages, ue.WSMessage{
			Type:    MsgTypeVideoStreamsSet,
			Seq:     seqStart + int64(len(messages)),
			Payload: BundleToVideoStreams(bundle),
		})
	}
	return messages
}
