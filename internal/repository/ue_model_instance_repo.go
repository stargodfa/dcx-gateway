package repository

import (
	"context"

	"dcx-gateway/internal/model/entity"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UEModelInstanceRepository interface {
	ListEnabled(ctx context.Context) ([]entity.UEModelInstance, error)
	ListByInstanceIDs(ctx context.Context, ids []string) ([]entity.UEModelInstance, error)
	CountEnabledByModelIDs(ctx context.Context, modelIDs []string) (map[string]int64, error)
	CreateManyAtomic(ctx context.Context, ids []string, mutate func(existing map[string]*entity.UEModelInstance) ([]entity.UEModelInstance, []entity.UEModelInstancePlacement, error)) error
	UpdateManyAtomic(ctx context.Context, ids []string, mutate func(existing map[string]*entity.UEModelInstance) ([]entity.UEModelInstance, error)) error
}

type ueModelInstanceRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewUEModelInstanceRepository(db *gorm.DB, log *zap.Logger) UEModelInstanceRepository {
	return &ueModelInstanceRepository{db: db, log: log}
}

func (r *ueModelInstanceRepository) ListEnabled(ctx context.Context) ([]entity.UEModelInstance, error) {
	var models []entity.UEModelInstance
	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Order("sort_order ASC, instance_id ASC").
		Find(&models).Error
	if err != nil {
		r.log.Error("query enabled ue model instances failed", zap.Error(err))
		return nil, err
	}
	return models, nil
}

func (r *ueModelInstanceRepository) ListByInstanceIDs(ctx context.Context, ids []string) ([]entity.UEModelInstance, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var models []entity.UEModelInstance
	err := r.db.WithContext(ctx).
		Where("instance_id IN ?", ids).
		Find(&models).Error
	if err != nil {
		r.log.Error("query ue model instances by ids failed", zap.Error(err))
		return nil, err
	}
	return models, nil
}

func (r *ueModelInstanceRepository) CountEnabledByModelIDs(ctx context.Context, modelIDs []string) (map[string]int64, error) {
	out := make(map[string]int64, len(modelIDs))
	if len(modelIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		ModelID string
		Count   int64
	}
	err := r.db.WithContext(ctx).
		Model(&entity.UEModelInstance{}).
		Select("model_id, COUNT(*) AS count").
		Where("model_id IN ? AND enabled = ?", modelIDs, true).
		Group("model_id").
		Scan(&rows).Error
	if err != nil {
		r.log.Error("count enabled ue model instances by model ids failed", zap.Error(err))
		return nil, err
	}
	for _, row := range rows {
		out[row.ModelID] = row.Count
	}
	return out, nil
}

func (r *ueModelInstanceRepository) CreateManyAtomic(
	ctx context.Context,
	ids []string,
	mutate func(existing map[string]*entity.UEModelInstance) ([]entity.UEModelInstance, []entity.UEModelInstancePlacement, error),
) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		existing, err := r.lockInstances(tx, ids)
		if err != nil {
			return err
		}
		instances, placements, err := mutate(existing)
		if err != nil {
			return err
		}
		if len(instances) > 0 {
			if err := tx.Create(&instances).Error; err != nil {
				r.log.Error("create ue model instances failed", zap.Error(err))
				return err
			}
		}
		if len(placements) > 0 {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&placements).Error; err != nil {
				r.log.Error("create default ue model placements failed", zap.Error(err))
				return err
			}
		}
		return nil
	})
}

func (r *ueModelInstanceRepository) UpdateManyAtomic(
	ctx context.Context,
	ids []string,
	mutate func(existing map[string]*entity.UEModelInstance) ([]entity.UEModelInstance, error),
) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		existing, err := r.lockInstances(tx, ids)
		if err != nil {
			return err
		}
		next, err := mutate(existing)
		if err != nil {
			return err
		}
		for _, model := range next {
			if err := tx.Save(&model).Error; err != nil {
				r.log.Error("save ue model instance failed", zap.String("instance_id", model.InstanceID), zap.Error(err))
				return err
			}
		}
		return nil
	})
}

func (r *ueModelInstanceRepository) lockInstances(tx *gorm.DB, ids []string) (map[string]*entity.UEModelInstance, error) {
	var models []entity.UEModelInstance
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("instance_id IN ?", ids).
		Find(&models).Error; err != nil {
		r.log.Error("query ue model instances for update failed", zap.Error(err))
		return nil, err
	}
	locked := make(map[string]*entity.UEModelInstance, len(models))
	for i := range models {
		locked[models[i].InstanceID] = &models[i]
	}
	return locked, nil
}
