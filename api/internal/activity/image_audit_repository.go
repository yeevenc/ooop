package activity

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type ImageAuditRepository interface {
	FindByID(ctx context.Context, id int64) (Activity, error)
	ClaimImageAuditTasks(ctx context.Context, now time.Time, limit int) ([]ImageAuditTask, error)
	SaveImageAuditDecision(
		ctx context.Context,
		taskID int64,
		attempts int,
		decision string,
		resultJSON string,
		rejectReason string,
	) error
	MarkImageAuditNotificationDone(ctx context.Context, taskID int64) error
	MarkImageAuditCompleted(ctx context.Context, taskID int64, status string, attempts int) error
	MarkImageAuditRetry(
		ctx context.Context,
		taskID int64,
		attempts int,
		nextRetryAt time.Time,
		reason string,
	) error
	RecoverStaleImageAuditTasks(ctx context.Context, before time.Time) error
}

func (r *GormRepository) ClaimImageAuditTasks(
	ctx context.Context,
	now time.Time,
	limit int,
) ([]ImageAuditTask, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	var candidates []ImageAuditTask
	err := r.db.WithContext(ctx).
		Where("status = ? AND next_retry_at <= ?", ImageAuditTaskPending, now).
		Order("next_retry_at ASC, id ASC").
		Limit(limit).
		Find(&candidates).Error
	if err != nil {
		return nil, err
	}

	claimed := make([]ImageAuditTask, 0, len(candidates))
	for _, task := range candidates {
		result := r.db.WithContext(ctx).
			Model(&ImageAuditTask{}).
			Where("id = ? AND status = ?", task.ID, ImageAuditTaskPending).
			Updates(map[string]interface{}{
				"status":    ImageAuditTaskProcessing,
				"locked_at": now,
			})
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected != 1 {
			continue
		}
		task.Status = ImageAuditTaskProcessing
		task.LockedAt = &now
		claimed = append(claimed, task)
	}
	return claimed, nil
}

func (r *GormRepository) SaveImageAuditDecision(
	ctx context.Context,
	taskID int64,
	attempts int,
	decision string,
	resultJSON string,
	rejectReason string,
) error {
	return r.updateImageAuditTask(ctx, taskID, map[string]interface{}{
		"decision":      decision,
		"result_json":   resultJSON,
		"reject_reason": rejectReason,
		"attempts":      attempts,
		"last_error":    "",
	})
}

func (r *GormRepository) MarkImageAuditNotificationDone(ctx context.Context, taskID int64) error {
	return r.updateImageAuditTask(ctx, taskID, map[string]interface{}{
		"notification_done": true,
		"last_error":        "",
	})
}

func (r *GormRepository) MarkImageAuditCompleted(
	ctx context.Context,
	taskID int64,
	status string,
	attempts int,
) error {
	now := time.Now()
	return r.updateImageAuditTask(ctx, taskID, map[string]interface{}{
		"status":       status,
		"attempts":     attempts,
		"locked_at":    nil,
		"last_error":   "",
		"completed_at": now,
	})
}

func (r *GormRepository) MarkImageAuditRetry(
	ctx context.Context,
	taskID int64,
	attempts int,
	nextRetryAt time.Time,
	reason string,
) error {
	return r.updateImageAuditTask(ctx, taskID, map[string]interface{}{
		"status":        ImageAuditTaskPending,
		"attempts":      attempts,
		"next_retry_at": nextRetryAt,
		"locked_at":     nil,
		"last_error":    truncateImageAuditError(reason),
	})
}

func (r *GormRepository) RecoverStaleImageAuditTasks(ctx context.Context, before time.Time) error {
	return r.db.WithContext(ctx).
		Model(&ImageAuditTask{}).
		Where("status = ? AND locked_at < ?", ImageAuditTaskProcessing, before).
		Updates(map[string]interface{}{
			"status":        ImageAuditTaskPending,
			"next_retry_at": time.Now(),
			"locked_at":     nil,
		}).Error
}

func (r *GormRepository) updateImageAuditTask(
	ctx context.Context,
	taskID int64,
	fields map[string]interface{},
) error {
	result := r.db.WithContext(ctx).
		Model(&ImageAuditTask{}).
		Where("id = ?", taskID).
		Updates(fields)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func truncateImageAuditError(value string) string {
	const maxLength = 500
	runes := []rune(value)
	if len(runes) <= maxLength {
		return value
	}
	return string(runes[:maxLength])
}
