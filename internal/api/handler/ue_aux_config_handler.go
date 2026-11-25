package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"dcx-gateway/internal/model/domain"
	"dcx-gateway/internal/repository"
	"dcx-gateway/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type UEAuxConfigHandler struct {
	service      service.UEAuxConfigService
	log          *zap.Logger
	maxBodyBytes int64
}

func NewUEAuxConfigHandler(svc service.UEAuxConfigService, log *zap.Logger, maxBodyBytes int64) *UEAuxConfigHandler {
	if maxBodyBytes <= 0 {
		maxBodyBytes = 8 * 1024 * 1024
	}
	return &UEAuxConfigHandler{service: svc, log: log, maxBodyBytes: maxBodyBytes}
}

func (h *UEAuxConfigHandler) Get(resource domain.AuxConfigResource) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := h.service.Get(c.Request.Context(), resource)
		h.respond(c, resource, result, err)
	}
}

func (h *UEAuxConfigHandler) Upsert(resource domain.AuxConfigResource, param string) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, ok := h.decodeObject(c)
		if !ok {
			return
		}
		result, err := h.service.Upsert(c.Request.Context(), resource, c.Param(param), body)
		h.respond(c, resource, result, err)
	}
}

func (h *UEAuxConfigHandler) Replace(resource domain.AuxConfigResource) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, ok := h.decodeObject(c)
		if !ok {
			return
		}
		result, err := h.service.Replace(c.Request.Context(), resource, body)
		h.respond(c, resource, result, err)
	}
}

func (h *UEAuxConfigHandler) Delete(resource domain.AuxConfigResource) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, ok := h.decodeObject(c)
		if !ok {
			return
		}
		err := h.service.Delete(c.Request.Context(), resource, body)
		h.respond(c, resource, gin.H{"ok": true}, err)
	}
}

func (h *UEAuxConfigHandler) decodeObject(c *gin.Context) (map[string]any, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxBodyBytes)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.UseNumber()
	var body map[string]any
	err := decoder.Decode(&body)
	if err == nil {
		var extra any
		if trailingErr := decoder.Decode(&extra); !errors.Is(trailingErr, io.EOF) {
			err = trailingErr
			if err == nil {
				err = errors.New("multiple JSON values")
			}
		}
	}
	if err != nil || body == nil {
		var sizeErr *http.MaxBytesError
		if errors.As(err, &sizeErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "body must be a single JSON object"})
		}
		return nil, false
	}
	return body, true
}

func (h *UEAuxConfigHandler) respond(c *gin.Context, resource domain.AuxConfigResource, result any, err error) {
	switch {
	case err == nil:
		c.JSON(http.StatusOK, result)
	case errors.Is(err, service.ErrInvalidAuxConfig):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, repository.ErrAuxConfigConflict):
		c.JSON(http.StatusConflict, gin.H{"error": "anchor revision conflict"})
	default:
		if h.log != nil {
			h.log.Error("auxiliary UE config request failed", zap.String("resource", string(resource)), zap.Error(err))
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "auxiliary config operation failed"})
	}
}
