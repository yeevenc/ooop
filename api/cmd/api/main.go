package main

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"ooop-admin-api/internal/activity"
	"ooop-admin-api/internal/admin"
	"ooop-admin-api/internal/auth"
	"ooop-admin-api/internal/config"
	"ooop-admin-api/internal/database"
	"ooop-admin-api/internal/feedback"
	"ooop-admin-api/internal/httpx"
	"ooop-admin-api/internal/legal"
	"ooop-admin-api/internal/logger"
	"ooop-admin-api/internal/message"
	"ooop-admin-api/internal/provider"
	"ooop-admin-api/internal/upload"
	"ooop-admin-api/internal/user"
)

func main() {
	cfg := config.Load()

	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.Connect(cfg.Database)
	if err != nil {
		logger.Fatalf("数据库连接失败: %v", err)
	}

	if cfg.Database.AutoMigrate {
		if err := database.AutoMigrate(db); err != nil {
			logger.Fatalf("数据库迁移失败: %v", err)
		}
	}

	tokenManager := auth.NewTokenManager(cfg.JWT)
	adminJWT := cfg.JWT
	adminJWT.Issuer = cfg.JWT.Issuer + "-admin"
	adminTokenManager := auth.NewTokenManager(adminJWT)
	passwordHasher := auth.NewBcryptHasher()

	adminRepo := admin.NewGormRepository(db)
	userRepo := user.NewGormUserRepository(db)
	codeRepo := user.NewGormLoginCodeRepository(db)
	activityRepo := activity.NewGormRepository(db)
	messageRepo := message.NewGormRepository(db)
	feedbackRepo := feedback.NewGormRepository(db)

	aliyunClient := provider.NewAliyunRPCClient(cfg.Aliyun.AccessKeyID, cfg.Aliyun.AccessKeySecret)
	mobileVerifier := provider.NewJiguangMobileVerifier(cfg.Jiguang)
	jpushPusher := provider.NewJiguangPusher(cfg.Jiguang)
	harmonyPusher := provider.NewHarmonyPusher(cfg.HarmonyPush)
	pushSender := provider.NewDualChannelPusher(jpushPusher, harmonyPusher)
	smsSender := provider.NewAliyunSMSSender(aliyunClient, cfg.Aliyun.SMS)
	realNameVerifier := provider.NewAliyunIDCardVerifier(cfg.Aliyun.IDCard)

	authService := user.NewAuthService(user.AuthServiceOptions{
		Users:            userRepo,
		Stats:            activityRepo,
		LoginCodes:       codeRepo,
		PasswordHasher:   passwordHasher,
		TokenManager:     tokenManager,
		MobileVerifier:   mobileVerifier,
		SMSSender:        smsSender,
		RealNameVerifier: realNameVerifier,
		CodeSecret:       cfg.Auth.CodeSecret,
	})
	adminService := admin.NewService(adminRepo, passwordHasher, adminTokenManager)
	activityService := activity.NewService(activityRepo, userRepo)
	messageService := message.NewService(messageRepo, pushSender, userRepo)
	feedbackService := feedback.NewService(feedbackRepo, authService)
	activityService.SetReviewNotifier(messageService)
	// 后台账号独立写入 admin_users，不再污染 APP 用户表。
	if _, err := adminService.EnsureDefaultAdmin(context.Background(), "admin", "admin"); err != nil {
		logger.Warnf("默认管理员初始化跳过: %v", err)
	} else {
		logger.Infof("默认管理员账号已就绪: admin")
	}
	if err := activityService.EnsureDefaultCategories(context.Background()); err != nil {
		logger.Warnf("默认活动分类初始化跳过: %v", err)
	} else {
		logger.Infof("默认活动分类已就绪")
	}

	router := gin.New()
	// 请求日志统一走彩色输出，便于本地排查接口状态和耗时。
	router.Use(logger.HTTPLogger(), gin.Recovery(), httpx.CORS(cfg.HTTP.AllowOrigins))

	router.GET("/health", func(c *gin.Context) {
		httpx.OK(c, gin.H{"status": "ok"})
	})
	router.Static("/uploads", "./uploads")
	legal.NewHandler().Register(router)

	api := router.Group("/api/v1")
	user.NewHandler(authService, tokenManager).Register(api)
	activity.NewHandler(activityService, tokenManager).Register(api)
	message.NewHandler(messageService, tokenManager).Register(api)
	feedback.NewHandler(feedbackService, tokenManager, adminTokenManager).Register(api)
	upload.NewHandlerWithConfig(cfg.Qiniu).Register(api)
	admin.NewHandler(adminService, authService, activityService, adminTokenManager).Register(api)

	server := &http.Server{
		Addr:              cfg.HTTP.Addr(),
		Handler:           router,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
	}

	logger.Infof("服务启动: %s", cfg.HTTP.Addr())
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("服务启动失败: %v", err)
	}
}
