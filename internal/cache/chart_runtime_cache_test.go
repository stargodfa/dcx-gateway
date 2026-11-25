package cache

import "testing"

func TestChartRuntimeDataEncodeDecode(t *testing.T) {
	data := ChartRuntimeData{
		DataKey:   "security.access.today",
		BindingID: "sim_3",
		ChartID:   "chart_h",
		CellValues: [][]string{
			{"指标", "数值"},
			{"今日进入", "1800"},
		},
		UpdatedAt: "2026-06-02T00:00:00Z",
	}

	encoded, err := encodeChartRuntimeData(data)
	if err != nil {
		t.Fatalf("encodeChartRuntimeData returned error: %v", err)
	}
	decoded, err := decodeChartRuntimeData(encoded)
	if err != nil {
		t.Fatalf("decodeChartRuntimeData returned error: %v", err)
	}
	if decoded.BindingID != "sim_3" || decoded.ChartID != "chart_h" {
		t.Fatalf("decoded metadata = %#v", decoded)
	}
	if decoded.CellValues[1][1] != "1800" {
		t.Fatalf("decoded cell values = %#v", decoded.CellValues)
	}
}
