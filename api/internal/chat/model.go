package chat

import (
	"strconv"
	"time"
)

const (
	MessageTypeText = "text"
	PushMessageType = "chat_message"

	PushTaskPending    = "pending"
	PushTaskProcessing = "processing"
	PushTaskSucceeded  = "succeeded"
	PushTaskSkipped    = "skipped"
	PushTaskDead       = "dead"
)

type Conversation struct {
	ID                     int64      `gorm:"primaryKey;autoIncrement"`
	UserAID                int64      `gorm:"not null;index;uniqueIndex:uniq_chat_conversation_users,priority:1"`
	UserBID                int64      `gorm:"not null;index;uniqueIndex:uniq_chat_conversation_users,priority:2"`
	LastMessageID          int64      `gorm:"not null;default:0"`
	LastMessageContent     string     `gorm:"size:2000;not null;default:''"`
	LastMessageAt          *time.Time `gorm:"index"`
	UserAUnread            int        `gorm:"not null;default:0"`
	UserBUnread            int        `gorm:"not null;default:0"`
	UserALastReadMessageID int64      `gorm:"not null;default:0"`
	UserBLastReadMessageID int64      `gorm:"not null;default:0"`
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type Message struct {
	ID              int64     `gorm:"primaryKey;autoIncrement"`
	ConversationID  int64     `gorm:"not null;index:idx_chat_message_conversation_id,priority:1"`
	SenderID        int64     `gorm:"not null;index;uniqueIndex:uniq_chat_sender_client_message,priority:1"`
	RecipientID     int64     `gorm:"not null;index"`
	ClientMessageID string    `gorm:"size:64;not null;uniqueIndex:uniq_chat_sender_client_message,priority:2"`
	Type            string    `gorm:"size:16;not null;default:'text'"`
	Content         string    `gorm:"size:2000;not null"`
	ExpiresAt       time.Time `gorm:"not null;index"`
	CreatedAt       time.Time `gorm:"index:idx_chat_message_conversation_id,priority:2"`
}

type PushTask struct {
	ID          int64      `gorm:"primaryKey;autoIncrement"`
	MessageID   int64      `gorm:"not null;index;uniqueIndex:uniq_chat_message_push_channel,priority:1"`
	RecipientID int64      `gorm:"not null;index"`
	Channel     string     `gorm:"size:16;not null;uniqueIndex:uniq_chat_message_push_channel,priority:2"`
	Status      string     `gorm:"size:16;not null;index:idx_chat_push_schedule,priority:1"`
	Attempts    int        `gorm:"not null;default:0"`
	NextRetryAt time.Time  `gorm:"not null;index:idx_chat_push_schedule,priority:2"`
	LockedAt    *time.Time `gorm:"index"`
	LastError   string     `gorm:"size:500;not null;default:''"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Conversation) TableName() string {
	return "chat_conversations"
}

func (Message) TableName() string {
	return "chat_messages"
}

func (PushTask) TableName() string {
	return "chat_push_tasks"
}

type PublicMessage struct {
	ID              string    `json:"id"`
	ConversationID  string    `json:"conversationId"`
	SenderID        string    `json:"senderId"`
	RecipientID     string    `json:"recipientId"`
	ClientMessageID string    `json:"clientMessageId"`
	Type            string    `json:"type"`
	Content         string    `json:"content"`
	CreatedAt       time.Time `json:"createdAt"`
}

type PublicConversationUser struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

type PublicConversation struct {
	ID                 string                 `json:"id"`
	OtherUser          PublicConversationUser `json:"otherUser"`
	LastMessageID      string                 `json:"lastMessageId"`
	LastMessageContent string                 `json:"lastMessageContent"`
	LastMessageAt      *time.Time             `json:"lastMessageAt"`
	UnreadCount        int                    `json:"unreadCount"`
	LastReadMessageID  string                 `json:"lastReadMessageId"`
}

type ConversationListResult struct {
	List     []PublicConversation `json:"list"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
}

func toPublicMessage(item Message) PublicMessage {
	return PublicMessage{
		ID:              formatID(item.ID),
		ConversationID:  formatID(item.ConversationID),
		SenderID:        formatID(item.SenderID),
		RecipientID:     formatID(item.RecipientID),
		ClientMessageID: item.ClientMessageID,
		Type:            item.Type,
		Content:         item.Content,
		CreatedAt:       item.CreatedAt,
	}
}

func formatID(id int64) string {
	return strconv.FormatInt(id, 10)
}
