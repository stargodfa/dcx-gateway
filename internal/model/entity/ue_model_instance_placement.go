package entity

import "time"

type UEModelInstancePlacement struct {
	InstanceID string `gorm:"primaryKey;size:128;not null" json:"instance_id"`

	Visible bool `gorm:"not null;default:true;index:idx_visible" json:"visible"`

	PositionX float64 `gorm:"column:position_x;not null;default:0" json:"position_x"`
	PositionY float64 `gorm:"column:position_y;not null;default:0" json:"position_y"`
	PositionZ float64 `gorm:"column:position_z;not null;default:0" json:"position_z"`

	RotationYaw   float64 `gorm:"column:rotation_yaw;not null;default:0" json:"rotation_yaw"`
	RotationPitch float64 `gorm:"column:rotation_pitch;not null;default:0" json:"rotation_pitch"`
	RotationRoll  float64 `gorm:"column:rotation_roll;not null;default:0" json:"rotation_roll"`

	ScaleX float64 `gorm:"column:scale_x;not null;default:1" json:"scale_x"`
	ScaleY float64 `gorm:"column:scale_y;not null;default:1" json:"scale_y"`
	ScaleZ float64 `gorm:"column:scale_z;not null;default:1" json:"scale_z"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (UEModelInstancePlacement) TableName() string {
	return "ue_model_instance_placements"
}
