package user

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"ooop-admin-api/internal/auth"
	"ooop-admin-api/internal/provider"
)

const LoginCodeSceneLogin = "login"

var (
	ErrInvalidPhone    = errors.New("手机号格式不正确")
	ErrInvalidPassword = errors.New("密码长度不能少于 8 位")
	ErrInvalidAccount  = errors.New("账号或密码错误")
	ErrInvalidCode     = errors.New("验证码错误或已过期")
	ErrDisabledUser    = errors.New("账号已被禁用")
	ErrPhoneExists     = errors.New("手机号已注册")
)

var phonePattern = regexp.MustCompile(`^1[3-9]\d{9}$`)

type AuthServiceOptions struct {
	Users          UserRepository
	LoginCodes     LoginCodeRepository
	RefreshTokens  RefreshTokenRepository
	PasswordHasher auth.PasswordHasher
	TokenManager   *auth.TokenManager
	MobileVerifier provider.MobileVerifier
	SMSSender      provider.SMSSender
	CodeSecret     string
}

type AuthService struct {
	users          UserRepository
	loginCodes     LoginCodeRepository
	refreshTokens  RefreshTokenRepository
	passwordHasher auth.PasswordHasher
	tokenManager   *auth.TokenManager
	mobileVerifier provider.MobileVerifier
	smsSender      provider.SMSSender
	codeSecret     string
}

type LoginResult struct {
	User   PublicUser     `json:"user"`
	Tokens auth.TokenPair `json:"tokens"`
}

type UserListQuery struct {
	Page     int
	PageSize int
	Keyword  string
	Status   *int
}

type UserListResult struct {
	List     []PublicUser `json:"list"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
}

type ClientMeta struct {
	Platform string
	DeviceNo string
}

func NewAuthService(opts AuthServiceOptions) *AuthService {
	return &AuthService{
		users:          opts.Users,
		loginCodes:     opts.LoginCodes,
		refreshTokens:  opts.RefreshTokens,
		passwordHasher: opts.PasswordHasher,
		tokenManager:   opts.TokenManager,
		mobileVerifier: opts.MobileVerifier,
		smsSender:      opts.SMSSender,
		codeSecret:     opts.CodeSecret,
	}
}

func (s *AuthService) AliyunMobileLogin(ctx context.Context, accessToken string, meta ClientMeta) (LoginResult, error) {
	phone, err := s.mobileVerifier.Verify(ctx, accessToken)
	if err != nil {
		return LoginResult{}, err
	}
	if !isValidPhone(phone) {
		return LoginResult{}, ErrInvalidPhone
	}
	return s.loginOrCreateByPhone(ctx, phone, RegisterSourceAliyunMobile, meta)
}

func (s *AuthService) SendLoginCode(ctx context.Context, phone string) error {
	phone = normalizePhone(phone)
	if !isValidPhone(phone) {
		return ErrInvalidPhone
	}

	code := randomCode()
	item := &LoginCode{
		Phone:     phone,
		Scene:     LoginCodeSceneLogin,
		CodeHash:  s.hashCode(phone, code),
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	if err := s.loginCodes.Create(ctx, item); err != nil {
		return err
	}
	return s.smsSender.SendCode(ctx, phone, code)
}

func (s *AuthService) MobileCodeLogin(ctx context.Context, phone string, code string, meta ClientMeta) (LoginResult, error) {
	phone = normalizePhone(phone)
	code = strings.TrimSpace(code)
	if !isValidPhone(phone) {
		return LoginResult{}, ErrInvalidPhone
	}
	if code == "" {
		return LoginResult{}, ErrInvalidCode
	}

	item, err := s.loginCodes.FindValid(ctx, phone, LoginCodeSceneLogin, s.hashCode(phone, code), time.Now())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return LoginResult{}, ErrInvalidCode
		}
		return LoginResult{}, err
	}
	if err := s.loginCodes.MarkUsed(ctx, item.ID, time.Now()); err != nil {
		return LoginResult{}, err
	}
	return s.loginOrCreateByPhone(ctx, phone, RegisterSourceMobileCode, meta)
}

func (s *AuthService) RegisterByPassword(ctx context.Context, phone string, username string, password string, meta ClientMeta) (LoginResult, error) {
	phone = normalizePhone(phone)
	username = strings.TrimSpace(username)
	if !isValidPhone(phone) {
		return LoginResult{}, ErrInvalidPhone
	}
	if len(password) < 8 {
		return LoginResult{}, ErrInvalidPassword
	}

	_, err := s.users.FindByPhone(ctx, phone)
	if err == nil {
		return LoginResult{}, ErrPhoneExists
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		return LoginResult{}, err
	}

	passwordHash, err := s.passwordHasher.Hash(password)
	if err != nil {
		return LoginResult{}, err
	}

	item := User{
		Phone:          phone,
		Nickname:       username,
		Status:         UserStatusEnabled,
		RegisterSource: RegisterSourcePassword,
		PasswordHash:   passwordHash,
		Platform:       normalizeMetaValue(meta.Platform),
		DeviceNo:       normalizeMetaValue(meta.DeviceNo),
	}
	if username != "" {
		item.Username = &username
	}

	if err := s.users.Create(ctx, &item); err != nil {
		return LoginResult{}, err
	}
	return s.issueLoginResult(ctx, item, meta)
}

func (s *AuthService) PasswordLogin(ctx context.Context, account string, password string, meta ClientMeta) (LoginResult, error) {
	account = strings.TrimSpace(account)
	if account == "" || password == "" {
		return LoginResult{}, ErrInvalidAccount
	}

	item, err := s.users.FindByUsernameOrPhone(ctx, account)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return LoginResult{}, ErrInvalidAccount
		}
		return LoginResult{}, err
	}
	if item.Status != UserStatusEnabled {
		return LoginResult{}, ErrDisabledUser
	}
	if item.PasswordHash == "" || !s.passwordHasher.Compare(item.PasswordHash, password) {
		return LoginResult{}, ErrInvalidAccount
	}
	return s.issueLoginResult(ctx, item, meta)
}

func (s *AuthService) SetPassword(ctx context.Context, userID int64, username string, password string) (PublicUser, error) {
	username = strings.TrimSpace(username)
	if len(password) < 8 {
		return PublicUser{}, ErrInvalidPassword
	}
	hash, err := s.passwordHasher.Hash(password)
	if err != nil {
		return PublicUser{}, err
	}
	if err := s.users.UpdatePassword(ctx, userID, username, hash); err != nil {
		return PublicUser{}, err
	}
	item, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return PublicUser{}, err
	}
	return ToPublicUser(item), nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (auth.TokenPair, error) {
	claims, err := s.tokenManager.Parse(refreshToken, auth.TokenTypeRefresh)
	if err != nil {
		return auth.TokenPair{}, err
	}

	tokenHash := s.tokenManager.RefreshTokenHash(claims.TokenID)
	item, err := s.refreshTokens.FindValid(ctx, tokenHash, time.Now())
	if err != nil {
		return auth.TokenPair{}, err
	}
	if item.UserID != claims.UserID {
		return auth.TokenPair{}, auth.ErrInvalidToken
	}

	if err := s.refreshTokens.Revoke(ctx, tokenHash, time.Now()); err != nil {
		return auth.TokenPair{}, err
	}
	tokens, refreshID, expiresAt, err := s.tokenManager.NewTokenPair(claims.UserID)
	if err != nil {
		return auth.TokenPair{}, err
	}
	if err := s.refreshTokens.Create(ctx, &RefreshToken{
		UserID:    claims.UserID,
		TokenHash: s.tokenManager.RefreshTokenHash(refreshID),
		ExpiresAt: expiresAt,
	}); err != nil {
		return auth.TokenPair{}, err
	}
	return tokens, nil
}

func (s *AuthService) Profile(ctx context.Context, userID int64) (PublicUser, error) {
	item, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return PublicUser{}, err
	}
	return ToPublicUser(item), nil
}

func (s *AuthService) ListUsers(ctx context.Context, query UserListQuery) (UserListResult, error) {
	query.Keyword = strings.TrimSpace(query.Keyword)
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 10
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}

	items, total, err := s.users.List(ctx, query)
	if err != nil {
		return UserListResult{}, err
	}

	list := make([]PublicUser, 0, len(items))
	for _, item := range items {
		list = append(list, ToPublicUser(item))
	}

	return UserListResult{
		List:     list,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

func (s *AuthService) loginOrCreateByPhone(ctx context.Context, phone string, source string, meta ClientMeta) (LoginResult, error) {
	item, err := s.users.FindByPhone(ctx, phone)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return LoginResult{}, err
	}
	if errors.Is(err, ErrNotFound) {
		item = User{
			Phone:          phone,
			Status:         UserStatusEnabled,
			RegisterSource: source,
			Platform:       normalizeMetaValue(meta.Platform),
			DeviceNo:       normalizeMetaValue(meta.DeviceNo),
		}
		if err := s.users.Create(ctx, &item); err != nil {
			return LoginResult{}, err
		}
	}
	return s.issueLoginResult(ctx, item, meta)
}

func (s *AuthService) issueLoginResult(ctx context.Context, item User, meta ClientMeta) (LoginResult, error) {
	if item.Status != UserStatusEnabled {
		return LoginResult{}, ErrDisabledUser
	}

	now := time.Now()
	if err := s.users.TouchLastLogin(ctx, item.ID, now, meta); err != nil {
		return LoginResult{}, err
	}
	item.LastLoginAt = &now
	if meta.Platform != "" {
		item.Platform = normalizeMetaValue(meta.Platform)
	}
	if meta.DeviceNo != "" {
		item.DeviceNo = normalizeMetaValue(meta.DeviceNo)
	}

	tokens, refreshID, expiresAt, err := s.tokenManager.NewTokenPair(item.ID)
	if err != nil {
		return LoginResult{}, err
	}
	if err := s.refreshTokens.Create(ctx, &RefreshToken{
		UserID:    item.ID,
		TokenHash: s.tokenManager.RefreshTokenHash(refreshID),
		ExpiresAt: expiresAt,
	}); err != nil {
		return LoginResult{}, err
	}

	return LoginResult{
		User:   ToPublicUser(item),
		Tokens: tokens,
	}, nil
}

func (s *AuthService) hashCode(phone string, code string) string {
	sum := sha256.Sum256([]byte(phone + ":" + code + ":" + s.codeSecret))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func normalizePhone(phone string) string {
	return strings.TrimSpace(phone)
}

func normalizeMetaValue(value string) string {
	return strings.TrimSpace(value)
}

func isValidPhone(phone string) bool {
	return phonePattern.MatchString(phone)
}

func randomCode() string {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	}
	value := int(bytes[0])<<24 | int(bytes[1])<<16 | int(bytes[2])<<8 | int(bytes[3])
	if value < 0 {
		value = -value
	}
	return fmt.Sprintf("%06d", value%1000000)
}
