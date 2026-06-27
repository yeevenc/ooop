package message

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ooop-admin-api/internal/provider"
)

var ErrNotFound = errors.New("消息不存在")

type Service struct {
	messages Repository
	pusher   PushSender
}

type PushSender interface {
	Push(ctx context.Context, payload provider.JiguangPushPayload) error
}

func NewService(messages Repository, pusher PushSender) *Service {
	return &Service{
		messages: messages,
		pusher:   pusher,
	}
}

func (s *Service) CreateActivityReviewMessage(ctx context.Context, userID int64, activityID int64, activityTitle string, approved bool) error {
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
		return err
	}

	return s.pushToUser(ctx, userID, title, content, activityID)
}

func (s *Service) CreateActivityRegistrationMessage(ctx context.Context, userID int64, activityID int64, activityTitle string, applicantName string) error {
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
		return err
	}

	return s.pushToUser(ctx, userID, title, content, activityID)
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

func formatID(id int64) string {
	return strconv.FormatInt(id, 10)
}

func (s *Service) pushToUser(ctx context.Context, userID int64, title string, content string, activityID int64) error {
	if s.pusher == nil {
		return nil
	}

	return s.pusher.Push(ctx, provider.JiguangPushPayload{
		Alias:      strconv.FormatInt(userID, 10),
		Title:      title,
		Alert:      content,
		ActivityID: activityID,
	})
}
