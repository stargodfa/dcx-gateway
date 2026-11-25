package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"dcx-gateway/internal/config"
	"dcx-gateway/internal/model/dto"
	"dcx-gateway/internal/service"

	"github.com/gin-gonic/gin"
)

func TestAuthMiddlewareDisabledAllowsAll(t *testing.T) {
	router := testRouter(config.SecurityConfig{Enabled: false}, nil)
	resp := performMiddlewareRequest(router, http.MethodPut, "/api/ue/config/path-config", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.Code)
	}
}

func TestAuthMiddlewareWriteNeedsToken(t *testing.T) {
	router := testRouter(config.SecurityConfig{Enabled: true, StaticTokenEnabled: true, StaticToken: "dev"}, nil)
	resp := performMiddlewareRequest(router, http.MethodPut, "/api/ue/config/path-config", nil)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.Code)
	}
	resp = performMiddlewareRequest(router, http.MethodPost, "/api/ue/config/report-scripts/scripts/create", nil)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("report_scripts create status = %d, want 401", resp.Code)
	}
}

func TestAuthMiddlewareStaticTokenPassesAndMarksContext(t *testing.T) {
	router := testRouter(config.SecurityConfig{Enabled: true, StaticTokenEnabled: true, StaticToken: "dev"}, nil)
	resp := performMiddlewareRequest(router, http.MethodPut, "/api/ue/config/path-config", map[string]string{"X-Gateway-Token": "dev"})
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.Code)
	}
	if resp.Body.String() != "static-token|operator|static" {
		t.Fatalf("body = %q", resp.Body.String())
	}
}

func TestAuthMiddlewareBearerPassesAndMarksContext(t *testing.T) {
	auth := &fakeAuthService{principal: &service.AuthPrincipal{UserID: 1, Username: "operator", Role: "operator", AuthType: "bearer"}}
	router := testRouter(config.SecurityConfig{Enabled: true}, auth)
	resp := performMiddlewareRequest(router, http.MethodPost, "/api/scripts/security_main/preload", map[string]string{"Authorization": "Bearer token"})
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.Code)
	}
	if resp.Body.String() != "operator|operator|bearer" {
		t.Fatalf("body = %q", resp.Body.String())
	}
}

func TestAuthMiddlewareBearerErrors(t *testing.T) {
	router := testRouter(config.SecurityConfig{Enabled: true}, &fakeAuthService{validateErr: service.ErrAuthUnauthorized})
	resp := performMiddlewareRequest(router, http.MethodPut, "/api/ue/config/path-config", map[string]string{"Authorization": "Bearer bad"})
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.Code)
	}

	router = testRouter(config.SecurityConfig{Enabled: true}, &fakeAuthService{validateErr: service.ErrAuthUnavailable})
	resp = performMiddlewareRequest(router, http.MethodPut, "/api/ue/config/path-config", map[string]string{"Authorization": "Bearer token"})
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", resp.Code)
	}
}

func TestAuthMiddlewareReadAndAuthRouteRules(t *testing.T) {
	auth := &fakeAuthService{principal: &service.AuthPrincipal{Username: "operator", Role: "operator", AuthType: "bearer"}}

	router := testRouter(config.SecurityConfig{Enabled: true, ProtectRead: false}, auth)
	if resp := performMiddlewareRequest(router, http.MethodGet, "/api/ue/config/model-table", nil); resp.Code != http.StatusOK {
		t.Fatalf("GET with protect_read=false status = %d, want 200", resp.Code)
	}
	if resp := performMiddlewareRequest(router, http.MethodGet, "/api/auth/me", nil); resp.Code != http.StatusUnauthorized {
		t.Fatalf("GET /me without token status = %d, want 401", resp.Code)
	}
	if resp := performMiddlewareRequest(router, http.MethodPost, "/api/auth/logout", nil); resp.Code != http.StatusUnauthorized {
		t.Fatalf("POST /logout without token status = %d, want 401", resp.Code)
	}
	if resp := performMiddlewareRequest(router, http.MethodPost, "/api/auth/login", nil); resp.Code != http.StatusOK {
		t.Fatalf("POST /login status = %d, want 200", resp.Code)
	}
	if resp := performMiddlewareRequest(router, http.MethodOptions, "/api/ue/config/path-config", nil); resp.Code != http.StatusOK {
		t.Fatalf("OPTIONS status = %d, want 200", resp.Code)
	}

	router = testRouter(config.SecurityConfig{Enabled: true, ProtectRead: true}, auth)
	if resp := performMiddlewareRequest(router, http.MethodGet, "/api/ue/config/model-table", nil); resp.Code != http.StatusUnauthorized {
		t.Fatalf("GET with protect_read=true status = %d, want 401", resp.Code)
	}
	if resp := performMiddlewareRequest(router, http.MethodGet, "/api/health", nil); resp.Code != http.StatusOK {
		t.Fatalf("GET /health status = %d, want 200", resp.Code)
	}
}

func testRouter(cfg config.SecurityConfig, auth service.AuthService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api")
	api.Use(Auth(cfg, auth))
	api.GET("/health", echoAuth)
	api.POST("/auth/login", echoAuth)
	api.GET("/auth/me", echoAuth)
	api.POST("/auth/logout", echoAuth)
	api.GET("/ue/config/model-table", echoAuth)
	api.PUT("/ue/config/path-config", echoAuth)
	api.POST("/ue/config/report-scripts/scripts/create", echoAuth)
	api.POST("/scripts/:id/preload", echoAuth)
	api.OPTIONS("/ue/config/path-config", echoAuth)
	return router
}

func echoAuth(c *gin.Context) {
	c.String(http.StatusOK, AuthUsername(c)+"|"+AuthRole(c)+"|"+AuthType(c))
}

func performMiddlewareRequest(router *gin.Engine, method string, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

type fakeAuthService struct {
	principal   *service.AuthPrincipal
	validateErr error
	logoutErr   error
}

func (s *fakeAuthService) Login(ctx context.Context, username string, password string) (*dto.LoginResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *fakeAuthService) Logout(ctx context.Context, token string) error {
	return s.logoutErr
}

func (s *fakeAuthService) ValidateBearerToken(ctx context.Context, token string) (*service.AuthPrincipal, error) {
	if s.validateErr != nil {
		return nil, s.validateErr
	}
	return s.principal, nil
}
