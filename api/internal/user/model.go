package user

import (
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	UserStatusEnabled = 1

	RegisterSourceAliyunMobile  = "aliyun_mobile"
	RegisterSourceJiguangMobile = "jiguang_mobile"
	RegisterSourceMobileCode    = "mobile_code"
	RegisterSourcePassword      = "password"

	// admin 是后台默认账号，禁止进入 App 用户体系。
	ReservedAdminUsername = "admin"

	DefaultAvatarPath = "/uploads/defaults/default_avatar.png"
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
	PushPlatform   string     `gorm:"size:32;not null;default:''" json:"push_platform"`
	RegistrationID string     `gorm:"size:128;not null;default:''" json:"registration_id"`
	PasswordHash   string     `gorm:"size:255" json:"-"`
	Status         int        `gorm:"not null;default:1" json:"status"`
	CreditScore    int        `gorm:"not null;default:100" json:"credit_score"` // 靠谱值（满分 100，评分逻辑后续接入）
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
	CreditScore    int        `json:"credit_score"`
	RegisterSource string     `json:"register_source"`
	HasPassword    bool       `json:"has_password"`
	LastLoginAt    *time.Time `json:"last_login_at"`
	CreatedAt      time.Time  `json:"created_at"`
	PublishedCount int        `json:"published_count"`
	JoinedCount    int        `json:"joined_count"`
	LikedCount     int        `json:"liked_count"`
}

func ToPublicUser(item User) PublicUser {
	return PublicUser{
		ID:             item.ID,
		Phone:          item.Phone,
		Username:       stringValue(item.Username),
		Nickname:       item.Nickname,
		Avatar:         AvatarURL(item.Avatar),
		Gender:         item.Gender,
		Region:         item.Region,
		Bio:            item.Bio,
		Platform:       item.Platform,
		DeviceNo:       item.DeviceNo,
		Status:         item.Status,
		CreditScore:    item.CreditScore,
		RegisterSource: item.RegisterSource,
		HasPassword:    item.PasswordHash != "",
		LastLoginAt:    item.LastLoginAt,
		CreatedAt:      item.CreatedAt,
	}
}

type UserStats struct {
	PublishedCount int
	JoinedCount    int
	LikedCount     int
}

func ToPublicUserWithStats(item User, stats UserStats) PublicUser {
	result := ToPublicUser(item)
	result.PublishedCount = stats.PublishedCount
	result.JoinedCount = stats.JoinedCount
	result.LikedCount = stats.LikedCount
	return result
}

// UserPublicProfile 是对外（查看他人主页）暴露的安全资料子集，
// 不含手机号/设备号/状态/注册来源等敏感字段。
type UserPublicProfile struct {
	ID             string    `json:"id"`
	Nickname       string    `json:"nickname"`
	Avatar         string    `json:"avatar"`
	Gender         string    `json:"gender"`
	Region         string    `json:"region"`
	Bio            string    `json:"bio"`
	CreditScore    int       `json:"creditScore"`
	CreatedAt      time.Time `json:"createdAt"`
	PublishedCount int       `json:"publishedCount"`
	JoinedCount    int       `json:"joinedCount"`
	LikedCount     int       `json:"likedCount"`
}

func ToUserPublicProfile(item User) UserPublicProfile {
	return UserPublicProfile{
		ID:          strconv.FormatInt(item.ID, 10),
		Nickname:    item.Nickname,
		Avatar:      AvatarURL(item.Avatar),
		Gender:      item.Gender,
		Region:      item.Region,
		Bio:         item.Bio,
		CreditScore: item.CreditScore,
		CreatedAt:   item.CreatedAt,
	}
}

func ToUserPublicProfileWithStats(item User, stats UserStats) UserPublicProfile {
	result := ToUserPublicProfile(item)
	result.PublishedCount = stats.PublishedCount
	result.JoinedCount = stats.JoinedCount
	result.LikedCount = stats.LikedCount
	return result
}

func AvatarURL(value string) string {
	avatar := strings.TrimSpace(value)
	if avatar == "" {
		return DefaultAvatarURL()
	}
	return avatar
}

func DefaultAvatarURL() string {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("APP_PUBLIC_BASE_URL")), "/")
	if baseURL == "" {
		return DefaultAvatarPath
	}
	return baseURL + DefaultAvatarPath
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
