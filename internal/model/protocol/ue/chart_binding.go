package ue

// ChartBinding maps the existing UE/FastAPI chart_binding.json.
// It binds binding_id to chart_id for real-time data.
type ChartBinding struct {
	Version  int                `json:"version"`
	Bindings []ChartBindingItem `json:"bindings"`
}

type ChartBindingItem struct {
	BindingID string `json:"binding_id"`
	ChartID   string `json:"chart_id"`
}
