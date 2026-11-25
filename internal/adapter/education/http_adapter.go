package education

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
	client             *http.Client
	baseURL            string
	classroomUsagePath string
	auth               config.OutboundAuthConfig
	log                *zap.Logger
}

func NewHTTPAdapter(baseURL string, timeout time.Duration, classroomUsagePath string, log *zap.Logger, auth ...config.OutboundAuthConfig) Adapter {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	if strings.TrimSpace(classroomUsagePath) == "" {
		classroomUsagePath = "/api/education/classroom-usage"
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &HTTPAdapter{
		client:             &http.Client{Timeout: timeout},
		baseURL:            strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		classroomUsagePath: classroomUsagePath,
		auth:               firstAuth(auth),
		log:                log,
	}
}

func (a *HTTPAdapter) GetClassroomUsage(ctx context.Context) (*domain.EducationSnapshot, error) {
	var payload classroomUsageResponse
	if err := a.getJSON(ctx, a.classroomUsagePath, &payload); err != nil {
		return nil, err
	}
	if payload.Classrooms == nil {
		return nil, fmt.Errorf("education classroom usage response missing classrooms")
	}
	classrooms := make([]domain.ClassroomUsage, 0, len(payload.Classrooms))
	for _, classroom := range payload.Classrooms {
		if classroom.RoomID == "" {
			return nil, fmt.Errorf("education classroom missing room_id")
		}
		if classroom.Status == "" {
			classroom.Status = "unknown"
		}
		timestamp := classroom.Timestamp
		if timestamp == "" {
			timestamp = payload.Timestamp
		}
		if timestamp == "" {
			timestamp = time.Now().Format(time.RFC3339)
		}
		classrooms = append(classrooms, domain.ClassroomUsage{
			ClassroomID: classroom.RoomID,
			Name:        classroom.RoomName,
			Status:      classroom.Status,
			CourseName:  classroom.CourseName,
			TeacherName: classroom.TeacherName,
			Timestamp:   timestamp,
		})
	}
	return &domain.EducationSnapshot{Classrooms: classrooms}, nil
}

func (a *HTTPAdapter) getJSON(ctx context.Context, path string, out any) error {
	if a.baseURL == "" {
		return fmt.Errorf("education http adapter base_url is empty")
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
		return fmt.Errorf("education http adapter returned status %d", resp.StatusCode)
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

type classroomUsageResponse struct {
	Classrooms []classroomUsageItemResponse `json:"classrooms"`
	Timestamp  string                       `json:"timestamp"`
}

type classroomUsageItemResponse struct {
	RoomID      string `json:"room_id"`
	RoomName    string `json:"room_name"`
	Status      string `json:"status"`
	CourseName  string `json:"course_name"`
	TeacherName string `json:"teacher_name"`
	Timestamp   string `json:"timestamp"`
}
