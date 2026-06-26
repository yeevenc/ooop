package message

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type UserMessageQuery struct {
	UserID   int64
	Page     int
	PageSize int
}

type Repository interface {
	Create(ctx context.Context, item *UserMessage) error
	ListByUser(ctx context.Context, query UserMessageQuery) ([]UserMessage, error)
	MarkRead(ctx context.Context, userID int64, id int64, readAt time.Time) error
}

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) Create(ctx context.Context, item *UserMessage) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *GormRepository) ListByUser(ctx context.Context, query UserMessageQuery) ([]UserMessage, error) {
	var items []UserMessage
	db := r.db.WithContext(ctx).
		Model(&UserMessage{}).
		Where("user_id = ?", query.UserID)

	err := paginate(db, query.Page, query.PageSize).
		Order("created_at DESC").
		Find(&items).Error
	return items, err
}

func (r *GormRepository) MarkRead(ctx context.Context, userID int64, id int64, readAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&UserMessage{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("read_at", readAt)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func paginate(db *gorm.DB, page int, pageSize int) *gorm.DB {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	return db.Offset((page - 1) * pageSize).Limit(pageSize)
}
