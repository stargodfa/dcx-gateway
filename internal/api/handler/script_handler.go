package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	apimiddleware "dcx-gateway/internal/api/middleware"
	"dcx-gateway/internal/model/dto"
	"dcx-gateway/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ScriptHandler struct {
	scriptService service.ScriptService
	log           *zap.Logger
}

func NewScriptHandler(scriptService service.ScriptService, log *zap.Logger) *ScriptHandler {
	return &ScriptHandler{
		scriptService: scriptService,
		log:           log,
	}
}

func (h *ScriptHandler) GetDataBundle(c *gin.Context) {
	var uri dto.GetDataBundleURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("BAD_REQUEST", err.Error()))
		return
	}

	var query dto.GetDataBundleQuery
	_ = c.ShouldBindQuery(&query)

	bundle, err := h.scriptService.BuildDataBundle(c.Request.Context(), uri.ScriptID, query.UseCache)
	if err != nil {
		h.log.Error("build data bundle failed",
			zap.String("script_id", uri.ScriptID),
			zap.Error(err),
		)

		c.JSON(http.StatusInternalServerError, dto.Fail("BUILD_DATA_BUNDLE_FAILED", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.OK(bundle))
}

func (h *ScriptHandler) Preload(c *gin.Context) {
	var uri dto.GetDataBundleURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("BAD_REQUEST", err.Error()))
		return
	}
	req, err := decodePreloadRequest(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("BAD_REQUEST", "body must be a JSON object with an optional boolean force"))
		return
	}

	bundle, err := h.scriptService.PreloadWithOptions(c.Request.Context(), uri.ScriptID, service.PreloadOptions{
		Force:    req.Force,
		PushWS:   true,
		Source:   "http_preload",
		AuthType: apimiddleware.AuthType(c),
		Username: apimiddleware.AuthUsername(c),
	})
	if err != nil {
		h.log.Error("preload script failed",
			zap.String("script_id", uri.ScriptID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, dto.Fail("PRELOAD_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.OK(dto.PreloadResponse{
		TaskID:   bundle.BundleID,
		ScriptID: bundle.ScriptID,
		Status:   "done",
	}))
}

func decodePreloadRequest(body io.Reader) (dto.PreloadRequest, error) {
	var req dto.PreloadRequest
	decoder := json.NewDecoder(body)
	var object map[string]json.RawMessage
	if err := decoder.Decode(&object); err != nil {
		if errors.Is(err, io.EOF) {
			return req, nil // The request body is optional; force defaults to false.
		}
		return req, err
	}
	if object == nil {
		return req, errors.New("preload body must be an object")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return req, errors.New("preload body must contain a single object")
	}
	if raw, exists := object["force"]; exists {
		var force *bool
		if err := json.Unmarshal(raw, &force); err != nil || force == nil {
			return req, errors.New("force must be a boolean")
		}
		req.Force = *force
	}
	return req, nil
}
