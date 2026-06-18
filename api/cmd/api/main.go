package main

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"ooop-admin-api/internal/admin"
	"ooop-admin-api/internal/auth"
	"ooop-admin-api/internal/config"
	"ooop-admin-api/internal/database"
	"ooop-admin-api/internal/httpx"
	"ooop-admin-api/internal/logger"
	"ooop-admin-api/internal/provider"
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
	refreshTokenRepo := user.NewGormRefreshTokenRepository(db)

	aliyunClient := provider.NewAliyunRPCClient(cfg.Aliyun.AccessKeyID, cfg.Aliyun.AccessKeySecret)
	mobileVerifier := provider.NewAliyunMobileVerifier(aliyunClient, cfg.Aliyun.Mobile)
	smsSender := provider.NewAliyunSMSSender(aliyunClient, cfg.Aliyun.SMS)

	authService := user.NewAuthService(user.AuthServiceOptions{
		Users:          userRepo,
		LoginCodes:     codeRepo,
		RefreshTokens:  refreshTokenRepo,
		PasswordHasher: passwordHasher,
		TokenManager:   tokenManager,
		MobileVerifier: mobileVerifier,
		SMSSender:      smsSender,
		CodeSecret:     cfg.Auth.CodeSecret,
	})
	adminService := admin.NewService(adminRepo, passwordHasher, adminTokenManager)
	// 后台账号独立写入 admin_users，不再污染 APP 用户表。
	if _, err := adminService.EnsureDefaultAdmin(context.Background(), "admin", "admin"); err != nil {
		logger.Warnf("默认管理员初始化跳过: %v", err)
	} else {
		logger.Infof("默认管理员账号已就绪: admin")
	}

	router := gin.New()
	// 请求日志统一走彩色输出，便于本地排查接口状态和耗时。
	router.Use(logger.HTTPLogger(), gin.Recovery(), httpx.CORS(cfg.HTTP.AllowOrigins))

	router.GET("/health", func(c *gin.Context) {
		httpx.OK(c, gin.H{"status": "ok"})
	})

	api := router.Group("/api/v1")
	user.NewHandler(authService, tokenManager).Register(api)
	admin.NewHandler(adminService, authService, adminTokenManager).Register(api)

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
