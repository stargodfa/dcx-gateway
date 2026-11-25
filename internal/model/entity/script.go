package entity

import "time"

// Script 是本地网关保存的汇报脚本配置。
// 它描述“有哪些脚本”，不保存实时业务数据。
type Script struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	ScriptID string `gorm:"uniqueIndex;size:64;not null" json:"script_id"`
	Name     string `gorm:"size:128;not null" json:"name"`
	SceneID  string `gorm:"size:64;not null" json:"scene_id"`
	Status   string `gorm:"size:32;not null;default:enabled" json:"status"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Script) TableName() string {
	return "scripts"
}
