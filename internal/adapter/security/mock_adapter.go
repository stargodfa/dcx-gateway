package security

import (
	"context"
	"dcx-gateway/internal/model/domain"
	"time"

	"go.uber.org/zap"
)

type MockAdapter struct {
	log *zap.Logger
}

func NewMockAdapter(log *zap.Logger) Adapter {
	return &MockAdapter{log: log}
}

func (a *MockAdapter) GetAccessSummary(ctx context.Context) (*domain.AccessSummary, error) {
	now := time.Now().Format(time.RFC3339)
	return &domain.AccessSummary{
		InCount:    1800,
		OutCount:   1480,
		TotalCount: 3280,
		Timestamp:  now,
	}, nil
}

func (a *MockAdapter) GetFireAlerts(ctx context.Context) ([]domain.FireAlert, error) {
	now := time.Now().Format(time.RFC3339)

	return []domain.FireAlert{
		{
			AlertID:   "fire_mock_001",
			Level:     "normal",
			Location:  "校门口",
			Message:   "消防状态正常",
			Timestamp: now,
		},
	}, nil
}
