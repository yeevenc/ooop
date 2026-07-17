package chat

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"ooop-admin-api/internal/contentmoderation"
	"ooop-admin-api/internal/user"
)

var (
	ErrNotFound              = errors.New("会话或消息不存在")
	ErrRecipientNotFound     = errors.New("接收用户不存在")
	ErrSendToSelf            = errors.New("不能给自己发送消息")
	ErrContentRequired       = errors.New("请输入消息内容")
	ErrContentTooLong        = errors.New("消息内容不能超过 2000 个字符")
	ErrClientMessageInvalid  = errors.New("客户端消息号格式不正确")
	ErrClientMessageConflict = errors.New("客户端消息号已被其他消息使用")
	ErrCursorConflict        = errors.New("before_id 与 after_id 不能同时使用")
	ErrRateLimited           = errors.New("发送过于频繁，请稍后再试")
)

const maxMessageLength = 2000

type SendMessageInput struct {
	RecipientID     int64
	ClientMessageID string
	Content         string
}

type SendMessageResult struct {
	Message PublicMessage `json:"message"`
	Created bool          `json:"created"`
}

type Service struct {
	messages       MessageRepository
	users          UserReader
	contentChecker *contentmoderation.Checker
	retention      time.Duration
	limiter        *messageRateLimiter
}

type UserReader interface {
	FindByID(ctx context.Context, id int64) (user.User, error)
	FindByIDs(ctx context.Context, ids []int64) ([]user.User, error)
}

func NewService(messages MessageRepository, users UserReader, checker *contentmoderation.Checker, retention time.Duration) *Service {
	return &Service{
		messages:       messages,
		users:          users,
		contentChecker: checker,
		retention:      normalizeRetention(retention),
		limiter:        newMessageRateLimiter(),
	}
}

func (s *Service) SendMessage(ctx context.Context, senderID int64, input SendMessageInput) (SendMessageResult, error) {
	content := strings.TrimSpace(input.Content)
	clientMessageID := strings.TrimSpace(input.ClientMessageID)
	if input.RecipientID <= 0 {
		return SendMessageResult{}, ErrRecipientNotFound
	}
	if input.RecipientID == senderID {
		return SendMessageResult{}, ErrSendToSelf
	}
	if content == "" {
		return SendMessageResult{}, ErrContentRequired
	}
	if utf8.RuneCountInString(content) > maxMessageLength {
		return SendMessageResult{}, ErrContentTooLong
	}
	if !validClientMessageID(clientMessageID) {
		return SendMessageResult{}, ErrClientMessageInvalid
	}
	if !s.limiter.Allow(senderID, time.Now()) {
		return SendMessageResult{}, ErrRateLimited
	}

	if _, err := s.users.FindByID(ctx, input.RecipientID); err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return SendMessageResult{}, ErrRecipientNotFound
		}
		return SendMessageResult{}, err
	}
	if err := s.contentChecker.Check(ctx, contentmoderation.SceneContent, contentmoderation.Field{
		Name:    "消息内容",
		Content: content,
	}); err != nil {
		return SendMessageResult{}, err
	}

	now := time.Now()
	_, item, created, err := s.messages.CreateMessage(ctx, CreateMessageParams{
		SenderID:        senderID,
		RecipientID:     input.RecipientID,
		ClientMessageID: clientMessageID,
		Content:         content,
		CreatedAt:       now,
		ExpiresAt:       now.Add(s.retention),
	})
	if err != nil {
		return SendMessageResult{}, err
	}

	return SendMessageResult{
		Message: toPublicMessage(item),
		Created: created,
	}, nil
}

func (s *Service) ListConversations(ctx context.Context, userID int64, page int, pageSize int) (ConversationListResult, error) {
	if page <= 0 {
		page = 1
	}
	pageSize = normalizePageSize(pageSize, 20, 100)
	items, err := s.messages.ListConversations(ctx, userID, page, pageSize)
	if err != nil {
		return ConversationListResult{}, err
	}

	otherUserIDs := make([]int64, 0, len(items))
	for _, item := range items {
		otherUserIDs = append(otherUserIDs, otherUserID(item, userID))
	}
	users, err := s.users.FindByIDs(ctx, otherUserIDs)
	if err != nil {
		return ConversationListResult{}, err
	}
	userMap := make(map[int64]user.User, len(users))
	for _, item := range users {
		userMap[item.ID] = item
	}

	list := make([]PublicConversation, 0, len(items))
	for _, item := range items {
		otherID := otherUserID(item, userID)
		other := userMap[otherID]
		unread, lastReadID := conversationReadState(item, userID)
		list = append(list, PublicConversation{
			ID: formatID(item.ID),
			OtherUser: PublicConversationUser{
				ID:       formatID(otherID),
				Nickname: other.Nickname,
				Avatar:   user.AvatarURL(other.Avatar),
			},
			LastMessageID:      formatID(item.LastMessageID),
			LastMessageContent: item.LastMessageContent,
			LastMessageAt:      item.LastMessageAt,
			UnreadCount:        unread,
			LastReadMessageID:  formatID(lastReadID),
		})
	}

	return ConversationListResult{List: list, Page: page, PageSize: pageSize}, nil
}

func (s *Service) ListMessages(ctx context.Context, userID int64, query MessageQuery) ([]PublicMessage, error) {
	if query.BeforeID > 0 && query.AfterID > 0 {
		return nil, ErrCursorConflict
	}
	if _, err := s.messages.FindConversationForUser(ctx, query.ConversationID, userID); err != nil {
		return nil, err
	}

	items, err := s.messages.ListMessages(ctx, query)
	if err != nil {
		return nil, err
	}
	result := make([]PublicMessage, 0, len(items))
	for _, item := range items {
		result = append(result, toPublicMessage(item))
	}
	return result, nil
}

func (s *Service) MarkRead(ctx context.Context, userID int64, conversationID int64, lastMessageID int64) error {
	conversation, err := s.messages.FindConversationForUser(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	return s.messages.MarkRead(ctx, conversation, userID, lastMessageID)
}

func (s *Service) CountUnread(ctx context.Context, userID int64) (int64, error) {
	return s.messages.CountUnread(ctx, userID)
}

func normalizeRetention(value time.Duration) time.Duration {
	const minimum = 72 * time.Hour
	const maximum = 168 * time.Hour
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func validClientMessageID(value string) bool {
	if len(value) < 8 || len(value) > 64 {
		return false
	}
	return strings.IndexFunc(value, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	}) == -1
}

func otherUserID(item Conversation, userID int64) int64 {
	if item.UserAID == userID {
		return item.UserBID
	}
	return item.UserAID
}

func conversationReadState(item Conversation, userID int64) (int, int64) {
	if item.UserAID == userID {
		return item.UserAUnread, item.UserALastReadMessageID
	}
	return item.UserBUnread, item.UserBLastReadMessageID
}
