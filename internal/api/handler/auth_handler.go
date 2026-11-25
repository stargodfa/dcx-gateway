package handler

import (
	"errors"
	"net/http"

	apimiddleware "dcx-gateway/internal/api/middleware"
	"dcx-gateway/internal/model/dto"
	"dcx-gateway/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AuthHandler struct {
	auth service.AuthService
	log  *zap.Logger
}

func NewAuthHandler(auth service.AuthService, log *zap.Logger) *AuthHandler {
	return &AuthHandler{auth: auth, log: log}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
		return
	}
	resp, err := h.auth.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		h.writeAuthError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	if apimiddleware.AuthType(c) == "bearer" {
		if err := h.auth.Logout(c.Request.Context(), apimiddleware.BearerToken(c)); err != nil {
			h.log.Warn("logout bearer token failed",
				zap.String("request_id", apimiddleware.RequestIDFromContext(c)),
				zap.Error(err),
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "logout_failed"})
			return
		}
	}
	c.JSON(http.StatusOK, dto.LogoutResponse{OK: true})
}

func (h *AuthHandler) Me(c *gin.Context) {
	c.JSON(http.StatusOK, dto.MeResponse{
		Username: apimiddleware.AuthUsername(c),
		Role:     apimiddleware.AuthRole(c),
	})
}

func (h *AuthHandler) writeAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrAuthUnauthorized):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	case errors.Is(err, service.ErrAuthForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	default:
		h.log.Warn("auth request failed",
			zap.String("request_id", apimiddleware.RequestIDFromContext(c)),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "auth_failed"})
	}
}
