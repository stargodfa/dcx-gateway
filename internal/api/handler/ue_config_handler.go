package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"dcx-gateway/internal/model/dto"
	"dcx-gateway/internal/model/protocol/ue"
	"dcx-gateway/internal/repository"
	"dcx-gateway/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type UEConfigHandler struct {
	ueConfigService service.UEConfigService
	log             *zap.Logger
	maxBodyBytes    int64
}

type UEConfigHandlerOption func(*UEConfigHandler)

func WithMaxRawConfigBodyBytes(maxBytes int64) UEConfigHandlerOption {
	return func(h *UEConfigHandler) {
		h.maxBodyBytes = maxBytes
	}
}

func NewUEConfigHandler(ueConfigService service.UEConfigService, log *zap.Logger, opts ...UEConfigHandlerOption) *UEConfigHandler {
	h := &UEConfigHandler{
		ueConfigService: ueConfigService,
		log:             log,
		maxBodyBytes:    8 * 1024 * 1024,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(h)
		}
	}
	return h
}

func (h *UEConfigHandler) GetChartBinding(c *gin.Context) {
	h.getModeConfig(c, "chart_binding", "generated", func() (any, error) {
		return h.ueConfigService.BuildChartBinding(c.Request.Context())
	})
}

func (h *UEConfigHandler) GetVideoStreams(c *gin.Context) {
	h.getModeConfig(c, "video_streams", "generated", func() (any, error) {
		return h.ueConfigService.BuildVideoStreams(c.Request.Context())
	})
}

func (h *UEConfigHandler) GetChartConfig(c *gin.Context) {
	mode := strings.ToLower(strings.TrimSpace(c.Query("mode")))
	switch mode {
	case "", "full", "raw":
		raw, err := h.ueConfigService.GetRawConfig(c.Request.Context(), "chart_config")
		if err == nil {
			c.Data(http.StatusOK, "application/json; charset=utf-8", raw)
			return
		}
		if mode != "" || !errors.Is(err, repository.ErrNotFound) {
			h.writeConfigError(c, "chart_config", err)
			return
		}
		h.log.Warn("raw chart_config missing, fallback to structured config", zap.Error(err))
		config, err := h.ueConfigService.BuildStructuredChartConfig(c.Request.Context())
		if err != nil {
			h.log.Error("build structured chart_config failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, config)
	case "light", "structured":
		config, err := h.ueConfigService.BuildStructuredChartConfig(c.Request.Context())
		if err != nil {
			h.log.Error("build structured chart_config failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, config)
	case "runtime":
		config, err := h.ueConfigService.BuildRuntimeChartConfig(c.Request.Context())
		if err != nil {
			h.log.Error("build runtime chart_config failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, config)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported mode"})
	}
}

func (h *UEConfigHandler) GetReportScripts(c *gin.Context) {
	h.getModeConfig(c, "report_scripts", "light", func() (any, error) {
		return h.ueConfigService.BuildReportScripts(c.Request.Context())
	})
}

func (h *UEConfigHandler) GetRawConfig(c *gin.Context) {
	key := normalizeConfigKey(c.Param("key"))
	h.writeRawConfig(c, key)
}

func (h *UEConfigHandler) GetConfigIndex(c *gin.Context) {
	index, err := h.ueConfigService.ListRawConfigs(c.Request.Context())
	if err != nil {
		h.log.Error("build ue config index failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, index)
}

func (h *UEConfigHandler) GetBootstrap(c *gin.Context) {
	config, err := h.ueConfigService.BuildBootstrap(c.Request.Context())
	if err != nil {
		h.log.Error("build ue config bootstrap failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, config)
}

func (h *UEConfigHandler) GetDroneConfig(c *gin.Context) {
	h.writeRawConfig(c, "drone_config")
}

func (h *UEConfigHandler) GetInitConfig(c *gin.Context) {
	h.writeRawConfig(c, "init_config")
}

func (h *UEConfigHandler) UpdateInitConfig(c *gin.Context) {
	var req dto.UpdateInitConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.writeInitConfigBadRequest(c, err)
		return
	}
	if req.HasInstanceField() {
		h.writeInitConfigPatchError(c, fmt.Errorf("%w: use POST /api/ue/config/init-config/instance_ids/create or DELETE /api/ue/config/init-config/instance_ids/delete for instance_ids", service.ErrInvalidInitConfigPatch))
		return
	}

	patch, err := initConfigUpdateFromRequest(req)
	if err != nil {
		h.writeInitConfigPatchError(c, err)
		return
	}
	raw, err := h.ueConfigService.UpdateInitConfig(c.Request.Context(), patch)
	h.writeInitConfigPatchResult(c, raw, err)
}

func (h *UEConfigHandler) CreateInitInstanceIDs(c *gin.Context) {
	var req dto.UpdateInstanceIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.writeInitConfigBadRequest(c, err)
		return
	}
	raw, err := h.ueConfigService.AddInitInstanceIDs(c.Request.Context(), instanceIDsFromRequest(req))
	h.writeInitConfigPatchResult(c, raw, err)
}

func (h *UEConfigHandler) DeleteInitInstanceIDs(c *gin.Context) {
	var req dto.UpdateInstanceIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.writeInitConfigBadRequest(c, err)
		return
	}
	raw, err := h.ueConfigService.DeleteInitInstanceIDs(c.Request.Context(), instanceIDsFromRequest(req))
	h.writeInitConfigPatchResult(c, raw, err)
}

func (h *UEConfigHandler) GetSceneDesired(c *gin.Context) {
	mode := strings.ToLower(strings.TrimSpace(c.Query("mode")))
	switch mode {
	case "", "generated":
		config, err := h.ueConfigService.BuildSceneDesiredConfig(c.Request.Context())
		if err != nil {
			h.writeModelConfigError(c, err)
			return
		}
		c.JSON(http.StatusOK, config)
	case "legacy":
		h.writeRawConfig(c, "scene_desired_config")
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported mode"})
	}
}

func (h *UEConfigHandler) PatchSceneDesired(c *gin.Context) {
	var req dto.SceneDesiredModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.writeSceneDesiredBadRequest(c, err)
		return
	}
	patches, err := sceneDesiredPatchesFromRequest(req.Models)
	if err != nil {
		h.writeSceneDesiredError(c, err)
		return
	}
	config, err := h.ueConfigService.PatchSceneDesiredModels(c.Request.Context(), patches)
	if err != nil {
		h.writeSceneDesiredError(c, err)
		return
	}
	c.JSON(http.StatusOK, config)
}

func (h *UEConfigHandler) CreateSceneDesiredModels(c *gin.Context) {
	var req dto.SceneDesiredModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.writeSceneDesiredBadRequest(c, err)
		return
	}
	patches, err := sceneDesiredPatchesFromRequest(req.Models)
	if err != nil {
		h.writeSceneDesiredError(c, err)
		return
	}
	config, err := h.ueConfigService.UpsertSceneDesiredModels(c.Request.Context(), patches)
	if err != nil {
		h.writeSceneDesiredError(c, err)
		return
	}
	c.JSON(http.StatusOK, config)
}

func (h *UEConfigHandler) DeleteSceneDesiredModels(c *gin.Context) {
	var req dto.UpdateInstanceIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.writeSceneDesiredBadRequest(c, err)
		return
	}
	config, err := h.ueConfigService.DeleteSceneDesiredModels(c.Request.Context(), instanceIDsFromRequest(req))
	if err != nil {
		h.writeSceneDesiredError(c, err)
		return
	}
	c.JSON(http.StatusOK, config)
}

func (h *UEConfigHandler) GetModelConfig(c *gin.Context) {
	mode := strings.ToLower(strings.TrimSpace(c.Query("mode")))
	switch mode {
	case "", "full", "raw", "legacy":
		h.writeRawConfig(c, "model_config")
	case "structured":
		config, err := h.ueConfigService.BuildStructuredModelConfig(c.Request.Context())
		if err != nil {
			h.writeModelConfigError(c, err)
			return
		}
		c.JSON(http.StatusOK, config)
	case "runtime":
		config, err := h.ueConfigService.BuildRuntimeModelConfig(c.Request.Context())
		if err != nil {
			h.writeModelConfigError(c, err)
			return
		}
		c.JSON(http.StatusOK, config)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported mode"})
	}
}

func (h *UEConfigHandler) GetModelTable(c *gin.Context) {
	mode := strings.ToLower(strings.TrimSpace(c.Query("mode")))
	switch mode {
	case "", "generated":
		config, err := h.ueConfigService.BuildModelTable(c.Request.Context())
		if err != nil {
			h.writeModelConfigError(c, err)
			return
		}
		c.JSON(http.StatusOK, config)
	case "legacy", "raw", "full":
		h.writeRawConfig(c, "model_table")
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported mode"})
	}
}

func (h *UEConfigHandler) GetModelTypes(c *gin.Context) {
	config, err := h.ueConfigService.BuildModelTypes(c.Request.Context())
	if err != nil {
		h.writeModelConfigError(c, err)
		return
	}
	c.JSON(http.StatusOK, config)
}

func (h *UEConfigHandler) CreateModelTypes(c *gin.Context) {
	var req dto.ModelTypesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.writeModelConfigBadRequest(c, err)
		return
	}
	patches, err := modelTypePatchesFromRequest(req.ModelTypes)
	if err != nil {
		h.writeModelConfigError(c, err)
		return
	}
	config, err := h.ueConfigService.CreateModelTypes(c.Request.Context(), patches)
	if err != nil {
		h.writeModelConfigError(c, err)
		return
	}
	c.JSON(http.StatusOK, config)
}

func (h *UEConfigHandler) UpdateModelTypes(c *gin.Context) {
	var req dto.ModelTypesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.writeModelConfigBadRequest(c, err)
		return
	}
	patches, err := modelTypePatchesFromRequest(req.ModelTypes)
	if err != nil {
		h.writeModelConfigError(c, err)
		return
	}
	config, err := h.ueConfigService.UpdateModelTypes(c.Request.Context(), patches)
	if err != nil {
		h.writeModelConfigError(c, err)
		return
	}
	c.JSON(http.StatusOK, config)
}

func (h *UEConfigHandler) DeleteModelTypes(c *gin.Context) {
	var req dto.ModelTypeIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.writeModelConfigBadRequest(c, err)
		return
	}
	config, err := h.ueConfigService.DeleteModelTypes(c.Request.Context(), modelIDsFromRequest(req))
	if err != nil {
		h.writeModelConfigError(c, err)
		return
	}
	c.JSON(http.StatusOK, config)
}

func (h *UEConfigHandler) CreateModelTableEntries(c *gin.Context) {
	var req dto.ModelTableEntriesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.writeModelConfigBadRequest(c, err)
		return
	}
	patches := modelInstancePatchesFromRequest(req.Entries)
	config, err := h.ueConfigService.CreateModelTableEntries(c.Request.Context(), patches)
	if err != nil {
		h.writeModelConfigError(c, err)
		return
	}
	c.JSON(http.StatusOK, config)
}

func (h *UEConfigHandler) UpdateModelTableEntries(c *gin.Context) {
	var req dto.ModelTableEntriesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.writeModelConfigBadRequest(c, err)
		return
	}
	patches := modelInstancePatchesFromRequest(req.Entries)
	config, err := h.ueConfigService.UpdateModelTableEntries(c.Request.Context(), patches)
	if err != nil {
		h.writeModelConfigError(c, err)
		return
	}
	c.JSON(http.StatusOK, config)
}

func (h *UEConfigHandler) DeleteModelTableEntries(c *gin.Context) {
	var req dto.UpdateInstanceIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.writeModelConfigBadRequest(c, err)
		return
	}
	config, err := h.ueConfigService.DeleteModelTableEntries(c.Request.Context(), instanceIDsFromRequest(req))
	if err != nil {
		h.writeModelConfigError(c, err)
		return
	}
	c.JSON(http.StatusOK, config)
}

func (h *UEConfigHandler) DeleteModelPlacements(c *gin.Context) {
	var req dto.UpdateInstanceIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.writeModelConfigBadRequest(c, err)
		return
	}
	config, err := h.ueConfigService.DeleteModelPlacements(c.Request.Context(), instanceIDsFromRequest(req))
	if err != nil {
		h.writeModelConfigError(c, err)
		return
	}
	c.JSON(http.StatusOK, config)
}

func (h *UEConfigHandler) UpdateModelPlacements(c *gin.Context) {
	var req dto.ModelPlacementsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.writeModelConfigBadRequest(c, err)
		return
	}
	patches, err := modelPlacementPatchesFromRequest(req.Placements, "")
	if err != nil {
		h.writeModelConfigError(c, err)
		return
	}
	config, err := h.ueConfigService.UpdateModelPlacements(c.Request.Context(), patches)
	if err != nil {
		h.writeModelConfigError(c, err)
		return
	}
	c.JSON(http.StatusOK, config)
}

func (h *UEConfigHandler) UpdateModelPlacement(c *gin.Context) {
	var req dto.ModelPlacementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.writeModelConfigBadRequest(c, err)
		return
	}
	patches, err := modelPlacementPatchesFromRequest([]dto.ModelPlacementRequest{req}, c.Param("instance_id"))
	if err != nil {
		h.writeModelConfigError(c, err)
		return
	}
	config, err := h.ueConfigService.UpdateModelPlacements(c.Request.Context(), patches)
	if err != nil {
		h.writeModelConfigError(c, err)
		return
	}
	c.JSON(http.StatusOK, config)
}

func (h *UEConfigHandler) GetPathConfig(c *gin.Context) {
	h.writeRawConfig(c, "path_config")
}

func (h *UEConfigHandler) ReplacePathConfig(c *gin.Context) {
	h.replaceRawConfig(c, "path-config")
}

func (h *UEConfigHandler) ReplaceChartConfig(c *gin.Context) {
	h.replaceRawConfig(c, "chart-config")
}

func (h *UEConfigHandler) ReplaceChartBinding(c *gin.Context) {
	h.replaceRawConfig(c, "chart-binding")
}

func (h *UEConfigHandler) ReplaceDroneConfig(c *gin.Context) {
	h.replaceRawConfig(c, "drone-config")
}

func (h *UEConfigHandler) GetRenderConfig(c *gin.Context) {
	h.writeRawConfig(c, "render_config")
}

func (h *UEConfigHandler) ReplaceRenderConfig(c *gin.Context) {
	h.replaceRawConfig(c, "render-config")
}

func (h *UEConfigHandler) ReplaceVideoStreams(c *gin.Context) {
	h.replaceRawConfig(c, "video-streams")
}

func (h *UEConfigHandler) ReplaceReportScripts(c *gin.Context) {
	h.replaceRawConfig(c, "report-scripts")
}

func (h *UEConfigHandler) CreateReportScript(c *gin.Context) {
	var req dto.ReportScriptRequest
	if !h.bindReportScriptsJSON(c, &req) {
		return
	}
	raw, err := h.ueConfigService.CreateReportScript(c.Request.Context(), reportScriptPatchFromRequest(req))
	h.writeReportScriptsResult(c, raw, err)
}

func (h *UEConfigHandler) UpdateReportScript(c *gin.Context) {
	var req dto.ReportScriptRequest
	if !h.bindReportScriptsJSON(c, &req) {
		return
	}
	raw, err := h.ueConfigService.UpdateReportScript(c.Request.Context(), reportScriptPatchFromRequest(req))
	h.writeReportScriptsResult(c, raw, err)
}

func (h *UEConfigHandler) DeleteReportScript(c *gin.Context) {
	var req dto.ReportScriptRequest
	if !h.bindReportScriptsJSON(c, &req) {
		return
	}
	raw, err := h.ueConfigService.DeleteReportScript(c.Request.Context(), req.ScriptID)
	h.writeReportScriptsResult(c, raw, err)
}

func (h *UEConfigHandler) CreateReportChapter(c *gin.Context) {
	var req dto.ReportChapterRequest
	if !h.bindReportScriptsJSON(c, &req) {
		return
	}
	raw, err := h.ueConfigService.CreateReportChapter(c.Request.Context(), reportChapterPatchFromRequest(req))
	h.writeReportScriptsResult(c, raw, err)
}

func (h *UEConfigHandler) UpdateReportChapter(c *gin.Context) {
	var req dto.ReportChapterRequest
	if !h.bindReportScriptsJSON(c, &req) {
		return
	}
	raw, err := h.ueConfigService.UpdateReportChapter(c.Request.Context(), reportChapterPatchFromRequest(req))
	h.writeReportScriptsResult(c, raw, err)
}

func (h *UEConfigHandler) DeleteReportChapter(c *gin.Context) {
	var req dto.ReportChapterRequest
	if !h.bindReportScriptsJSON(c, &req) {
		return
	}
	raw, err := h.ueConfigService.DeleteReportChapter(c.Request.Context(), reportChapterPatchFromRequest(req))
	h.writeReportScriptsResult(c, raw, err)
}

func (h *UEConfigHandler) CreateReportScene(c *gin.Context) {
	var req dto.ReportSceneRequest
	if !h.bindReportScriptsJSON(c, &req) {
		return
	}
	patch, err := reportScenePatchFromRequest(req)
	if err != nil {
		h.writeReportScriptsError(c, err)
		return
	}
	raw, err := h.ueConfigService.CreateReportScene(c.Request.Context(), patch)
	h.writeReportScriptsResult(c, raw, err)
}

func (h *UEConfigHandler) UpdateReportScene(c *gin.Context) {
	var req dto.ReportSceneRequest
	if !h.bindReportScriptsJSON(c, &req) {
		return
	}
	if req.HasField("camera") || req.HasField("bindings") || req.HasField("warmup") {
		h.writeReportScriptsError(c, fmt.Errorf("%w: PUT /api/ue/config/report-scripts/scenes only updates scene_id/label; use /scenes/camera, /scenes/bindings/*, or /scenes/warmup for nested fields", service.ErrInvalidReportScriptsPatch))
		return
	}
	patch, err := reportScenePatchFromRequest(req)
	if err != nil {
		h.writeReportScriptsError(c, err)
		return
	}
	raw, err := h.ueConfigService.UpdateReportScene(c.Request.Context(), patch)
	h.writeReportScriptsResult(c, raw, err)
}

func (h *UEConfigHandler) DeleteReportScene(c *gin.Context) {
	var req dto.ReportSceneRequest
	if !h.bindReportScriptsJSON(c, &req) {
		return
	}
	raw, err := h.ueConfigService.DeleteReportScene(c.Request.Context(), service.ReportScenePatch{
		ScriptID:  req.ScriptID,
		ChapterID: req.ChapterID,
		SceneID:   req.SceneID,
	})
	h.writeReportScriptsResult(c, raw, err)
}

func (h *UEConfigHandler) UpdateReportSceneCamera(c *gin.Context) {
	var req dto.ReportSceneCameraRequest
	if !h.bindReportScriptsJSON(c, &req) {
		return
	}
	camera, err := reportCameraFromRequest(req.Camera)
	if err != nil {
		h.writeReportScriptsError(c, err)
		return
	}
	raw, err := h.ueConfigService.UpdateReportSceneCamera(c.Request.Context(), service.ReportScenePatch{
		ScriptID:  req.ScriptID,
		ChapterID: req.ChapterID,
		SceneID:   req.SceneID,
		Camera:    camera,
		CameraSet: true,
	})
	h.writeReportScriptsResult(c, raw, err)
}

func (h *UEConfigHandler) CreateReportSceneBindings(c *gin.Context) {
	var req dto.ReportBindingsRequest
	if !h.bindReportScriptsJSON(c, &req) {
		return
	}
	bindings := reportBindingsFromRequest(req.Bindings)
	raw, err := h.ueConfigService.CreateReportSceneBindings(c.Request.Context(), service.ReportBindingPatch{
		ScriptID:  req.ScriptID,
		ChapterID: req.ChapterID,
		SceneID:   req.SceneID,
		Bindings:  bindings,
	})
	h.writeReportScriptsResult(c, raw, err)
}

func (h *UEConfigHandler) UpdateReportSceneBindings(c *gin.Context) {
	var req dto.ReportBindingsRequest
	if !h.bindReportScriptsJSON(c, &req) {
		return
	}
	bindings := reportBindingsFromRequest(req.Bindings)
	raw, err := h.ueConfigService.UpdateReportSceneBindings(c.Request.Context(), service.ReportBindingPatch{
		ScriptID:  req.ScriptID,
		ChapterID: req.ChapterID,
		SceneID:   req.SceneID,
		Bindings:  bindings,
	})
	h.writeReportScriptsResult(c, raw, err)
}

func (h *UEConfigHandler) DeleteReportSceneBindings(c *gin.Context) {
	var req dto.ReportBindingsRequest
	if !h.bindReportScriptsJSON(c, &req) {
		return
	}
	ids := append([]string{}, req.InstanceIDs...)
	if req.InstanceID != "" {
		ids = append(ids, req.InstanceID)
	}
	raw, err := h.ueConfigService.DeleteReportSceneBindings(c.Request.Context(), service.ReportBindingDeletePatch{
		ScriptID:    req.ScriptID,
		ChapterID:   req.ChapterID,
		SceneID:     req.SceneID,
		InstanceIDs: ids,
	})
	h.writeReportScriptsResult(c, raw, err)
}

func (h *UEConfigHandler) UpdateReportSceneWarmup(c *gin.Context) {
	var req dto.ReportSceneWarmupRequest
	if !h.bindReportScriptsJSON(c, &req) {
		return
	}
	if req.Warmup == nil {
		h.writeReportScriptsError(c, fmt.Errorf("%w: warmup is required", service.ErrInvalidReportScriptsPatch))
		return
	}
	raw, err := h.ueConfigService.UpdateReportSceneWarmup(c.Request.Context(), service.ReportWarmupPatch{
		ScriptID:  req.ScriptID,
		ChapterID: req.ChapterID,
		SceneID:   req.SceneID,
		Warmup:    reportWarmupFromRequest(req.Warmup),
	})
	h.writeReportScriptsResult(c, raw, err)
}

func (h *UEConfigHandler) GetNDisplayConfig(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))
	name = strings.TrimSuffix(name, ".json")
	key := "ndisplay/" + name
	h.writeRawConfig(c, key)
}

func (h *UEConfigHandler) getModeConfig(c *gin.Context, rawKey string, generatedMode string, buildGenerated func() (any, error)) {
	mode := strings.ToLower(strings.TrimSpace(c.Query("mode")))
	switch mode {
	case "", "full", "raw":
		raw, err := h.ueConfigService.GetRawConfig(c.Request.Context(), rawKey)
		if err == nil {
			c.Data(http.StatusOK, "application/json; charset=utf-8", raw)
			return
		}
		if mode != "" || !errors.Is(err, repository.ErrNotFound) {
			h.writeConfigError(c, rawKey, err)
			return
		}
		h.log.Warn("full ue config missing, fallback to generated config",
			zap.String("config_key", rawKey),
			zap.String("fallback_mode", generatedMode),
			zap.Error(err),
		)
	case "generated", "light":
		if mode != generatedMode && !(generatedMode == "light" && mode == "generated") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported mode"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported mode"})
		return
	}

	config, err := buildGenerated()
	if err != nil {
		h.log.Error("build generated ue config failed", zap.String("config_key", rawKey), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, config)
}

func (h *UEConfigHandler) writeRawConfig(c *gin.Context, key string) {
	raw, err := h.ueConfigService.GetRawConfig(c.Request.Context(), key)
	if err != nil {
		h.writeConfigError(c, key, err)
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", raw)
}

func (h *UEConfigHandler) replaceRawConfig(c *gin.Context, routeKey string) {
	if h.maxBodyBytes > 0 {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxBodyBytes)
	}
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	next, err := h.ueConfigService.ReplaceRawConfig(c.Request.Context(), routeKey, raw)
	if err != nil {
		h.writeModelConfigError(c, err)
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", next)
}

func (h *UEConfigHandler) writeConfigError(c *gin.Context, key string, err error) {
	if errors.Is(err, repository.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "ue config not found", "config_key": key})
		return
	}
	h.log.Error("get raw ue config failed", zap.String("config_key", key), zap.Error(err))
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "config_key": key})
}

func (h *UEConfigHandler) writeInitConfigPatchResult(c *gin.Context, raw []byte, err error) {
	if err != nil {
		h.writeInitConfigPatchError(c, err)
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", raw)
}

func (h *UEConfigHandler) writeInitConfigPatchError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrInvalidInitConfigPatch) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, repository.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "ue config not found", "config_key": "init_config"})
		return
	}
	h.log.Error("update init_config failed", zap.Error(err))
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "config_key": "init_config"})
}

func (h *UEConfigHandler) writeSceneDesiredError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrInvalidSceneDesiredPatch) || errors.Is(err, service.ErrInvalidInitConfigPatch) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var missing *service.MissingSceneDesiredModelsError
	if errors.As(err, &missing) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":                "scene desired model not found",
			"missing_instance_ids": missing.InstanceIDs,
		})
		return
	}
	if errors.Is(err, repository.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "scene desired model not found"})
		return
	}
	h.log.Error("update scene_desired_config failed", zap.Error(err))
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "config_key": "scene_desired"})
}

func (h *UEConfigHandler) writeInitConfigBadRequest(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}

func (h *UEConfigHandler) writeSceneDesiredBadRequest(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}

func (h *UEConfigHandler) writeModelConfigBadRequest(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}

func (h *UEConfigHandler) writeModelConfigError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrInvalidModelConfigPatch) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, service.ErrModelConfigConflict) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, repository.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "model config resource not found"})
		return
	}
	h.log.Error("model config operation failed", zap.Error(err))
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func (h *UEConfigHandler) writeReportScriptsResult(c *gin.Context, raw []byte, err error) {
	if err != nil {
		h.writeReportScriptsError(c, err)
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", raw)
}

func (h *UEConfigHandler) writeReportScriptsError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrInvalidReportScriptsPatch) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, service.ErrReportScriptsConflict) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, repository.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "report_scripts resource not found"})
		return
	}
	h.log.Error("report_scripts operation failed", zap.Error(err))
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func (h *UEConfigHandler) bindReportScriptsJSON(c *gin.Context, out any) bool {
	if h.maxBodyBytes > 0 {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxBodyBytes)
	}
	if err := c.ShouldBindJSON(out); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) || strings.Contains(err.Error(), "request body too large") {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large"})
			return false
		}
		h.writeModelConfigBadRequest(c, err)
		return false
	}
	return true
}

func initConfigUpdateFromRequest(req dto.UpdateInitConfigRequest) (service.InitConfigUpdate, error) {
	var patch service.InitConfigUpdate
	if req.Camera != nil {
		if req.Camera.CameraID != nil {
			patch.CameraID = req.Camera.CameraID
		}
		if req.Camera.Position != nil {
			position, err := vectorFromRequest(req.Camera.Position, "camera.position")
			if err != nil {
				return service.InitConfigUpdate{}, err
			}
			patch.Position = &position
		}
		if req.Camera.Rotation != nil {
			rotation, err := rotationFromRequest(req.Camera.Rotation, "camera.rotation")
			if err != nil {
				return service.InitConfigUpdate{}, err
			}
			patch.Rotation = &rotation
		}
	}
	if req.CameraID != nil {
		patch.CameraID = req.CameraID
	}
	if req.Position != nil {
		position, err := vectorFromRequest(req.Position, "position")
		if err != nil {
			return service.InitConfigUpdate{}, err
		}
		patch.Position = &position
	}
	if req.Rotation != nil {
		rotation, err := rotationFromRequest(req.Rotation, "rotation")
		if err != nil {
			return service.InitConfigUpdate{}, err
		}
		patch.Rotation = &rotation
	}
	if req.AutoLoadOnStartup != nil {
		patch.AutoLoad = req.AutoLoadOnStartup
	}
	return patch, nil
}

func sceneDesiredPatchesFromRequest(reqs []dto.SceneDesiredModelRequest) ([]service.SceneDesiredModelPatch, error) {
	patches := make([]service.SceneDesiredModelPatch, 0, len(reqs))
	for _, req := range reqs {
		patch := service.SceneDesiredModelPatch{
			InstanceID:  req.InstanceID,
			ModelID:     req.ModelID,
			URI:         req.URI,
			Format:      req.Format,
			SourcePath:  req.SourcePath,
			DisplayName: req.DisplayName,
			Visible:     req.Visible,
			ConfigJSON:  req.ConfigJSON,
		}
		if req.BoundsSize != nil {
			value, err := sceneVectorFromRequest(req.BoundsSize, "bounds_size")
			if err != nil {
				return nil, err
			}
			patch.BoundsSize = &value
		}
		if req.Transform != nil {
			if req.Transform.Position != nil {
				value, err := sceneVectorFromRequest(req.Transform.Position, "transform.position")
				if err != nil {
					return nil, err
				}
				patch.Position = &value
			}
			if req.Transform.Rotation != nil {
				value, err := sceneRotationFromRequest(req.Transform.Rotation, "transform.rotation")
				if err != nil {
					return nil, err
				}
				patch.Rotation = &value
			}
			if req.Transform.Scale != nil {
				value, err := sceneVectorFromRequest(req.Transform.Scale, "transform.scale")
				if err != nil {
					return nil, err
				}
				patch.Scale = &value
			}
		}
		if req.Position != nil {
			value, err := sceneVectorFromRequest(req.Position, "position")
			if err != nil {
				return nil, err
			}
			patch.Position = &value
		}
		if req.Rotation != nil {
			value, err := sceneRotationFromRequest(req.Rotation, "rotation")
			if err != nil {
				return nil, err
			}
			patch.Rotation = &value
		}
		if req.Scale != nil {
			value, err := sceneVectorFromRequest(req.Scale, "scale")
			if err != nil {
				return nil, err
			}
			patch.Scale = &value
		}
		patches = append(patches, patch)
	}
	return patches, nil
}

func modelTypePatchesFromRequest(reqs []dto.ModelTypeRequest) ([]service.ModelTypePatch, error) {
	patches := make([]service.ModelTypePatch, 0, len(reqs))
	for _, req := range reqs {
		patch := service.ModelTypePatch{
			ModelID:                  req.ModelID,
			URI:                      req.URI,
			Format:                   req.Format,
			SourcePath:               req.SourcePath,
			BoundsNegativeExtentAxis: req.BoundsNegativeExtentAxis,
			BoundsAxisSet:            req.HasField("bounds_negative_extent_axis"),
			Enabled:                  req.Enabled,
		}
		if req.BaseBoundsSize != nil {
			value, err := modelVectorFromRequest(req.BaseBoundsSize, "base_bounds_size")
			if err != nil {
				return nil, err
			}
			patch.BaseBoundsSize = &value
		}
		patches = append(patches, patch)
	}
	return patches, nil
}

func modelInstancePatchesFromRequest(reqs []dto.ModelTableEntryRequest) []service.ModelInstancePatch {
	patches := make([]service.ModelInstancePatch, 0, len(reqs))
	for _, req := range reqs {
		patches = append(patches, service.ModelInstancePatch{
			InstanceID:  req.InstanceID,
			ModelID:     req.ModelID,
			DisplayName: req.DisplayName,
			SortOrder:   req.SortOrder,
			Enabled:     req.Enabled,
		})
	}
	return patches
}

func modelPlacementPatchesFromRequest(reqs []dto.ModelPlacementRequest, pathInstanceID string) ([]service.ModelPlacementPatch, error) {
	forbidden := []string{"bounds_size", "uri", "format", "source_path", "model_id", "display_name"}
	patches := make([]service.ModelPlacementPatch, 0, len(reqs))
	for _, req := range reqs {
		for _, field := range forbidden {
			if req.HasField(field) {
				return nil, fmt.Errorf("%w: %s is not allowed in placement patch", service.ErrInvalidModelConfigPatch, field)
			}
		}
		if req.HasField("transform") && (req.HasField("position") || req.HasField("rotation") || req.HasField("scale")) {
			return nil, fmt.Errorf("%w: do not mix transform with top-level position/rotation/scale", service.ErrInvalidModelConfigPatch)
		}
		instanceID := req.InstanceID
		if pathInstanceID != "" {
			instanceID = pathInstanceID
		}
		patch := service.ModelPlacementPatch{
			InstanceID: instanceID,
			Visible:    req.Visible,
		}
		if req.Transform != nil {
			if req.Transform.Position != nil {
				value, err := modelVectorFromRequest(req.Transform.Position, "transform.position")
				if err != nil {
					return nil, err
				}
				patch.Position = &value
			}
			if req.Transform.Rotation != nil {
				value, err := modelRotationFromRequest(req.Transform.Rotation, "transform.rotation")
				if err != nil {
					return nil, err
				}
				patch.Rotation = &value
			}
			if req.Transform.Scale != nil {
				value, err := modelVectorFromRequest(req.Transform.Scale, "transform.scale")
				if err != nil {
					return nil, err
				}
				patch.Scale = &value
			}
		}
		if req.Position != nil {
			value, err := modelVectorFromRequest(req.Position, "position")
			if err != nil {
				return nil, err
			}
			patch.Position = &value
		}
		if req.Rotation != nil {
			value, err := modelRotationFromRequest(req.Rotation, "rotation")
			if err != nil {
				return nil, err
			}
			patch.Rotation = &value
		}
		if req.Scale != nil {
			value, err := modelVectorFromRequest(req.Scale, "scale")
			if err != nil {
				return nil, err
			}
			patch.Scale = &value
		}
		patches = append(patches, patch)
	}
	return patches, nil
}

func reportScriptPatchFromRequest(req dto.ReportScriptRequest) service.ReportScriptPatch {
	return service.ReportScriptPatch{
		ScriptID:    req.ScriptID,
		NewScriptID: req.NewScriptID,
		Label:       req.Label,
		LabelSet:    req.HasField("label"),
	}
}

func reportChapterPatchFromRequest(req dto.ReportChapterRequest) service.ReportChapterPatch {
	return service.ReportChapterPatch{
		ScriptID:     req.ScriptID,
		ChapterID:    req.ChapterID,
		NewChapterID: req.NewChapterID,
		Label:        req.Label,
		LabelSet:     req.HasField("label"),
	}
}

func reportScenePatchFromRequest(req dto.ReportSceneRequest) (service.ReportScenePatch, error) {
	patch := service.ReportScenePatch{
		ScriptID:    req.ScriptID,
		ChapterID:   req.ChapterID,
		SceneID:     req.SceneID,
		NewSceneID:  req.NewSceneID,
		Label:       req.Label,
		LabelSet:    req.HasField("label"),
		CameraSet:   req.HasField("camera"),
		BindingsSet: req.HasField("bindings"),
		WarmupSet:   req.HasField("warmup"),
	}
	if req.Camera != nil {
		camera, err := reportCameraFromRequest(req.Camera)
		if err != nil {
			return service.ReportScenePatch{}, err
		}
		patch.Camera = camera
	}
	if req.HasField("bindings") {
		patch.Bindings = reportBindingsFromRequest(req.Bindings)
	}
	if req.Warmup != nil {
		patch.Warmup = reportWarmupFromRequest(req.Warmup)
	}
	return patch, nil
}

func reportCameraFromRequest(req *dto.ReportCameraRequest) (*ue.Camera, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: camera is required", service.ErrInvalidReportScriptsPatch)
	}
	cameraID := strings.TrimSpace(req.CameraID)
	if cameraID == "" {
		return nil, fmt.Errorf("%w: camera.camera_id is required", service.ErrInvalidReportScriptsPatch)
	}
	position, err := reportVectorFromRequest(req.Position, "camera.position")
	if err != nil {
		return nil, err
	}
	rotation, err := reportRotationFromRequest(req.Rotation, "camera.rotation")
	if err != nil {
		return nil, err
	}
	return &ue.Camera{CameraID: cameraID, Position: position, Rotation: rotation}, nil
}

func reportBindingsFromRequest(reqs []dto.ReportBindingDTO) []ue.ReportModelBinding {
	bindings := make([]ue.ReportModelBinding, 0, len(reqs))
	for _, req := range reqs {
		visible := true
		if req.Visible != nil {
			visible = *req.Visible
		}
		bindings = append(bindings, ue.ReportModelBinding{
			InstanceID: req.InstanceID,
			ModelID:    req.ModelID,
			Visible:    visible,
			Transform:  req.Transform,
		})
	}
	return bindings
}

func reportWarmupFromRequest(req *dto.ReportWarmupRequest) ue.ReportWarmup {
	if req == nil {
		return ue.ReportWarmup{}
	}
	return ue.ReportWarmup{Data: req.Data, Videos: req.Videos, Charts: req.Charts}
}

func reportVectorFromRequest(req *dto.Vector3Request, field string) (ue.Vector3, error) {
	if req == nil || req.X == nil || req.Y == nil || req.Z == nil {
		return ue.Vector3{}, fmt.Errorf("%w: %s x/y/z are required", service.ErrInvalidReportScriptsPatch, field)
	}
	return ue.Vector3{X: *req.X, Y: *req.Y, Z: *req.Z}, nil
}

func reportRotationFromRequest(req *dto.RotationRequest, field string) (ue.Rotation, error) {
	if req == nil || req.Yaw == nil || req.Pitch == nil || req.Roll == nil {
		return ue.Rotation{}, fmt.Errorf("%w: %s yaw/pitch/roll are required", service.ErrInvalidReportScriptsPatch, field)
	}
	return ue.Rotation{Yaw: *req.Yaw, Pitch: *req.Pitch, Roll: *req.Roll}, nil
}

func vectorFromRequest(req *dto.Vector3Request, field string) (ue.Vector3, error) {
	if req == nil || req.X == nil || req.Y == nil || req.Z == nil {
		return ue.Vector3{}, fmt.Errorf("%w: %s x/y/z are required", service.ErrInvalidInitConfigPatch, field)
	}
	return ue.Vector3{X: *req.X, Y: *req.Y, Z: *req.Z}, nil
}

func rotationFromRequest(req *dto.RotationRequest, field string) (ue.Rotation, error) {
	if req == nil || req.Yaw == nil || req.Pitch == nil || req.Roll == nil {
		return ue.Rotation{}, fmt.Errorf("%w: %s yaw/pitch/roll are required", service.ErrInvalidInitConfigPatch, field)
	}
	return ue.Rotation{Yaw: *req.Yaw, Pitch: *req.Pitch, Roll: *req.Roll}, nil
}

func sceneVectorFromRequest(req *dto.Vector3Request, field string) (ue.Vector3, error) {
	if req == nil || req.X == nil || req.Y == nil || req.Z == nil {
		return ue.Vector3{}, fmt.Errorf("%w: %s x/y/z are required", service.ErrInvalidSceneDesiredPatch, field)
	}
	return ue.Vector3{X: *req.X, Y: *req.Y, Z: *req.Z}, nil
}

func sceneRotationFromRequest(req *dto.RotationRequest, field string) (ue.Rotation, error) {
	if req == nil || req.Yaw == nil || req.Pitch == nil || req.Roll == nil {
		return ue.Rotation{}, fmt.Errorf("%w: %s yaw/pitch/roll are required", service.ErrInvalidSceneDesiredPatch, field)
	}
	return ue.Rotation{Yaw: *req.Yaw, Pitch: *req.Pitch, Roll: *req.Roll}, nil
}

func modelVectorFromRequest(req *dto.Vector3Request, field string) (ue.Vector3, error) {
	if req == nil || req.X == nil || req.Y == nil || req.Z == nil {
		return ue.Vector3{}, fmt.Errorf("%w: %s x/y/z are required", service.ErrInvalidModelConfigPatch, field)
	}
	return ue.Vector3{X: *req.X, Y: *req.Y, Z: *req.Z}, nil
}

func modelRotationFromRequest(req *dto.RotationRequest, field string) (ue.Rotation, error) {
	if req == nil || req.Yaw == nil || req.Pitch == nil || req.Roll == nil {
		return ue.Rotation{}, fmt.Errorf("%w: %s yaw/pitch/roll are required", service.ErrInvalidModelConfigPatch, field)
	}
	return ue.Rotation{Yaw: *req.Yaw, Pitch: *req.Pitch, Roll: *req.Roll}, nil
}

func instanceIDsFromRequest(req dto.UpdateInstanceIDsRequest) []string {
	ids := make([]string, 0, len(req.InstanceIDs)+1)
	ids = append(ids, req.InstanceIDs...)
	if req.InstanceID != "" {
		ids = append(ids, req.InstanceID)
	}
	return ids
}

func modelIDsFromRequest(req dto.ModelTypeIDsRequest) []string {
	ids := make([]string, 0, len(req.ModelIDs)+1)
	ids = append(ids, req.ModelIDs...)
	if req.ModelID != "" {
		ids = append(ids, req.ModelID)
	}
	return ids
}

func normalizeConfigKey(key string) string {
	key = strings.TrimSpace(key)
	key = strings.Trim(key, "/")
	key = strings.ReplaceAll(key, "-", "_")
	return key
}
