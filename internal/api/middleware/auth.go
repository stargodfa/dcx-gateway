package middleware

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"

	"dcx-gateway/internal/config"
	"dcx-gateway/internal/service"

	"github.com/gin-gonic/gin"
)

const (
	ContextUserID      = "auth_user_id"
	ContextUsername    = "auth_username"
	ContextRole        = "auth_role"
	ContextAuthType    = "auth_type"
	ContextBearerToken = "auth_bearer_token"
)

func Auth(cfg config.SecurityConfig, auth service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if shouldSkipAuth(c, cfg) {
			c.Next()
			return
		}

		if tryBearer(c, auth) {
			if !c.IsAborted() {
				c.Next()
			}
			return
		}
		if tryStaticToken(c, cfg) {
			c.Next()
			return
		}

		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		c.Abort()
	}
}

func shouldSkipAuth(c *gin.Context, cfg config.SecurityConfig) bool {
	if !cfg.Enabled {
		return true
	}
	method := c.Request.Method
	path := c.FullPath()
	if path == "" {
		path = c.Request.URL.Path
	}
	if method == http.MethodOptions {
		return true
	}
	if method == http.MethodGet && path == "/api/health" {
		return true
	}
	if method == http.MethodPost && path == "/api/auth/login" {
		return true
	}
	if path == "/api/auth/me" || path == "/api/auth/logout" {
		return false
	}
	if method == http.MethodGet && !cfg.ProtectRead {
		return true
	}
	return false
}

func tryBearer(c *gin.Context, auth service.AuthService) bool {
	token := bearerTokenFromHeader(c.GetHeader("Authorization"))
	if token == "" || auth == nil {
		return false
	}
	principal, err := auth.ValidateBearerToken(c.Request.Context(), token)
	if err != nil {
		if errors.Is(err, service.ErrAuthUnavailable) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "auth_unavailable"})
			c.Abort()
			return true
		}
		if errors.Is(err, service.ErrAuthForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			c.Abort()
			return true
		}
		return false
	}
	c.Set(ContextUserID, principal.UserID)
	c.Set(ContextUsername, principal.Username)
	c.Set(ContextRole, principal.Role)
	c.Set(ContextAuthType, "bearer")
	c.Set(ContextBearerToken, token)
	return true
}

func tryStaticToken(c *gin.Context, cfg config.SecurityConfig) bool {
	if !cfg.StaticTokenEnabled {
		return false
	}
	got := c.GetHeader("X-Gateway-Token")
	want := strings.TrimSpace(cfg.StaticToken)
	if got == "" || want == "" {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
		return false
	}
	c.Set(ContextUsername, "static-token")
	c.Set(ContextRole, "operator")
	c.Set(ContextAuthType, "static")
	return true
}

func bearerTokenFromHeader(header string) string {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func AuthUsername(c *gin.Context) string {
	v, _ := c.Get(ContextUsername)
	s, _ := v.(string)
	return s
}

func AuthRole(c *gin.Context) string {
	v, _ := c.Get(ContextRole)
	s, _ := v.(string)
	return s
}

func AuthType(c *gin.Context) string {
	v, _ := c.Get(ContextAuthType)
	s, _ := v.(string)
	return s
}

func BearerToken(c *gin.Context) string {
	v, _ := c.Get(ContextBearerToken)
	s, _ := v.(string)
	return s
}
