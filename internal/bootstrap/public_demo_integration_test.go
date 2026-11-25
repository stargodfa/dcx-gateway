package bootstrap

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"dcx-gateway/internal/model/protocol/ue"

	drivermysql "github.com/go-sql-driver/mysql"
	"go.yaml.in/yaml/v3"
	"golang.org/x/crypto/bcrypt"
)

// These endpoints must refer to disposable test services. The test creates and
// drops only its own database and boots the public sample without external UE.
func TestPublicDemoIntegration(t *testing.T) {
	dsn := os.Getenv("DCX_GATEWAY_TEST_MYSQL_DSN")
	redisAddr := os.Getenv("DCX_GATEWAY_TEST_REDIS_ADDR")
	if dsn == "" || redisAddr == "" {
		t.Skip("set DCX_GATEWAY_TEST_MYSQL_DSN and DCX_GATEWAY_TEST_REDIS_ADDR for public demo integration")
	}
	connection, err := drivermysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	connection.DBName = ""
	connection.ParseTime = true
	connection.MultiStatements = true
	admin, err := sql.Open("mysql", connection.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	database := fmt.Sprintf("dcx-gateway-public-demo-%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE `" + database + "` CHARACTER SET utf8mb4"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec("DROP DATABASE `" + database + "`"); err != nil {
			t.Error(err)
		}
	}()
	connection.DBName = database
	db, err := sql.Open("mysql", connection.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..")
	migrations, err := filepath.Glob(filepath.Join(root, "migrations", "[0-9][0-9][0-9]_*.sql"))
	if err != nil || len(migrations) != 14 {
		t.Fatalf("expected 14 numbered migrations: count=%d err=%v", len(migrations), err)
	}
	sort.Strings(migrations)
	for range 2 {
		for _, migration := range migrations {
			raw, err := os.ReadFile(migration)
			if err != nil {
				t.Fatal(err)
			}
			hasSQL := false
			for _, line := range strings.Split(string(raw), "\n") {
				line = strings.TrimSpace(line)
				if line != "" && !strings.HasPrefix(line, "--") {
					hasSQL = true
					break
				}
			}
			if hasSQL {
				if _, err := db.Exec(string(raw)); err != nil {
					t.Fatalf("migration %s: %v", filepath.Base(migration), err)
				}
			}
		}
	}
	var users int
	if err := db.QueryRow("SELECT COUNT(*) FROM ue_users").Scan(&users); err != nil || users != 0 {
		t.Fatalf("public demo must not seed login users: count=%d err=%v", users, err)
	}
	rows, err := db.Query("SELECT config_key, content_json FROM ue_config_documents")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var key, payload string
		if err := rows.Scan(&key, &payload); err != nil {
			t.Fatal(err)
		}
		if !json.Valid([]byte(payload)) {
			t.Errorf("public config %s is not valid JSON", key)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()
	raw, err := os.ReadFile(filepath.Join(root, "configs", "config.example.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	cfg["mysql"] = map[string]any{"dsn": connection.FormatDSN()}
	cfg["redis"] = map[string]any{"addr": redisAddr, "key_prefix": database + ":"}
	raw, err = yaml.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	app, err := NewApp(configPath)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		iterator := app.Redis.Scan(ctx, 0, database+":*", 100).Iterator()
		var keys []string
		for iterator.Next(ctx) {
			keys = append(keys, iterator.Val())
		}
		if err := iterator.Err(); err != nil {
			t.Error(err)
		}
		if len(keys) > 0 {
			if err := app.Redis.Del(ctx, keys...).Err(); err != nil {
				t.Error(err)
			}
		}
	}()
	if app.FastAPIClient != nil || app.Config.Security.StaticTokenEnabled || app.Config.Security.StaticToken != "" {
		t.Fatal("public template must not enable external push or ship a static token")
	}
	for _, path := range []string{
		"/api/health?deep=true", "/api/ue/config/bootstrap", "/api/ue/config/model-table",
		"/api/ue/config/scene-desired?mode=generated", "/api/ue/config/model-anchors",
		"/api/ue/config/sensor-devices", "/api/ue/config/cluster-agents-catalog",
		"/api/ue/config/cluster-agent-registry", "/api/scripts/security_main/data-bundle",
		"/api/scripts/energy_main/data-bundle", "/api/scripts/education_main/data-bundle",
		"/api/ue/config/ndisplay/cluster_1node_2viewport.ndisplay.template",
	} {
		t.Run(path, func(t *testing.T) {
			response := httptest.NewRecorder()
			app.Router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			if response.Code != http.StatusOK || !json.Valid(response.Body.Bytes()) {
				t.Fatalf("status=%d body=%s", response.Code, response.Body)
			}
		})
	}
	// Create a test-only account after checking that the public seed has no users.
	hash, err := bcrypt.GenerateFromPassword([]byte("demo-test-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO ue_users (username, password_hash, role, enabled) VALUES (?, ?, ?, ?)", "demo-integration", string(hash), "operator", true); err != nil {
		t.Fatal(err)
	}
	login := demoAPIRequest(t, app.Router, http.MethodPost, "/api/auth/login", `{"username":"demo-integration","password":"demo-test-password"}`, "", http.StatusOK)
	var credentials struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(login.Body.Bytes(), &credentials); err != nil || credentials.AccessToken == "" {
		t.Fatalf("test account login failed: %v", err)
	}
	token := credentials.AccessToken
	t.Run("seed model edit reset disable and restore", func(t *testing.T) {
		demoAPIRequest(t, app.Router, http.MethodPut, "/api/ue/config/model-config/placements/demo_model_1", `{"visible":false}`, token, http.StatusOK)
		response := demoAPIRequest(t, app.Router, http.MethodGet, "/api/ue/config/model-config?mode=structured", "", token, http.StatusOK)
		var config ue.StructuredModelConfig
		if err := json.Unmarshal(response.Body.Bytes(), &config); err != nil {
			t.Fatal(err)
		}
		if placement, exists := config.Placements["demo_model_1"]; !exists || placement.Visible {
			t.Fatalf("seed model visibility was not updated: %+v", config)
		}
		demoAPIRequest(t, app.Router, http.MethodDelete, "/api/ue/config/model-config/placements/delete", `{"instance_id":"demo_model_1"}`, token, http.StatusOK)
		response = demoAPIRequest(t, app.Router, http.MethodGet, "/api/ue/config/model-config?mode=structured", "", token, http.StatusOK)
		if err := json.Unmarshal(response.Body.Bytes(), &config); err != nil {
			t.Fatal(err)
		}
		if placement, exists := config.Placements["demo_model_1"]; !exists || !placement.Visible {
			t.Fatalf("placement reset did not restore default visibility: %+v", config)
		}
		demoAPIRequest(t, app.Router, http.MethodDelete, "/api/ue/config/model-table/entries/delete", `{"instance_id":"demo_model_1"}`, token, http.StatusOK)
		response = demoAPIRequest(t, app.Router, http.MethodGet, "/api/ue/config/model-table", "", token, http.StatusOK)
		var table ue.ModelTable
		if err := json.Unmarshal(response.Body.Bytes(), &table); err != nil || len(table.Entries) != 0 {
			t.Fatalf("disabled model remains in catalog: %+v err=%v", table, err)
		}
		demoAPIRequest(t, app.Router, http.MethodPut, "/api/ue/config/model-table/entries", `{"instance_id":"demo_model_1","enabled":true}`, token, http.StatusOK)
		response = demoAPIRequest(t, app.Router, http.MethodGet, "/api/ue/config/model-table", "", token, http.StatusOK)
		if err := json.Unmarshal(response.Body.Bytes(), &table); err != nil || len(table.Entries) != 1 || table.Entries[0].InstanceID != "demo_model_1" {
			t.Fatalf("seed model was not restored: %+v err=%v", table, err)
		}
	})
	t.Run("seed report scene warmup succeeds", func(t *testing.T) {
		response := demoAPIRequest(t, app.Router, http.MethodPost, "/api/ue/runtime/report-scenes/preload", `{"script_id":"security_main","chapter_id":"demo_chapter","scene_id":"demo_scene","force":true,"push_ws":false}`, token, http.StatusOK)
		var result ue.ReportScenePreloadResponse
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if len(result.Results) != 1 || !result.Results[0].Success || result.Results[0].DataScriptID != "security_main" || result.Results[0].BundleID == "" {
			t.Fatalf("seed warmup contains unresolved data scripts: %+v", result)
		}
		if len(result.Warmup.Charts) != 2 || result.Warmup.Charts[0] != "sim_3" || result.Warmup.Charts[1] != "sim_4" {
			t.Fatalf("seed chart resources must be binding IDs: %+v", result.Warmup)
		}
	})
	t.Run("preload input contract", func(t *testing.T) {
		demoAPIRequest(t, app.Router, http.MethodPost, "/api/scripts/security_main/preload", `{"force":"invalid-type"}`, token, http.StatusBadRequest)
		demoAPIRequest(t, app.Router, http.MethodPost, "/api/scripts/security_main/preload", `{"force":true}`, token, http.StatusOK)
		demoAPIRequest(t, app.Router, http.MethodPost, "/api/scripts/security_main/preload", "", token, http.StatusOK)
	})
	t.Run("ndisplay aliases and missing document", func(t *testing.T) {
		path := "/api/ue/config/ndisplay/cluster_1node_2viewport.ndisplay.template"
		original := demoAPIRequest(t, app.Router, http.MethodGet, path, "", token, http.StatusOK)
		alias := demoAPIRequest(t, app.Router, http.MethodGet, path+".json", "", token, http.StatusOK)
		if original.Body.String() != alias.Body.String() {
			t.Fatal("nDisplay .json suffix did not resolve to the same stored document")
		}
		demoAPIRequest(t, app.Router, http.MethodGet, "/api/ue/config/ndisplay/missing-template", "", token, http.StatusNotFound)
	})
}

func demoAPIRequest(t *testing.T, router http.Handler, method, path, body, token string, status int) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != status || !json.Valid(response.Body.Bytes()) {
		t.Fatalf("%s %s: status=%d want=%d body=%s", method, path, response.Code, status, response.Body)
	}
	return response
}
