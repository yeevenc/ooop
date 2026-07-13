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

	_, err := service.CreateActivityReviewMessage(context.Background(), 3000, 99, "周末徒步", true)
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
