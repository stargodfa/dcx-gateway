package api

import (
	"dcx-gateway/internal/api/handler"
	"dcx-gateway/internal/model/domain"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	Script      *handler.ScriptHandler
	UEConfig    *handler.UEConfigHandler
	Auth        *handler.AuthHandler
	Health      *handler.HealthHandler
	Runtime     *handler.RuntimeHandler
	UEAuxConfig *handler.UEAuxConfigHandler
}

func RegisterRoutes(r *gin.Engine, h Handlers, middlewares ...gin.HandlerFunc) {
	api := r.Group("/api")
	for _, middleware := range middlewares {
		if middleware != nil {
			api.Use(middleware)
		}
	}

	if h.Health != nil {
		api.GET("/health", h.Health.Get)
	} else {
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})
	}

	api.POST("/auth/login", h.Auth.Login)
	api.POST("/auth/logout", h.Auth.Logout)
	api.GET("/auth/me", h.Auth.Me)

	if h.Runtime != nil {
		api.GET("/runtime/ws-data/status", h.Runtime.GetWSDataStatus)
		api.GET("/ue/runtime/warmup-options", h.Runtime.GetWarmupOptions)
		api.POST("/ue/runtime/report-scenes/preload", h.Runtime.PreloadReportScene)
	}

	api.GET("/scripts/:id/data-bundle", h.Script.GetDataBundle)
	api.POST("/scripts/:id/preload", h.Script.Preload)

	api.GET("/ue/config/chart-binding", h.UEConfig.GetChartBinding)
	api.PUT("/ue/config/chart-binding", h.UEConfig.ReplaceChartBinding)
	api.GET("/ue/config/video-streams", h.UEConfig.GetVideoStreams)
	api.PUT("/ue/config/video-streams", h.UEConfig.ReplaceVideoStreams)
	api.GET("/ue/config/chart-config", h.UEConfig.GetChartConfig)
	api.PUT("/ue/config/chart-config", h.UEConfig.ReplaceChartConfig)
	api.GET("/ue/config/report-scripts", h.UEConfig.GetReportScripts)
	api.PUT("/ue/config/report-scripts", h.UEConfig.ReplaceReportScripts)
	api.POST("/ue/config/report-scripts/scripts/create", h.UEConfig.CreateReportScript)
	api.PUT("/ue/config/report-scripts/scripts", h.UEConfig.UpdateReportScript)
	api.DELETE("/ue/config/report-scripts/scripts/delete", h.UEConfig.DeleteReportScript)
	api.POST("/ue/config/report-scripts/chapters/create", h.UEConfig.CreateReportChapter)
	api.PUT("/ue/config/report-scripts/chapters", h.UEConfig.UpdateReportChapter)
	api.DELETE("/ue/config/report-scripts/chapters/delete", h.UEConfig.DeleteReportChapter)
	api.POST("/ue/config/report-scripts/scenes/create", h.UEConfig.CreateReportScene)
	api.PUT("/ue/config/report-scripts/scenes", h.UEConfig.UpdateReportScene)
	api.DELETE("/ue/config/report-scripts/scenes/delete", h.UEConfig.DeleteReportScene)
	api.PUT("/ue/config/report-scripts/scenes/camera", h.UEConfig.UpdateReportSceneCamera)
	api.POST("/ue/config/report-scripts/scenes/bindings/create", h.UEConfig.CreateReportSceneBindings)
	api.PUT("/ue/config/report-scripts/scenes/bindings", h.UEConfig.UpdateReportSceneBindings)
	api.DELETE("/ue/config/report-scripts/scenes/bindings/delete", h.UEConfig.DeleteReportSceneBindings)
	api.PUT("/ue/config/report-scripts/scenes/warmup", h.UEConfig.UpdateReportSceneWarmup)
	api.GET("/ue/config/raw/:key", h.UEConfig.GetRawConfig)
	api.GET("/ue/config/index", h.UEConfig.GetConfigIndex)
	api.GET("/ue/config/drone-config", h.UEConfig.GetDroneConfig)
	api.PUT("/ue/config/drone-config", h.UEConfig.ReplaceDroneConfig)
	api.GET("/ue/config/bootstrap", h.UEConfig.GetBootstrap)
	api.GET("/ue/config/init-config", h.UEConfig.GetInitConfig)
	api.PUT("/ue/config/init-config", h.UEConfig.UpdateInitConfig)
	api.POST("/ue/config/init-config/instance_ids/create", h.UEConfig.CreateInitInstanceIDs)
	api.DELETE("/ue/config/init-config/instance_ids/delete", h.UEConfig.DeleteInitInstanceIDs)
	api.GET("/ue/config/scene-desired", h.UEConfig.GetSceneDesired)
	api.GET("/ue/config/model-types", h.UEConfig.GetModelTypes)
	api.POST("/ue/config/model-types/create", h.UEConfig.CreateModelTypes)
	api.PUT("/ue/config/model-types", h.UEConfig.UpdateModelTypes)
	api.DELETE("/ue/config/model-types/delete", h.UEConfig.DeleteModelTypes)
	api.GET("/ue/config/model-config", h.UEConfig.GetModelConfig)
	api.PUT("/ue/config/model-config/placements", h.UEConfig.UpdateModelPlacements)
	api.PUT("/ue/config/model-config/placements/:instance_id", h.UEConfig.UpdateModelPlacement)
	api.DELETE("/ue/config/model-config/placements/delete", h.UEConfig.DeleteModelPlacements)
	api.GET("/ue/config/model-table", h.UEConfig.GetModelTable)
	api.POST("/ue/config/model-table/entries/create", h.UEConfig.CreateModelTableEntries)
	api.PUT("/ue/config/model-table/entries", h.UEConfig.UpdateModelTableEntries)
	api.DELETE("/ue/config/model-table/entries/delete", h.UEConfig.DeleteModelTableEntries)
	api.GET("/ue/config/path-config", h.UEConfig.GetPathConfig)
	api.PUT("/ue/config/path-config", h.UEConfig.ReplacePathConfig)
	api.GET("/ue/config/render-config", h.UEConfig.GetRenderConfig)
	api.PUT("/ue/config/render-config", h.UEConfig.ReplaceRenderConfig)
	api.GET("/ue/config/ndisplay/:name", h.UEConfig.GetNDisplayConfig)

	if h.UEAuxConfig != nil {
		api.GET("/ue/config/model-anchors", h.UEAuxConfig.Get(domain.ModelAnchors))
		api.PUT("/ue/config/model-anchors/:anchor_id", h.UEAuxConfig.Upsert(domain.ModelAnchors, "anchor_id"))
		api.PUT("/ue/config/model-anchors", h.UEAuxConfig.Replace(domain.ModelAnchors))
		api.DELETE("/ue/config/model-anchors/delete", h.UEAuxConfig.Delete(domain.ModelAnchors))

		api.GET("/ue/config/sensor-devices", h.UEAuxConfig.Get(domain.SensorDevices))
		api.PUT("/ue/config/sensor-devices/:device_id", h.UEAuxConfig.Upsert(domain.SensorDevices, "device_id"))
		api.PUT("/ue/config/sensor-devices", h.UEAuxConfig.Replace(domain.SensorDevices))
		api.DELETE("/ue/config/sensor-devices/delete", h.UEAuxConfig.Delete(domain.SensorDevices))

		api.GET("/ue/config/cluster-agents-catalog", h.UEAuxConfig.Get(domain.ClusterAgentsCatalog))
		api.PUT("/ue/config/cluster-agents-catalog/agents/:node_id", h.UEAuxConfig.Upsert(domain.ClusterAgentsCatalog, "node_id"))
		api.PUT("/ue/config/cluster-agents-catalog", h.UEAuxConfig.Replace(domain.ClusterAgentsCatalog))
		api.DELETE("/ue/config/cluster-agents-catalog/agents/delete", h.UEAuxConfig.Delete(domain.ClusterAgentsCatalog))

		api.GET("/ue/config/cluster-agent-registry", h.UEAuxConfig.Get(domain.ClusterAgentRegistry))
		api.PUT("/ue/config/cluster-agent-registry/agents/:node_id", h.UEAuxConfig.Upsert(domain.ClusterAgentRegistry, "node_id"))
		api.PUT("/ue/config/cluster-agent-registry", h.UEAuxConfig.Replace(domain.ClusterAgentRegistry))
		api.DELETE("/ue/config/cluster-agent-registry/agents/delete", h.UEAuxConfig.Delete(domain.ClusterAgentRegistry))
	}
}
