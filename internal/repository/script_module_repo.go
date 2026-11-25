package repository

import (
	"context"
	"dcx-gateway/internal/model/entity"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ScriptModuleRepository interface {
	ListByScriptID(ctx context.Context, scriptID string) ([]entity.ScriptModule, error)
	ListEnabled(ctx context.Context) ([]entity.ScriptModule, error)
}

type scriptModuleRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewScriptModuleRepository(db *gorm.DB, log *zap.Logger) ScriptModuleRepository {
	return &scriptModuleRepository{db: db, log: log}
}

func (r *scriptModuleRepository) ListByScriptID(ctx context.Context, scriptID string) ([]entity.ScriptModule, error) {
	var modules []entity.ScriptModule

	err := r.db.WithContext(ctx).
		Where("script_id = ? AND enabled = ?", scriptID, true).
		Order("sort_order ASC, id ASC").
		Find(&modules).Error

	if err != nil {

		r.log.Error("query scriptModule error", zap.String("script_id", scriptID), zap.Error(err))

		return nil, err
	}

	return modules, nil

}

func (r *scriptModuleRepository) ListEnabled(ctx context.Context) ([]entity.ScriptModule, error) {
	var modules []entity.ScriptModule

	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Order("script_id ASC, sort_order ASC, id ASC").
		Find(&modules).Error

	if err != nil {
		r.log.Error("query enabled script modules failed", zap.Error(err))
		return nil, err
	}

	return modules, nil
}
