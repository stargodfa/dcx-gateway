package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apimiddleware "dcx-gateway/internal/api/middleware"
	"dcx-gateway/internal/model/dto"
	"dcx-gateway/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestAuthHandlerLoginSuccess(t *testing.T) {
	auth := &fakeHandlerAuthService{loginResp: &dto.LoginResponse{
		AccessToken: "token",
		TokenType:   "Bearer",
		ExpiresIn:   7200,
		User:        dto.AuthUser{Username: "operator", Role: "operator"},
	}}
	router := authHandlerRouter(NewAuthHandler(auth, zap.NewNop()))

	resp := performAuthHandlerRequest(router, http.MethodPost, "/api/auth/login", `{"username":"operator","password":"test-password"}`, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), `"access_token":"token"`) {
		t.Fatalf("body does not contain access token: %s", resp.Body.String())
	}
	if strings.Contains(resp.Body.String(), "password") || strings.Contains(resp.Body.String(), "token_hash") {
		t.Fatalf("body leaked sensitive fields: %s", resp.Body.String())
	}
}

func TestAuthHandlerLoginPasswordError(t *testing.T) {
	auth := &fakeHandlerAuthService{loginErr: service.ErrAuthUnauthorized}
	router := authHandlerRouter(NewAuthHandler(auth, zap.NewNop()))

	resp := performAuthHandlerRequest(router, http.MethodPost, "/api/auth/login", `{"username":"operator","password":"wrong"}`, nil)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.Code)
	}
}

func TestAuthHandlerLoginDependencyErrorsStayInServerLogs(t *testing.T) {
	for _, detail := range []string{"database unavailable: diagnostic-only", "redis unavailable: diagnostic-only", ""} {
		t.Run(detail, func(t *testing.T) {
			core, logs := observer.New(zap.WarnLevel)
			auth := &fakeHandlerAuthService{loginErr: errors.New(detail)}
			router := authHandlerRouter(NewAuthHandler(auth, zap.New(core)))
			response := performAuthHandlerRequest(router, http.MethodPost, "/api/auth/login", `{"username":"operator","password":"test-password"}`, nil)
			if response.Code != http.StatusInternalServerError || response.Body.String() != `{"error":"auth_failed"}` {
				t.Fatalf("status=%d body=%s", response.Code, response.Body)
			}
			entries := logs.FilterMessage("auth request failed").All()
			if len(entries) != 1 || entries[0].ContextMap()["error"] != detail {
				t.Fatalf("dependency error was not retained in server logs: %+v", entries)
			}
		})
	}
}

func TestAuthHandlerLoginForbiddenStillReturns403(t *testing.T) {
	auth := &fakeHandlerAuthService{loginErr: service.ErrAuthForbidden}
	response := performAuthHandlerRequest(authHandlerRouter(NewAuthHandler(auth, zap.NewNop())), http.MethodPost, "/api/auth/login", `{"username":"operator","password":"test-password"}`, nil)
	if response.Code != http.StatusForbidden || response.Body.String() != `{"error":"forbidden"}` {
		t.Fatalf("status=%d body=%s", response.Code, response.Body)
	}
}

func TestAuthHandlerMeAndLogout(t *testing.T) {
	auth := &fakeHandlerAuthService{}
	router := authHandlerRouter(NewAuthHandler(auth, zap.NewNop()))

	headers := map[string]string{
		apimiddleware.ContextUsername:    "operator",
		apimiddleware.ContextRole:        "operator",
		apimiddleware.ContextAuthType:    "bearer",
		apimiddleware.ContextBearerToken: "token",
	}
	resp := performAuthHandlerRequest(router, http.MethodGet, "/api/auth/me", "", headers)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), `"username":"operator"`) {
		t.Fatalf("unexpected me response: status=%d body=%s", resp.Code, resp.Body.String())
	}

	resp = performAuthHandlerRequest(router, http.MethodPost, "/api/auth/logout", "", headers)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), `"ok":true`) {
		t.Fatalf("unexpected logout response: status=%d body=%s", resp.Code, resp.Body.String())
	}
	if auth.logoutToken != "token" {
		t.Fatalf("logout token = %q, want token", auth.logoutToken)
	}
}

func authHandlerRouter(h *AuthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api")
	api.Use(func(c *gin.Context) {
		for _, key := range []string{
			apimiddleware.ContextUsername,
			apimiddleware.ContextRole,
			apimiddleware.ContextAuthType,
			apimiddleware.ContextBearerToken,
		} {
			if value := c.GetHeader(key); value != "" {
				c.Set(key, value)
			}
		}
		c.Next()
	})
	api.POST("/auth/login", h.Login)
	api.GET("/auth/me", h.Me)
	api.POST("/auth/logout", h.Logout)
	return router
}

func performAuthHandlerRequest(router *gin.Engine, method string, path string, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

type fakeHandlerAuthService struct {
	loginResp   *dto.LoginResponse
	loginErr    error
	logoutErr   error
	logoutToken string
}

func (s *fakeHandlerAuthService) Login(ctx context.Context, username string, password string) (*dto.LoginResponse, error) {
	if s.loginErr != nil {
		return nil, s.loginErr
	}
	return s.loginResp, nil
}

func (s *fakeHandlerAuthService) Logout(ctx context.Context, token string) error {
	s.logoutToken = token
	return s.logoutErr
}

func (s *fakeHandlerAuthService) ValidateBearerToken(ctx context.Context, token string) (*service.AuthPrincipal, error) {
	return nil, errors.New("not implemented")
}
