package database

import (
	"context"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"ooop-admin-api/internal/activity"
	"ooop-admin-api/internal/admin"
	"ooop-admin-api/internal/config"
	"ooop-admin-api/internal/feedback"
	"ooop-admin-api/internal/message"
	"ooop-admin-api/internal/user"
)

func Connect(cfg config.DatabaseConfig) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&admin.AdminUser{},
		&user.User{},
		&user.LoginCode{},
		&activity.ActivityCategory{},
		&activity.Activity{},
		&activity.ActivityParticipant{},
		&message.UserMessage{},
		&feedback.Feedback{},
	); err != nil {
		return err
	}

	// APP 用户 ID 从 3000 起步，只影响后续新增用户，不修改历史数据。
	return db.Exec("ALTER TABLE users AUTO_INCREMENT = 3000").Error
}

func SeedDefaultActivityCategories(db *gorm.DB) error {
	repo := activity.NewGormRepository(db)
	return activity.EnsureDefaultCategories(context.Background(), repo)
}
