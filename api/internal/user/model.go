package user

import "time"

const (
	UserStatusEnabled = 1

	RegisterSourceAliyunMobile = "aliyun_mobile"
	RegisterSourceMobileCode   = "mobile_code"
	RegisterSourcePassword     = "password"

	// admin 是后台默认账号，禁止进入 App 用户体系。
	ReservedAdminUsername = "admin"
)

type User struct {
	ID             int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	Phone          string     `gorm:"size:20;not null;uniqueIndex" json:"phone"`
	Username       *string    `gorm:"size:64;uniqueIndex" json:"username"`
	Nickname       string     `gorm:"size:64;not null;default:''" json:"nickname"`
	Avatar         string     `gorm:"size:255;not null;default:''" json:"avatar"`
	Gender         string     `gorm:"size:16;not null;default:''" json:"gender"`
	Region         string     `gorm:"size:64;not null;default:''" json:"region"`
	Bio            string     `gorm:"size:255;not null;default:''" json:"bio"`
	Platform       string     `gorm:"size:32;not null;default:''" json:"platform"`
	DeviceNo       string     `gorm:"size:128;not null;default:''" json:"device_no"`
	PasswordHash   string     `gorm:"size:255" json:"-"`
	Status         int        `gorm:"not null;default:1" json:"status"`
	RegisterSource string     `gorm:"size:32;not null" json:"register_source"`
	LastLoginAt    *time.Time `json:"last_login_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type LoginCode struct {
	ID        int64      `gorm:"primaryKey;autoIncrement"`
	Phone     string     `gorm:"size:20;not null;index"`
	Scene     string     `gorm:"size:32;not null;index"`
	CodeHash  string     `gorm:"size:128;not null"`
	UsedAt    *time.Time `gorm:"index"`
	ExpiresAt time.Time  `gorm:"not null;index"`
	CreatedAt time.Time
}

type RefreshToken struct {
	ID        int64      `gorm:"primaryKey;autoIncrement"`
	UserID    int64      `gorm:"not null;index"`
	TokenHash string     `gorm:"size:128;not null;uniqueIndex"`
	ExpiresAt time.Time  `gorm:"not null;index"`
	RevokedAt *time.Time `gorm:"index"`
	CreatedAt time.Time
}

type PublicUser struct {
	ID             int64      `json:"id"`
	Phone          string     `json:"phone"`
	Username       string     `json:"username"`
	Nickname       string     `json:"nickname"`
	Avatar         string     `json:"avatar"`
	Gender         string     `json:"gender"`
	Region         string     `json:"region"`
	Bio            string     `json:"bio"`
	Platform       string     `json:"platform"`
	DeviceNo       string     `json:"device_no"`
	Status         int        `json:"status"`
	RegisterSource string     `json:"register_source"`
	LastLoginAt    *time.Time `json:"last_login_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

func ToPublicUser(item User) PublicUser {
	return PublicUser{
		ID:             item.ID,
		Phone:          item.Phone,
		Username:       stringValue(item.Username),
		Nickname:       item.Nickname,
		Avatar:         item.Avatar,
		Gender:         item.Gender,
		Region:         item.Region,
		Bio:            item.Bio,
		Platform:       item.Platform,
		DeviceNo:       item.DeviceNo,
		Status:         item.Status,
		RegisterSource: item.RegisterSource,
		LastLoginAt:    item.LastLoginAt,
		CreatedAt:      item.CreatedAt,
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
