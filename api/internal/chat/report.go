package chat

import "time"

const (
	ReportReasonSpam        = "spam"
	ReportReasonHarassment  = "harassment"
	ReportReasonPornography = "pornography"
	ReportReasonFraud       = "fraud"
	ReportReasonIllegal     = "illegal"
	ReportReasonOther       = "other"

	ReportStatusPending   = "pending"
	ReportStatusResolved  = "resolved"
	ReportStatusDismissed = "dismissed"
)

type ChatReport struct {
	ID               int64      `gorm:"primaryKey;autoIncrement"`
	ConversationID   int64      `gorm:"not null;index"`
	ReporterID       int64      `gorm:"not null;index"`
	ReportedUserID   int64      `gorm:"not null;index"`
	Reason           string     `gorm:"size:32;not null;index"`
	Description      string     `gorm:"size:500;not null;default:''"`
	EvidenceJSON     string     `gorm:"type:longtext;not null"`
	Status           string     `gorm:"size:20;not null;default:'pending';index"`
	HandleResult     string     `gorm:"size:500;not null;default:''"`
	HandlerAdminID   *int64     `gorm:"index"`
	HandledAt        *time.Time `gorm:"index"`
	RestrictionUntil *time.Time `gorm:"index"`
	CreatedAt        time.Time  `gorm:"index"`
	UpdatedAt        time.Time
}

func (ChatReport) TableName() string {
	return "chat_reports"
}

type SubmitReportInput struct {
	Reason      string
	Description string
}

type ReportEvidenceMessage struct {
	ID        string    `json:"id"`
	SenderID  string    `json:"senderId"`
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type ReportReceipt struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type AdminReportUser struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	Phone    string `json:"phone"`
	Avatar   string `json:"avatar"`
}

type AdminChatReport struct {
	ID               string                  `json:"id"`
	ConversationID   string                  `json:"conversationId"`
	Reporter         AdminReportUser         `json:"reporter"`
	ReportedUser     AdminReportUser         `json:"reportedUser"`
	Reason           string                  `json:"reason"`
	Description      string                  `json:"description"`
	EvidenceCount    int                     `json:"evidenceCount"`
	Evidence         []ReportEvidenceMessage `json:"evidence,omitempty"`
	Status           string                  `json:"status"`
	HandleResult     string                  `json:"handleResult"`
	HandlerAdminID   string                  `json:"handlerAdminId,omitempty"`
	HandledAt        *time.Time              `json:"handledAt"`
	RestrictionUntil *time.Time              `json:"restrictionUntil"`
	CreatedAt        time.Time               `json:"createdAt"`
	UpdatedAt        time.Time               `json:"updatedAt"`
}

type AdminReportQuery struct {
	Page     int
	PageSize int
	Status   string
	Keyword  string
}

type AdminReportListResult struct {
	List     []AdminChatReport `json:"list"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

type ResolveReportInput struct {
	Status           string
	Result           string
	RestrictionUntil *time.Time
}

type ReportResolution struct {
	Report         ChatReport
	MessageID      int64
	MessageTitle   string
	MessageContent string
}
