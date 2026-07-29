package activity

import (
	"errors"
	"testing"
	"time"
)

func TestValidateActivityRepublishAllowsCancelledFutureActivity(t *testing.T) {
	now := time.Date(2026, time.July, 29, 10, 0, 0, 0, time.Local)
	activityDate := now.Add(24 * time.Hour)

	err := validateActivityRepublish(Activity{
		Status:       StatusCancelled,
		ActivityDate: &activityDate,
	}, now)
	if err != nil {
		t.Fatalf("validateActivityRepublish() error = %v", err)
	}
}

func TestValidateActivityRepublishAllowsTakenDownFutureActivity(t *testing.T) {
	now := time.Date(2026, time.July, 29, 10, 0, 0, 0, time.Local)
	activityDate := now.Add(time.Hour)

	err := validateActivityRepublish(Activity{
		Status:       StatusTakenDown,
		ActivityDate: &activityDate,
	}, now)
	if err != nil {
		t.Fatalf("validateActivityRepublish() error = %v", err)
	}
}

func TestValidateActivityRepublishRejectsExpiredActivity(t *testing.T) {
	now := time.Date(2026, time.July, 29, 10, 0, 0, 0, time.Local)
	activityDate := now.Add(-time.Minute)

	err := validateActivityRepublish(Activity{
		Status:       StatusCancelled,
		ActivityDate: &activityDate,
	}, now)
	if !errors.Is(err, ErrActivityExpired) {
		t.Fatalf("validateActivityRepublish() error = %v, want ErrActivityExpired", err)
	}
}

func TestValidateActivityRepublishRejectsInvalidStatus(t *testing.T) {
	now := time.Date(2026, time.July, 29, 10, 0, 0, 0, time.Local)
	activityDate := now.Add(time.Hour)

	err := validateActivityRepublish(Activity{
		Status:       StatusOngoing,
		ActivityDate: &activityDate,
	}, now)
	if !errors.Is(err, ErrInvalidStatus) {
		t.Fatalf("validateActivityRepublish() error = %v, want ErrInvalidStatus", err)
	}
}

func TestValidateActivityRepublishRequiresActivityTime(t *testing.T) {
	now := time.Date(2026, time.July, 29, 10, 0, 0, 0, time.Local)

	err := validateActivityRepublish(Activity{Status: StatusTakenDown}, now)
	if !errors.Is(err, ErrInvalidActivityTime) {
		t.Fatalf("validateActivityRepublish() error = %v, want ErrInvalidActivityTime", err)
	}
}
