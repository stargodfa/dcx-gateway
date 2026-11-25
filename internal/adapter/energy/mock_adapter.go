package energy

import (
	"context"
	"time"

	"dcx-gateway/internal/model/domain"

	"go.uber.org/zap"
)

type MockAdapter struct {
	log *zap.Logger
}

func NewMockAdapter(log *zap.Logger) Adapter {
	return &MockAdapter{log: log}
}

func (a *MockAdapter) GetEnergySnapshot(ctx context.Context) (*domain.EnergySnapshot, error) {
	now := time.Now().Format(time.RFC3339)
	return &domain.EnergySnapshot{
		Buildings: []domain.BuildingEnergyMetric{
			{BuildingID: "library", Name: "图书馆", Value: 1280.5, Unit: "kWh", Timestamp: now},
			{BuildingID: "teaching_a", Name: "教学楼A", Value: 2460.0, Unit: "kWh", Timestamp: now},
			{BuildingID: "canteen", Name: "食堂", Value: 980.8, Unit: "kWh", Timestamp: now},
			{BuildingID: "dormitory", Name: "宿舍区", Value: 3120.2, Unit: "kWh", Timestamp: now},
		},
	}, nil
}
