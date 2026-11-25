package entity

import "time"

type UEModelType struct {
	ModelID string `gorm:"primaryKey;size:128;not null" json:"model_id"`

	URI        string `gorm:"size:1024;not null" json:"uri"`
	Format     string `gorm:"size:64;not null;default:udatasmith" json:"format"`
	SourcePath string `gorm:"size:1024;not null" json:"source_path"`

	BaseBoundsX float64 `gorm:"column:base_bounds_x;not null" json:"base_bounds_x"`
	BaseBoundsY float64 `gorm:"column:base_bounds_y;not null" json:"base_bounds_y"`
	BaseBoundsZ float64 `gorm:"column:base_bounds_z;not null" json:"base_bounds_z"`

	BoundsNegativeExtentAxis *int `gorm:"column:bounds_negative_extent_axis" json:"bounds_negative_extent_axis,omitempty"`

	Enabled bool `gorm:"not null;default:true" json:"enabled"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (UEModelType) TableName() string {
	return "ue_model_types"
}
