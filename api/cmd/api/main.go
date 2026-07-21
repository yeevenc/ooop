package main

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"ooop-admin-api/internal/activity"
	"ooop-admin-api/internal/admin"
	"ooop-admin-api/internal/auth"
	"ooop-admin-api/internal/chat"
	"ooop-admin-api/internal/config"
	"ooop-admin-api/internal/contentmoderation"
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
	chatRepo := chat.NewGormRepository(db)
	feedbackRepo := feedback.NewGormRepository(db)

	aliyunClient := provider.NewAliyunRPCClient(cfg.Aliyun.AccessKeyID, cfg.Aliyun.AccessKeySecret)
	mobileVerifier := provider.NewJiguangMobileVerifier(cfg.Jiguang)
	jpushPusher := provider.NewJiguangPusher(cfg.Jiguang)
	harmonyPusher := provider.NewHarmonyPusher(cfg.HarmonyPush)
	if cfg.HarmonyPush.ServiceAccountFile == "" {
		logger.Warnf("鸿蒙推送未配置：请在 .env 设置 HARMONY_PUSH_SERVICE_ACCOUNT_FILE 为 AGC 服务账号 JSON 绝对路径")
	} else if err := harmonyPusher.ValidateServiceAccount(); err != nil {
		logger.Warnf("鸿蒙推送 Service Account 预检失败: %v", err)
	} else {
		logger.Infof("鸿蒙推送 Service Account 预检通过: %s", cfg.HarmonyPush.ServiceAccountFile)
	}
	pushSender := provider.NewDualChannelPusher(jpushPusher, harmonyPusher)
	smsSender := provider.NewAliyunSMSSender(aliyunClient, cfg.Aliyun.SMS)
	realNameVerifier := provider.NewAliyunIDCardVerifier(cfg.Aliyun.IDCard)
	contentChecker, err := contentmoderation.NewChecker(cfg.ContentModeration.BlockedWords)
	if err != nil {
		logger.Fatalf("敏感词过滤器初始化失败: %v", err)
	}

	authService := user.NewAuthService(user.AuthServiceOptions{
		Users:            userRepo,
		Stats:            activityRepo,
		LoginCodes:       codeRepo,
		PasswordHasher:   passwordHasher,
		TokenManager:     tokenManager,
		MobileVerifier:   mobileVerifier,
		SMSSender:        smsSender,
		RealNameVerifier: realNameVerifier,
		ContentChecker:   contentChecker,
		CodeSecret:       cfg.Auth.CodeSecret,
	})
	adminService := admin.NewService(adminRepo, passwordHasher, adminTokenManager)
	activityService := activity.NewService(activityRepo, userRepo, contentChecker)
	messageService := message.NewService(messageRepo, pushSender, userRepo)
	chatService := chat.NewService(chatRepo, userRepo, contentChecker, cfg.Chat.MessageRetention)
	chatReportService := chat.NewReportService(chatRepo, chatRepo, userRepo, messageService)
	chatWorker := chat.NewWorker(chatRepo, userRepo, pushSender, chat.WorkerOptions{
		PushInterval:    cfg.Chat.PushInterval,
		CleanupInterval: cfg.Chat.CleanupInterval,
		BatchSize:       cfg.Chat.PushBatchSize,
		Workers:         cfg.Chat.PushWorkers,
		Retention:       cfg.Chat.MessageRetention,
		PushCategory:    cfg.Chat.PushCategory,
	})
	chatWorker.Start(context.Background())
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
	// APP 鉴权统一注入 authService：封禁用户（含已登录）返回 403002，不落业务逻辑
	user.NewHandler(authService, tokenManager).Register(api)
	activity.NewHandler(activityService, tokenManager, authService).Register(api)
	message.NewHandler(messageService, tokenManager, authService).Register(api)
	chat.NewHandler(chatService, chatReportService, tokenManager, authService).Register(api)
	feedback.NewHandler(feedbackService, tokenManager, adminTokenManager, authService).Register(api)
	upload.NewHandlerWithConfig(cfg.Qiniu).Register(api)
	admin.NewHandler(adminService, authService, activityService, chatReportService, adminTokenManager).Register(api)

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
