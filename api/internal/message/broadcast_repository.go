package message

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BroadcastRepository interface {
	Create(ctx context.Context, item *SystemBroadcast) error
	ClaimNext(ctx context.Context, now time.Time) (*SystemBroadcast, error)
	SaveProgress(ctx context.Context, item *SystemBroadcast) error
	List(ctx context.Context, query BroadcastQuery) ([]SystemBroadcast, int64, error)
}

type GormBroadcastRepository struct {
	db *gorm.DB
}

func NewGormBroadcastRepository(db *gorm.DB) *GormBroadcastRepository {
	return &GormBroadcastRepository{db: db}
}

func (r *GormBroadcastRepository) Create(ctx context.Context, item *SystemBroadcast) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *GormBroadcastRepository) ClaimNext(ctx context.Context, now time.Time) (*SystemBroadcast, error) {
	var item SystemBroadcast
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("status IN ?", []string{BroadcastStatusPending, BroadcastStatusRunning}).
			Order("id ASC").
			First(&item).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if item.Status != BroadcastStatusPending {
			return nil
		}
		startedAt := now
		item.Status = BroadcastStatusRunning
		item.StartedAt = &startedAt
		return tx.Model(&item).Select("status", "started_at").Updates(item).Error
	})
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *GormBroadcastRepository) SaveProgress(ctx context.Context, item *SystemBroadcast) error {
	return r.db.WithContext(ctx).Model(item).Select(
		"status",
		"target_count",
		"message_count",
		"push_success",
		"push_skipped",
		"push_failed",
		"last_user_id",
		"error_message",
		"started_at",
		"finished_at",
	).Updates(item).Error
}

func (r *GormBroadcastRepository) List(ctx context.Context, query BroadcastQuery) ([]SystemBroadcast, int64, error) {
	var items []SystemBroadcast
	var total int64
	db := r.db.WithContext(ctx).Model(&SystemBroadcast{})
	if status := strings.TrimSpace(query.Status); status != "" {
		db = db.Where("status = ?", status)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := paginate(db, query.Page, query.PageSize).
		Order("id DESC").
		Find(&items).Error
	return items, total, err
}
