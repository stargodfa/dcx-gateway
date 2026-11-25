package entity

import "time"

// UEChartConfigSettings stores chart_config top-level display settings.
type UEChartConfigSettings struct {
	ID                      uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Version                 int       `gorm:"not null;default:1" json:"version"`
	OverlayHUDMaxPanelWidth int       `gorm:"not null;default:480" json:"overlay_hud_max_panel_width"`
	Enabled                 bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

func (UEChartConfigSettings) TableName() string {
	return "ue_chart_config_settings"
}

// UEChartConfig stores one chart_config.charts[] entry as static layout config.
type UEChartConfig struct {
	ID                    uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ChartID               string    `gorm:"uniqueIndex;size:128;not null" json:"chart_id"`
	ChartName             string    `gorm:"size:255;not null" json:"chart_name"`
	PanelKey              *string   `gorm:"size:128" json:"panel_key,omitempty"`
	Visible               bool      `gorm:"not null;default:true" json:"visible"`
	Applied               bool      `gorm:"not null;default:true" json:"applied"`
	LayoutMode            *string   `gorm:"size:64" json:"layout_mode,omitempty"`
	RowsCount             *int      `json:"rows_count,omitempty"`
	ColsCount             *int      `json:"cols_count,omitempty"`
	ColFieldsJSON         string    `gorm:"type:json" json:"col_fields_json,omitempty"`
	RowFieldsJSON         string    `gorm:"type:json" json:"row_fields_json,omitempty"`
	PairLabelColsJSON     string    `gorm:"type:json" json:"pair_label_cols_json,omitempty"`
	PairLabelRowsJSON     string    `gorm:"type:json" json:"pair_label_rows_json,omitempty"`
	DefaultCellValuesJSON string    `gorm:"type:json" json:"default_cell_values_json,omitempty"`
	Template              *string   `gorm:"size:128" json:"template,omitempty"`
	Enabled               bool      `gorm:"not null;default:true" json:"enabled"`
	SortOrder             int       `gorm:"not null;default:0" json:"sort_order"`
	ConfigJSON            string    `gorm:"type:json" json:"config_json,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

func (UEChartConfig) TableName() string {
	return "ue_chart_configs"
}
