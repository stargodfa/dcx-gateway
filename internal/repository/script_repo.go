package repository

import (
	"context"
	"dcx-gateway/internal/model/entity"
	"errors"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ScriptRepository interface {
	FindByScriptID(ctx context.Context, scriptID string) (*entity.Script, error)
	ListEnabled(ctx context.Context) ([]entity.Script, error)
}

type scriptRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewScriptRepository(db *gorm.DB, log *zap.Logger) ScriptRepository {
	return &scriptRepository{db: db, log: log}
}

func (r *scriptRepository) FindByScriptID(ctx context.Context, scriptID string) (*entity.Script, error) {
	var script entity.Script

	err := r.db.WithContext(ctx).
		Where("script_id = ?", scriptID).
		First(&script).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Warn("script not found", zap.String("script_id", scriptID))
			return nil, ErrNotFound
		}
		r.log.Error("query script failed",
			zap.String("script_id", scriptID),
			zap.Error(err),
		)
		return nil, err
	}
	return &script, nil
}

func (r *scriptRepository) ListEnabled(ctx context.Context) ([]entity.Script, error) {
	var scripts []entity.Script

	err := r.db.WithContext(ctx).
		Where("status = ?", "enabled").
		Order("id ASC").
		Find(&scripts).Error
	if err != nil {
		r.log.Error("query enabled scripts failed", zap.Error(err))
		return nil, err
	}

	return scripts, nil
}
