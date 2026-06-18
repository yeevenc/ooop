package main

import (
	"ooop-admin-api/internal/config"
	"ooop-admin-api/internal/database"
	"ooop-admin-api/internal/logger"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg.Database)
	if err != nil {
		logger.Fatalf("数据库连接失败: %v", err)
	}

	if err := database.AutoMigrate(db); err != nil {
		logger.Fatalf("数据表迁移失败: %v", err)
	}

	logger.Infof("数据表迁移完成")
}
