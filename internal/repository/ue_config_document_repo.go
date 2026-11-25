package repository

import (
	"context"
	"errors"

	"dcx-gateway/internal/model/entity"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UEConfigDocumentRepository interface {
	FindByKey(ctx context.Context, key string) (*entity.UEConfigDocument, error)
	ListEnabled(ctx context.Context) ([]entity.UEConfigDocument, error)
	UpdateContentJSONAtomic(ctx context.Context, key string, mutate func(raw []byte) ([]byte, error)) ([]byte, error)
	ReplaceContentJSONAtomic(ctx context.Context, key string, fileName string, raw []byte) ([]byte, error)
}

type ueConfigDocumentRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewUEConfigDocumentRepository(db *gorm.DB, log *zap.Logger) UEConfigDocumentRepository {
	return &ueConfigDocumentRepository{db: db, log: log}
}

func (r *ueConfigDocumentRepository) FindByKey(ctx context.Context, key string) (*entity.UEConfigDocument, error) {
	var doc entity.UEConfigDocument
	err := r.db.WithContext(ctx).
		Where("config_key = ? AND enabled = ?", key, true).
		First(&doc).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Warn("ue config document not found", zap.String("config_key", key))
			return nil, ErrNotFound
		}
		r.log.Error("query ue config document failed", zap.String("config_key", key), zap.Error(err))
		return nil, err
	}

	return &doc, nil
}

func (r *ueConfigDocumentRepository) ListEnabled(ctx context.Context) ([]entity.UEConfigDocument, error) {
	var docs []entity.UEConfigDocument
	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Order("config_key ASC").
		Find(&docs).Error
	if err != nil {
		r.log.Error("query enabled ue config documents failed", zap.Error(err))
		return nil, err
	}

	return docs, nil
}

func (r *ueConfigDocumentRepository) UpdateContentJSONAtomic(
	ctx context.Context,
	key string,
	mutate func(raw []byte) ([]byte, error),
) ([]byte, error) {
	var newRaw []byte
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var doc entity.UEConfigDocument
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("config_key = ? AND enabled = ?", key, true).
			First(&doc).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				r.log.Warn("ue config document not found for update", zap.String("config_key", key))
				return ErrNotFound
			}
			r.log.Error("query ue config document for update failed", zap.String("config_key", key), zap.Error(err))
			return err
		}

		next, err := mutate([]byte(doc.ContentJSON))
		if err != nil {
			return err
		}
		if err := tx.Model(&doc).Update("content_json", string(next)).Error; err != nil {
			r.log.Error("update ue config document content json failed", zap.String("config_key", key), zap.Error(err))
			return err
		}
		newRaw = next
		return nil
	})
	if err != nil {
		return nil, err
	}
	return newRaw, nil
}

func (r *ueConfigDocumentRepository) ReplaceContentJSONAtomic(
	ctx context.Context,
	key string,
	fileName string,
	raw []byte,
) ([]byte, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var doc entity.UEConfigDocument
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("config_key = ?", key).
			First(&doc).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				doc = entity.UEConfigDocument{
					ConfigKey:   key,
					FileName:    fileName,
					ContentJSON: string(raw),
					Source:      "gateway_preset",
					Enabled:     true,
					Version:     "v1",
				}
				if err := tx.Create(&doc).Error; err != nil {
					r.log.Error("create ue config document failed", zap.String("config_key", key), zap.Error(err))
					return err
				}
				return nil
			}
			r.log.Error("query ue config document for replace failed", zap.String("config_key", key), zap.Error(err))
			return err
		}

		updates := map[string]any{
			"file_name":    fileName,
			"content_json": string(raw),
			"source":       "gateway_preset",
			"enabled":      true,
			"version":      "v1",
		}
		if err := tx.Model(&doc).Updates(updates).Error; err != nil {
			r.log.Error("replace ue config document content json failed", zap.String("config_key", key), zap.Error(err))
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return raw, nil
}
