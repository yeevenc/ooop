package message

import "time"

const (
	BroadcastStatusPending   = "pending"
	BroadcastStatusRunning   = "running"
	BroadcastStatusCompleted = "completed"
	BroadcastStatusFailed    = "failed"

	maxBroadcastTitleLength   = 80
	maxBroadcastContentLength = 500
	defaultBroadcastBatchSize = 50
	maxBroadcastBatchSize     = 200
)

type SystemBroadcast struct {
	ID           int64  `gorm:"primaryKey;autoIncrement"`
	AdminID      int64  `gorm:"not null;index"`
	Title        string `gorm:"size:80;not null"`
	Content      string `gorm:"size:500;not null;default:''"`
	Status       string `gorm:"size:16;not null;index"`
	TargetCount  int    `gorm:"not null;default:0"`
	MessageCount int    `gorm:"not null;default:0"`
	PushSuccess  int    `gorm:"not null;default:0"`
	PushSkipped  int    `gorm:"not null;default:0"`
	PushFailed   int    `gorm:"not null;default:0"`
	LastUserID   int64  `gorm:"not null;default:0"`
	ErrorMessage string `gorm:"size:500;not null;default:''"`
	StartedAt    *time.Time
	FinishedAt   *time.Time
	CreatedAt    time.Time `gorm:"index"`
	UpdatedAt    time.Time
}

func (SystemBroadcast) TableName() string {
	return "system_broadcasts"
}

type BroadcastQuery struct {
	Page     int
	PageSize int
	Status   string
}

type AdminSystemBroadcast struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	Content      string     `json:"content"`
	Status       string     `json:"status"`
	TargetCount  int        `json:"targetCount"`
	MessageCount int        `json:"messageCount"`
	PushSuccess  int        `json:"pushSuccess"`
	PushSkipped  int        `json:"pushSkipped"`
	PushFailed   int        `json:"pushFailed"`
	ErrorMessage string     `json:"errorMessage,omitempty"`
	StartedAt    *time.Time `json:"startedAt"`
	FinishedAt   *time.Time `json:"finishedAt"`
	CreatedAt    time.Time  `json:"createdAt"`
}

type AdminSystemBroadcastList struct {
	List     []AdminSystemBroadcast `json:"list"`
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
}

func toAdminSystemBroadcast(item SystemBroadcast) AdminSystemBroadcast {
	return AdminSystemBroadcast{
		ID:           formatID(item.ID),
		Title:        item.Title,
		Content:      item.Content,
		Status:       item.Status,
		TargetCount:  item.TargetCount,
		MessageCount: item.MessageCount,
		PushSuccess:  item.PushSuccess,
		PushSkipped:  item.PushSkipped,
		PushFailed:   item.PushFailed,
		ErrorMessage: item.ErrorMessage,
		StartedAt:    item.StartedAt,
		FinishedAt:   item.FinishedAt,
		CreatedAt:    item.CreatedAt,
	}
}
