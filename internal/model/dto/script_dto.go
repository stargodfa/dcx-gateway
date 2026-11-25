package dto

// GetDataBundleURI 用于 GET /api/scripts/:id/data-bundle。
type GetDataBundleURI struct {
	ScriptID string `uri:"id" binding:"required"`
}

type GetDataBundleQuery struct {
	UseCache bool `form:"use_cache"`
}
