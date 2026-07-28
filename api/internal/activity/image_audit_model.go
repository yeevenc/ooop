package activity

import "time"

const (
	ImageAuditTaskPending      = "pending"
	ImageAuditTaskProcessing   = "processing"
	ImageAuditTaskPassed       = "passed"
	ImageAuditTaskRejected     = "rejected"
	ImageAuditTaskManualReview = "manual_review"
	ImageAuditTaskSkipped      = "skipped"
)

type ImageAuditTask struct {
	ID               int64      `gorm:"primaryKey;autoIncrement"`
	ActivityID       int64      `gorm:"not null;uniqueIndex:uniq_activity_image_audit_tasks_activity_id"`
	ImageURLsJSON    string     `gorm:"type:text;not null"`
	Status           string     `gorm:"size:24;not null;index:idx_activity_image_audit_schedule,priority:1;index:idx_activity_image_audit_recovery,priority:1"`
	Decision         string     `gorm:"size:16;not null;default:''"`
	Attempts         int        `gorm:"not null;default:0"`
	NextRetryAt      time.Time  `gorm:"not null;index:idx_activity_image_audit_schedule,priority:2"`
	LockedAt         *time.Time `gorm:"index;index:idx_activity_image_audit_recovery,priority:2"`
	ResultJSON       string     `gorm:"type:longtext"`
	RejectReason     string     `gorm:"size:500;not null;default:''"`
	NotificationDone bool       `gorm:"not null;default:false"`
	LastError        string     `gorm:"size:500;not null;default:''"`
	CompletedAt      *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (ImageAuditTask) TableName() string {
	return "activity_image_audit_tasks"
}
