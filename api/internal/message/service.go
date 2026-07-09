package message

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ooop-admin-api/internal/logger"
	"ooop-admin-api/internal/provider"
	"ooop-admin-api/internal/user"
)

var ErrNotFound = errors.New("消息不存在")

type Service struct {
	messages Repository
	pusher   PushSender
	users    user.UserRepository
}

type PushSender interface {
	Push(ctx context.Context, payload provider.JiguangPushPayload) (provider.JiguangPushResult, error)
}

func NewService(messages Repository, pusher PushSender, users user.UserRepository) *Service {
	return &Service{
		messages: messages,
		pusher:   pusher,
		users:    users,
	}
}

func (s *Service) CreateActivityReviewMessage(ctx context.Context, userID int64, activityID int64, activityTitle string, approved bool) (provider.JiguangPushResult, error) {
	title := "活动审核通知"
	content := fmt.Sprintf("您发布的%s审核拒绝。", strings.TrimSpace(activityTitle))
	if approved {
		content = fmt.Sprintf("您发布的%s审核成功。", strings.TrimSpace(activityTitle))
	}

	if err := s.messages.Create(ctx, &UserMessage{
		UserID:     userID,
		Type:       TypeActivityReview,
		Title:      title,
		Content:    content,
		ActivityID: &activityID,
	}); err != nil {
		return provider.JiguangPushResult{}, err
	}

	return s.pushToUser(ctx, userID, TypeActivityReview, title, content, activityID)
}

func (s *Service) CreateActivityRegistrationMessage(ctx context.Context, userID int64, activityID int64, activityTitle string, applicantName string) (provider.JiguangPushResult, error) {
	title := "活动报名通知"
	content := fmt.Sprintf("%s报名参加了您发布的%s。", strings.TrimSpace(applicantName), strings.TrimSpace(activityTitle))
	if strings.TrimSpace(applicantName) == "" {
		content = fmt.Sprintf("有人报名参加了您发布的%s。", strings.TrimSpace(activityTitle))
	}

	if err := s.messages.Create(ctx, &UserMessage{
		UserID:     userID,
		Type:       TypeRegistration,
		Title:      title,
		Content:    content,
		ActivityID: &activityID,
	}); err != nil {
		return provider.JiguangPushResult{}, err
	}

	return s.pushToUser(ctx, userID, TypeRegistration, title, content, activityID)
}

func (s *Service) ListUserMessages(ctx context.Context, userID int64, page int, pageSize int) ([]PublicMessage, error) {
	items, err := s.messages.ListByUser(ctx, UserMessageQuery{
		UserID:   userID,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, err
	}

	list := make([]PublicMessage, 0, len(items))
	for _, item := range items {
		list = append(list, toPublicMessage(item))
	}
	return list, nil
}

func (s *Service) MarkRead(ctx context.Context, userID int64, id int64) error {
	return s.messages.MarkRead(ctx, userID, id, time.Now())
}

func (s *Service) MarkAllRead(ctx context.Context, userID int64) (int64, error) {
	return s.messages.MarkAllRead(ctx, userID, time.Now())
}

func (s *Service) DeleteMessage(ctx context.Context, userID int64, id int64) error {
	return s.messages.DeleteByID(ctx, userID, id)
}

func (s *Service) ClearMessages(ctx context.Context, userID int64) (int64, error) {
	return s.messages.DeleteByUser(ctx, userID)
}

func formatID(id int64) string {
	return strconv.FormatInt(id, 10)
}

func (s *Service) pushToUser(ctx context.Context, userID int64, messageType string, title string, content string, activityID int64) (provider.JiguangPushResult, error) {
	alias := strconv.FormatInt(userID, 10)
	allowed, err := s.allowsPush(ctx, userID, messageType)
	if err != nil {
		return provider.JiguangPushResult{}, err
	}
	if !allowed {
		logger.Infof("极光推送跳过: 用户通知权限关闭, user_id=%d, activity_id=%d, type=%s", userID, activityID, messageType)
		return provider.JiguangPushResult{
			Triggered: false,
			Success:   false,
			Alias:     alias,
			Message:   "用户通知权限关闭",
		}, nil
	}

	if s.pusher == nil {
		logger.Warnf("极光推送跳过: pusher 未初始化, user_id=%d, activity_id=%d", userID, activityID)
		return provider.JiguangPushResult{
			Triggered: false,
			Success:   false,
			Alias:     alias,
			Message:   "极光推送未初始化",
		}, nil
	}

	logger.Infof("准备发送极光推送: user_id=%d, alias=%s, activity_id=%d, title=%s", userID, alias, activityID, title)

	result, err := s.pusher.Push(ctx, provider.JiguangPushPayload{
		Alias:      alias,
		Title:      title,
		Alert:      content,
		ActivityID: activityID,
	})
	if err != nil {
		logger.Errorf("极光推送发送失败: user_id=%d, alias=%s, activity_id=%d, error=%v", userID, alias, activityID, err)
		return result, err
	}

	return result, nil
}

func (s *Service) allowsPush(ctx context.Context, userID int64, messageType string) (bool, error) {
	if s.users == nil {
		return true, nil
	}

	item, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return false, err
	}

	switch messageType {
	case TypeActivityReview, TypeRegistration:
		return item.AllowsActivityReminderPush(), nil
	case TypeSystem:
		return item.AllowsSystemMessagePush(), nil
	default:
		return item.IsNotificationPermissionEnabled(), nil
	}
}
