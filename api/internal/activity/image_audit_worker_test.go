package activity

import (
	"context"
	"errors"
	"testing"
	"time"

	"ooop-admin-api/internal/provider"
)

type imageAuditTestRepository struct {
	activity            Activity
	decision            string
	rejectReason        string
	notificationDone    bool
	completedStatus     string
	retryAttempts       int
	retryReason         string
	recoverCalled       bool
	claimedTasks        []ImageAuditTask
	saveDecisionError   error
	markNotificationErr error
	markCompletedError  error
	markRetryError      error
	findActivityError   error
}

func (r *imageAuditTestRepository) FindByID(context.Context, int64) (Activity, error) {
	return r.activity, r.findActivityError
}

func (r *imageAuditTestRepository) ClaimImageAuditTasks(
	context.Context,
	time.Time,
	int,
) ([]ImageAuditTask, error) {
	return r.claimedTasks, nil
}

func (r *imageAuditTestRepository) SaveImageAuditDecision(
	_ context.Context,
	_ int64,
	_ int,
	decision string,
	_ string,
	rejectReason string,
) error {
	r.decision = decision
	r.rejectReason = rejectReason
	return r.saveDecisionError
}

func (r *imageAuditTestRepository) MarkImageAuditNotificationDone(context.Context, int64) error {
	r.notificationDone = true
	return r.markNotificationErr
}

func (r *imageAuditTestRepository) MarkImageAuditCompleted(
	_ context.Context,
	_ int64,
	status string,
	_ int,
) error {
	r.completedStatus = status
	return r.markCompletedError
}

func (r *imageAuditTestRepository) MarkImageAuditRetry(
	_ context.Context,
	_ int64,
	attempts int,
	_ time.Time,
	reason string,
) error {
	r.retryAttempts = attempts
	r.retryReason = reason
	return r.markRetryError
}

func (r *imageAuditTestRepository) RecoverStaleImageAuditTasks(context.Context, time.Time) error {
	r.recoverCalled = true
	return nil
}

type imageAuditTestModerator struct {
	result provider.ImageAuditResult
	err    error
}

func (m imageAuditTestModerator) Audit(
	context.Context,
	[]string,
) (provider.ImageAuditResult, error) {
	return m.result, m.err
}

type imageAuditTestReviewer struct {
	approve           bool
	rejectReason      string
	reviewSource      string
	idempotencyKey    string
	reviewCalls       int
	retryNotification int
	reviewResult      ReviewActivityResult
	reviewError       error
	notificationError error
}

func (r *imageAuditTestReviewer) ReviewActivity(
	_ context.Context,
	_ int64,
	approve bool,
	rejectReason string,
	reviewSource string,
	idempotencyKey string,
) (ReviewActivityResult, error) {
	r.reviewCalls++
	r.approve = approve
	r.rejectReason = rejectReason
	r.reviewSource = reviewSource
	r.idempotencyKey = idempotencyKey
	return r.reviewResult, r.reviewError
}

func (r *imageAuditTestReviewer) RetryActivityReviewNotification(
	context.Context,
	int64,
	bool,
	string,
	string,
) error {
	r.retryNotification++
	return r.notificationError
}

func TestImageAuditWorkerRejectsBlockedActivityThroughReviewFlow(t *testing.T) {
	repository := &imageAuditTestRepository{
		activity: Activity{ID: 99, Status: StatusPending},
	}
	reviewer := &imageAuditTestReviewer{}
	moderator := imageAuditTestModerator{
		result: provider.ImageAuditResult{
			Suggestion: provider.ImageAuditSuggestionBlock,
			Hits: []provider.ImageAuditHit{
				{
					ImageIndex: 1,
					Scene:      "porn",
					Label:      "porn",
					Suggestion: provider.ImageAuditSuggestionBlock,
					Rate:       99.8,
				},
			},
			RawJSON: `{"Data":{"Results":[]}}`,
		},
	}
	worker := NewImageAuditWorker(repository, reviewer, moderator, ImageAuditWorkerOptions{})

	worker.processTask(context.Background(), ImageAuditTask{
		ID:            7,
		ActivityID:    99,
		ImageURLsJSON: `["https://source.example.com/activity.jpg"]`,
		Status:        ImageAuditTaskProcessing,
	})

	if repository.decision != provider.ImageAuditSuggestionBlock {
		t.Fatalf("decision = %s, want block", repository.decision)
	}
	if reviewer.reviewCalls != 1 || reviewer.approve {
		t.Fatalf("review calls = %d, approve = %t", reviewer.reviewCalls, reviewer.approve)
	}
	if reviewer.reviewSource != ReviewSourceImageAudit {
		t.Fatalf("review source = %s", reviewer.reviewSource)
	}
	if reviewer.rejectReason == "" {
		t.Fatal("reject reason is empty")
	}
	if !repository.notificationDone {
		t.Fatal("notification was not marked done")
	}
	if repository.completedStatus != ImageAuditTaskRejected {
		t.Fatalf("completed status = %s, want rejected", repository.completedStatus)
	}
}

func TestImageAuditWorkerRetriesProviderFailureWithoutCompletingTask(t *testing.T) {
	repository := &imageAuditTestRepository{
		activity: Activity{ID: 99, Status: StatusPending},
	}
	reviewer := &imageAuditTestReviewer{}
	worker := NewImageAuditWorker(
		repository,
		reviewer,
		imageAuditTestModerator{err: errors.New("request timeout")},
		ImageAuditWorkerOptions{},
	)

	worker.processTask(context.Background(), ImageAuditTask{
		ID:            7,
		ActivityID:    99,
		ImageURLsJSON: `["https://source.example.com/activity.jpg"]`,
		Status:        ImageAuditTaskProcessing,
		Attempts:      2,
	})

	if repository.retryAttempts != 3 {
		t.Fatalf("retry attempts = %d, want 3", repository.retryAttempts)
	}
	if repository.completedStatus != "" {
		t.Fatalf("completed status = %s, want empty", repository.completedStatus)
	}
	if reviewer.reviewCalls != 0 {
		t.Fatalf("review calls = %d, want 0", reviewer.reviewCalls)
	}
}

func TestImageAuditWorkerResumesNotificationAfterStatusWasApplied(t *testing.T) {
	repository := &imageAuditTestRepository{
		activity: Activity{
			ID:           99,
			Status:       StatusRejected,
			ReviewSource: ReviewSourceImageAudit,
		},
	}
	reviewer := &imageAuditTestReviewer{}
	worker := NewImageAuditWorker(
		repository,
		reviewer,
		imageAuditTestModerator{},
		ImageAuditWorkerOptions{},
	)

	worker.processTask(context.Background(), ImageAuditTask{
		ID:            7,
		ActivityID:    99,
		ImageURLsJSON: `["https://source.example.com/activity.jpg"]`,
		Status:        ImageAuditTaskProcessing,
		Decision:      provider.ImageAuditSuggestionBlock,
		RejectReason:  "图片内容审核未通过",
		Attempts:      1,
	})

	if reviewer.reviewCalls != 0 {
		t.Fatalf("review calls = %d, want 0", reviewer.reviewCalls)
	}
	if reviewer.retryNotification != 1 {
		t.Fatalf("notification retries = %d, want 1", reviewer.retryNotification)
	}
	if !repository.notificationDone {
		t.Fatal("notification was not marked done")
	}
	if repository.completedStatus != ImageAuditTaskRejected {
		t.Fatalf("completed status = %s, want rejected", repository.completedStatus)
	}
}
