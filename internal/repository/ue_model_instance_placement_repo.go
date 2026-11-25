package repository

import (
	"context"

	"dcx-gateway/internal/model/entity"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UEModelInstancePlacementRepository interface {
	ListByInstanceIDs(ctx context.Context, ids []string) ([]entity.UEModelInstancePlacement, error)
	UpdateManyAtomic(ctx context.Context, ids []string, mutate func(existing map[string]*entity.UEModelInstancePlacement) ([]entity.UEModelInstancePlacement, error)) error
	DeleteByInstanceIDsAtomic(ctx context.Context, ids []string) error
}

type ueModelInstancePlacementRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewUEModelInstancePlacementRepository(db *gorm.DB, log *zap.Logger) UEModelInstancePlacementRepository {
	return &ueModelInstancePlacementRepository{db: db, log: log}
}

func (r *ueModelInstancePlacementRepository) ListByInstanceIDs(ctx context.Context, ids []string) ([]entity.UEModelInstancePlacement, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var models []entity.UEModelInstancePlacement
	err := r.db.WithContext(ctx).
		Where("instance_id IN ?", ids).
		Find(&models).Error
	if err != nil {
		r.log.Error("query ue model placements by ids failed", zap.Error(err))
		return nil, err
	}
	return models, nil
}

func (r *ueModelInstancePlacementRepository) UpdateManyAtomic(
	ctx context.Context,
	ids []string,
	mutate func(existing map[string]*entity.UEModelInstancePlacement) ([]entity.UEModelInstancePlacement, error),
) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureDefaultPlacementRows(tx, ids); err != nil {
			r.log.Error("ensure default ue model placements failed", zap.Error(err))
			return err
		}

		var models []entity.UEModelInstancePlacement
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("instance_id IN ?", ids).
			Find(&models).Error; err != nil {
			r.log.Error("query ue model placements for update failed", zap.Error(err))
			return err
		}
		locked := make(map[string]*entity.UEModelInstancePlacement, len(models))
		for i := range models {
			locked[models[i].InstanceID] = &models[i]
		}
		next, err := mutate(locked)
		if err != nil {
			return err
		}
		for _, model := range next {
			if err := tx.Table(entity.UEModelInstancePlacement{}.TableName()).Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "instance_id"}},
				DoUpdates: clause.Assignments(placementUpdateMap(model)),
			}).Create(placementInsertMap(model)).Error; err != nil {
				r.log.Error("upsert ue model placement failed", zap.String("instance_id", model.InstanceID), zap.Error(err))
				return err
			}
		}
		return nil
	})
}

func ensureDefaultPlacementRows(tx *gorm.DB, ids []string) error {
	for _, id := range ids {
		model := defaultPlacementRow(id)
		if err := tx.Table(entity.UEModelInstancePlacement{}.TableName()).
			Clauses(defaultPlacementConflictClause()).
			Create(placementInsertMap(model)).Error; err != nil {
			return err
		}
	}
	return nil
}

func defaultPlacementConflictClause() clause.OnConflict {
	return clause.OnConflict{
		Columns: []clause.Column{{Name: "instance_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"instance_id": gorm.Expr("instance_id"),
		}),
	}
}

func defaultPlacementRow(instanceID string) entity.UEModelInstancePlacement {
	return entity.UEModelInstancePlacement{
		InstanceID: instanceID,
		Visible:    true,
		ScaleX:     1,
		ScaleY:     1,
		ScaleZ:     1,
	}
}

func placementInsertMap(model entity.UEModelInstancePlacement) map[string]any {
	values := placementUpdateMap(model)
	values["instance_id"] = model.InstanceID
	return values
}

func placementUpdateMap(model entity.UEModelInstancePlacement) map[string]any {
	return map[string]any{
		"visible":        model.Visible,
		"position_x":     model.PositionX,
		"position_y":     model.PositionY,
		"position_z":     model.PositionZ,
		"rotation_yaw":   model.RotationYaw,
		"rotation_pitch": model.RotationPitch,
		"rotation_roll":  model.RotationRoll,
		"scale_x":        model.ScaleX,
		"scale_y":        model.ScaleY,
		"scale_z":        model.ScaleZ,
	}
}

func (r *ueModelInstancePlacementRepository) DeleteByInstanceIDsAtomic(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("instance_id IN ?", ids).Delete(&entity.UEModelInstancePlacement{}).Error; err != nil {
			r.log.Error("delete ue model placements failed", zap.Error(err))
			return err
		}
		return nil
	})
}
