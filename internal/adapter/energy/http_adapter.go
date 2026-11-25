package energy

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
	client       *http.Client
	baseURL      string
	snapshotPath string
	auth         config.OutboundAuthConfig
	log          *zap.Logger
}

func NewHTTPAdapter(baseURL string, timeout time.Duration, snapshotPath string, log *zap.Logger, auth ...config.OutboundAuthConfig) Adapter {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	if strings.TrimSpace(snapshotPath) == "" {
		snapshotPath = "/api/energy/snapshot"
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &HTTPAdapter{
		client:       &http.Client{Timeout: timeout},
		baseURL:      strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		snapshotPath: snapshotPath,
		auth:         firstAuth(auth),
		log:          log,
	}
}

func (a *HTTPAdapter) GetEnergySnapshot(ctx context.Context) (*domain.EnergySnapshot, error) {
	var payload energySnapshotResponse
	if err := a.getJSON(ctx, a.snapshotPath, &payload); err != nil {
		return nil, err
	}
	if payload.Buildings == nil {
		return nil, fmt.Errorf("energy snapshot response missing buildings")
	}
	buildings := make([]domain.BuildingEnergyMetric, 0, len(payload.Buildings))
	for _, building := range payload.Buildings {
		if building.BuildingID == "" {
			return nil, fmt.Errorf("energy building missing building_id")
		}
		if building.Unit == "" {
			building.Unit = "kWh"
		}
		timestamp := building.Timestamp
		if timestamp == "" {
			timestamp = payload.Timestamp
		}
		if timestamp == "" {
			timestamp = time.Now().Format(time.RFC3339)
		}
		buildings = append(buildings, domain.BuildingEnergyMetric{
			BuildingID: building.BuildingID,
			Name:       building.BuildingName,
			Value:      building.Value,
			Unit:       building.Unit,
			Timestamp:  timestamp,
		})
	}
	return &domain.EnergySnapshot{Buildings: buildings}, nil
}

func (a *HTTPAdapter) getJSON(ctx context.Context, path string, out any) error {
	if a.baseURL == "" {
		return fmt.Errorf("energy http adapter base_url is empty")
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
		return fmt.Errorf("energy http adapter returned status %d", resp.StatusCode)
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

type energySnapshotResponse struct {
	Buildings []energyBuildingResponse `json:"buildings"`
	Timestamp string                   `json:"timestamp"`
}

type energyBuildingResponse struct {
	BuildingID   string  `json:"building_id"`
	BuildingName string  `json:"building_name"`
	Value        float64 `json:"value"`
	Unit         string  `json:"unit"`
	Timestamp    string  `json:"timestamp"`
}
