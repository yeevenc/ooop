package user

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"ooop-admin-api/internal/auth"
	"ooop-admin-api/internal/config"
)

func TestAliyunMobileLoginCreatesUserAndReturnsTokens(t *testing.T) {
	service := newTestAuthService()

	result, err := service.AliyunMobileLogin(context.Background(), "aliyun-token", ClientMeta{Platform: "ios", DeviceNo: "device-001"})
	if err != nil {
		t.Fatalf("AliyunMobileLogin() error = %v", err)
	}

	if result.User.Phone != "13800138000" {
		t.Fatalf("phone = %s, want 13800138000", result.User.Phone)
	}
	if result.User.RegisterSource != RegisterSourceAliyunMobile {
		t.Fatalf("register source = %s", result.User.RegisterSource)
	}
	if result.Tokens.AccessToken == "" || result.Tokens.RefreshToken == "" {
		t.Fatalf("token pair should not be empty")
	}
	if result.User.Platform != "ios" || result.User.DeviceNo != "device-001" {
		t.Fatalf("client meta = %s/%s, want ios/device-001", result.User.Platform, result.User.DeviceNo)
	}
}

func TestPasswordLoginSupportsUsernameOrPhone(t *testing.T) {
	service := newTestAuthService()
	ctx := context.Background()

	loginResult, err := service.AliyunMobileLogin(ctx, "aliyun-token", ClientMeta{})
	if err != nil {
		t.Fatalf("AliyunMobileLogin() error = %v", err)
	}
	if _, err := service.SetPassword(ctx, loginResult.User.ID, "test_user", "password123"); err != nil {
		t.Fatalf("SetPassword() error = %v", err)
	}

	phoneLogin, err := service.PasswordLogin(ctx, "13800138000", "password123", ClientMeta{})
	if err != nil {
		t.Fatalf("PasswordLogin(phone) error = %v", err)
	}
	if phoneLogin.User.ID != loginResult.User.ID {
		t.Fatalf("phone login user id = %d, want %d", phoneLogin.User.ID, loginResult.User.ID)
	}

	usernameLogin, err := service.PasswordLogin(ctx, "test_user", "password123", ClientMeta{})
	if err != nil {
		t.Fatalf("PasswordLogin(username) error = %v", err)
	}
	if usernameLogin.User.ID != loginResult.User.ID {
		t.Fatalf("username login user id = %d, want %d", usernameLogin.User.ID, loginResult.User.ID)
	}
}

func TestRegisterByPasswordCreatesUserAndRejectsDuplicatePhone(t *testing.T) {
	service := newTestAuthService()
	ctx := context.Background()

	result, err := service.RegisterByPassword(ctx, "13700137000", "new_user", "password123", ClientMeta{Platform: "android"})
	if err != nil {
		t.Fatalf("RegisterByPassword() error = %v", err)
	}
	if result.User.Phone != "13700137000" {
		t.Fatalf("phone = %s, want 13700137000", result.User.Phone)
	}
	if result.User.Username != "new_user" {
		t.Fatalf("username = %s, want new_user", result.User.Username)
	}
	if result.User.RegisterSource != RegisterSourcePassword {
		t.Fatalf("register source = %s, want %s", result.User.RegisterSource, RegisterSourcePassword)
	}
	if result.User.Nickname != "new_user" || result.User.Platform != "android" {
		t.Fatalf("profile = %s/%s, want new_user/android", result.User.Nickname, result.User.Platform)
	}

	if _, err := service.RegisterByPassword(ctx, "13700137000", "other_user", "password123", ClientMeta{}); !errors.Is(err, ErrPhoneExists) {
		t.Fatalf("duplicate RegisterByPassword() error = %v, want ErrPhoneExists", err)
	}
}

func TestRegisterByPasswordRejectsReservedUsername(t *testing.T) {
	service := newTestAuthService()

	_, err := service.RegisterByPassword(context.Background(), "13500135000", ReservedAdminUsername, "password123", ClientMeta{})
	if !errors.Is(err, ErrReservedUsername) {
		t.Fatalf("RegisterByPassword() error = %v, want ErrReservedUsername", err)
	}
}

func TestMobileCodeLoginConsumesValidCode(t *testing.T) {
	service := newTestAuthService()
	codeRepo := service.loginCodes.(*memoryLoginCodeRepository)
	ctx := context.Background()

	codeRepo.seed("13900139000", LoginCodeSceneLogin, service.hashCode("13900139000", "123456"))

	result, err := service.MobileCodeLogin(ctx, "13900139000", "123456", ClientMeta{})
	if err != nil {
		t.Fatalf("MobileCodeLogin() error = %v", err)
	}
	if result.User.Phone != "13900139000" {
		t.Fatalf("phone = %s, want 13900139000", result.User.Phone)
	}

	if _, err := service.MobileCodeLogin(ctx, "13900139000", "123456", ClientMeta{}); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("second MobileCodeLogin() error = %v, want ErrInvalidCode", err)
	}
}

func TestListUsersReturnsPagedPublicUsers(t *testing.T) {
	service := newTestAuthService()
	ctx := context.Background()

	if _, err := service.RegisterByPassword(ctx, "13700137000", "first_user", "password123", ClientMeta{}); err != nil {
		t.Fatalf("RegisterByPassword(first) error = %v", err)
	}
	if _, err := service.RegisterByPassword(ctx, "13600136000", "second_user", "password123", ClientMeta{}); err != nil {
		t.Fatalf("RegisterByPassword(second) error = %v", err)
	}
	if err := service.users.Create(ctx, &User{
		Phone:          "13500135000",
		Username:       stringPtr(ReservedAdminUsername),
		Status:         UserStatusEnabled,
		RegisterSource: RegisterSourcePassword,
	}); err != nil {
		t.Fatalf("seed reserved user error = %v", err)
	}

	status := UserStatusEnabled
	result, err := service.ListUsers(ctx, UserListQuery{
		Page:     1,
		PageSize: 10,
		Keyword:  "second",
		Status:   &status,
	})
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("total = %d, want 1", result.Total)
	}
	if len(result.List) != 1 || result.List[0].Username != "second_user" {
		t.Fatalf("list = %+v, want second_user", result.List)
	}
}

func newTestAuthService() *AuthService {
	tokenManager := auth.NewTokenManager(config.JWTConfig{
		Secret:             "test-secret",
		AccessTokenTTL:     time.Hour,
		RefreshTokenTTL:    24 * time.Hour,
		RefreshTokenPepper: "test-pepper",
		Issuer:             "test",
	})

	return NewAuthService(AuthServiceOptions{
		Users:          newMemoryUserRepository(),
		LoginCodes:     newMemoryLoginCodeRepository(),
		RefreshTokens:  newMemoryRefreshTokenRepository(),
		PasswordHasher: auth.NewBcryptHasher(),
		TokenManager:   tokenManager,
		MobileVerifier: fixedMobileVerifier{phone: "13800138000"},
		SMSSender:      noopSMSSender{},
		CodeSecret:     "test-code-secret",
	})
}

type fixedMobileVerifier struct {
	phone string
}

func (v fixedMobileVerifier) Verify(ctx context.Context, accessToken string) (string, error) {
	return v.phone, nil
}

type noopSMSSender struct{}

func (noopSMSSender) SendCode(ctx context.Context, phone string, code string) error {
	return nil
}

type memoryUserRepository struct {
	mu     sync.Mutex
	nextID int64
	items  map[int64]User
}

func newMemoryUserRepository() *memoryUserRepository {
	return &memoryUserRepository{
		nextID: 1,
		items:  map[int64]User{},
	}
}

func (r *memoryUserRepository) FindByID(ctx context.Context, id int64) (User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return item, nil
}

func (r *memoryUserRepository) FindByPhone(ctx context.Context, phone string) (User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, item := range r.items {
		if item.Phone == phone {
			return item, nil
		}
	}
	return User{}, ErrNotFound
}

func (r *memoryUserRepository) FindByUsernameOrPhone(ctx context.Context, account string) (User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, item := range r.items {
		if item.Phone == account || (item.Username != nil && *item.Username == account) {
			return item, nil
		}
	}
	return User{}, ErrNotFound
}

func (r *memoryUserRepository) List(ctx context.Context, query UserListQuery) ([]User, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	list := make([]User, 0, len(r.items))
	for _, item := range r.items {
		if item.Username != nil && strings.EqualFold(*item.Username, ReservedAdminUsername) {
			continue
		}
		if query.Keyword != "" &&
			!strings.Contains(item.Phone, query.Keyword) &&
			(item.Username == nil || !strings.Contains(*item.Username, query.Keyword)) &&
			!strings.Contains(item.Nickname, query.Keyword) {
			continue
		}
		if query.Status != nil && item.Status != *query.Status {
			continue
		}
		list = append(list, item)
	}

	total := int64(len(list))
	start := (query.Page - 1) * query.PageSize
	if start >= len(list) {
		return []User{}, total, nil
	}
	end := start + query.PageSize
	if end > len(list) {
		end = len(list)
	}
	return list[start:end], total, nil
}

func (r *memoryUserRepository) Create(ctx context.Context, item *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item.ID = r.nextID
	now := time.Now()
	item.CreatedAt = now
	item.UpdatedAt = now
	r.items[item.ID] = *item
	r.nextID++
	return nil
}

func (r *memoryUserRepository) UpdatePassword(ctx context.Context, id int64, username string, passwordHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok {
		return ErrNotFound
	}
	if username != "" {
		item.Username = &username
	}
	item.PasswordHash = passwordHash
	item.UpdatedAt = time.Now()
	r.items[id] = item
	return nil
}

func (r *memoryUserRepository) TouchLastLogin(ctx context.Context, id int64, loginAt time.Time, meta ClientMeta) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok {
		return ErrNotFound
	}
	item.LastLoginAt = &loginAt
	if meta.Platform != "" {
		item.Platform = meta.Platform
	}
	if meta.DeviceNo != "" {
		item.DeviceNo = meta.DeviceNo
	}
	r.items[id] = item
	return nil
}

type memoryLoginCodeRepository struct {
	mu     sync.Mutex
	nextID int64
	items  map[int64]LoginCode
}

func newMemoryLoginCodeRepository() *memoryLoginCodeRepository {
	return &memoryLoginCodeRepository{
		nextID: 1,
		items:  map[int64]LoginCode{},
	}
}

func (r *memoryLoginCodeRepository) Create(ctx context.Context, item *LoginCode) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item.ID = r.nextID
	item.CreatedAt = time.Now()
	r.items[item.ID] = *item
	r.nextID++
	return nil
}

func (r *memoryLoginCodeRepository) FindValid(ctx context.Context, phone string, scene string, codeHash string, now time.Time) (LoginCode, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, item := range r.items {
		if item.Phone == phone && item.Scene == scene && item.CodeHash == codeHash && item.UsedAt == nil && item.ExpiresAt.After(now) {
			return item, nil
		}
	}
	return LoginCode{}, ErrNotFound
}

func (r *memoryLoginCodeRepository) MarkUsed(ctx context.Context, id int64, usedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok {
		return ErrNotFound
	}
	item.UsedAt = &usedAt
	r.items[id] = item
	return nil
}

func (r *memoryLoginCodeRepository) seed(phone string, scene string, codeHash string) {
	_ = r.Create(context.Background(), &LoginCode{
		Phone:     phone,
		Scene:     scene,
		CodeHash:  codeHash,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	})
}

type memoryRefreshTokenRepository struct {
	mu    sync.Mutex
	items map[string]RefreshToken
}

func stringPtr(value string) *string {
	return &value
}

func newMemoryRefreshTokenRepository() *memoryRefreshTokenRepository {
	return &memoryRefreshTokenRepository{items: map[string]RefreshToken{}}
}

func (r *memoryRefreshTokenRepository) Create(ctx context.Context, item *RefreshToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[item.TokenHash] = *item
	return nil
}

func (r *memoryRefreshTokenRepository) FindValid(ctx context.Context, tokenHash string, now time.Time) (RefreshToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[tokenHash]
	if !ok || item.RevokedAt != nil || !item.ExpiresAt.After(now) {
		return RefreshToken{}, ErrNotFound
	}
	return item, nil
}

func (r *memoryRefreshTokenRepository) Revoke(ctx context.Context, tokenHash string, revokedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[tokenHash]
	if !ok {
		return ErrNotFound
	}
	item.RevokedAt = &revokedAt
	r.items[tokenHash] = item
	return nil
}
