package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	HTTP     HTTPConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Auth     AuthConfig
	Aliyun   AliyunConfig
}

type AppConfig struct {
	Env string
}

type HTTPConfig struct {
	Host              string
	Port              string
	AllowOrigins      []string
	ReadHeaderTimeout time.Duration
}

func (c HTTPConfig) Addr() string {
	return c.Host + ":" + c.Port
}

type DatabaseConfig struct {
	DSN         string
	AutoMigrate bool
}

type JWTConfig struct {
	Secret             string
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
	RefreshTokenPepper string
	Issuer             string
}

type AuthConfig struct {
	CodeSecret string
}

type AliyunConfig struct {
	AccessKeyID     string
	AccessKeySecret string
	Mobile          AliyunMobileConfig
	SMS             AliyunSMSConfig
}

type AliyunMobileConfig struct {
	Endpoint string
}

type AliyunSMSConfig struct {
	Endpoint     string
	SignName     string
	TemplateCode string
}

func Load() Config {
	_ = godotenv.Load()

	return Config{
		App: AppConfig{
			Env: getEnv("APP_ENV", "development"),
		},
		HTTP: HTTPConfig{
			Host:              getEnv("HTTP_HOST", "0.0.0.0"),
			Port:              getEnv("HTTP_PORT", "8080"),
			AllowOrigins:      getListEnv("HTTP_ALLOW_ORIGINS", "*"),
			ReadHeaderTimeout: getDurationEnv("HTTP_READ_HEADER_TIMEOUT", 5*time.Second),
		},
		Database: DatabaseConfig{
			DSN:         getEnv("MYSQL_DSN", "root:root@tcp(127.0.0.1:3306)/ooop_admin?charset=utf8mb4&parseTime=True&loc=Local"),
			AutoMigrate: getBoolEnv("MYSQL_AUTO_MIGRATE", false),
		},
		JWT: JWTConfig{
			Secret:             getEnv("JWT_SECRET", "change-me-before-production"),
			AccessTokenTTL:     getDurationEnv("JWT_ACCESS_TOKEN_TTL", 2*time.Hour),
			RefreshTokenTTL:    getDurationEnv("JWT_REFRESH_TOKEN_TTL", 30*24*time.Hour),
			RefreshTokenPepper: getEnv("JWT_REFRESH_TOKEN_PEPPER", "change-me-refresh-pepper"),
			Issuer:             getEnv("JWT_ISSUER", "ooop-admin-api"),
		},
		Auth: AuthConfig{
			CodeSecret: getEnv("AUTH_CODE_SECRET", "change-me-code-secret"),
		},
		Aliyun: AliyunConfig{
			AccessKeyID:     getEnv("ALIYUN_ACCESS_KEY_ID", ""),
			AccessKeySecret: getEnv("ALIYUN_ACCESS_KEY_SECRET", ""),
			Mobile: AliyunMobileConfig{
				Endpoint: getEnv("ALIYUN_MOBILE_ENDPOINT", "dypnsapi.aliyuncs.com"),
			},
			SMS: AliyunSMSConfig{
				Endpoint:     getEnv("ALIYUN_SMS_ENDPOINT", "dysmsapi.aliyuncs.com"),
				SignName:     getEnv("ALIYUN_SMS_SIGN_NAME", ""),
				TemplateCode: getEnv("ALIYUN_SMS_TEMPLATE_CODE", ""),
			},
		},
	}
}

func getEnv(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getListEnv(key string, fallback string) []string {
	value := getEnv(key, fallback)
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err == nil {
		return parsed
	}
	seconds, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}

func getBoolEnv(key string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes" || value == "on"
}
