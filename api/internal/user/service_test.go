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
	"ooop-admin-api/internal/provider"
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
	if result.Tokens.AccessToken == "" {
		t.Fatalf("access token should not be empty")
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
	if _, err := service.SetPassword(ctx, loginResult.User.ID, "test_user", "", "password123"); err != nil {
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

func TestSetPasswordRequiresOldPasswordWhenAlreadySet(t *testing.T) {
	service := newTestAuthService()
	ctx := context.Background()

	loginResult, err := service.AliyunMobileLogin(ctx, "aliyun-token", ClientMeta{})
	if err != nil {
		t.Fatalf("AliyunMobileLogin() error = %v", err)
	}
	if _, err := service.SetPassword(ctx, loginResult.User.ID, "", "", "password123"); err != nil {
		t.Fatalf("SetPassword() first error = %v", err)
	}
	if _, err := service.SetPassword(ctx, loginResult.User.ID, "", "", "password456"); !errors.Is(err, ErrInvalidOldPass) {
		t.Fatalf("SetPassword() error = %v, want ErrInvalidOldPass", err)
	}
	if _, err := service.SetPassword(ctx, loginResult.User.ID, "", "password123", "password456"); err != nil {
		t.Fatalf("SetPassword() second error = %v", err)
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
	smsSender := service.smsSender.(*noopSMSSender)
	ctx := context.Background()

	smsSender.allow("13900139000", provider.SMSSceneLogin, "123456")

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

func TestUpdateProfilePartiallyUpdatesFields(t *testing.T) {
	service := newTestAuthService()
	ctx := context.Background()

	login, err := service.AliyunMobileLogin(ctx, "aliyun-token", ClientMeta{})
	if err != nil {
		t.Fatalf("AliyunMobileLogin() error = %v", err)
	}

	nickname := "  新昵称  "
	bio := "热爱生活"
	updated, err := service.UpdateProfile(ctx, login.User.ID, ProfileUpdate{Nickname: &nickname, Bio: &bio})
	if err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}
	if updated.Nickname != "新昵称" {
		t.Fatalf("nickname = %q, want 新昵称(已去空白)", updated.Nickname)
	}
	if updated.Bio != "热爱生活" {
		t.Fatalf("bio = %q, want 热爱生活", updated.Bio)
	}

	// 仅传 region，其余字段保持不变
	region := "上海"
	updated, err = service.UpdateProfile(ctx, login.User.ID, ProfileUpdate{Region: &region})
	if err != nil {
		t.Fatalf("UpdateProfile(region) error = %v", err)
	}
	if updated.Nickname != "新昵称" || updated.Region != "上海" {
		t.Fatalf("部分更新丢字段: %+v", updated)
	}

	// 空昵称应被拒绝
	blank := "   "
	if _, err := service.UpdateProfile(ctx, login.User.ID, ProfileUpdate{Nickname: &blank}); !errors.Is(err, ErrInvalidProfile) {
		t.Fatalf("空昵称 error = %v, want ErrInvalidProfile", err)
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
		Secret:         "test-secret",
		AccessTokenTTL: time.Hour,
		Issuer:         "test",
	})

	return NewAuthService(AuthServiceOptions{
		Users:          newMemoryUserRepository(),
		LoginCodes:     newMemoryLoginCodeRepository(),
		PasswordHasher: auth.NewBcryptHasher(),
		TokenManager:   tokenManager,
		MobileVerifier: fixedMobileVerifier{phone: "13800138000"},
		SMSSender:      newNoopSMSSender(),
		CodeSecret:     "test-code-secret",
	})
}

type fixedMobileVerifier struct {
	phone string
}

func (v fixedMobileVerifier) Verify(ctx context.Context, accessToken string) (provider.MobileVerifyResult, error) {
	return provider.MobileVerifyResult{Phone: v.phone}, nil
}

type noopSMSSender struct {
	mu    sync.Mutex
	codes map[string]bool
}

func newNoopSMSSender() *noopSMSSender {
	return &noopSMSSender{codes: map[string]bool{}}
}

func (s *noopSMSSender) allow(phone string, scene provider.SMSScene, code string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[phone+":"+string(scene)+":"+code] = true
}

func (s *noopSMSSender) SendCode(ctx context.Context, phone string, scene provider.SMSScene) error {
	return nil
}

func (s *noopSMSSender) CheckCode(ctx context.Context, phone string, scene provider.SMSScene, code string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := phone + ":" + string(scene) + ":" + code
	if !s.codes[key] {
		return false, nil
	}
	delete(s.codes, key)
	return true, nil
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

func (r *memoryUserRepository) FindByIDs(ctx context.Context, ids []int64) ([]User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []User
	for _, id := range ids {
		if item, ok := r.items[id]; ok {
			result = append(result, item)
		}
	}
	return result, nil
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

func (r *memoryUserRepository) UpdatePhone(ctx context.Context, id int64, phone string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok {
		return ErrNotFound
	}
	item.Phone = phone
	item.UpdatedAt = time.Now()
	r.items[id] = item
	return nil
}

func (r *memoryUserRepository) UpdateProfile(ctx context.Context, id int64, update ProfileUpdate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok {
		return ErrNotFound
	}
	if update.Nickname != nil {
		item.Nickname = *update.Nickname
	}
	if update.Gender != nil {
		item.Gender = *update.Gender
	}
	if update.Region != nil {
		item.Region = *update.Region
	}
	if update.Bio != nil {
		item.Bio = *update.Bio
	}
	if update.Avatar != nil {
		item.Avatar = *update.Avatar
	}
	item.UpdatedAt = time.Now()
	r.items[id] = item
	return nil
}

func (r *memoryUserRepository) UpdatePushRegistration(ctx context.Context, id int64, platform string, registrationID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok {
		return ErrNotFound
	}
	item.PushPlatform = platform
	item.RegistrationID = registrationID
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

func stringPtr(value string) *string {
	return &value
}
