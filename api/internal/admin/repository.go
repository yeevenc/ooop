package admin

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("数据不存在")

type Repository interface {
	FindByUsername(ctx context.Context, username string) (AdminUser, error)
	FindByID(ctx context.Context, id int64) (AdminUser, error)
	Create(ctx context.Context, item *AdminUser) error
	UpdatePassword(ctx context.Context, id int64, passwordHash string) error
}

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) FindByUsername(ctx context.Context, username string) (AdminUser, error) {
	var item AdminUser
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&item).Error
	return item, normalizeNotFound(err)
}

func (r *GormRepository) FindByID(ctx context.Context, id int64) (AdminUser, error) {
	var item AdminUser
	err := r.db.WithContext(ctx).First(&item, id).Error
	return item, normalizeNotFound(err)
}

func (r *GormRepository) Create(ctx context.Context, item *AdminUser) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *GormRepository) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	result := r.db.WithContext(ctx).Model(&AdminUser{}).Where("id = ?", id).Update("password_hash", passwordHash)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func normalizeNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}
