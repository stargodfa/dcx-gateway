package ue

// ChartDataPush 是运行时图表数据推送 payload。
// Go 网关可通过 /ws_data 推给 FastAPI，再由 FastAPI 根据 binding_id 更新 chart_id。
type ChartDataPush struct {
	BindingID  string     `json:"binding_id"`
	Timestamp  string     `json:"timestamp,omitempty"`
	Value      any        `json:"value,omitempty"`
	Rows       [][]string `json:"rows,omitempty"`
	CellValues [][]string `json:"cell_values,omitempty"`
	Payload    any        `json:"payload,omitempty"`
}
