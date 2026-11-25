package security

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"dcx-gateway/internal/config"
	"dcx-gateway/internal/model/domain"

	"go.uber.org/zap"
)

type HTTPAdapter struct {
	client            *http.Client
	baseURL           string
	accessSummaryPath string
	fireAlertsPath    string
	auth              config.OutboundAuthConfig
	log               *zap.Logger
}

func NewHTTPAdapter(baseURL string, timeout time.Duration, accessSummaryPath string, fireAlertsPath string, log *zap.Logger, auth ...config.OutboundAuthConfig) Adapter {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	if strings.TrimSpace(accessSummaryPath) == "" {
		accessSummaryPath = "/api/security/access-summary"
	}
	if strings.TrimSpace(fireAlertsPath) == "" {
		fireAlertsPath = "/api/security/fire-alerts"
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &HTTPAdapter{
		client:            &http.Client{Timeout: timeout},
		baseURL:           strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		accessSummaryPath: accessSummaryPath,
		fireAlertsPath:    fireAlertsPath,
		auth:              firstAuth(auth),
		log:               log,
	}
}

func (a *HTTPAdapter) GetAccessSummary(ctx context.Context) (*domain.AccessSummary, error) {
	var payload accessSummaryResponse
	if err := a.getJSON(ctx, a.accessSummaryPath, &payload); err != nil {
		return nil, err
	}
	if payload.Timestamp == "" {
		payload.Timestamp = time.Now().Format(time.RFC3339)
	}
	if payload.TotalCount == 0 {
		payload.TotalCount = payload.InCount + payload.OutCount
	}
	return &domain.AccessSummary{
		InCount:    payload.InCount,
		OutCount:   payload.OutCount,
		TotalCount: payload.TotalCount,
		Timestamp:  payload.Timestamp,
	}, nil
}

func (a *HTTPAdapter) GetFireAlerts(ctx context.Context) ([]domain.FireAlert, error) {
	var payload fireAlertsResponse
	if err := a.getJSON(ctx, a.fireAlertsPath, &payload); err != nil {
		return nil, err
	}
	if payload.Alerts == nil {
		return nil, fmt.Errorf("security fire alerts response missing alerts")
	}
	alerts := make([]domain.FireAlert, 0, len(payload.Alerts))
	for _, alert := range payload.Alerts {
		if alert.AlertID == "" {
			return nil, fmt.Errorf("security fire alert missing alert_id")
		}
		if alert.Level == "" {
			alert.Level = "unknown"
		}
		if alert.Timestamp == "" {
			alert.Timestamp = time.Now().Format(time.RFC3339)
		}
		alerts = append(alerts, domain.FireAlert{
			AlertID:   alert.AlertID,
			Level:     alert.Level,
			Location:  alert.Location,
			Message:   alert.Message,
			Timestamp: alert.Timestamp,
		})
	}
	return alerts, nil
}

func (a *HTTPAdapter) getJSON(ctx context.Context, path string, out any) error {
	if a.baseURL == "" {
		return fmt.Errorf("security http adapter base_url is empty")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, joinURL(a.baseURL, path), nil)
	if err != nil {
		return err
	}
	config.ApplyOutboundAuthHeader(req.Header, a.auth)
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("security http adapter returned status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return err
	}
	return nil
}

func firstAuth(auth []config.OutboundAuthConfig) config.OutboundAuthConfig {
	if len(auth) == 0 {
		return config.OutboundAuthConfig{Type: "none"}
	}
	return config.NormalizeOutboundAuth(auth[0])
}

func joinURL(baseURL string, path string) string {
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/")
}

type accessSummaryResponse struct {
	InCount    int64  `json:"in_count"`
	OutCount   int64  `json:"out_count"`
	TotalCount int64  `json:"total_count"`
	Timestamp  string `json:"timestamp"`
}

type fireAlertsResponse struct {
	Alerts []fireAlertResponse `json:"alerts"`
}

type fireAlertResponse struct {
	AlertID   string `json:"alert_id"`
	Level     string `json:"level"`
	Location  string `json:"location"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}
