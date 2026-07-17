package chat

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"ooop-admin-api/internal/user"
)

type serviceTestRepository struct {
	params CreateMessageParams
}

func (r *serviceTestRepository) CreateMessage(_ context.Context, params CreateMessageParams) (Conversation, Message, bool, error) {
	r.params = params
	return Conversation{ID: 9}, Message{
		ID:              10,
		ConversationID:  9,
		SenderID:        params.SenderID,
		RecipientID:     params.RecipientID,
		ClientMessageID: params.ClientMessageID,
		Type:            MessageTypeText,
		Content:         params.Content,
		CreatedAt:       params.CreatedAt,
		ExpiresAt:       params.ExpiresAt,
	}, true, nil
}

func (*serviceTestRepository) ListConversations(context.Context, int64, int, int) ([]Conversation, error) {
	return nil, nil
}

func (*serviceTestRepository) FindConversationForUser(context.Context, int64, int64) (Conversation, error) {
	return Conversation{}, nil
}

func (*serviceTestRepository) ListMessages(context.Context, MessageQuery) ([]Message, error) {
	return nil, nil
}

func (*serviceTestRepository) MarkRead(context.Context, Conversation, int64, int64) error {
	return nil
}

func (*serviceTestRepository) CountUnread(context.Context, int64) (int64, error) {
	return 0, nil
}

type serviceTestUsers struct {
	items map[int64]user.User
}

func (s serviceTestUsers) FindByID(_ context.Context, id int64) (user.User, error) {
	item, ok := s.items[id]
	if !ok {
		return user.User{}, user.ErrNotFound
	}
	return item, nil
}

func (s serviceTestUsers) FindByIDs(_ context.Context, ids []int64) ([]user.User, error) {
	result := make([]user.User, 0, len(ids))
	for _, id := range ids {
		if item, ok := s.items[id]; ok {
			result = append(result, item)
		}
	}
	return result, nil
}

func TestSendMessageCreatesSevenDayMessage(t *testing.T) {
	repository := &serviceTestRepository{}
	service := NewService(repository, serviceTestUsers{items: map[int64]user.User{
		3001: {ID: 3001},
	}}, nil, 7*24*time.Hour)

	result, err := service.SendMessage(context.Background(), 3000, SendMessageInput{
		RecipientID:     3001,
		ClientMessageID: "0190f25d-6b71-7b68",
		Content:         "你好 😊",
	})
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	if !result.Created || result.Message.Content != "你好 😊" {
		t.Fatalf("result = %+v", result)
	}
	retention := repository.params.ExpiresAt.Sub(repository.params.CreatedAt)
	if retention != 7*24*time.Hour {
		t.Fatalf("retention = %s", retention)
	}
}

func TestSendMessageRejectsSelfAndMissingRecipient(t *testing.T) {
	service := NewService(&serviceTestRepository{}, serviceTestUsers{items: map[int64]user.User{}}, nil, 7*24*time.Hour)

	_, err := service.SendMessage(context.Background(), 3000, SendMessageInput{
		RecipientID:     3000,
		ClientMessageID: "0190f25d-6b71-7b68",
		Content:         "你好",
	})
	if !errors.Is(err, ErrSendToSelf) {
		t.Fatalf("error = %v, want ErrSendToSelf", err)
	}

	_, err = service.SendMessage(context.Background(), 3000, SendMessageInput{
		RecipientID:     3001,
		ClientMessageID: "0190f25d-6b71-7b68",
		Content:         "你好",
	})
	if !errors.Is(err, ErrRecipientNotFound) {
		t.Fatalf("error = %v, want ErrRecipientNotFound", err)
	}
}

func TestListMessagesRejectsConflictingCursors(t *testing.T) {
	service := NewService(&serviceTestRepository{}, serviceTestUsers{}, nil, 7*24*time.Hour)
	_, err := service.ListMessages(context.Background(), 3000, MessageQuery{
		ConversationID: 9,
		BeforeID:       10,
		AfterID:        8,
	})
	if !errors.Is(err, ErrCursorConflict) {
		t.Fatalf("error = %v, want ErrCursorConflict", err)
	}
}

func TestSendMessageLimitsSingleUserBurst(t *testing.T) {
	service := NewService(&serviceTestRepository{}, serviceTestUsers{items: map[int64]user.User{
		3001: {ID: 3001},
	}}, nil, 7*24*time.Hour)

	for index := 0; index < int(perUserMessageBurst); index++ {
		_, err := service.SendMessage(context.Background(), 3000, SendMessageInput{
			RecipientID:     3001,
			ClientMessageID: fmt.Sprintf("message-%08d", index),
			Content:         "你好",
		})
		if err != nil {
			t.Fatalf("message %d error = %v", index, err)
		}
	}

	_, err := service.SendMessage(context.Background(), 3000, SendMessageInput{
		RecipientID:     3001,
		ClientMessageID: "message-over-limit",
		Content:         "你好",
	})
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("error = %v, want ErrRateLimited", err)
	}
}
