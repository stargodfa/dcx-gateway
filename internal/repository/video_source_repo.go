package repository

import (
	"context"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"dcx-gateway/internal/model/entity"

	"errors"
)

type VideoSourceRepository interface {
	FindByVideoID(ctx context.Context, videoID string) (*entity.VideoSource, error)
	ListEnabled(ctx context.Context) ([]entity.VideoSource, error)
	ListByVideoIDs(ctx context.Context, videoIDs []string) ([]entity.VideoSource, error)
}

type videoSourceRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewVideoSourceRepository(db *gorm.DB, log *zap.Logger) VideoSourceRepository {
	return &videoSourceRepository{db: db, log: log}
}

func (r *videoSourceRepository) FindByVideoID(ctx context.Context, videoID string) (*entity.VideoSource, error) {
	var video entity.VideoSource

	err := r.db.WithContext(ctx).
		Where("video_id = ? AND enabled = ?", videoID, true).
		First(&video).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Warn("video source not found", zap.String("video_id", videoID))
			return nil, ErrNotFound
		}
		r.log.Error("query video source failed",
			zap.String("video_id", videoID),
			zap.Error(err))
		return nil, err
	}

	return &video, nil
}

func (r *videoSourceRepository) ListEnabled(ctx context.Context) ([]entity.VideoSource, error) {
	var videos []entity.VideoSource

	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Order("id ASC").
		Find(&videos).Error

	if err != nil {
		r.log.Error("query video source failed",
			zap.Error(err))
		return nil, err
	}

	return videos, nil
}

func (r *videoSourceRepository) ListByVideoIDs(ctx context.Context, videoIDs []string) ([]entity.VideoSource, error) {
	if len(videoIDs) == 0 {
		return []entity.VideoSource{}, nil
	}

	var videos []entity.VideoSource

	err := r.db.WithContext(ctx).
		Where("video_id IN ? AND enabled = ?", videoIDs, true).
		Find(&videos).Error

	if err != nil {
		r.log.Error("query video source failed",
			zap.Strings("videoIDs", videoIDs),
			zap.Error(err))
		return nil, err
	}

	return videos, nil
}
