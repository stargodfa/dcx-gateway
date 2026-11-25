package repository

import (
	"context"
	"dcx-gateway/internal/model/entity"
	"errors"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type DataBindingRepository interface {
	FindByDataKey(ctx context.Context, dataKey string) (*entity.DataBinding, error)
	ListByDataKeys(ctx context.Context, dataKeys []string) ([]entity.DataBinding, error)
	ListEnabled(ctx context.Context) ([]entity.DataBinding, error)
}

type dataBindingRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewDataBindingRepository(db *gorm.DB, log *zap.Logger) DataBindingRepository {
	return &dataBindingRepository{db: db, log: log}
}

func (r *dataBindingRepository) FindByDataKey(ctx context.Context, dataKey string) (*entity.DataBinding, error) {
	var dataBind entity.DataBinding
	err := r.db.WithContext(ctx).
		Where("data_key = ? AND enabled = ?", dataKey, true).
		First(&dataBind).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Warn("data binding not found", zap.String("data_key", dataKey))
			return nil, ErrNotFound
		}

		r.log.Error("query data binding failed", zap.String("data_key", dataKey), zap.Error(err))
		return nil, err
	}

	return &dataBind, err
}

func (r *dataBindingRepository) ListByDataKeys(ctx context.Context, dataKeys []string) ([]entity.DataBinding, error) {
	if len(dataKeys) == 0 {
		return []entity.DataBinding{}, nil
	}
	var dataBindings []entity.DataBinding

	err := r.db.WithContext(ctx).
		Where("data_key IN ? AND enabled = ?", dataKeys, true).
		Find(&dataBindings).Error
	if err != nil {

		r.log.Error("query databinding list failed",
			zap.Strings("data_key", dataKeys),
			zap.Error(err))
		return nil, err
	}

	return dataBindings, err

}

func (r *dataBindingRepository) ListEnabled(ctx context.Context) ([]entity.DataBinding, error) {
	var dataBindings []entity.DataBinding

	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Order("id ASC").
		Find(&dataBindings).Error
	if err != nil {
		r.log.Error("query enabled data bindings failed", zap.Error(err))
		return nil, err
	}

	return dataBindings, nil
}
