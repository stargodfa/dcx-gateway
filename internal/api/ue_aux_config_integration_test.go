package api_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"dcx-gateway/internal/api"
	"dcx-gateway/internal/api/handler"
	apimiddleware "dcx-gateway/internal/api/middleware"
	"dcx-gateway/internal/config"
	"dcx-gateway/internal/repository"
	"dcx-gateway/internal/service"

	"github.com/gin-gonic/gin"
	drivermysql "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// The optional DSN must point at a disposable MySQL instance with CREATE DATABASE
// permission. Every run creates and drops only its own dcx-gateway-aux-test-* database.
func auxIntegrationRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	dsn := os.Getenv("DCX_GATEWAY_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set DCX_GATEWAY_TEST_MYSQL_DSN to run MySQL HTTP integration tests")
	}
	cfg, err := drivermysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.DBName = ""
	cfg.ParseTime = true
	admin, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = admin.Close() })
	database := fmt.Sprintf("dcx-gateway-aux-test-%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE `" + database + "` CHARACTER SET utf8mb4"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec("DROP DATABASE `" + database + "`"); err != nil {
			t.Error(err)
		}
	})
	cfg.DBName = database
	db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	_, file, _, _ := runtime.Caller(0)
	migration, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", "migrations", "014_init_ue_aux_configs.sql"))
	if err != nil {
		t.Fatal(err)
	}
	// Verify migration can also be safely rerun on an existing deployment.
	for range 2 {
		for _, statement := range strings.Split(string(migration), ";") {
			if strings.TrimSpace(statement) != "" {
				if err := db.Exec(statement).Error; err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()
	svc := service.NewUEAuxConfigService(repository.NewUEAuxConfigRepository(db, log))
	router := gin.New()
	api.RegisterRoutes(router, api.Handlers{
		Auth:        handler.NewAuthHandler(nil, log),
		Script:      handler.NewScriptHandler(nil, log),
		UEConfig:    handler.NewUEConfigHandler(nil, log),
		UEAuxConfig: handler.NewUEAuxConfigHandler(svc, log, 1024*1024),
	}, apimiddleware.RequestID(), apimiddleware.Auth(config.SecurityConfig{
		Enabled: true, StaticTokenEnabled: true, StaticToken: "integration-token",
	}, nil))
	return router, db
}

func auxRequest(t *testing.T, router http.Handler, method, path, body string, want int) map[string]any {
	t.Helper()
	req := httptest.NewRequest(method, "/api/ue/config/"+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gateway-Token", "integration-token")
	req.Header.Set("X-Request-Id", "aux-integration")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != want {
		t.Fatalf("%s %s: status=%d want=%d body=%s", method, path, w.Code, want, w.Body)
	}
	if w.Header().Get("X-Request-Id") != "aux-integration" {
		t.Fatal("request ID middleware missing")
	}
	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestAuxConfigMySQLHTTPIntegration(t *testing.T) {
	router, db := auxIntegrationRouter(t)
	t.Run("authentication and empty documents", func(t *testing.T) {
		for _, resource := range []string{"model-anchors", "sensor-devices", "cluster-agents-catalog", "cluster-agent-registry"} {
			auxRequest(t, router, "GET", resource, "", 200)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest("PUT", "/api/ue/config/"+resource, strings.NewReader(`{}`)))
			if w.Code != 401 {
				t.Fatalf("unprotected resource %s: %d", resource, w.Code)
			}
		}
	})
	t.Run("anchor optimistic locking and atomic rollback", func(t *testing.T) {
		auxRequest(t, router, "PUT", "model-anchors/a", `{"anchor_id":"wrong"}`, 400)
		auxRequest(t, router, "PUT", "model-anchors/a", `{"anchor_id":"a","revision":2}`, 409)
		result := auxRequest(t, router, "PUT", "model-anchors/a", `{"anchor_id":"a","instance_id":"i","title":"old","extra":{"retained":true}}`, 200)
		if result["revision"] != float64(1) {
			t.Fatalf("create revision = %v", result)
		}
		result = auxRequest(t, router, "PUT", "model-anchors/a", `{"anchor_id":"a","instance_id":"i","title":"old","revision":1,"extra":{"retained":true}}`, 200)
		if result["revision"] != float64(1) {
			t.Fatalf("idempotent revision = %v", result)
		}
		var wg sync.WaitGroup
		statuses := make(chan int, 2)
		for _, title := range []string{"concurrent-one", "concurrent-two"} {
			wg.Go(func() {
				body := fmt.Sprintf(`{"anchor_id":"a","instance_id":"i","revision":1,"title":%q}`, title)
				req := httptest.NewRequest("PUT", "/api/ue/config/model-anchors/a", strings.NewReader(body))
				req.Header.Set("X-Gateway-Token", "integration-token")
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				statuses <- w.Code
			})
		}
		wg.Wait()
		close(statuses)
		counts := map[int]int{}
		for status := range statuses {
			counts[status]++
		}
		if counts[200] != 1 || counts[409] != 1 {
			t.Fatalf("concurrent status counts = %v", counts)
		}
		auxRequest(t, router, "PUT", "model-anchors/b", `{"anchor_id":"b","instance_id":"i","title":"b"}`, 200)
		auxRequest(t, router, "PUT", "model-anchors/c", `{"anchor_id":"c","instance_id":"i","title":"c"}`, 200)
		before := auxRequest(t, router, "GET", "model-anchors", "", 200)
		auxRequest(t, router, "PUT", "model-anchors", `{"anchors":[{"anchor_id":"a","instance_id":"i","revision":2,"title":"must-rollback"},{"anchor_id":"b","revision":0}]}`, 409)
		after := auxRequest(t, router, "GET", "model-anchors", "", 200)
		beforeJSON, _ := json.Marshal(before)
		afterJSON, _ := json.Marshal(after)
		if string(beforeJSON) != string(afterJSON) {
			t.Fatalf("failed batch changed records: before=%s after=%s", beforeJSON, afterJSON)
		}
		result = auxRequest(t, router, "PUT", "model-anchors", `{"anchors":[{"anchor_id":"b","instance_id":"i","title":"b","revision":1}]}`, 200)
		if len(result["anchors"].([]any)) != 1 {
			t.Fatalf("replacement = %v", result)
		}
		auxRequest(t, router, "DELETE", "model-anchors/delete", `{"anchor_id":"b","anchor_ids":["b","missing"]}`, 200)
		result = auxRequest(t, router, "GET", "model-anchors", "", 200)
		if len(result["anchors"].([]any)) != 0 {
			t.Fatalf("delete = %v", result)
		}
	})
	t.Run("sensor last write wins and whole document replacement", func(t *testing.T) {
		auxRequest(t, router, "PUT", "sensor-devices/d", `{"device_id":"other"}`, 400)
		auxRequest(t, router, "PUT", "sensor-devices/d", `{"device_id":"d","status":"normal","extension":{"retained":true}}`, 200)
		auxRequest(t, router, "PUT", "sensor-devices/d", `{"device_id":"d","status":"fault","extension":{"retained":true}}`, 200)
		result := auxRequest(t, router, "GET", "sensor-devices", "", 200)
		device := result["devices"].([]any)[0].(map[string]any)
		if device["status"] != "fault" || device["extension"] == nil {
			t.Fatalf("sensor readback = %v", result)
		}
		auxRequest(t, router, "PUT", "sensor-devices", `{"devices":[{"device_id":"e","status":"normal"}]}`, 200)
		auxRequest(t, router, "PUT", "sensor-devices", `{}`, 400)
		auxRequest(t, router, "PUT", "sensor-devices", `{"devices":[{"device_id":"f"},{}]}`, 400)
		result = auxRequest(t, router, "GET", "sensor-devices", "", 200)
		if device := result["devices"].([]any)[0].(map[string]any); device["device_id"] != "e" || len(result["devices"].([]any)) != 1 {
			t.Fatalf("invalid batch modified records = %v", result)
		}
		auxRequest(t, router, "DELETE", "sensor-devices/delete", `{"device_ids":["e","missing","e"]}`, 200)
		auxRequest(t, router, "PUT", "sensor-devices", `{"devices":[]}`, 200)
	})
	t.Run("cluster catalog merge and registry derived whitelist", func(t *testing.T) {
		auxRequest(t, router, "PUT", "cluster-agents-catalog/agents/n", `{"ip":"192.0.2.10","metadata":{"x":1}}`, 200)
		result := auxRequest(t, router, "PUT", "cluster-agents-catalog/agents/n", `{"status":"offline","nodeId":"wrong"}`, 200)
		if result["ip"] != "192.0.2.10" || result["nodeId"] != "n" || result["metadata"] == nil {
			t.Fatalf("catalog merge = %v", result)
		}
		result = auxRequest(t, router, "GET", "cluster-agents-catalog", "", 200)
		if result["updatedAt"].(float64) <= 0 {
			t.Fatalf("catalog timestamp = %v", result)
		}
		auxRequest(t, router, "PUT", "cluster-agents-catalog", `{"agents":{"z":{"status":"online"}}}`, 200)
		auxRequest(t, router, "DELETE", "cluster-agents-catalog/agents/delete", `{"node_id":"z"}`, 200)
		auxRequest(t, router, "PUT", "cluster-agents-catalog", `{"agents":{}}`, 200)
		result = auxRequest(t, router, "PUT", "cluster-agent-registry", `{"agents":{"b":{"enabled":true,"unknown":42},"a":{"enabled":true},"c":{"enabled":false}},"authorizedWhitelist":["c"]}`, 200)
		if whitelist := result["authorizedWhitelist"].([]any); len(whitelist) != 2 || whitelist[0] != "a" || whitelist[1] != "b" {
			t.Fatalf("derived whitelist = %v", result)
		}
		result = auxRequest(t, router, "PUT", "cluster-agent-registry/agents/a", `{"enabled":false}`, 200)
		if whitelist := result["authorizedWhitelist"].([]any); len(whitelist) != 1 || whitelist[0] != "b" {
			t.Fatalf("disabled whitelist = %v", result)
		}
		result = auxRequest(t, router, "GET", "cluster-agent-registry", "", 200)
		if result["agents"].(map[string]any)["b"].(map[string]any)["unknown"] != float64(42) {
			t.Fatal("registry unknown field lost")
		}
		auxRequest(t, router, "DELETE", "cluster-agent-registry/agents/delete", `{"node_ids":["b","missing"]}`, 200)
		auxRequest(t, router, "PUT", "cluster-agent-registry", `{"agents":{}}`, 200)
		result = auxRequest(t, router, "GET", "cluster-agent-registry", "", 200)
		if len(result["authorizedWhitelist"].([]any)) != 0 || len(result["agents"].(map[string]any)) != 0 {
			t.Fatalf("cleared registry = %v", result)
		}
	})
	t.Run("database write failure rolls back entire replacement", func(t *testing.T) {
		auxRequest(t, router, "PUT", "sensor-devices/keep", `{"device_id":"keep","status":"normal"}`, 200)
		err := db.Exec(`CREATE TRIGGER reject_aux_test_device BEFORE INSERT ON sensor_devices
			FOR EACH ROW BEGIN
				IF NEW.device_id = 'z_fail' THEN
					SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'injected test write failure';
				END IF;
			END`).Error
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = db.Exec("DROP TRIGGER IF EXISTS reject_aux_test_device").Error })
		auxRequest(t, router, "PUT", "sensor-devices", `{"devices":[{"device_id":"a_new"},{"device_id":"z_fail"}]}`, 500)
		result := auxRequest(t, router, "GET", "sensor-devices", "", 200)
		devices := result["devices"].([]any)
		if len(devices) != 1 || devices[0].(map[string]any)["device_id"] != "keep" {
			t.Fatalf("failed database write changed collection: %v", result)
		}
	})
	t.Run("concurrent first writes preserve last-write-wins and merge", func(t *testing.T) {
		for _, resource := range []string{"sensor-devices", "cluster-agents-catalog", "cluster-agent-registry"} {
			path := resource + "/agents/first-writer"
			if resource == "sensor-devices" {
				path = resource + "/first-writer"
			}
			start := make(chan struct{})
			failures := make(chan string, 4)
			var wg sync.WaitGroup
			for i := range 4 {
				wg.Go(func() {
					<-start
					body := fmt.Sprintf(`{"device_id":"first-writer","enabled":true,"field_%d":true}`, i)
					req := httptest.NewRequest("PUT", "/api/ue/config/"+path, strings.NewReader(body))
					req.Header.Set("X-Gateway-Token", "integration-token")
					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)
					if w.Code != 200 {
						failures <- fmt.Sprintf("%s status=%d body=%s", resource, w.Code, w.Body)
					}
				})
			}
			close(start)
			wg.Wait()
			close(failures)
			for failure := range failures {
				t.Error(failure)
			}
			if resource == "cluster-agents-catalog" {
				result := auxRequest(t, router, "GET", resource, "", 200)
				agent := result["agents"].(map[string]any)["first-writer"].(map[string]any)
				for i := range 4 {
					if agent[fmt.Sprintf("field_%d", i)] != true {
						t.Fatalf("concurrent catalog merge lost fields: %v", agent)
					}
				}
			}
		}
	})
}
