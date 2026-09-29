package admin

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

func TestEnsureDefaultAdminCreatesAdminAndLogin(t *testing.T) {
	service := newTestService()
	ctx := context.Background()

	adminUser, err := service.EnsureDefaultAdmin(ctx, "admin", "admin")
	if err != nil {
		t.Fatalf("EnsureDefaultAdmin() error = %v", err)
	}
	if adminUser.Username != "admin" {
		t.Fatalf("username = %s, want admin", adminUser.Username)
	}

	result, err := service.Login(ctx, "admin", "admin")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if result.Token == "" {
		t.Fatalf("token should not be empty")
	}
	if result.User.ID != adminUser.ID {
		t.Fatalf("login user id = %d, want %d", result.User.ID, adminUser.ID)
	}
}

func TestChangePasswordUpdatesLoginPassword(t *testing.T) {
	service := newTestService()
	ctx := context.Background()

	adminUser, err := service.EnsureDefaultAdmin(ctx, "admin", "admin")
	if err != nil {
		t.Fatalf("EnsureDefaultAdmin() error = %v", err)
	}

	if err := service.ChangePassword(ctx, adminUser.ID, "admin", "new-password"); err != nil {
		t.Fatalf("ChangePassword() error = %v", err)
	}
	if _, err := service.Login(ctx, "admin", "admin"); !errors.Is(err, ErrInvalidAccount) {
		t.Fatalf("old password error = %v, want ErrInvalidAccount", err)
	}
	result, err := service.Login(ctx, "admin", "new-password")
	if err != nil {
		t.Fatalf("Login() with new password error = %v", err)
	}
	if result.User.ID != adminUser.ID {
		t.Fatalf("login user id = %d, want %d", result.User.ID, adminUser.ID)
	}
	if err := service.ChangePassword(ctx, adminUser.ID, "new-password", "new-password"); !errors.Is(err, ErrSamePassword) {
		t.Fatalf("same password error = %v, want ErrSamePassword", err)
	}
}

func TestChangePasswordRejectsInvalidInput(t *testing.T) {
	service := newTestService()
	ctx := context.Background()

	adminUser, err := service.EnsureDefaultAdmin(ctx, "admin", "admin")
	if err != nil {
		t.Fatalf("EnsureDefaultAdmin() error = %v", err)
	}

	cases := []struct {
		name        string
		oldPassword string
		newPassword string
		want        error
	}{
		{name: "wrong old password", oldPassword: "wrong-password", newPassword: "new-password", want: ErrInvalidOldPassword},
		{name: "empty old password", oldPassword: "", newPassword: "new-password", want: ErrInvalidOldPassword},
		{name: "short new password", oldPassword: "admin", newPassword: "short", want: ErrInvalidNewPassword},
		{name: "long new password", oldPassword: "admin", newPassword: strings.Repeat("a", maxAdminPasswordLength+1), want: ErrPasswordTooLong},
		{name: "password exceeds bcrypt byte limit", oldPassword: "admin", newPassword: strings.Repeat("😀", 20), want: ErrPasswordTooLong},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := service.ChangePassword(ctx, adminUser.ID, tc.oldPassword, tc.newPassword)
			if !errors.Is(err, tc.want) {
				t.Fatalf("ChangePassword() error = %v, want %v", err, tc.want)
			}
		})
	}

	if _, err := service.Login(ctx, "admin", "admin"); err != nil {
		t.Fatalf("password should stay unchanged, Login() error = %v", err)
	}
}

func TestChangePasswordRejectsDisabledAdmin(t *testing.T) {
	repo := newMemoryRepository()
	service := NewService(repo, auth.NewBcryptHasher(), auth.NewTokenManager(config.JWTConfig{
		Secret:         "test-secret",
		AccessTokenTTL: time.Hour,
		Issuer:         "test-admin",
	}))
	ctx := context.Background()

	adminUser, err := service.EnsureDefaultAdmin(ctx, "admin", "admin")
	if err != nil {
		t.Fatalf("EnsureDefaultAdmin() error = %v", err)
	}

	repo.mu.Lock()
	item := repo.items[adminUser.ID]
	item.Status = 0
	repo.items[adminUser.ID] = item
	repo.mu.Unlock()

	err = service.ChangePassword(ctx, adminUser.ID, "admin", "new-password")
	if !errors.Is(err, ErrDisabledAdmin) {
		t.Fatalf("ChangePassword() error = %v, want ErrDisabledAdmin", err)
	}
}

func newTestService() *Service {
	tokenManager := auth.NewTokenManager(config.JWTConfig{
		Secret:         "test-secret",
		AccessTokenTTL: time.Hour,
		Issuer:         "test-admin",
	})

	return NewService(newMemoryRepository(), auth.NewBcryptHasher(), tokenManager)
}

type memoryRepository struct {
	mu     sync.Mutex
	nextID int64
	items  map[int64]AdminUser
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		nextID: 1,
		items:  map[int64]AdminUser{},
	}
}

func (r *memoryRepository) FindByUsername(ctx context.Context, username string) (AdminUser, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, item := range r.items {
		if item.Username == username {
			return item, nil
		}
	}
	return AdminUser{}, ErrNotFound
}

func (r *memoryRepository) Create(ctx context.Context, item *AdminUser) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, exists := range r.items {
		if exists.Username == item.Username {
			return errors.New("用户名已存在")
		}
	}
	item.ID = r.nextID
	now := time.Now()
	item.CreatedAt = now
	item.UpdatedAt = now
	r.items[item.ID] = *item
	r.nextID++
	return nil
}

func (r *memoryRepository) FindByID(ctx context.Context, id int64) (AdminUser, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok {
		return AdminUser{}, ErrNotFound
	}
	return item, nil
}

func (r *memoryRepository) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok {
		return ErrNotFound
	}
	item.PasswordHash = passwordHash
	item.UpdatedAt = time.Now()
	r.items[id] = item
	return nil
}
