package entity

import "time"

// VideoSource 是视频源的长期配置。
// 在线状态、延迟等运行时状态应放 Redis/cache，不主要放 MySQL。
type VideoSource struct {
	ID          uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	VideoID     string `gorm:"uniqueIndex;size:64;not null" json:"video_id"`
	Name        string `gorm:"size:128;not null" json:"name"`
	PlayURL     string `gorm:"size:512;not null" json:"play_url"`
	SourceType  string `gorm:"size:64;not null;default:mock" json:"source_type"`
	BindModelID string `gorm:"size:128" json:"bind_model_id"`

	Enabled bool `gorm:"not null;default:true" json:"enabled"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (VideoSource) TableName() string {
	return "video_sources"
}
