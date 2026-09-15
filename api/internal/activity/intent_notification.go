package activity

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"ooop-admin-api/internal/logger"
	"ooop-admin-api/internal/user"
	"time"
)

type IntentNotification struct {
	ID            int64  `gorm:"primaryKey;autoIncrement"`
	ActivityID    int64  `gorm:"not null;uniqueIndex:uniq_intent_notice"`
	UserID        int64  `gorm:"not null;uniqueIndex:uniq_intent_notice"`
	Status        string `gorm:"size:20;not null;index"`
	Attempts      int
	NextAttemptAt time.Time `gorm:"index"`
	UpdatedAt     time.Time
	CreatedAt     time.Time
	LastError     string `gorm:"size:500"`
}
type IntentNotifier interface {
	NotifyActivityIntent(context.Context, int64, int64, string, string) error
}

// 持久化匹配记录，审核通过后才通知，重启和重复审核不会重复建任务。
func (r *IntentStore) prepareNotifications(ctx context.Context) error {
	var activities []Activity
	if err := r.db.WithContext(ctx).Where("is_official = ? AND status = ? AND intent_notifications_prepared = ? AND reviewed_at IS NOT NULL", true, StatusOngoing, false).Order("id").Limit(20).Find(&activities).Error; err != nil {
		return err
	}
	for _, a := range activities {
		err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var current Activity
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, a.ID).Error; err != nil {
				return err
			}
			if current.IntentNotificationsPrepared || current.Status != StatusOngoing {
				return nil
			}
			// 数据库内生成任务，按活动和用户唯一去重；只匹配发布前已经登记的类别。
			if current.ActivityDate == nil || current.ActivityDate.After(time.Now()) {
				if err := tx.Exec(`INSERT INTO intent_notifications (activity_id,user_id,status,attempts,next_attempt_at,created_at,updated_at,last_error)
 SELECT ?, i.user_id, 'pending', 0, NOW(), NOW(), NOW(), '' FROM activity_intents i
 JOIN activity_intent_categories c ON c.intent_id=i.id
 JOIN users u ON u.id=i.user_id
 WHERE i.active=1 AND i.notify_enabled=1 AND i.city=? AND c.category_id=? AND c.created_at<=? AND i.user_id<>? AND u.status=1
 ON DUPLICATE KEY UPDATE activity_id=VALUES(activity_id)`, current.ID, current.City, current.CategoryID, current.ReviewedAt, current.UserID).Error; err != nil {
					return err
				}
			}
			return tx.Model(&current).Update("intent_notifications_prepared", true).Error
		})
		if err != nil {
			return err
		}
	}
	return nil
}
func (r *IntentStore) deliverNotification(ctx context.Context, n IntentNotifier) (bool, error) {
	var job IntentNotification
	now := time.Now()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("(status = ? AND next_attempt_at <= ?) OR (status = ? AND updated_at < ?)", "pending", now, "processing", now.Add(-2*time.Minute)).Order("id").First(&job).Error; err != nil {
			return err
		}
		job.Status = "processing"
		job.Attempts++
		return tx.Save(&job).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var item Activity
	err = r.db.WithContext(ctx).First(&item, job.ActivityID).Error
	status := "sent"
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = nil
		status = "skipped"
	} else if err == nil {
		var count int64
		err = r.db.WithContext(ctx).Table("activity_intents i").Joins("JOIN activity_intent_categories c ON c.intent_id=i.id").Joins("JOIN users u ON u.id=i.user_id").Where("i.user_id=? AND i.city=? AND c.category_id=? AND i.active=1 AND i.notify_enabled=1 AND u.status=?", job.UserID, item.City, item.CategoryID, user.UserStatusEnabled).Count(&count).Error
		if err == nil {
			if count == 0 || item.Status != StatusOngoing || !item.IsOfficial || (item.ActivityDate != nil && !item.ActivityDate.After(now)) {
				status = "skipped"
			} else {
				sendCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				err = n.NotifyActivityIntent(sendCtx, job.UserID, item.ID, item.Title, fmt.Sprintf("activity-intent:%d:%d", item.ID, job.UserID))
				cancel()
			}
		}
	}
	fields := map[string]interface{}{"status": status, "last_error": ""}
	if err != nil {
		fields["status"] = "pending"
		if job.Attempts >= 5 {
			fields["status"] = "failed"
		}
		fields["next_attempt_at"] = now.Add(time.Duration(job.Attempts) * time.Minute)
		message := []rune(err.Error())
		if len(message) > 500 {
			message = message[:500]
		}
		fields["last_error"] = string(message)
	}
	return true, r.db.WithContext(ctx).Model(&IntentNotification{}).Where("id=?", job.ID).Updates(fields).Error
}
func (r *IntentStore) StartNotifications(ctx context.Context, n IntentNotifier) {
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := r.prepareNotifications(ctx); err != nil {
					logger.Errorf("意向通知匹配失败: %v", err)
					continue
				}
				for i := 0; i < 50; i++ {
					found, err := r.deliverNotification(ctx, n)
					if err != nil {
						logger.Errorf("意向通知处理失败: %v", err)
						break
					}
					if !found {
						break
					}
				}
			}
		}
	}()
}
