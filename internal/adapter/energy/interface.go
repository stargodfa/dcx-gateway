package energy

import (
	"context"

	"dcx-gateway/internal/model/domain"
)

type Adapter interface {
	GetEnergySnapshot(ctx context.Context) (*domain.EnergySnapshot, error)
}
