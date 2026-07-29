package database

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"ooop-admin-api/internal/activity"
	"ooop-admin-api/internal/admin"
	"ooop-admin-api/internal/chat"
	"ooop-admin-api/internal/config"
	"ooop-admin-api/internal/feedback"
	"ooop-admin-api/internal/message"
	"ooop-admin-api/internal/user"
)

var ErrLegacyActivityCategorySchema = errors.New(
	"检测到活动分类仍使用英文 ID，请先执行 docs/sql/20260729_migrate_activity_category_numeric_id.sql，再重新运行数据表迁移",
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
	legacyCategorySchema, err := hasLegacyActivityCategorySchema(db)
	if err != nil {
		return err
	}
	if legacyCategorySchema {
		// 英文 ID 到数字 ID 的转换必须通过映射 SQL 完成，禁止交给 GORM 原地修改字段类型。
		return ErrLegacyActivityCategorySchema
	}

	if err := db.AutoMigrate(
		&admin.AdminUser{},
		&user.User{},
		&user.LoginCode{},
		&activity.ActivityCategory{},
		// 旧版 App 分类 ID 兼容表，完成兼容下线流程前必须保留。
		&activity.ActivityCategoryLegacyID{},
		&activity.Activity{},
		&activity.ImageAuditTask{},
		&activity.ActivityFavorite{},
		&activity.ActivityParticipant{},
		&message.UserMessage{},
		&chat.Conversation{},
		&chat.Message{},
		&chat.PushTask{},
		&chat.ChatReport{},
		&feedback.Feedback{},
	); err != nil {
		return err
	}

	// APP 用户 ID 从 3000 起步，只影响后续新增用户，不修改历史数据。
	return db.Exec("ALTER TABLE users AUTO_INCREMENT = 3000").Error
}

func hasLegacyActivityCategorySchema(db *gorm.DB) (bool, error) {
	if !db.Migrator().HasTable(&activity.ActivityCategory{}) {
		return false, nil
	}

	columns, err := db.Migrator().ColumnTypes(&activity.ActivityCategory{})
	if err != nil {
		return false, err
	}
	for _, column := range columns {
		if !strings.EqualFold(column.Name(), "id") {
			continue
		}
		return !isIntegerColumnType(column.DatabaseTypeName()), nil
	}
	return false, fmt.Errorf("activity_categories 表缺少 id 字段")
}

func isIntegerColumnType(dataType string) bool {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "tinyint", "smallint", "mediumint", "int", "integer", "bigint":
		return true
	default:
		return false
	}
}

func SeedDefaultActivityCategories(db *gorm.DB) error {
	repo := activity.NewGormRepository(db)
	return activity.EnsureDefaultCategories(context.Background(), repo)
}
