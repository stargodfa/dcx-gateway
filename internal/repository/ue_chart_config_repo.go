package repository

import (
	"context"
	"errors"

	"dcx-gateway/internal/model/entity"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type UEChartConfigRepository interface {
	GetSettings(ctx context.Context) (*entity.UEChartConfigSettings, error)
	ListEnabledCharts(ctx context.Context) ([]entity.UEChartConfig, error)
	FindByChartID(ctx context.Context, chartID string) (*entity.UEChartConfig, error)
}

type ueChartConfigRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewUEChartConfigRepository(db *gorm.DB, log *zap.Logger) UEChartConfigRepository {
	return &ueChartConfigRepository{db: db, log: log}
}

func (r *ueChartConfigRepository) GetSettings(ctx context.Context) (*entity.UEChartConfigSettings, error) {
	var settings entity.UEChartConfigSettings
	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Order("id ASC").
		First(&settings).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Warn("ue chart config settings not found")
			return nil, ErrNotFound
		}
		r.log.Error("query ue chart config settings failed", zap.Error(err))
		return nil, err
	}
	return &settings, nil
}

func (r *ueChartConfigRepository) ListEnabledCharts(ctx context.Context) ([]entity.UEChartConfig, error) {
	var charts []entity.UEChartConfig
	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Order("sort_order ASC, id ASC").
		Find(&charts).Error
	if err != nil {
		r.log.Error("query enabled ue chart configs failed", zap.Error(err))
		return nil, err
	}
	return charts, nil
}

func (r *ueChartConfigRepository) FindByChartID(ctx context.Context, chartID string) (*entity.UEChartConfig, error) {
	var chart entity.UEChartConfig
	err := r.db.WithContext(ctx).
		Where("chart_id = ? AND enabled = ?", chartID, true).
		First(&chart).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Warn("ue chart config not found", zap.String("chart_id", chartID))
			return nil, ErrNotFound
		}
		r.log.Error("query ue chart config failed", zap.String("chart_id", chartID), zap.Error(err))
		return nil, err
	}
	return &chart, nil
}
