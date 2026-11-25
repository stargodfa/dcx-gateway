package education

import (
	"context"

	"dcx-gateway/internal/model/domain"
)

type Adapter interface {
	GetClassroomUsage(ctx context.Context) (*domain.EducationSnapshot, error)
}
