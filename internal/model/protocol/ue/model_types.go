package ue

type ModelTypesConfig struct {
	Version    int         `json:"version"`
	ModelTypes []ModelType `json:"model_types"`
}

type ModelType struct {
	ModelID                  string     `json:"model_id"`
	URI                      string     `json:"uri"`
	Format                   string     `json:"format"`
	SourcePath               string     `json:"source_path"`
	BaseBoundsSize           BoundsSize `json:"base_bounds_size"`
	BoundsNegativeExtentAxis *int       `json:"bounds_negative_extent_axis"`
	Enabled                  bool       `json:"enabled"`
}

type StructuredModelConfig struct {
	Version    int                            `json:"version"`
	Placements map[string]ModelPlacementState `json:"placements"`
}

type ModelPlacementState struct {
	Visible   bool       `json:"visible"`
	Transform *Transform `json:"transform,omitempty"`
}
