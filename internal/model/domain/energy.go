package domain

// BuildingEnergyMetric	能耗尽系统进入网关后的业务对象
type BuildingEnergyMetric struct {
	BuildingID string  `json:"building_id"`
	Name       string  `json:"name"`
	Value      float64 `json:"value"`
	Unit       string  `json:"unit"`
	Timestamp  string  `json:"timestamp"`
}

type EnergySnapshot struct {
	Buildings []BuildingEnergyMetric `json:"buildings"`
}
