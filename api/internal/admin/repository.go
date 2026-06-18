package admin

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("数据不存在")

type Repository interface {
	FindByUsername(ctx context.Context, username string) (AdminUser, error)
	Create(ctx context.Context, item *AdminUser) error
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

func (r *GormRepository) Create(ctx context.Context, item *AdminUser) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func normalizeNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}
