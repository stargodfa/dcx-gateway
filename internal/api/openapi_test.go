package api_test

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"dcx-gateway/internal/api"
	"dcx-gateway/internal/api/handler"

	"github.com/gin-gonic/gin"
	"go.yaml.in/yaml/v3"
)

func TestOpenAPIMatchesRegisteredRoutes(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", "docs", "api", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := yaml.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	paths, ok := document["paths"].(map[string]any)
	if !ok {
		t.Fatal("OpenAPI paths must be an object")
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// All optional handlers are present so every production route is registered.
	api.RegisterRoutes(router, api.Handlers{
		Script:      &handler.ScriptHandler{},
		UEConfig:    &handler.UEConfigHandler{},
		Auth:        &handler.AuthHandler{},
		Health:      &handler.HealthHandler{},
		Runtime:     &handler.RuntimeHandler{},
		UEAuxConfig: &handler.UEAuxConfigHandler{},
	})
	parameter := regexp.MustCompile(`:([A-Za-z0-9_]+)`)
	registered := map[string]bool{}
	for _, route := range router.Routes() {
		registered[route.Method+" "+parameter.ReplaceAllString(route.Path, `{$1}`)] = true
	}
	documented := map[string]bool{}
	for path, item := range paths {
		operations, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("invalid path item %s", path)
		}
		for method := range operations {
			switch method {
			case "get", "put", "post", "delete", "patch", "head", "options", "trace":
				documented[strings.ToUpper(method)+" "+path] = true
			}
		}
	}
	for route := range registered {
		if !documented[route] {
			t.Errorf("registered route missing from OpenAPI: %s", route)
		}
	}
	for route := range documented {
		if !registered[route] {
			t.Errorf("OpenAPI operation has no registered route: %s", route)
		}
	}
	var checkReferences func(any)
	checkReferences = func(value any) {
		switch value := value.(type) {
		case map[string]any:
			if reference, ok := value["$ref"].(string); ok && strings.HasPrefix(reference, "#/") {
				var target any = document
				for _, part := range strings.Split(strings.TrimPrefix(reference, "#/"), "/") {
					object, ok := target.(map[string]any)
					if !ok || object[part] == nil {
						t.Errorf("unresolved OpenAPI reference: %s", reference)
						break
					}
					target = object[part]
				}
			}
			for _, child := range value {
				checkReferences(child)
			}
		case []any:
			for _, child := range value {
				checkReferences(child)
			}
		}
	}
	checkReferences(document)
}
