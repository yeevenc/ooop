package activity

import (
	"context"
	"errors"
	"sync/atomic"
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
	reconcileCalls      int
	reconcileResults    []ImageAuditQueueReconcileResult
	reconcileError      error
	reconcileSignal     chan struct{}
	reconcileWake       []bool
	claimCalls          int
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

func (r *imageAuditTestRepository) ReconcilePendingImageAuditTasks(
	_ context.Context,
	_ time.Time,
	_ int,
	wakeRetries bool,
) (ImageAuditQueueReconcileResult, error) {
	r.reconcileCalls++
	r.reconcileWake = append(r.reconcileWake, wakeRetries)
	if r.reconcileSignal != nil {
		select {
		case r.reconcileSignal <- struct{}{}:
		default:
		}
	}
	if r.reconcileError != nil {
		return ImageAuditQueueReconcileResult{}, r.reconcileError
	}
	if len(r.reconcileResults) == 0 {
		return ImageAuditQueueReconcileResult{}, nil
	}
	result := r.reconcileResults[0]
	r.reconcileResults = r.reconcileResults[1:]
	return result, nil
}

func (r *imageAuditTestRepository) ClaimImageAuditTasks(
	context.Context,
	time.Time,
	int,
) ([]ImageAuditTask, error) {
	r.claimCalls++
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
	calls  *int
}

func (m imageAuditTestModerator) Audit(
	context.Context,
	[]string,
) (provider.ImageAuditResult, error) {
	if m.calls != nil {
		*m.calls = *m.calls + 1
	}
	return m.result, m.err
}

type imageAuditSerialTestModerator struct {
	current atomic.Int32
	maximum atomic.Int32
}

func (m *imageAuditSerialTestModerator) Audit(
	context.Context,
	[]string,
) (provider.ImageAuditResult, error) {
	current := m.current.Add(1)
	for {
		maximum := m.maximum.Load()
		if current <= maximum || m.maximum.CompareAndSwap(maximum, current) {
			break
		}
	}
	time.Sleep(5 * time.Millisecond)
	m.current.Add(-1)
	return provider.ImageAuditResult{
		Suggestion: provider.ImageAuditSuggestionPass,
		RawJSON:    `{"Data":{"Results":[]}}`,
	}, nil
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

func TestImageAuditWorkerSkipsReviewedActivityBeforeProviderCall(t *testing.T) {
	repository := &imageAuditTestRepository{
		activity: Activity{ID: 99, Status: StatusOngoing},
	}
	moderatorCalls := 0
	worker := NewImageAuditWorker(
		repository,
		&imageAuditTestReviewer{},
		imageAuditTestModerator{calls: &moderatorCalls},
		ImageAuditWorkerOptions{},
	)

	worker.processTask(context.Background(), ImageAuditTask{
		ID:            7,
		ActivityID:    99,
		ImageURLsJSON: `["https://source.example.com/activity.jpg"]`,
		Status:        ImageAuditTaskProcessing,
	})

	if moderatorCalls != 0 {
		t.Fatalf("moderator calls = %d, want 0", moderatorCalls)
	}
	if repository.completedStatus != ImageAuditTaskSkipped {
		t.Fatalf(
			"completed status = %s, want skipped",
			repository.completedStatus,
		)
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

func TestImageAuditPollingDoesNotRecoverLocksEveryTime(t *testing.T) {
	repository := &imageAuditTestRepository{}
	worker := NewImageAuditWorker(
		repository,
		&imageAuditTestReviewer{},
		imageAuditTestModerator{},
		ImageAuditWorkerOptions{},
	)

	worker.process(context.Background())
	if repository.recoverCalled {
		t.Fatal("normal polling should not recover stale locks")
	}

	worker.recoverStaleTasks(context.Background())
	if !repository.recoverCalled {
		t.Fatal("scheduled recovery was not executed")
	}
}

func TestImageAuditWorkerProcessesClaimedTasksSerially(t *testing.T) {
	repository := &imageAuditTestRepository{
		activity: Activity{ID: 99, Status: StatusPending},
		claimedTasks: []ImageAuditTask{
			{
				ID:            7,
				ActivityID:    99,
				ImageURLsJSON: `["https://source.example.com/first.jpg"]`,
				Status:        ImageAuditTaskProcessing,
			},
			{
				ID:            8,
				ActivityID:    100,
				ImageURLsJSON: `["https://source.example.com/second.jpg"]`,
				Status:        ImageAuditTaskProcessing,
			},
		},
	}
	moderator := &imageAuditSerialTestModerator{}
	worker := NewImageAuditWorker(
		repository,
		&imageAuditTestReviewer{},
		moderator,
		ImageAuditWorkerOptions{},
	)

	worker.process(context.Background())

	if maximum := moderator.maximum.Load(); maximum != 1 {
		t.Fatalf("maximum audit concurrency = %d, want 1", maximum)
	}
}

func TestImageAuditWorkerReconcilesPendingTasksOnStartup(t *testing.T) {
	repository := &imageAuditTestRepository{
		reconcileSignal: make(chan struct{}, 1),
	}
	worker := NewImageAuditWorker(
		repository,
		&imageAuditTestReviewer{},
		imageAuditTestModerator{},
		ImageAuditWorkerOptions{},
	)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		worker.run(ctx)
	}()

	select {
	case <-repository.reconcileSignal:
		cancel()
	case <-time.After(time.Second):
		cancel()
		t.Fatal("pending task repair did not start")
	}
	<-done

	if repository.reconcileCalls != 1 {
		t.Fatalf("reconcile calls = %d, want 1", repository.reconcileCalls)
	}
	if len(repository.reconcileWake) != 1 || !repository.reconcileWake[0] {
		t.Fatalf("startup wake flags = %v, want [true]", repository.reconcileWake)
	}
}

func TestImageAuditWorkerReconcilesAllHistoricalPendingActivities(t *testing.T) {
	repository := &imageAuditTestRepository{
		reconcileResults: []ImageAuditQueueReconcileResult{
			{
				Missing:   100,
				Created:   100,
				Awakened:  2,
				Recovered: 1,
			},
			{
				Missing: 2,
				Created: 2,
			},
		},
	}
	worker := NewImageAuditWorker(
		repository,
		&imageAuditTestReviewer{},
		imageAuditTestModerator{},
		ImageAuditWorkerOptions{},
	)

	worker.reconcilePendingTasks(context.Background(), true, false)

	if repository.reconcileCalls != 2 {
		t.Fatalf("reconcile calls = %d, want 2", repository.reconcileCalls)
	}
	if len(repository.reconcileWake) != 2 ||
		!repository.reconcileWake[0] ||
		repository.reconcileWake[1] {
		t.Fatalf("wake flags = %v, want [true false]", repository.reconcileWake)
	}
}

func TestImageAuditURLsJSONUsesGalleryAndFallsBackToMainImage(t *testing.T) {
	tests := []struct {
		name     string
		activity Activity
		expected string
	}{
		{
			name: "gallery",
			activity: Activity{
				GalleryJSON: `["https://source.example.com/first.jpg","https://source.example.com/second.jpg"]`,
				ImageURL:    "https://source.example.com/main.jpg",
			},
			expected: `["https://source.example.com/first.jpg","https://source.example.com/second.jpg"]`,
		},
		{
			name: "fallback",
			activity: Activity{
				GalleryJSON: `invalid`,
				ImageURL:    "https://source.example.com/main.jpg",
			},
			expected: `["https://source.example.com/main.jpg"]`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if result := imageAuditURLsJSON(test.activity); result != test.expected {
				t.Fatalf("imageAuditURLsJSON() = %s, want %s", result, test.expected)
			}
		})
	}
}
