package message

import (
	"context"
	"fmt"
)

func (s *Service) NotifyActivityIntent(ctx context.Context, uid, aid int64, activityTitle, key string) error {
	title := "你登记的活动意向有新消息"
	content := truncateMessageContent(fmt.Sprintf("官方发布了「%s」。本活动为意向征集，官方不到场，不保证举行，可私聊运营了解安排。", activityTitle), 500)
	item := &UserMessage{UserID: uid, Type: TypeInteraction, Title: title, Content: content, ActivityID: &aid, IdempotencyKey: &key}
	if err := s.messages.CreateIdempotent(ctx, item); err != nil {
		return err
	}
	result, err := s.pushToUser(ctx, uid, TypeInteraction, title, content, item.ID, aid)
	if err != nil {
		return err
	}
	if result.Triggered && !result.Success {
		return fmt.Errorf("意向通知推送失败: %s", result.Message)
	}
	return nil
}
