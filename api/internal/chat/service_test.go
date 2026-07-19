package chat

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"ooop-admin-api/internal/contentmoderation"
	"ooop-admin-api/internal/user"
)

type serviceTestRepository struct {
	params                    CreateMessageParams
	conversations             []Conversation
	conversation              Conversation
	messageQuery              MessageQuery
	deletedConversationID     int64
	deletedConversationUserID int64
}

func (r *serviceTestRepository) CreateMessage(_ context.Context, params CreateMessageParams) (Conversation, Message, bool, error) {
	r.params = params
	return Conversation{ID: 9}, Message{
		ID:              10,
		ConversationID:  9,
		SenderID:        params.SenderID,
		RecipientID:     params.RecipientID,
		ClientMessageID: params.ClientMessageID,
		Type:            params.Type,
		Content:         params.Content,
		CreatedAt:       params.CreatedAt,
		ExpiresAt:       params.ExpiresAt,
	}, true, nil
}

func (r *serviceTestRepository) ListConversations(context.Context, int64, int, int) ([]Conversation, error) {
	return r.conversations, nil
}

func (r *serviceTestRepository) FindConversationForUser(context.Context, int64, int64) (Conversation, error) {
	return r.conversation, nil
}

func (r *serviceTestRepository) ListMessages(_ context.Context, query MessageQuery) ([]Message, error) {
	r.messageQuery = query
	return nil, nil
}

func (*serviceTestRepository) MarkRead(context.Context, Conversation, int64, int64) error {
	return nil
}

func (r *serviceTestRepository) DeleteConversation(_ context.Context, conversationID int64, userID int64) error {
	r.deletedConversationID = conversationID
	r.deletedConversationUserID = userID
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

func TestSendImageMessage(t *testing.T) {
	repository := &serviceTestRepository{}
	service := NewService(repository, serviceTestUsers{items: map[int64]user.User{
		3001: {ID: 3001},
	}}, nil, 7*24*time.Hour)

	result, err := service.SendMessage(context.Background(), 3000, SendMessageInput{
		RecipientID:     3001,
		ClientMessageID: "image-message-0001",
		Type:            MessageTypeImage,
		Content:         "https://cdn.example.com/chat/image.jpg",
	})
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	if result.Message.Type != MessageTypeImage || repository.params.Type != MessageTypeImage {
		t.Fatalf("result = %+v, params = %+v", result, repository.params)
	}
}

func TestSendMessageRejectsSensitiveContentBeforeStorage(t *testing.T) {
	repository := &serviceTestRepository{}
	checker, err := contentmoderation.NewChecker([]string{"聊天禁用测试词"})
	if err != nil {
		t.Fatalf("NewChecker() error = %v", err)
	}
	service := NewService(repository, serviceTestUsers{items: map[int64]user.User{
		3001: {ID: 3001},
	}}, checker, 7*24*time.Hour)

	_, err = service.SendMessage(context.Background(), 3000, SendMessageInput{
		RecipientID:     3001,
		ClientMessageID: "sensitive-message-0001",
		Content:         "这是一条聊天禁用测试词消息",
	})
	if !errors.Is(err, contentmoderation.ErrRejected) {
		t.Fatalf("error = %v, want ErrRejected", err)
	}
	if repository.params.Content != "" {
		t.Fatalf("rejected content was stored: %q", repository.params.Content)
	}
}

func TestSendImageMessageRejectsInvalidURL(t *testing.T) {
	service := NewService(&serviceTestRepository{}, serviceTestUsers{items: map[int64]user.User{
		3001: {ID: 3001},
	}}, nil, 7*24*time.Hour)

	_, err := service.SendMessage(context.Background(), 3000, SendMessageInput{
		RecipientID:     3001,
		ClientMessageID: "image-message-0002",
		Type:            MessageTypeImage,
		Content:         "file:///private/image.jpg",
	})
	if !errors.Is(err, ErrImageURLInvalid) {
		t.Fatalf("error = %v, want ErrImageURLInvalid", err)
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

func TestListConversationsReturnsOtherUserGender(t *testing.T) {
	repository := &serviceTestRepository{conversations: []Conversation{{
		ID:            9,
		UserAID:       3000,
		UserBID:       3001,
		LastMessageID: 10,
	}}}
	service := NewService(repository, serviceTestUsers{items: map[int64]user.User{
		3001: {ID: 3001, Nickname: "小欧", Gender: "女"},
	}}, nil, 7*24*time.Hour)

	result, err := service.ListConversations(context.Background(), 3000, 1, 20)
	if err != nil {
		t.Fatalf("ListConversations() error = %v", err)
	}
	if len(result.List) != 1 || result.List[0].OtherUser.Gender != "女" {
		t.Fatalf("result = %+v", result)
	}
}

func TestListMessagesAppliesUserDeleteBoundary(t *testing.T) {
	repository := &serviceTestRepository{conversation: Conversation{
		ID:                   9,
		UserAID:              3000,
		UserBID:              3001,
		UserADeletedBeforeID: 42,
	}}
	service := NewService(repository, serviceTestUsers{}, nil, 7*24*time.Hour)

	if _, err := service.ListMessages(context.Background(), 3000, MessageQuery{ConversationID: 9}); err != nil {
		t.Fatalf("ListMessages() error = %v", err)
	}
	if repository.messageQuery.DeletedBeforeID != 42 {
		t.Fatalf("DeletedBeforeID = %d, want 42", repository.messageQuery.DeletedBeforeID)
	}
}

func TestDeleteConversationUsesCurrentUser(t *testing.T) {
	repository := &serviceTestRepository{}
	service := NewService(repository, serviceTestUsers{}, nil, 7*24*time.Hour)

	if err := service.DeleteConversation(context.Background(), 3000, 9); err != nil {
		t.Fatalf("DeleteConversation() error = %v", err)
	}
	if repository.deletedConversationID != 9 || repository.deletedConversationUserID != 3000 {
		t.Fatalf("delete conversation = %d, user = %d", repository.deletedConversationID, repository.deletedConversationUserID)
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
