package feedback

import (
	"encoding/json"
	"strconv"
	"time"
)

const (
	TypeProduct  = "product"
	TypeAccount  = "account"
	TypeActivity = "activity"
)

type Feedback struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         int64     `gorm:"not null;index" json:"user_id"`
	UserPhone      string    `gorm:"size:20;not null;default:''" json:"user_phone"`
	UserNickname   string    `gorm:"size:64;not null;default:''" json:"user_nickname"`
	Type           string    `gorm:"size:32;not null;index" json:"type"`
	Content        string    `gorm:"size:1200;not null" json:"content"`
	ImageURLs      string    `gorm:"type:text" json:"image_urls"`
	DevicePlatform string    `gorm:"size:32;not null;default:''" json:"device_platform"`
	DeviceVersion  string    `gorm:"size:64;not null;default:''" json:"device_version"`
	AppVersion     string    `gorm:"size:32;not null;default:''" json:"app_version"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type PublicFeedback struct {
	ID             string    `json:"id"`
	UserID         string    `json:"userId"`
	UserPhone      string    `json:"userPhone"`
	UserNickname   string    `json:"userNickname"`
	Type           string    `json:"type"`
	Content        string    `json:"content"`
	ImageURLs      []string  `json:"imageUrls"`
	DevicePlatform string    `json:"devicePlatform"`
	DeviceVersion  string    `json:"deviceVersion"`
	AppVersion     string    `json:"appVersion"`
	CreatedAt      time.Time `json:"createdAt"`
}

func toPublicFeedback(item Feedback) PublicFeedback {
	return PublicFeedback{
		ID:             formatID(item.ID),
		UserID:         formatID(item.UserID),
		UserPhone:      item.UserPhone,
		UserNickname:   item.UserNickname,
		Type:           item.Type,
		Content:        item.Content,
		ImageURLs:      parseImageURLs(item.ImageURLs),
		DevicePlatform: item.DevicePlatform,
		DeviceVersion:  item.DeviceVersion,
		AppVersion:     item.AppVersion,
		CreatedAt:      item.CreatedAt,
	}
}

func formatID(id int64) string {
	return strconv.FormatInt(id, 10)
}

func parseImageURLs(value string) []string {
	if value == "" {
		return []string{}
	}
	var result []string
	if err := json.Unmarshal([]byte(value), &result); err != nil {
		return []string{}
	}
	return result
}
