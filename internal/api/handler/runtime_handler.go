package handler

import (
	"errors"
	"net/http"

	apimiddleware "dcx-gateway/internal/api/middleware"
	"dcx-gateway/internal/model/dto"
	"dcx-gateway/internal/repository"
	"dcx-gateway/internal/service"
	"dcx-gateway/internal/websocket"

	"github.com/gin-gonic/gin"
)

type RuntimeHandler struct {
	fastAPIClient  websocket.FastAPIClient
	runtimeService service.RuntimeService
}

func NewRuntimeHandler(fastAPIClient websocket.FastAPIClient, runtimeService service.RuntimeService) *RuntimeHandler {
	return &RuntimeHandler{fastAPIClient: fastAPIClient, runtimeService: runtimeService}
}

func (h *RuntimeHandler) GetWSDataStatus(c *gin.Context) {
	if h == nil || h.fastAPIClient == nil {
		c.JSON(http.StatusOK, websocket.FastAPIClientStats{
			Enabled:            false,
			Connected:          false,
			MessageCountByType: map[string]int64{},
		})
		return
	}
	c.JSON(http.StatusOK, h.fastAPIClient.Stats())
}

func (h *RuntimeHandler) GetWarmupOptions(c *gin.Context) {
	if h == nil || h.runtimeService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "runtime service unavailable"})
		return
	}
	options, err := h.runtimeService.BuildWarmupOptions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, options)
}

func (h *RuntimeHandler) PreloadReportScene(c *gin.Context) {
	if h == nil || h.runtimeService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "runtime service unavailable"})
		return
	}
	var req dto.ReportScenePreloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	force := true
	if req.Force != nil {
		force = *req.Force
	}
	pushWS := true
	if req.PushWS != nil {
		pushWS = *req.PushWS
	}
	resp, err := h.runtimeService.PreloadReportScene(c.Request.Context(), service.ReportScenePreloadRequest{
		ScriptID:  req.ScriptID,
		ChapterID: req.ChapterID,
		SceneID:   req.SceneID,
		Force:     force,
		PushWS:    pushWS,
		Source:    "report_scene_preload",
		AuthType:  apimiddleware.AuthType(c),
		Username:  apimiddleware.AuthUsername(c),
	})
	if err != nil {
		h.writeRuntimeError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *RuntimeHandler) writeRuntimeError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrInvalidRuntimeRequest) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, repository.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "runtime resource not found"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}
