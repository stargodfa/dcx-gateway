package education

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

func (a *MockAdapter) GetClassroomUsage(ctx context.Context) (*domain.EducationSnapshot, error) {
	now := time.Now().Format(time.RFC3339)
	return &domain.EducationSnapshot{
		Classrooms: []domain.ClassroomUsage{
			{ClassroomID: "A101", Name: "A101", Status: "occupied", CourseName: "高等数学", TeacherName: "演示教师A", Timestamp: now},
			{ClassroomID: "A102", Name: "A102", Status: "free", Timestamp: now},
			{ClassroomID: "B201", Name: "B201", Status: "occupied", CourseName: "大学英语", TeacherName: "演示教师B", Timestamp: now},
			{ClassroomID: "B305", Name: "B305", Status: "unknown", Timestamp: now},
		},
	}, nil
}
