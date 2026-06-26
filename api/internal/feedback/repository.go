package feedback

import (
	"context"

	"gorm.io/gorm"
)

type ListQuery struct {
	Page     int
	PageSize int
	Type     string
	Keyword  string
}

type Repository interface {
	Create(ctx context.Context, item *Feedback) error
	List(ctx context.Context, query ListQuery) ([]Feedback, int64, error)
}

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) Create(ctx context.Context, item *Feedback) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *GormRepository) List(ctx context.Context, query ListQuery) ([]Feedback, int64, error) {
	var items []Feedback
	var total int64

	db := r.db.WithContext(ctx).Model(&Feedback{})
	if query.Type != "" {
		db = db.Where("type = ?", query.Type)
	}
	if query.Keyword != "" {
		keyword := "%" + query.Keyword + "%"
		db = db.Where("content LIKE ? OR user_phone LIKE ? OR user_nickname LIKE ?", keyword, keyword, keyword)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Order("id DESC").
		Offset((query.Page - 1) * query.PageSize).
		Limit(query.PageSize).
		Find(&items).Error
	return items, total, err
}
