package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"ooop-admin-api/internal/auth"
	"ooop-admin-api/internal/config"
)

func TestBanUserPermanentAndUnban(t *testing.T) {
	repo := newMemoryUserRepository()
	svc := NewAuthService(AuthServiceOptions{
		Users: repo,
		TokenManager: auth.NewTokenManager(config.JWTConfig{
			Secret:         "test",
			AccessTokenTTL: time.Hour,
			Issuer:         "test",
		}),
	})
	item := &User{Phone: "13800000001", Status: UserStatusEnabled, RegisterSource: RegisterSourcePassword}
	if err := repo.Create(context.Background(), item); err != nil {
		t.Fatal(err)
	}

	got, err := svc.BanUser(context.Background(), item.ID, BanUserInput{Type: BanTypePermanent, Reason: "违规"})
	if err != nil {
		t.Fatalf("ban: %v", err)
	}
	if got.Status != UserStatusDisabled || got.BannedUntil != nil || got.BanReason != "违规" {
		t.Fatalf("ban result = %+v", got)
	}
	if err := svc.CheckAppUserAccess(context.Background(), item.ID); !errors.Is(err, auth.ErrAccountBanned) {
		t.Fatalf("access after ban = %v", err)
	}

	got, err = svc.UnbanUser(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("unban: %v", err)
	}
	if got.Status != UserStatusEnabled {
		t.Fatalf("unban status = %d", got.Status)
	}
	if err := svc.CheckAppUserAccess(context.Background(), item.ID); err != nil {
		t.Fatalf("access after unban = %v", err)
	}
}

func TestTemporaryBanAutoUnban(t *testing.T) {
	repo := newMemoryUserRepository()
	svc := NewAuthService(AuthServiceOptions{
		Users: repo,
		TokenManager: auth.NewTokenManager(config.JWTConfig{
			Secret:         "test",
			AccessTokenTTL: time.Hour,
			Issuer:         "test",
		}),
	})
	item := &User{Phone: "13800000002", Status: UserStatusEnabled, RegisterSource: RegisterSourcePassword}
	if err := repo.Create(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Hour)
	if err := repo.UpdateBanStatus(context.Background(), item.ID, UserStatusDisabled, &past, "临时"); err != nil {
		t.Fatal(err)
	}
	if err := svc.CheckAppUserAccess(context.Background(), item.ID); err != nil {
		t.Fatalf("expired ban should auto unban, got %v", err)
	}
	stored, _ := repo.FindByID(context.Background(), item.ID)
	if stored.Status != UserStatusEnabled {
		t.Fatalf("status after auto unban = %d", stored.Status)
	}
}
