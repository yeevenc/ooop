package activity

import "time"

const (
	StatusPending   = "pending"    // 待审核
	StatusOngoing   = "ongoing"    // 已通过/进行中，App 可见
	StatusRejected  = "rejected"   // 审核拒绝
	StatusTakenDown = "taken_down" // 已下架
	StatusCancelled = "cancelled"  // 发起人取消
	CategoryEnabled = 1
)

type Activity struct {
	ID                int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID            int64      `gorm:"not null;index" json:"user_id"`
	Title             string     `gorm:"size:80;not null" json:"title"`
	CategoryID        string     `gorm:"size:32;not null;index" json:"category_id"`
	CategoryLabel     string     `gorm:"size:32;not null" json:"category_label"`
	ActivityDate      *time.Time `gorm:"index" json:"activity_date"`
	ActivityTime      string     `gorm:"size:16;not null;default:''" json:"activity_time"`
	DeadlineAt        *time.Time `gorm:"index" json:"deadline_at"`
	LocationText      string     `gorm:"size:255;not null" json:"location_text"`
	City              string     `gorm:"size:64;not null;index" json:"city"`
	Latitude          float64    `gorm:"not null" json:"latitude"`
	Longitude         float64    `gorm:"not null" json:"longitude"`
	TotalCount        int        `gorm:"not null;default:2" json:"total_count"`
	CurrentCount      int        `gorm:"not null;default:1" json:"current_count"`
	CostType          string     `gorm:"size:32;not null;default:''" json:"cost_type"`
	FeeDetail         string     `gorm:"size:80;not null;default:''" json:"fee_detail"`
	GenderRequirement string     `gorm:"size:32;not null;default:''" json:"gender_requirement"`
	Intro             string     `gorm:"size:1000;not null" json:"intro"`
	Notice            string     `gorm:"size:500;not null;default:''" json:"notice"`
	ImageURL          string     `gorm:"size:500;not null;default:''" json:"image_url"`
	GalleryJSON       string     `gorm:"type:text" json:"gallery_json"`
	Status            string     `gorm:"size:32;not null;default:'ongoing';index" json:"status"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type Organizer struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Avatar           string  `json:"avatar"`
	Gender           string  `json:"gender"`
	AvatarColor      string  `json:"avatarColor"`
	CreditLabel      string  `json:"creditLabel"`
	Rating           float64 `json:"rating"`
	PersonalityLabel string  `json:"personalityLabel"`
	CompletionRate   int     `json:"completionRate"`
	Verified         bool    `json:"verified"`
}

type PublicActivity struct {
	ID                string          `json:"id"`
	Title             string          `json:"title"`
	CategoryID        string          `json:"categoryId"`
	CategoryLabel     string          `json:"categoryLabel"`
	ImageURL          string          `json:"imageUrl"`
	Gallery           []string        `json:"gallery"`
	Status            string          `json:"status"`
	CostLabel         string          `json:"costLabel"`
	CostType          string          `json:"costType"`
	Time              string          `json:"time"`
	CurrentCount      int             `json:"currentCount"`
	TotalCount        int             `json:"totalCount"`
	NeedCount         int             `json:"needCount"`
	DeadlineText      string          `json:"deadlineText"`
	DateText          string          `json:"dateText"`
	TimeRange         string          `json:"timeRange"`
	ActivityTime      string          `json:"activityTime"`
	ActivityDate      *time.Time      `json:"activityDate"`
	LocationText      string          `json:"locationText"`
	City              string          `json:"city"`
	Latitude          float64         `json:"latitude"`
	Longitude         float64         `json:"longitude"`
	FeeDetail         string          `json:"feeDetail"`
	GenderRequirement string          `json:"genderRequirement"`
	Intro             string          `json:"intro"`
	Notice            string          `json:"notice"`
	Organizer         Organizer       `json:"organizer"`
	Participants      []any           `json:"participants"`
	ActionType        string          `json:"actionType"`
	PendingCount      int             `json:"pendingCount"`
	JoinInfo          *PublicJoinInfo `json:"joinInfo,omitempty"`
	CreatedAt         time.Time       `json:"createdAt"`
}

type ActivityCategory struct {
	ID        string    `gorm:"primaryKey;size:32" json:"id"`
	Label     string    `gorm:"size:32;not null" json:"label"`
	Icon      string    `gorm:"size:255;not null;default:''" json:"icon"`
	Sort      int       `gorm:"not null;default:0;index" json:"sort"`
	Status    int       `gorm:"not null;default:1;index" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PublicActivityCategory struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Icon  string `json:"icon"`
	Sort  int    `json:"sort"`
}
