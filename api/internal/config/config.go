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
	Jiguang  JiguangConfig
	Qiniu    QiniuConfig
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
	Secret         string
	AccessTokenTTL time.Duration
	Issuer         string
}

type AuthConfig struct {
	CodeSecret string
}

type AliyunConfig struct {
	AccessKeyID     string
	AccessKeySecret string
	Mobile          AliyunMobileConfig
	SMS             AliyunSMSConfig
	IDCard          AliyunIDCardConfig
}

type AliyunMobileConfig struct {
	Endpoint string
}

type AliyunSMSConfig struct {
	Endpoint                    string
	SignName                    string
	LoginTemplateCode           string
	ChangePhoneTemplateCode     string
	ResetPasswordTemplateCode   string
	BindNewPhoneTemplateCode    string
	VerifyBindPhoneTemplateCode string
	ValidSeconds                int
	CodeLength                  int
	IntervalSeconds             int
	DuplicatePolicy             int
	SchemeName                  string
}

type AliyunIDCardConfig struct {
	Endpoint  string
	AppCode   string
	AppKey    string
	AppSecret string
}

type JiguangConfig struct {
	AppKey       string
	MasterSecret string
	VerifyURL    string
	PushURL      string
	PrivateKey   string
}

type QiniuConfig struct {
	AccessKey string
	SecretKey string
	Bucket    string
	Domain    string
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
			Secret:         getEnv("JWT_SECRET", "change-me-before-production"),
			AccessTokenTTL: getDurationEnv("JWT_ACCESS_TOKEN_TTL", 30*24*time.Hour),
			Issuer:         getEnv("JWT_ISSUER", "ooop-admin-api"),
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
				Endpoint:                    getEnv("ALIYUN_SMS_ENDPOINT", "dypnsapi.aliyuncs.com"),
				SignName:                    getEnv("ALIYUN_SMS_SIGN_NAME", "速通互联验证码"),
				LoginTemplateCode:           getEnv("ALIYUN_SMS_LOGIN_TEMPLATE_CODE", "100001"),
				ChangePhoneTemplateCode:     getEnv("ALIYUN_SMS_CHANGE_PHONE_TEMPLATE_CODE", "100002"),
				ResetPasswordTemplateCode:   getEnv("ALIYUN_SMS_RESET_PASSWORD_TEMPLATE_CODE", "100003"),
				BindNewPhoneTemplateCode:    getEnv("ALIYUN_SMS_BIND_NEW_PHONE_TEMPLATE_CODE", "100004"),
				VerifyBindPhoneTemplateCode: getEnv("ALIYUN_SMS_VERIFY_BIND_PHONE_TEMPLATE_CODE", "100005"),
				ValidSeconds:                getIntEnv("ALIYUN_SMS_VALID_SECONDS", 300),
				CodeLength:                  getIntEnv("ALIYUN_SMS_CODE_LENGTH", 6),
				IntervalSeconds:             getIntEnv("ALIYUN_SMS_INTERVAL_SECONDS", 60),
				DuplicatePolicy:             getIntEnv("ALIYUN_SMS_DUPLICATE_POLICY", 1),
				SchemeName:                  getEnv("ALIYUN_SMS_SCHEME_NAME", ""),
			},
			IDCard: AliyunIDCardConfig{
				Endpoint:  getEnv("ALIYUN_ID_CARD_ENDPOINT", "https://kzidcardv1.market.alicloudapi.com/api-mall/api/id_card/check"),
				AppCode:   getEnv("ALIYUN_ID_CARD_APP_CODE", ""),
				AppKey:    getEnv("ALIYUN_ID_CARD_APP_KEY", ""),
				AppSecret: getEnv("ALIYUN_ID_CARD_APP_SECRET", ""),
			},
		},
		Jiguang: JiguangConfig{
			AppKey:       getEnv("JIGUANG_APP_KEY", ""),
			MasterSecret: getEnv("JIGUANG_MASTER_SECRET", ""),
			VerifyURL:    getEnv("JIGUANG_VERIFY_URL", "https://api.verification.jpush.cn/v1/web/loginTokenVerify"),
			PushURL:      getEnv("JIGUANG_PUSH_URL", "https://api.jpush.cn/v3/push"),
			PrivateKey:   normalizePrivateKey(getEnv("JIGUANG_PRIVATE_KEY", "")),
		},
		Qiniu: QiniuConfig{
			AccessKey: getEnv("QINIU_ACCESS_KEY", ""),
			SecretKey: getEnv("QINIU_SECRET_KEY", ""),
			Bucket:    getEnv("QINIU_BUCKET", "ooop"),
			Domain:    getEnv("QINIU_DOMAIN", "https://source.ooopai.cn"),
		},
	}
}

func normalizePrivateKey(value string) string {
	return strings.ReplaceAll(strings.TrimSpace(value), `\n`, "\n")
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

func getIntEnv(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getBoolEnv(key string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes" || value == "on"
}
