package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gateway "dcx-gateway/internal/model/protocol/gateway"
	"dcx-gateway/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type fakeScriptPreloadService struct {
	service.ScriptService
	calls   int
	options service.PreloadOptions
}

func (s *fakeScriptPreloadService) PreloadWithOptions(_ context.Context, id string, options service.PreloadOptions) (*gateway.DataBundle, error) {
	s.calls++
	s.options = options
	return &gateway.DataBundle{BundleID: "test-bundle", ScriptID: id}, nil
}

func TestScriptHandlerPreloadValidatesBodyBeforeCallingService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tt := range []struct {
		name  string
		body  string
		valid bool
		force bool
	}{
		{name: "omitted", valid: true},
		{name: "whitespace", body: " \n\t", valid: true},
		{name: "empty object", body: `{}`, valid: true},
		{name: "force false", body: `{"force":false}`, valid: true},
		{name: "force true", body: `{"force":true}`, valid: true, force: true},
		{name: "unknown field", body: `{"extension":"value"}`, valid: true},
		{name: "wrong force type", body: `{"force":"invalid-type"}`},
		{name: "numeric force", body: `{"force":1}`},
		{name: "null force", body: `{"force":null}`},
		{name: "null object", body: `null`},
		{name: "array", body: `[]`},
		{name: "boolean", body: `true`},
		{name: "truncated", body: `{"force":`},
		{name: "trailing object", body: `{"force":true} {}`},
		{name: "trailing garbage", body: `{"force":true} garbage`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeScriptPreloadService{}
			router := gin.New()
			router.POST("/api/scripts/:id/preload", NewScriptHandler(svc, zap.NewNop()).Preload)
			req := httptest.NewRequest(http.MethodPost, "/api/scripts/security_main/preload", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			if !tt.valid {
				if response.Code != http.StatusBadRequest || svc.calls != 0 || !strings.Contains(response.Body.String(), `"code":"BAD_REQUEST"`) {
					t.Fatalf("invalid input: status=%d calls=%d body=%s", response.Code, svc.calls, response.Body)
				}
				return
			}
			if response.Code != http.StatusOK || svc.calls != 1 || svc.options.Force != tt.force || !svc.options.PushWS {
				t.Fatalf("valid input: status=%d calls=%d options=%+v body=%s", response.Code, svc.calls, svc.options, response.Body)
			}
		})
	}
}
