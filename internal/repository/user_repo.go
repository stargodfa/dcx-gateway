package repository

import (
	"context"
	"errors"

	"dcx-gateway/internal/model/entity"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type UserRepository interface {
	FindByUsername(ctx context.Context, username string) (*entity.UEUser, error)
	FindByID(ctx context.Context, id uint64) (*entity.UEUser, error)
}

type userRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewUserRepository(db *gorm.DB, log *zap.Logger) UserRepository {
	return &userRepository{db: db, log: log}
}

func (r *userRepository) FindByUsername(ctx context.Context, username string) (*entity.UEUser, error) {
	var user entity.UEUser
	err := r.db.WithContext(ctx).
		Where("username = ?", username).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		r.log.Error("query user by username failed", zap.String("username", username), zap.Error(err))
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByID(ctx context.Context, id uint64) (*entity.UEUser, error) {
	var user entity.UEUser
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		r.log.Error("query user by id failed", zap.Uint64("user_id", id), zap.Error(err))
		return nil, err
	}
	return &user, nil
}
