package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ooop-admin-api/internal/message"
	"ooop-admin-api/internal/user"
)

type ReportRepository interface {
	CreateReport(ctx context.Context, item *ChatReport) error
	ListReports(ctx context.Context, query AdminReportQuery) ([]ChatReport, int64, error)
	FindReport(ctx context.Context, id int64) (ChatReport, error)
	ResolveReport(ctx context.Context, id int64, adminID int64, status string, result string, restrictionUntil *time.Time, handledAt time.Time) (ReportResolution, error)
}

func (r *GormRepository) CreateReport(ctx context.Context, item *ChatReport) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&ChatReport{}).
			Where("conversation_id = ? AND reporter_id = ? AND status = ?", item.ConversationID, item.ReporterID, ReportStatusPending).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrReportPending
		}
		return tx.Create(item).Error
	})
}

func (r *GormRepository) ListReports(ctx context.Context, query AdminReportQuery) ([]ChatReport, int64, error) {
	db := r.db.WithContext(ctx).Model(&ChatReport{})
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where(
			"CAST(id AS CHAR) LIKE ? OR CAST(reporter_id AS CHAR) LIKE ? OR CAST(reported_user_id AS CHAR) LIKE ? OR reason LIKE ? OR description LIKE ? OR handle_result LIKE ?",
			like,
			like,
			like,
			like,
			like,
			like,
		)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []ChatReport
	if err := paginate(db, query.Page, query.PageSize).
		Order("created_at DESC, id DESC").
		Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *GormRepository) FindReport(ctx context.Context, id int64) (ChatReport, error) {
	var item ChatReport
	err := r.db.WithContext(ctx).First(&item, id).Error
	return item, normalizeReportNotFound(err)
}

func (r *GormRepository) ResolveReport(ctx context.Context, id int64, adminID int64, status string, result string, restrictionUntil *time.Time, handledAt time.Time) (ReportResolution, error) {
	var resolution ReportResolution
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item ChatReport
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, id).Error
		if err != nil {
			return normalizeReportNotFound(err)
		}
		if item.Status != ReportStatusPending {
			return ErrReportProcessed
		}

		updates := map[string]interface{}{
			"status":            status,
			"handle_result":     result,
			"handler_admin_id":  adminID,
			"handled_at":        handledAt,
			"restriction_until": restrictionUntil,
		}
		if err := tx.Model(&ChatReport{}).Where("id = ? AND status = ?", item.ID, ReportStatusPending).Updates(updates).Error; err != nil {
			return err
		}
		if status == ReportStatusResolved && restrictionUntil != nil {
			var reportedUser user.User
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&reportedUser, item.ReportedUserID).Error; err != nil {
				return err
			}
			if reportedUser.ChatRestrictedUntil == nil || reportedUser.ChatRestrictedUntil.Before(*restrictionUntil) {
				userUpdates := map[string]interface{}{
					"chat_restriction_reason": result,
					"chat_restricted_until":   *restrictionUntil,
				}
				if err := tx.Model(&user.User{}).Where("id = ?", item.ReportedUserID).Updates(userUpdates).Error; err != nil {
					return err
				}
			}
		}

		title := "举报处理结果"
		content := fmt.Sprintf("您提交的聊天举报（编号 %d）已处理。处理结果：%s", item.ID, result)
		notification := &message.UserMessage{
			UserID:  item.ReporterID,
			Type:    message.TypeChatReport,
			Title:   title,
			Content: content,
		}
		if err := tx.Create(notification).Error; err != nil {
			return err
		}

		item.Status = status
		item.HandleResult = result
		item.HandlerAdminID = &adminID
		item.HandledAt = &handledAt
		item.RestrictionUntil = restrictionUntil
		item.UpdatedAt = handledAt
		resolution = ReportResolution{
			Report:         item,
			MessageID:      notification.ID,
			MessageTitle:   title,
			MessageContent: content,
		}
		return nil
	})
	return resolution, err
}

func normalizeReportNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrReportNotFound
	}
	return err
}
