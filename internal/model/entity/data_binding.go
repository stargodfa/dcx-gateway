package entity

import "time"

//绑定数据

// DataBinding保存业务数据 key与 UE/FastAPI图表绑定关系
type DataBinding struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	DataKey   string `gorm:"uniqueIndex;size:128;not null" json:"data_key"` //security.access.today
	BindingID string `gorm:"size:128;not null" json:"binding_id"`           //chart_data_push 使用
	ChartID   string `gorm:"size:128" json:"chart_id"`
	ModuleID  string `gorm:"size:128" json:"module_id"`
	Enabled   bool   `gorm:"not null;default:true" json:"enabled"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (DataBinding) TableName() string {
	return "data_bindings"
}
