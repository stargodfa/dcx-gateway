package entity

import "time"

// UEConfigDocument stores the original UE/FastAPI config JSON document.
type UEConfigDocument struct {
	ID          uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	ConfigKey   string `gorm:"uniqueIndex;size:128;not null" json:"config_key"`
	FileName    string `gorm:"size:255;not null" json:"file_name"`
	ContentJSON string `gorm:"type:longtext;not null" json:"content_json"`
	Source      string `gorm:"size:64;not null;default:ue_config" json:"source"`
	Enabled     bool   `gorm:"not null;default:true" json:"enabled"`
	Version     string `gorm:"size:64;not null;default:v1" json:"version"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (UEConfigDocument) TableName() string {
	return "ue_config_documents"
}
