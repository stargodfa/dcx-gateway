package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"go.yaml.in/yaml/v3"
)

type demoContainer struct {
	id   string
	addr string
}

func startContainer(ctx context.Context, runID, name, image, port string, environment ...string) (result demoContainer, err error) {
	args := []string{"run", "--detach", "--rm", "--name", "dcx-gateway-demo-" + runID + "-" + name, "--label", "dcx-gateway.demo=" + runID, "--publish", "127.0.0.1::" + port}
	for _, value := range environment {
		args = append(args, "--env", value)
	}
	id, err := dockerOutput(ctx, append(args, image)...)
	if err != nil {
		return result, err
	}
	result.id = id
	defer func() {
		if err != nil {
			err = errors.Join(err, result.close())
		}
	}()
	address, err := dockerOutput(ctx, "port", id, port)
	if err != nil {
		return result, err
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil || host != "127.0.0.1" {
		return result, errors.New("demo container did not publish a loopback-only port")
	}
	result.addr = address
	return result, nil
}

func dockerOutput(ctx context.Context, arguments ...string) (string, error) {
	command := exec.CommandContext(ctx, "docker", arguments...)
	var diagnostics bytes.Buffer
	command.Stderr = &diagnostics
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("docker %s failed; ensure Docker is installed and running: %w: %s", arguments[0], err, strings.TrimSpace(diagnostics.String()))
	}
	return strings.TrimSpace(string(output)), nil
}

func (c demoContainer) close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	// Remove only the container ID returned by this demo's successful docker run.
	_, err := dockerOutput(ctx, "rm", "--force", c.id)
	return err
}

func initializeDatabase(ctx context.Context, root, runID, address string) (string, error) {
	connection := mysql.NewConfig()
	connection.User = "root"
	connection.Net = "tcp"
	connection.Addr = address
	connection.ParseTime = true
	connection.MultiStatements = true
	connection.Timeout = time.Second
	connection.ReadTimeout = 5 * time.Second
	connection.WriteTimeout = 5 * time.Second
	// A fresh container may close connections during initialization. Keep these
	// expected readiness retries out of the interview transcript.
	readiness := *connection
	readiness.Logger = log.New(io.Discard, "", 0)
	connector, err := mysql.NewConnector(&readiness)
	if err != nil {
		return "", err
	}
	admin := sql.OpenDB(connector)
	defer admin.Close()
	ready, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for admin.PingContext(ready) != nil {
		select {
		case <-ready.Done():
			return "", fmt.Errorf("MySQL did not become ready: %w", ready.Err())
		case <-ticker.C:
		}
	}
	database := "dcx-gateway-demo-" + runID
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE `"+database+"` CHARACTER SET utf8mb4"); err != nil {
		return "", err
	}
	connection.DBName = database
	db, err := sql.Open("mysql", connection.FormatDSN())
	if err != nil {
		return "", err
	}
	defer db.Close()
	migrations, err := filepath.Glob(filepath.Join(root, "migrations", "[0-9][0-9][0-9]_*.sql"))
	if err != nil || len(migrations) == 0 {
		return "", errors.New("no numbered migrations found in the repository")
	}
	for _, path := range migrations { // Glob returns filenames in sorted order.
		raw, err := os.ReadFile(path)
		if err != nil {
			return "", err
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
			if _, err := db.ExecContext(ctx, string(raw)); err != nil {
				return "", fmt.Errorf("migration %s: %w", filepath.Base(path), err)
			}
		}
	}
	return connection.FormatDSN(), nil
}

func prepareDemoConfig(root, runID, dsn, redisAddr, wsURL, wsToken, apiToken string) (string, func() error, error) {
	raw, err := os.ReadFile(filepath.Join(root, "configs", "config.example.yaml"))
	if err != nil {
		return "", nil, err
	}
	var config map[string]any
	if err := yaml.Unmarshal(raw, &config); err != nil {
		return "", nil, err
	}
	config["app"] = map[string]any{"name": "dcx-gateway-demo", "env": "prod", "port": "0"}
	config["mysql"] = map[string]any{"dsn": dsn}
	config["redis"] = map[string]any{"addr": redisAddr, "key_prefix": "dcx-gateway-demo-" + runID + ":"}
	config["security"] = map[string]any{"enabled": true, "protect_read": false, "static_token_enabled": true, "static_token": apiToken}
	config["scheduler"] = map[string]any{"preload": map[string]any{"enabled": false}}
	config["mock_push"] = map[string]any{"enabled": false}
	config["fastapi"] = map[string]any{"ws_data_url": wsURL, "push_video_streams": false, "ws_auth": map[string]any{"type": "bearer", "token": wsToken}}
	raw, err = yaml.Marshal(config)
	if err != nil {
		return "", nil, err
	}
	directory, err := os.MkdirTemp("", "dcx-gateway-demo-")
	if err != nil {
		return "", nil, err
	}
	path := filepath.Join(directory, "config.yaml")
	cleanup := func() error {
		fileErr := os.Remove(path)
		if errors.Is(fileErr, os.ErrNotExist) {
			fileErr = nil
		}
		return errors.Join(fileErr, os.Remove(directory))
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		return "", nil, errors.Join(err, cleanup())
	}
	return path, cleanup, nil
}
