package entity

import "time"

// ModelBinding 保存现实业务对象与 UE 模型/点位的绑定关系。
type ModelBinding struct {
	ID           uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	BindModelID  string `gorm:"uniqueIndex;size:128;not null" json:"bind_model_id"` // UE instance_id 或自定义绑定点位 ID
	ObjectType   string `gorm:"size:64;not null" json:"object_type"`                // gate / building / room / device
	ObjectID     string `gorm:"size:128;not null" json:"object_id"`                 // gate_01
	DataKeysJSON string `gorm:"type:json" json:"data_keys_json"`                    // ["security.access.today"]

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (ModelBinding) TableName() string {
	return "model_bindings"
}
