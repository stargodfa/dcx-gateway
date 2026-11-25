package ue

// UE 渲染/室内显示调节。enabled=true时会转成UE ConsoleCVar下发，false时不干预UE默认光照/后处理
// render_config.json
type RenderConfig struct {
	Version  int            `json:"version"`
	Enabled  bool           `json:"enabled"`
	Settings RenderSettings `json:"settings"`
}

type RenderSettings struct {
	Shadows              bool `json:"shadows"`
	AO                   bool `json:"ao"`
	Bloom                bool `json:"bloom"`
	EyeAdaptation        bool `json:"eye_adaptation"`
	ExtendLuminanceRange bool `json:"extend_luminance_range"`

	LocalExposureHighlight float64 `json:"local_exposure_highlight"`
	LocalExposureShadow    float64 `json:"local_exposure_shadow"`
	ExposureBias           float64 `json:"exposure_bias"`

	TonemapperGamma float64 `json:"tonemapper_gamma"`
	Contrast        float64 `json:"contrast"`
	Saturation      float64 `json:"saturation"`
	Sharpen         float64 `json:"sharpen"`
	Vignette        float64 `json:"vignette"`
}
