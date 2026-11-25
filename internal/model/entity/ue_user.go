package entity

import "time"

type UEUser struct {
	ID           uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string `gorm:"uniqueIndex:uk_username;size:64;not null" json:"username"`
	PasswordHash string `gorm:"size:255;not null" json:"-"`
	Role         string `gorm:"size:32;not null;default:operator" json:"role"`
	Enabled      bool   `gorm:"not null;default:true;index:idx_enabled" json:"enabled"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (UEUser) TableName() string {
	return "ue_users"
}
