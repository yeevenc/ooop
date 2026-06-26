package admin

import (
	"context"
	"errors"
	"strings"

	"ooop-admin-api/internal/auth"
)

var (
	ErrInvalidAccount = errors.New("账号或密码错误")
	ErrDisabledAdmin  = errors.New("管理员账号已禁用")
)

type Service struct {
	repo           Repository
	passwordHasher auth.PasswordHasher
	tokenManager   *auth.TokenManager
}

type LoginResult struct {
	Token string          `json:"token"`
	User  PublicAdminUser `json:"user"`
}

func NewService(repo Repository, passwordHasher auth.PasswordHasher, tokenManager *auth.TokenManager) *Service {
	return &Service{
		repo:           repo,
		passwordHasher: passwordHasher,
		tokenManager:   tokenManager,
	}
}

func (s *Service) EnsureDefaultAdmin(ctx context.Context, username string, password string) (PublicAdminUser, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return PublicAdminUser{}, ErrInvalidAccount
	}

	item, err := s.repo.FindByUsername(ctx, username)
	if err == nil {
		return ToPublicAdminUser(item), nil
	}
	if !errors.Is(err, ErrNotFound) {
		return PublicAdminUser{}, err
	}

	passwordHash, err := s.passwordHasher.Hash(password)
	if err != nil {
		return PublicAdminUser{}, err
	}

	item = AdminUser{
		Username:     username,
		PasswordHash: passwordHash,
		Status:       AdminStatusEnabled,
	}
	if err := s.repo.Create(ctx, &item); err != nil {
		return PublicAdminUser{}, err
	}
	return ToPublicAdminUser(item), nil
}

func (s *Service) Login(ctx context.Context, username string, password string) (LoginResult, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return LoginResult{}, ErrInvalidAccount
	}

	item, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return LoginResult{}, ErrInvalidAccount
		}
		return LoginResult{}, err
	}
	if item.Status != AdminStatusEnabled {
		return LoginResult{}, ErrDisabledAdmin
	}
	if !s.passwordHasher.Compare(item.PasswordHash, password) {
		return LoginResult{}, ErrInvalidAccount
	}

	tokens, err := s.tokenManager.NewToken(item.ID)
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{
		Token: tokens.AccessToken,
		User:  ToPublicAdminUser(item),
	}, nil
}
