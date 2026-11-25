package security

import (
	"context"
	"dcx-gateway/internal/model/domain"
)

type Adapter interface {
	GetAccessSummary(ctx context.Context) (*domain.AccessSummary, error) //今日出入人数
	GetFireAlerts(ctx context.Context) ([]domain.FireAlert, error)       //消防告警
}
