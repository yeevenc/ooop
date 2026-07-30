package activity

import (
	"context"
	"errors"
	"testing"

	"ooop-admin-api/internal/user"
)

type managedParticipantsActivityRepository struct {
	Repository
	activity     Activity
	participants []ActivityParticipant
}

func (r managedParticipantsActivityRepository) FindByID(context.Context, int64) (Activity, error) {
	return r.activity, nil
}

func (r managedParticipantsActivityRepository) ListParticipantsByActivity(
	context.Context,
	int64,
	[]string,
	int,
) ([]ActivityParticipant, error) {
	return r.participants, nil
}

type managedParticipantsUserRepository struct {
	user.UserRepository
	users []user.User
}

func (r managedParticipantsUserRepository) FindByIDs(context.Context, []int64) ([]user.User, error) {
	return r.users, nil
}

func TestListManagedParticipantsReturnsEntryCodeForOrganizer(t *testing.T) {
	service := newManagedParticipantsTestService()

	result, err := service.ListManagedParticipants(context.Background(), 100, 200)
	if err != nil {
		t.Fatalf("ListManagedParticipants() error = %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("participants length = %d, want 1", len(result))
	}
	if result[0].EntryCode != "AB12CD34" {
		t.Fatalf("entry code = %q, want %q", result[0].EntryCode, "AB12CD34")
	}
}

func TestListManagedParticipantsRejectsNonOrganizer(t *testing.T) {
	service := newManagedParticipantsTestService()

	_, err := service.ListManagedParticipants(context.Background(), 101, 200)
	if !errors.Is(err, ErrNotOrganizer) {
		t.Fatalf("ListManagedParticipants() error = %v, want ErrNotOrganizer", err)
	}
}

func TestListApprovedParticipantsHidesEntryCode(t *testing.T) {
	service := newManagedParticipantsTestService()

	result, err := service.ListApprovedParticipants(context.Background(), 200)
	if err != nil {
		t.Fatalf("ListApprovedParticipants() error = %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("participants length = %d, want 1", len(result))
	}
	if result[0].EntryCode != "" {
		t.Fatalf("entry code = %q, want empty", result[0].EntryCode)
	}
}

func newManagedParticipantsTestService() *Service {
	activities := managedParticipantsActivityRepository{
		activity: Activity{
			ID:     200,
			UserID: 100,
			Status: StatusOngoing,
		},
		participants: []ActivityParticipant{
			{
				ID:         300,
				ActivityID: 200,
				UserID:     400,
				Count:      1,
				EntryCode:  "AB12CD34",
				Status:     ParticipantStatusApproved,
			},
		},
	}
	users := managedParticipantsUserRepository{
		users: []user.User{
			{
				ID:       400,
				Nickname: "参加人",
			},
		},
	}
	return NewService(activities, users)
}
