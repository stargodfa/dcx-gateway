package entity

import "time"

type UEModelInstance struct {
	InstanceID string `gorm:"primaryKey;size:128;not null" json:"instance_id"`
	ModelID    string `gorm:"size:128;not null;index:idx_model_id" json:"model_id"`

	DisplayName string `gorm:"size:255;not null;default:''" json:"display_name"`
	SortOrder   int    `gorm:"not null;default:0" json:"sort_order"`
	Enabled     bool   `gorm:"not null;default:true;index:idx_enabled" json:"enabled"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (UEModelInstance) TableName() string {
	return "ue_model_instances"
}
