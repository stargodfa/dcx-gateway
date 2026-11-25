// Command dcx-gateway-demo runs the gateway against disposable local services.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"dcx-gateway/internal/bootstrap"
	"dcx-gateway/internal/model/dto"
	"dcx-gateway/internal/model/protocol/ue"
	"dcx-gateway/internal/websocket"

	"github.com/gin-gonic/gin"
)

func main() {
	keepRunning := flag.Bool("keep-running", false, "keep the demo HTTP service running until Ctrl+C")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := runDemo(ctx, *keepRunning); err != nil {
		fmt.Fprintln(os.Stderr, "demo failed:", err)
		os.Exit(1)
	}
}

func runDemo(ctx context.Context, keepRunning bool) (err error) {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(root, "configs", "config.example.yaml")); err != nil {
		return fmt.Errorf("run this command from the dcx-gateway repository root: %w", err)
	}
	runID, err := randomID()
	if err != nil {
		return err
	}
	fmt.Println("[1/4] Starting disposable MySQL, Redis and the gateway...")
	mysqlContainer, err := startContainer(ctx, runID, "mysql", "mysql:8.4", "3306", "MYSQL_ALLOW_EMPTY_PASSWORD=yes")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, mysqlContainer.close()) }()
	redisContainer, err := startContainer(ctx, runID, "redis", "redis:7-alpine", "6379")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, redisContainer.close()) }()
	dsn, err := initializeDatabase(ctx, root, runID, mysqlContainer.addr)
	if err != nil {
		return err
	}
	wsToken, err := randomID()
	if err != nil {
		return err
	}
	receiver, err := startDemoReceiver(wsToken)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, receiver.close()) }()
	apiToken, err := randomID()
	if err != nil {
		return err
	}
	configPath, removeConfig, err := prepareDemoConfig(root, runID, dsn, redisContainer.addr, receiver.url, wsToken, apiToken)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, removeConfig()) }()
	gin.SetMode(gin.ReleaseMode)
	app, err := bootstrap.NewApp(configPath)
	if err != nil {
		return err
	}
	defer app.Close()
	baseURL, stopHTTP, err := serveLocal(app.Router)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, stopHTTP()) }()
	client := demoClient{baseURL: baseURL, token: apiToken, http: &http.Client{Timeout: 10 * time.Second}}
	var health struct {
		Status string `json:"status"`
	}
	if err := client.request(ctx, http.MethodGet, "/api/health?deep=true", nil, &health); err != nil {
		return err
	}
	if health.Status != "ok" {
		return fmt.Errorf("deep health status: %s", health.Status)
	}
	fmt.Printf("  Gateway: %s, deep health: %s\n", baseURL, health.Status)

	fmt.Println("[2/4] Updating demo_model_1 and reading its persisted placement...")
	patch := map[string]any{
		"visible":   true,
		"transform": map[string]any{"position": ue.Vector3{X: 100, Y: 200, Z: 300}},
	}
	var updated ue.StructuredModelConfig
	if err := client.request(ctx, http.MethodPut, "/api/ue/config/model-config/placements/demo_model_1", patch, &updated); err != nil {
		return err
	}
	var readback ue.StructuredModelConfig
	if err := client.request(ctx, http.MethodGet, "/api/ue/config/model-config?mode=structured", nil, &readback); err != nil {
		return err
	}
	placement, exists := readback.Placements["demo_model_1"]
	if !exists || !placement.Visible || placement.Transform == nil || placement.Transform.Position == nil || *placement.Transform.Position != (ue.Vector3{X: 100, Y: 200, Z: 300}) {
		return errors.New("model placement readback did not match the update")
	}
	fmt.Println("  demo_model_1: visible=true, position=(100, 200, 300), readback verified")

	fmt.Println("[3/4] Preloading security_main and pushing to the mock FastAPI receiver...")
	var preload dto.APIResponse[dto.PreloadResponse]
	if err := client.request(ctx, http.MethodPost, "/api/scripts/security_main/preload", dto.PreloadRequest{Force: true}, &preload); err != nil {
		return err
	}
	if preload.Code != "OK" || preload.Data.Status != "done" || preload.Data.TaskID == "" {
		return errors.New("preload response did not report a completed task")
	}
	fmt.Printf("  preload: status=%s, task_id=%s\n", preload.Data.Status, preload.Data.TaskID)
	if _, ok, err := bootstrap.NewCaches(app.Redis, app.Config.Redis.KeyPrefix).Script.GetPreloadBundle(ctx, "security_main"); err != nil || !ok {
		return fmt.Errorf("preload cache verification failed: found=%v err=%v", ok, err)
	}
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for receiver.messages.Load() == 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return errors.New("mock FastAPI did not receive chart_data_push")
		case <-ticker.C:
		}
	}

	fmt.Println("[4/4] Querying WebSocket connection and push counters...")
	var status websocket.FastAPIClientStats
	if err := client.request(ctx, http.MethodGet, "/api/runtime/ws-data/status", nil, &status); err != nil {
		return err
	}
	if !status.Enabled || !status.Connected || status.SuccessCount == 0 || status.FailedCount != 0 {
		return fmt.Errorf("unexpected push status: connected=%v success=%d failed=%d", status.Connected, status.SuccessCount, status.FailedCount)
	}
	fmt.Printf("  enabled=%v connected=%v success_count=%d failed_count=%d received_chart_messages=%d\n", status.Enabled, status.Connected, status.SuccessCount, status.FailedCount, receiver.messages.Load())
	fmt.Println("DEMO PASSED: startup -> model update -> preload -> status query")
	if keepRunning {
		fmt.Printf("Read-only inspection: %s/api/runtime/ws-data/status\nPress Ctrl+C to clean up the demo.\n", baseURL)
		<-ctx.Done()
	}
	fmt.Println("Cleaning up temporary configuration and demo containers...")
	return nil
}

func randomID() (string, error) {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(data[:]), nil
}

type demoClient struct {
	baseURL string
	token   string
	http    *http.Client
}

func (c demoClient) request(ctx context.Context, method, path string, body any, target any) error {
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("X-Gateway-Token", c.token)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.http.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%s %s returned %s", method, path, response.Status)
	}
	return json.NewDecoder(response.Body).Decode(target)
}

func serveLocal(handler http.Handler) (string, func() error, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	closeServer := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		shutdownErr := server.Shutdown(ctx)
		if shutdownErr != nil {
			_ = server.Close()
		}
		serveErr := <-done
		if errors.Is(serveErr, http.ErrServerClosed) {
			serveErr = nil
		}
		return errors.Join(shutdownErr, serveErr)
	}
	return "http://" + listener.Addr().String(), closeServer, nil
}
