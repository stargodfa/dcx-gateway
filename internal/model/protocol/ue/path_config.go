package ue

// 导航路径预设和播放参数
// 对应path_config.go
type PathConfig struct {
	Version     int                 `json:"version"`
	NavPlayback NavPlayback         `json:"nav_playback"`
	PlansByID   map[string]PathPlan `json:"plans_by_id"`
}

type NavPlayback struct {
	UseDuration      bool    `json:"use_duration"`
	PlannedDurationS float64 `json:"planned_duration_s"`
	PlannedSpeedCMS  float64 `json:"planned_speed_cm_s"`
}

type PathPlan struct {
	PathID    string    `json:"path_id"`
	Name      string    `json:"name"`
	Waypoints []Vector3 `json:"waypoints"`
}
