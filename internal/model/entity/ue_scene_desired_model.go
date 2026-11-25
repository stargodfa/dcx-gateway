package entity

import "time"

type UESceneDesiredModel struct {
	ID uint64 `gorm:"primaryKey;autoIncrement" json:"id"`

	InstanceID string `gorm:"uniqueIndex;size:128;not null" json:"instance_id"`
	ModelID    string `gorm:"size:128;not null" json:"model_id"`

	URI        string `gorm:"size:1024;not null" json:"uri"`
	Format     string `gorm:"size:64;not null;default:udatasmith" json:"format"`
	SourcePath string `gorm:"size:1024;not null" json:"source_path"`

	DisplayName string `gorm:"size:255;not null" json:"display_name"`
	Visible     bool   `gorm:"not null;default:true" json:"visible"`

	BoundsX *float64 `gorm:"column:bounds_x" json:"bounds_x,omitempty"`
	BoundsY *float64 `gorm:"column:bounds_y" json:"bounds_y,omitempty"`
	BoundsZ *float64 `gorm:"column:bounds_z" json:"bounds_z,omitempty"`

	PositionX float64 `gorm:"column:position_x;not null;default:0" json:"position_x"`
	PositionY float64 `gorm:"column:position_y;not null;default:0" json:"position_y"`
	PositionZ float64 `gorm:"column:position_z;not null;default:0" json:"position_z"`

	RotationYaw   float64 `gorm:"column:rotation_yaw;not null;default:0" json:"rotation_yaw"`
	RotationPitch float64 `gorm:"column:rotation_pitch;not null;default:0" json:"rotation_pitch"`
	RotationRoll  float64 `gorm:"column:rotation_roll;not null;default:0" json:"rotation_roll"`

	ScaleX float64 `gorm:"column:scale_x;not null;default:1" json:"scale_x"`
	ScaleY float64 `gorm:"column:scale_y;not null;default:1" json:"scale_y"`
	ScaleZ float64 `gorm:"column:scale_z;not null;default:1" json:"scale_z"`

	Enabled    bool   `gorm:"not null;default:true" json:"enabled"`
	SortOrder  int    `gorm:"not null;default:0" json:"sort_order"`
	ConfigJSON string `gorm:"type:longtext" json:"config_json,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (UESceneDesiredModel) TableName() string {
	return "ue_scene_desired_models"
}
