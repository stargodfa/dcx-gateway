package domain

// AccessSummary是外部安防/门禁系统进入网关后的业务对象
type AccessSummary struct {
	InCount    int64  `json:"in_count"`
	OutCount   int64  `json:"out_count"`
	TotalCount int64  `json:"total_count"`
	Timestamp  string `json:"timestamp"`
}

type FireAlert struct {
	AlertID   string `json:"alert_id"`
	Level     string `json:"level"` //info/warning/critical
	Location  string `json:"location"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

type SecuritySnapshot struct {
	AccessSummary AccessSummary `json:"access_summary"`
	FireAlerts    []FireAlert   `json:"fire_alerts"`
}
