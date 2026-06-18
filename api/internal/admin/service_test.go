package admin

import (
	"context"
	"errors"
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

func newTestService() *Service {
	tokenManager := auth.NewTokenManager(config.JWTConfig{
		Secret:             "test-secret",
		AccessTokenTTL:     time.Hour,
		RefreshTokenTTL:    24 * time.Hour,
		RefreshTokenPepper: "test-pepper",
		Issuer:             "test-admin",
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
