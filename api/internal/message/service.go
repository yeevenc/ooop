package message

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var ErrNotFound = errors.New("消息不存在")

type Service struct {
	messages Repository
}

func NewService(messages Repository) *Service {
	return &Service{messages: messages}
}

func (s *Service) CreateActivityReviewMessage(ctx context.Context, userID int64, activityID int64, activityTitle string, approved bool) error {
	title := "活动审核未通过"
	content := fmt.Sprintf("你发布的「%s」未通过审核，请调整内容后重新提交。", strings.TrimSpace(activityTitle))
	if approved {
		title = "活动审核已通过"
		content = fmt.Sprintf("你发布的「%s」已通过审核，其他用户现在可以看到并报名。", strings.TrimSpace(activityTitle))
	}

	return s.messages.Create(ctx, &UserMessage{
		UserID:     userID,
		Type:       TypeActivityReview,
		Title:      title,
		Content:    content,
		ActivityID: &activityID,
	})
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
