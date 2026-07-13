package message

import "time"

const (
	TypeActivityReview     = "activity_review"
	TypeRegistration       = "registration"
	TypeRegistrationReview = "registration_review"
	TypeSystem             = "system"
	TypeInteraction        = "interaction"
)

type UserMessage struct {
	ID         int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     int64      `gorm:"not null;index" json:"user_id"`
	Type       string     `gorm:"size:32;not null;index" json:"type"`
	Title      string     `gorm:"size:80;not null" json:"title"`
	Content    string     `gorm:"size:500;not null;default:''" json:"content"`
	ActivityID *int64     `gorm:"index" json:"activity_id"`
	ReadAt     *time.Time `json:"read_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type PublicMessage struct {
	ID         string     `json:"id"`
	Type       string     `json:"type"`
	Title      string     `json:"title"`
	Content    string     `json:"content"`
	ActivityID string     `json:"activityId,omitempty"`
	IsRead     bool       `json:"isRead"`
	ReadAt     *time.Time `json:"readAt"`
	CreatedAt  time.Time  `json:"createdAt"`
}

func toPublicMessage(item UserMessage) PublicMessage {
	result := PublicMessage{
		ID:        formatID(item.ID),
		Type:      item.Type,
		Title:     item.Title,
		Content:   item.Content,
		IsRead:    item.ReadAt != nil,
		ReadAt:    item.ReadAt,
		CreatedAt: item.CreatedAt,
	}
	if item.ActivityID != nil {
		result.ActivityID = formatID(*item.ActivityID)
	}
	return result
}
