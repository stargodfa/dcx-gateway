package entity

import "time"

// UEAuxConfigRecord is the common read model for the four supplemental tables.
// Payload retains the complete JSON record, including fields unknown to Gateway.
type UEAuxConfigRecord struct {
	ID         string    `gorm:"column:id"`
	InstanceID string    `gorm:"column:instance_id"`
	Revision   int       `gorm:"column:revision"`
	Enabled    bool      `gorm:"column:enabled"`
	Payload    string    `gorm:"column:payload"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}
