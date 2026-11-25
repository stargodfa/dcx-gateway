package domain

// LabDeviceStatus 实验室/科研设备进入网关后端业务对象
type LabDeviceStatus struct {
	DeviceID  string  `json:"device_id"`
	LabID     string  `json:"lab_id"`
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	Value     float64 `json:"value"`
	Unit      string  `json:"unit,omitempty"`
	Timestamp string  `json:"timestamp"`
}

type LabSnapshot struct {
	Devices []LabDeviceStatus `json:"devices"`
}
