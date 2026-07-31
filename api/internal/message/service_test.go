package message

import (
	"context"
	"testing"
	"time"

	"ooop-admin-api/internal/provider"
)

type pushTestRepository struct{}

func (pushTestRepository) Create(_ context.Context, item *UserMessage) error {
	item.ID = 77
	return nil
}

func (pushTestRepository) CreateIdempotent(ctx context.Context, item *UserMessage) error {
	return pushTestRepository{}.Create(ctx, item)
}

func (pushTestRepository) ListByUser(context.Context, UserMessageQuery) ([]UserMessage, error) {
	return nil, nil
}

func (pushTestRepository) MarkRead(context.Context, int64, int64, time.Time) error {
	return nil
}

func (pushTestRepository) MarkAllRead(context.Context, int64, time.Time) (int64, error) {
	return 0, nil
}

func (pushTestRepository) DeleteByID(context.Context, int64, int64) error {
	return nil
}

func (pushTestRepository) DeleteByUser(context.Context, int64) (int64, error) {
	return 0, nil
}

type messageQueryCaptureRepository struct {
	pushTestRepository
	query UserMessageQuery
}

func (r *messageQueryCaptureRepository) ListByUser(_ context.Context, query UserMessageQuery) ([]UserMessage, error) {
	r.query = query
	return []UserMessage{{
		ID:      88,
		UserID:  query.UserID,
		Type:    TypeSystem,
		Title:   "当前用户消息",
		Content: "仅返回当前用户的数据",
	}}, nil
}

type capturePushSender struct {
	payload provider.PushPayload
}

func (s *capturePushSender) Push(_ context.Context, payload provider.PushPayload) (provider.PushResult, error) {
	s.payload = payload
	return provider.PushResult{Success: true}, nil
}

func TestCreatedMessageIDIsSharedWithPushChannels(t *testing.T) {
	pusher := &capturePushSender{}
	service := NewService(pushTestRepository{}, pusher, nil)

	_, err := service.CreateActivityReviewMessage(context.Background(), 3000, 99, "周末徒步", true, "", "")
	if err != nil {
		t.Fatalf("CreateActivityReviewMessage() error = %v", err)
	}
	if pusher.payload.MessageID != 77 {
		t.Fatalf("MessageID = %d, want 77", pusher.payload.MessageID)
	}
	if pusher.payload.Alias != "3000" || pusher.payload.ActivityID != 99 {
		t.Fatalf("push payload = %+v", pusher.payload)
	}
}

func TestRegistrationReviewUsesSharedPushChannel(t *testing.T) {
	pusher := &capturePushSender{}
	service := NewService(pushTestRepository{}, pusher, nil)

	_, err := service.CreateRegistrationReviewMessage(
		context.Background(),
		3001,
		99,
		"周末徒步",
		true,
		"ABC12345",
		"",
	)
	if err != nil {
		t.Fatalf("CreateRegistrationReviewMessage() error = %v", err)
	}
	if pusher.payload.MessageType != TypeRegistrationReview {
		t.Fatalf("MessageType = %s, want %s", pusher.payload.MessageType, TypeRegistrationReview)
	}
	if pusher.payload.MessageID != 77 || pusher.payload.ActivityID != 99 {
		t.Fatalf("push payload = %+v", pusher.payload)
	}
	if pusher.payload.Alert != "您报名的周末徒步已通过审核，参加编号为 ABC12345。" {
		t.Fatalf("Alert = %s", pusher.payload.Alert)
	}
}

func TestActivityRejectMessageContainsImageAuditReason(t *testing.T) {
	pusher := &capturePushSender{}
	service := NewService(pushTestRepository{}, pusher, nil)

	_, err := service.CreateActivityReviewMessage(
		context.Background(),
		3000,
		99,
		"周末徒步",
		false,
		"第1张图片命中色情低俗内容",
		"activity-image-audit:7",
	)
	if err != nil {
		t.Fatalf("CreateActivityReviewMessage() error = %v", err)
	}
	if pusher.payload.Alert != "您发布的周末徒步审核拒绝：第1张图片命中色情低俗内容" {
		t.Fatalf("Alert = %s", pusher.payload.Alert)
	}
}

func TestListUserMessagesUsesCurrentUser(t *testing.T) {
	repository := &messageQueryCaptureRepository{}
	service := NewService(repository, nil, nil)

	result, err := service.ListUserMessages(context.Background(), 3001, 1, 20)
	if err != nil {
		t.Fatalf("ListUserMessages() error = %v", err)
	}
	if repository.query.UserID != 3001 {
		t.Fatalf("ListByUser user = %d, want 3001", repository.query.UserID)
	}
	if len(result) != 1 || result[0].ID != "88" {
		t.Fatalf("result = %+v", result)
	}
}
