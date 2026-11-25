package entity

import "time"

// ScriptModule 是脚本内的展示模块配置。
// module_id 是 Go 网关自己的业务展示模块 ID，不要求 UE 直接识别。
// target_kind/target_id 用于把业务模块映射到 UE/FastAPI 已有资源，例如 chart_id、stream_id、instance_id。
type ScriptModule struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	ModuleID string `gorm:"uniqueIndex;size:64;not null" json:"module_id"`
	ScriptID string `gorm:"index;size:64;not null" json:"script_id"`

	Type     string `gorm:"size:64;not null" json:"type"` // stat_card / chart / video / alert_list / model_point / hud
	Title    string `gorm:"size:128;not null" json:"title"`
	DataKey  string `gorm:"size:128" json:"data_key"` // security.access.today
	Template string `gorm:"size:64" json:"template"`  // stat_card_basic / bar_basic / table_basic

	TargetKind string `gorm:"size:32" json:"target_kind"` // chart / video / model / hud
	TargetID   string `gorm:"size:128" json:"target_id"`  // chart_id / stream_id / instance_id

	SortOrder  int    `gorm:"not null;default:0" json:"sort_order"`
	Enabled    bool   `gorm:"not null;default:true" json:"enabled"`
	ConfigJSON string `gorm:"type:json" json:"config_json"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (ScriptModule) TableName() string {
	return "script_modules"
}
