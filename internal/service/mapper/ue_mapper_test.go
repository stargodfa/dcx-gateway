package mapper

import (
	"testing"

	gateway "dcx-gateway/internal/model/protocol/gateway"
	"dcx-gateway/internal/model/protocol/ue"
)

func TestBundleToChartDataPushMessagesRowsToCellValues(t *testing.T) {
	rows := [][]string{
		{"metric", "value"},
		{"today", "1800"},
	}
	bundle := &gateway.DataBundle{
		GeneratedAt: "2026-06-02T00:00:00Z",
		Charts: []gateway.ChartData{
			{
				BindingID: "security.access.today",
				Rows:      rows,
			},
		},
	}

	messages := BundleToChartDataPushMessages(bundle, 7)
	if len(messages) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(messages))
	}
	if messages[0].Type != MsgTypeChartDataPush {
		t.Fatalf("message type = %q", messages[0].Type)
	}
	if messages[0].Seq != 7 {
		t.Fatalf("seq = %d, want 7", messages[0].Seq)
	}
	payload, ok := messages[0].Payload.(ue.ChartDataPush)
	if !ok {
		t.Fatalf("payload type = %T, want ue.ChartDataPush", messages[0].Payload)
	}
	if len(payload.CellValues) != len(rows) {
		t.Fatalf("len(CellValues) = %d, want %d", len(payload.CellValues), len(rows))
	}
	if payload.CellValues[1][1] != "1800" {
		t.Fatalf("CellValues[1][1] = %q", payload.CellValues[1][1])
	}
	if len(payload.Rows) != len(rows) {
		t.Fatalf("Rows should be preserved for compatibility")
	}
}

func TestBundleToChartDataPushMessagesSkipsEmptyBindingID(t *testing.T) {
	bundle := &gateway.DataBundle{
		Charts: []gateway.ChartData{
			{Rows: [][]string{{"metric", "value"}}},
		},
	}

	messages := BundleToChartDataPushMessages(bundle, 0)
	if len(messages) != 0 {
		t.Fatalf("len(messages) = %d, want 0", len(messages))
	}
}

func TestBundleToWSMessagesSkipsVideoStreamsSetByDefault(t *testing.T) {
	bundle := &gateway.DataBundle{
		Videos: []gateway.VideoSource{
			{
				VideoID: "vs_cam_65",
				Title:   "gate camera",
				PlayURL: "http://example.test/stream.html?src=cam_65",
			},
		},
	}

	messages := BundleToWSMessages(bundle, 0)
	if len(messages) != 0 {
		t.Fatalf("len(messages) = %d, want 0", len(messages))
	}
}

func TestBundleToWSMessagesKeepsVideoStreamsSetWhenEnabled(t *testing.T) {
	bundle := &gateway.DataBundle{
		Videos: []gateway.VideoSource{
			{
				VideoID: "vs_cam_65",
				Title:   "gate camera",
				PlayURL: "http://example.test/stream.html?src=cam_65",
			},
		},
	}

	messages := BundleToWSMessages(bundle, 0, true)
	if len(messages) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(messages))
	}
	if messages[0].Type != MsgTypeVideoStreamsSet {
		t.Fatalf("message type = %q, want %q", messages[0].Type, MsgTypeVideoStreamsSet)
	}
	payload, ok := messages[0].Payload.(ue.VideoStreams)
	if !ok {
		t.Fatalf("payload type = %T, want ue.VideoStreams", messages[0].Payload)
	}
	if len(payload.Streams) != 1 || payload.Streams[0].StreamID != "vs_cam_65" {
		t.Fatalf("payload = %#v", payload)
	}
}
