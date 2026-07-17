package chat

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ooop-admin-api/internal/provider"
)

type CreateMessageParams struct {
	SenderID        int64
	RecipientID     int64
	ClientMessageID string
	Content         string
	CreatedAt       time.Time
	ExpiresAt       time.Time
}

type MessageQuery struct {
	ConversationID int64
	BeforeID       int64
	AfterID        int64
	PageSize       int
}

type MessageRepository interface {
	CreateMessage(ctx context.Context, params CreateMessageParams) (Conversation, Message, bool, error)
	ListConversations(ctx context.Context, userID int64, page int, pageSize int) ([]Conversation, error)
	FindConversationForUser(ctx context.Context, conversationID int64, userID int64) (Conversation, error)
	ListMessages(ctx context.Context, query MessageQuery) ([]Message, error)
	MarkRead(ctx context.Context, conversation Conversation, userID int64, lastMessageID int64) error
	CountUnread(ctx context.Context, userID int64) (int64, error)
}

type PushRepository interface {
	ClaimPushTasks(ctx context.Context, now time.Time, limit int) ([]PushTask, error)
	FindPushMessage(ctx context.Context, messageID int64) (Message, error)
	MarkPushSucceeded(ctx context.Context, taskID int64, attempts int) error
	MarkPushSkipped(ctx context.Context, taskID int64, attempts int, reason string) error
	MarkPushRetry(ctx context.Context, taskID int64, attempts int, nextRetryAt time.Time, reason string, dead bool) error
	RecoverStalePushTasks(ctx context.Context, before time.Time) error
	DeleteExpiredMessages(ctx context.Context, now time.Time, limit int) (int64, error)
	DeleteEmptyConversations(ctx context.Context) error
	DeleteExpiredPushTasks(ctx context.Context, before time.Time, limit int) (int64, error)
}

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) CreateMessage(ctx context.Context, params CreateMessageParams) (Conversation, Message, bool, error) {
	var conversation Conversation
	var message Message
	created := false

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing Message
		err := tx.Where("sender_id = ? AND client_message_id = ?", params.SenderID, params.ClientMessageID).
			First(&existing).Error
		if err == nil {
			if existing.RecipientID != params.RecipientID || existing.Content != params.Content {
				return ErrClientMessageConflict
			}
			message = existing
			return tx.First(&conversation, existing.ConversationID).Error
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		userAID, userBID := normalizeUsers(params.SenderID, params.RecipientID)
		conversation = Conversation{UserAID: userAID, UserBID: userBID}
		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_a_id"}, {Name: "user_b_id"}},
			DoNothing: true,
		}).Create(&conversation)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			if err := tx.Where("user_a_id = ? AND user_b_id = ?", userAID, userBID).
				First(&conversation).Error; err != nil {
				return err
			}
		}

		message = Message{
			ConversationID:  conversation.ID,
			SenderID:        params.SenderID,
			RecipientID:     params.RecipientID,
			ClientMessageID: params.ClientMessageID,
			Type:            MessageTypeText,
			Content:         params.Content,
			CreatedAt:       params.CreatedAt,
			ExpiresAt:       params.ExpiresAt,
		}
		result = tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "sender_id"}, {Name: "client_message_id"}},
			DoNothing: true,
		}).Create(&message)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			if err := tx.Where("sender_id = ? AND client_message_id = ?", params.SenderID, params.ClientMessageID).
				First(&message).Error; err != nil {
				return err
			}
			if message.RecipientID != params.RecipientID || message.Content != params.Content {
				return ErrClientMessageConflict
			}
			return tx.First(&conversation, message.ConversationID).Error
		}

		updates := map[string]interface{}{
			"last_message_id":      message.ID,
			"last_message_content": message.Content,
			"last_message_at":      message.CreatedAt,
		}
		if params.RecipientID == conversation.UserAID {
			updates["user_a_unread"] = gorm.Expr("user_a_unread + 1")
		} else {
			updates["user_b_unread"] = gorm.Expr("user_b_unread + 1")
		}
		if err := tx.Model(&Conversation{}).Where("id = ?", conversation.ID).Updates(updates).Error; err != nil {
			return err
		}

		tasks := []PushTask{
			newPushTask(message.ID, params.RecipientID, provider.PushChannelJiguang, params.CreatedAt),
			newPushTask(message.ID, params.RecipientID, provider.PushChannelHarmony, params.CreatedAt),
		}
		if err := tx.Create(&tasks).Error; err != nil {
			return err
		}

		conversation.LastMessageID = message.ID
		conversation.LastMessageContent = message.Content
		conversation.LastMessageAt = &message.CreatedAt
		created = true
		return nil
	})

	return conversation, message, created, err
}

func (r *GormRepository) ListConversations(ctx context.Context, userID int64, page int, pageSize int) ([]Conversation, error) {
	var items []Conversation
	err := paginate(
		r.db.WithContext(ctx).
			Where("last_message_id > 0").
			Where("user_a_id = ? OR user_b_id = ?", userID, userID),
		page,
		pageSize,
	).
		Order("last_message_at DESC, id DESC").
		Find(&items).Error
	return items, err
}

func (r *GormRepository) FindConversationForUser(ctx context.Context, conversationID int64, userID int64) (Conversation, error) {
	var item Conversation
	err := r.db.WithContext(ctx).
		Where("id = ? AND (user_a_id = ? OR user_b_id = ?)", conversationID, userID, userID).
		First(&item).Error
	return item, normalizeNotFound(err)
}

func (r *GormRepository) ListMessages(ctx context.Context, query MessageQuery) ([]Message, error) {
	var items []Message
	db := r.db.WithContext(ctx).
		Where("conversation_id = ? AND expires_at > ?", query.ConversationID, time.Now())
	order := "id DESC"
	if query.AfterID > 0 {
		db = db.Where("id > ?", query.AfterID)
		order = "id ASC"
	} else if query.BeforeID > 0 {
		db = db.Where("id < ?", query.BeforeID)
	}

	err := db.Order(order).Limit(normalizePageSize(query.PageSize, 50, 100)).Find(&items).Error
	return items, err
}

func (r *GormRepository) MarkRead(ctx context.Context, conversation Conversation, userID int64, lastMessageID int64) error {
	if lastMessageID <= 0 || lastMessageID > conversation.LastMessageID {
		lastMessageID = conversation.LastMessageID
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var unread int64
		if err := tx.Model(&Message{}).
			Where("conversation_id = ? AND recipient_id = ? AND id > ? AND expires_at > ?", conversation.ID, userID, lastMessageID, time.Now()).
			Count(&unread).Error; err != nil {
			return err
		}

		updates := map[string]interface{}{}
		if userID == conversation.UserAID {
			updates["user_a_last_read_message_id"] = gorm.Expr("GREATEST(user_a_last_read_message_id, ?)", lastMessageID)
			updates["user_a_unread"] = unread
		} else {
			updates["user_b_last_read_message_id"] = gorm.Expr("GREATEST(user_b_last_read_message_id, ?)", lastMessageID)
			updates["user_b_unread"] = unread
		}
		return tx.Model(&Conversation{}).Where("id = ?", conversation.ID).Updates(updates).Error
	})
}

func (r *GormRepository) CountUnread(ctx context.Context, userID int64) (int64, error) {
	var result struct {
		Total int64
	}
	err := r.db.WithContext(ctx).
		Model(&Conversation{}).
		Select("COALESCE(SUM(CASE WHEN user_a_id = ? THEN user_a_unread ELSE user_b_unread END), 0) AS total", userID).
		Where("user_a_id = ? OR user_b_id = ?", userID, userID).
		Scan(&result).Error
	return result.Total, err
}

func (r *GormRepository) ClaimPushTasks(ctx context.Context, now time.Time, limit int) ([]PushTask, error) {
	var candidates []PushTask
	err := r.db.WithContext(ctx).
		Where("status = ? AND next_retry_at <= ?", PushTaskPending, now).
		Order("next_retry_at ASC, id ASC").
		Limit(normalizePageSize(limit, 100, 500)).
		Find(&candidates).Error
	if err != nil {
		return nil, err
	}

	claimed := make([]PushTask, 0, len(candidates))
	for _, task := range candidates {
		result := r.db.WithContext(ctx).Model(&PushTask{}).
			Where("id = ? AND status = ?", task.ID, PushTaskPending).
			Updates(map[string]interface{}{
				"status":    PushTaskProcessing,
				"locked_at": now,
			})
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected == 1 {
			task.Status = PushTaskProcessing
			task.LockedAt = &now
			claimed = append(claimed, task)
		}
	}
	return claimed, nil
}

func (r *GormRepository) FindPushMessage(ctx context.Context, messageID int64) (Message, error) {
	var item Message
	err := r.db.WithContext(ctx).First(&item, messageID).Error
	return item, normalizeNotFound(err)
}

func (r *GormRepository) MarkPushSucceeded(ctx context.Context, taskID int64, attempts int) error {
	return r.updatePushTask(ctx, taskID, map[string]interface{}{
		"status":     PushTaskSucceeded,
		"attempts":   attempts,
		"locked_at":  nil,
		"last_error": "",
	})
}

func (r *GormRepository) MarkPushSkipped(ctx context.Context, taskID int64, attempts int, reason string) error {
	return r.updatePushTask(ctx, taskID, map[string]interface{}{
		"status":     PushTaskSkipped,
		"attempts":   attempts,
		"locked_at":  nil,
		"last_error": truncateError(reason),
	})
}

func (r *GormRepository) MarkPushRetry(ctx context.Context, taskID int64, attempts int, nextRetryAt time.Time, reason string, dead bool) error {
	status := PushTaskPending
	if dead {
		status = PushTaskDead
	}
	return r.updatePushTask(ctx, taskID, map[string]interface{}{
		"status":        status,
		"attempts":      attempts,
		"next_retry_at": nextRetryAt,
		"locked_at":     nil,
		"last_error":    truncateError(reason),
	})
}

func (r *GormRepository) RecoverStalePushTasks(ctx context.Context, before time.Time) error {
	return r.db.WithContext(ctx).Model(&PushTask{}).
		Where("status = ? AND locked_at < ?", PushTaskProcessing, before).
		Updates(map[string]interface{}{
			"status":        PushTaskPending,
			"next_retry_at": time.Now(),
			"locked_at":     nil,
		}).Error
}

func (r *GormRepository) DeleteExpiredMessages(ctx context.Context, now time.Time, limit int) (int64, error) {
	var ids []int64
	if err := r.db.WithContext(ctx).Model(&Message{}).
		Where("expires_at <= ?", now).
		Order("id ASC").
		Limit(normalizePageSize(limit, 1000, 5000)).
		Pluck("id", &ids).Error; err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}
	result := r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&Message{})
	return result.RowsAffected, result.Error
}

func (r *GormRepository) DeleteEmptyConversations(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Where("NOT EXISTS (SELECT 1 FROM chat_messages WHERE chat_messages.conversation_id = chat_conversations.id)").
		Delete(&Conversation{}).Error
}

func (r *GormRepository) DeleteExpiredPushTasks(ctx context.Context, before time.Time, limit int) (int64, error) {
	var ids []int64
	if err := r.db.WithContext(ctx).Model(&PushTask{}).
		Where("created_at < ?", before).
		Order("id ASC").
		Limit(normalizePageSize(limit, 1000, 5000)).
		Pluck("id", &ids).Error; err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}
	result := r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&PushTask{})
	return result.RowsAffected, result.Error
}

func (r *GormRepository) updatePushTask(ctx context.Context, taskID int64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&PushTask{}).Where("id = ?", taskID).Updates(updates).Error
}

func newPushTask(messageID int64, recipientID int64, channel string, now time.Time) PushTask {
	return PushTask{
		MessageID:   messageID,
		RecipientID: recipientID,
		Channel:     channel,
		Status:      PushTaskPending,
		NextRetryAt: now,
	}
}

func normalizeUsers(first int64, second int64) (int64, int64) {
	if first < second {
		return first, second
	}
	return second, first
}

func normalizeNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

func normalizePageSize(value int, fallback int, maximum int) int {
	if value <= 0 {
		return fallback
	}
	if value > maximum {
		return maximum
	}
	return value
}

func paginate(db *gorm.DB, page int, pageSize int) *gorm.DB {
	if page <= 0 {
		page = 1
	}
	pageSize = normalizePageSize(pageSize, 20, 100)
	return db.Offset((page - 1) * pageSize).Limit(pageSize)
}

func truncateError(value string) string {
	const maxLength = 500
	runes := []rune(value)
	if len(runes) <= maxLength {
		return value
	}
	return string(runes[:maxLength])
}
