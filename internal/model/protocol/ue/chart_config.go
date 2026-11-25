package ue

// ChartConfig 对应 UE/FastAPI 现有 chart_config.json。chart表格配置
// 前端面板渲染的表格配置
type ChartConfig struct {
	Version                 int          `json:"version"`
	OverlayHUDMaxPanelWidth int          `json:"overlay_hud_max_panel_width"`
	Charts                  []ChartPanel `json:"charts"`
}

type ChartPanel struct {
	ChartID   string `json:"chart_id"`
	ChartName string `json:"chart_name"`
	Visible   bool   `json:"visible"`
	PanelKey  string `json:"panel_key,omitempty"`

	Rows int `json:"rows,omitempty"`
	Cols int `json:"cols,omitempty"`

	ColFields []string `json:"col_fields,omitempty"`
	RowFields []string `json:"row_fields,omitempty"`

	Applied    bool   `json:"applied,omitempty"`
	LayoutMode string `json:"layout_mode,omitempty"`

	PairLabelCols []int `json:"pair_label_cols,omitempty"`
	PairLabelRows []int `json:"pair_label_rows,omitempty"`

	ColAttrEnabled bool `json:"col_attr_enabled,omitempty"`
	RowAttrEnabled bool `json:"row_attr_enabled,omitempty"`

	HeaderRow int `json:"header_row,omitempty"`
	LabelCol  int `json:"label_col,omitempty"`

	// JSON 里是二维数组 cell_values，不是 cell_value，也不是 []string。
	CellValues [][]string `json:"cell_values,omitempty"`
}
