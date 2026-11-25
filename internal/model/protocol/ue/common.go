package ue

// Vector3 / Rotation / Transform 是 UE 协议中的基础几何结构。
// 坐标，视角，模型大小
type Vector3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type Rotation struct {
	Yaw   float64 `json:"yaw"`
	Pitch float64 `json:"pitch"`
	Roll  float64 `json:"roll"`
}

// Transform 在 report_scripts.json 里可能为 null，也可能只有 position。
// 所以外层字段用 *Transform，内部字段也用指针，兼容“只给 position”的情况。
type Transform struct {
	Position *Vector3  `json:"position,omitempty"`
	Rotation *Rotation `json:"rotation,omitempty"`
	Scale    *Vector3  `json:"scale,omitempty"`
}

type Camera struct {
	CameraID string   `json:"camera_id"`
	Position Vector3  `json:"position"`
	Rotation Rotation `json:"rotation"`
}

type BoundsSize struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}
