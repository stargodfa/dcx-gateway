package entity

//数据库图表模板

import "time"

// ChartTemplate 保存图表/卡片/表格模板配置。
type ChartTemplate struct {
	ID         uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	TemplateID string `gorm:"uniqueIndex;size:64;not null" json:"template_id"`
	Name       string `gorm:"size:128;not null" json:"name"`
	Type       string `gorm:"size:64;not null" json:"type"` // chart / stat_card / table / alert_list
	ConfigJSON string `gorm:"type:json" json:"config_json"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (ChartTemplate) TableName() string {
	return "chart_templates"
}
