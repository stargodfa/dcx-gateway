package repository

import (
	"context"
	"database/sql"
	"errors"

	"dcx-gateway/internal/model/entity"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UESceneDesiredModelRepository interface {
	ListEnabled(ctx context.Context) ([]entity.UESceneDesiredModel, error)
	ListByInstanceIDs(ctx context.Context, ids []string) ([]entity.UESceneDesiredModel, error)
	ExistsEnabledByInstanceIDs(ctx context.Context, ids []string) (map[string]bool, error)
	MaxSortOrder(ctx context.Context) (int, error)
	UpsertMany(ctx context.Context, models []entity.UESceneDesiredModel) error
	SaveManyAtomic(ctx context.Context, models []entity.UESceneDesiredModel) error
	UpdateManyAtomic(ctx context.Context, ids []string, mutate func(models map[string]*entity.UESceneDesiredModel) ([]entity.UESceneDesiredModel, error)) error
	SoftDeleteByInstanceIDs(ctx context.Context, ids []string) error
}

type ueSceneDesiredModelRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewUESceneDesiredModelRepository(db *gorm.DB, log *zap.Logger) UESceneDesiredModelRepository {
	return &ueSceneDesiredModelRepository{db: db, log: log}
}

func (r *ueSceneDesiredModelRepository) ListEnabled(ctx context.Context) ([]entity.UESceneDesiredModel, error) {
	var models []entity.UESceneDesiredModel
	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Order("sort_order ASC, id ASC").
		Find(&models).Error
	if err != nil {
		r.log.Error("query enabled ue scene desired models failed", zap.Error(err))
		return nil, err
	}
	return models, nil
}

func (r *ueSceneDesiredModelRepository) ListByInstanceIDs(ctx context.Context, ids []string) ([]entity.UESceneDesiredModel, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var models []entity.UESceneDesiredModel
	err := r.db.WithContext(ctx).
		Where("instance_id IN ?", ids).
		Find(&models).Error
	if err != nil {
		r.log.Error("query ue scene desired models by instance ids failed", zap.Error(err))
		return nil, err
	}
	return models, nil
}

func (r *ueSceneDesiredModelRepository) ExistsEnabledByInstanceIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	out := make(map[string]bool, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	var rows []struct {
		InstanceID string
	}
	err := r.db.WithContext(ctx).
		Model(&entity.UESceneDesiredModel{}).
		Select("instance_id").
		Where("instance_id IN ? AND enabled = ?", ids, true).
		Find(&rows).Error
	if err != nil {
		r.log.Error("query ue scene desired model existence failed", zap.Error(err))
		return nil, err
	}
	for _, row := range rows {
		out[row.InstanceID] = true
	}
	return out, nil
}

func (r *ueSceneDesiredModelRepository) MaxSortOrder(ctx context.Context) (int, error) {
	var maxSort sql.NullInt64
	err := r.db.WithContext(ctx).
		Model(&entity.UESceneDesiredModel{}).
		Select("MAX(sort_order)").
		Scan(&maxSort).Error
	if err != nil {
		r.log.Error("query max ue scene desired sort order failed", zap.Error(err))
		return 0, err
	}
	if !maxSort.Valid {
		return 0, nil
	}
	return int(maxSort.Int64), nil
}

func (r *ueSceneDesiredModelRepository) UpsertMany(ctx context.Context, models []entity.UESceneDesiredModel) error {
	if len(models) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "instance_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"model_id", "uri", "format", "source_path", "display_name", "visible",
				"bounds_x", "bounds_y", "bounds_z",
				"position_x", "position_y", "position_z",
				"rotation_yaw", "rotation_pitch", "rotation_roll",
				"scale_x", "scale_y", "scale_z",
				"enabled", "sort_order", "config_json",
			}),
		}).Create(&models).Error
		if err != nil {
			r.log.Error("upsert ue scene desired models failed", zap.Error(err))
		}
		return err
	})
}

func (r *ueSceneDesiredModelRepository) UpdateManyAtomic(
	ctx context.Context,
	ids []string,
	mutate func(models map[string]*entity.UESceneDesiredModel) ([]entity.UESceneDesiredModel, error),
) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var models []entity.UESceneDesiredModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("instance_id IN ?", ids).
			Find(&models).Error; err != nil {
			r.log.Error("query ue scene desired models for update failed", zap.Error(err))
			return err
		}

		locked := make(map[string]*entity.UESceneDesiredModel, len(models))
		for i := range models {
			locked[models[i].InstanceID] = &models[i]
		}

		next, err := mutate(locked)
		if err != nil {
			return err
		}
		for _, model := range next {
			if model.ID == 0 {
				return ErrNotFound
			}
			if err := tx.Save(&model).Error; err != nil {
				r.log.Error("save locked ue scene desired model failed", zap.String("instance_id", model.InstanceID), zap.Error(err))
				return err
			}
		}
		return nil
	})
}

func (r *ueSceneDesiredModelRepository) SaveManyAtomic(ctx context.Context, models []entity.UESceneDesiredModel) error {
	if len(models) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, model := range models {
			if model.ID == 0 {
				return ErrNotFound
			}
			if err := tx.Save(&model).Error; err != nil {
				r.log.Error("save ue scene desired model failed", zap.String("instance_id", model.InstanceID), zap.Error(err))
				return err
			}
		}
		return nil
	})
}

func (r *ueSceneDesiredModelRepository) SoftDeleteByInstanceIDs(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	err := r.db.WithContext(ctx).
		Model(&entity.UESceneDesiredModel{}).
		Where("instance_id IN ?", ids).
		Update("enabled", false).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		r.log.Error("soft delete ue scene desired models failed", zap.Error(err))
		return err
	}
	return nil
}
