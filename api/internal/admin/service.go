package admin

import (
	"context"
	"errors"
	"strings"

	"ooop-admin-api/internal/auth"
)

const (
	minAdminPasswordLength = 8
	maxAdminPasswordLength = 64
)

var (
	ErrInvalidAccount     = errors.New("账号或密码错误")
	ErrDisabledAdmin      = errors.New("管理员账号已禁用")
	ErrInvalidOldPassword = errors.New("原密码错误")
	ErrInvalidNewPassword = errors.New("新密码长度不能少于 8 位")
	ErrPasswordTooLong    = errors.New("新密码长度不能超过 64 位")
	ErrSamePassword       = errors.New("新密码不能与原密码相同")
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

func (s *Service) ChangePassword(ctx context.Context, adminID int64, oldPassword string, newPassword string) error {
	if adminID <= 0 || oldPassword == "" {
		return ErrInvalidOldPassword
	}
	if len([]rune(newPassword)) < minAdminPasswordLength {
		return ErrInvalidNewPassword
	}
	// bcrypt 只接受 72 字节。多字节字符可能在 64 个字符以内就超限。
	if len([]rune(newPassword)) > maxAdminPasswordLength || len(newPassword) > 72 {
		return ErrPasswordTooLong
	}
	if oldPassword == newPassword {
		return ErrSamePassword
	}

	item, err := s.repo.FindByID(ctx, adminID)
	if err != nil {
		return err
	}
	if item.Status != AdminStatusEnabled {
		return ErrDisabledAdmin
	}
	if !s.passwordHasher.Compare(item.PasswordHash, oldPassword) {
		return ErrInvalidOldPassword
	}

	passwordHash, err := s.passwordHasher.Hash(newPassword)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, adminID, passwordHash)
}
