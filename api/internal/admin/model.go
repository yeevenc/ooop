package admin

import "time"

const AdminStatusEnabled = 1

type AdminUser struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string    `gorm:"size:64;not null;uniqueIndex" json:"username"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Status       int       `gorm:"not null;default:1" json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type PublicAdminUser struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Status    int       `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func ToPublicAdminUser(item AdminUser) PublicAdminUser {
	return PublicAdminUser{
		ID:        item.ID,
		Username:  item.Username,
		Status:    item.Status,
		CreatedAt: item.CreatedAt,
	}
}
