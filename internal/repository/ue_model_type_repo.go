package repository

import (
	"context"
	"errors"

	"dcx-gateway/internal/model/entity"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UEModelTypeRepository interface {
	ListEnabled(ctx context.Context) ([]entity.UEModelType, error)
	ListByModelIDs(ctx context.Context, ids []string) ([]entity.UEModelType, error)
	CreateManyAtomic(ctx context.Context, ids []string, mutate func(existing map[string]*entity.UEModelType) ([]entity.UEModelType, error)) error
	UpdateManyAtomic(ctx context.Context, ids []string, mutate func(existing map[string]*entity.UEModelType) ([]entity.UEModelType, error)) error
}

type ueModelTypeRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewUEModelTypeRepository(db *gorm.DB, log *zap.Logger) UEModelTypeRepository {
	return &ueModelTypeRepository{db: db, log: log}
}

func (r *ueModelTypeRepository) ListEnabled(ctx context.Context) ([]entity.UEModelType, error) {
	var models []entity.UEModelType
	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Order("model_id ASC").
		Find(&models).Error
	if err != nil {
		r.log.Error("query enabled ue model types failed", zap.Error(err))
		return nil, err
	}
	return models, nil
}

func (r *ueModelTypeRepository) ListByModelIDs(ctx context.Context, ids []string) ([]entity.UEModelType, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var models []entity.UEModelType
	err := r.db.WithContext(ctx).
		Where("model_id IN ?", ids).
		Find(&models).Error
	if err != nil {
		r.log.Error("query ue model types by ids failed", zap.Error(err))
		return nil, err
	}
	return models, nil
}

func (r *ueModelTypeRepository) CreateManyAtomic(
	ctx context.Context,
	ids []string,
	mutate func(existing map[string]*entity.UEModelType) ([]entity.UEModelType, error),
) error {
	return r.withLockedModels(ctx, ids, mutate, func(tx *gorm.DB, models []entity.UEModelType) error {
		if len(models) == 0 {
			return nil
		}
		if err := tx.Create(&models).Error; err != nil {
			r.log.Error("create ue model types failed", zap.Error(err))
			return err
		}
		return nil
	})
}

func (r *ueModelTypeRepository) UpdateManyAtomic(
	ctx context.Context,
	ids []string,
	mutate func(existing map[string]*entity.UEModelType) ([]entity.UEModelType, error),
) error {
	return r.withLockedModels(ctx, ids, mutate, func(tx *gorm.DB, models []entity.UEModelType) error {
		for _, model := range models {
			if err := tx.Save(&model).Error; err != nil {
				r.log.Error("save ue model type failed", zap.String("model_id", model.ModelID), zap.Error(err))
				return err
			}
		}
		return nil
	})
}

func (r *ueModelTypeRepository) withLockedModels(
	ctx context.Context,
	ids []string,
	mutate func(existing map[string]*entity.UEModelType) ([]entity.UEModelType, error),
	apply func(tx *gorm.DB, models []entity.UEModelType) error,
) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var models []entity.UEModelType
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("model_id IN ?", ids).
			Find(&models).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			r.log.Error("query ue model types for update failed", zap.Error(err))
			return err
		}
		locked := make(map[string]*entity.UEModelType, len(models))
		for i := range models {
			locked[models[i].ModelID] = &models[i]
		}
		next, err := mutate(locked)
		if err != nil {
			return err
		}
		return apply(tx, next)
	})
}
