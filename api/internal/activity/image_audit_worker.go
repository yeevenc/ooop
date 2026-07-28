package activity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"

	"ooop-admin-api/internal/logger"
	"ooop-admin-api/internal/provider"
)

type ImageAuditReviewer interface {
	ReviewActivity(
		ctx context.Context,
		id int64,
		approve bool,
		rejectReason string,
		reviewSource string,
		idempotencyKey string,
	) (ReviewActivityResult, error)
	RetryActivityReviewNotification(
		ctx context.Context,
		id int64,
		approve bool,
		rejectReason string,
		idempotencyKey string,
	) error
}

type ImageAuditWorkerLease interface {
	TryAcquire(ctx context.Context, ttl time.Duration) (bool, error)
	Release(ctx context.Context) error
}

type ImageAuditWorkerOptions struct {
	PollInterval       time.Duration
	LockTimeout        time.Duration
	RecoveryInterval   time.Duration
	LeaseTTL           time.Duration
	LeaseRenewInterval time.Duration
	LeaseRetryInterval time.Duration
	BatchSize          int
	Workers            int
}

type ImageAuditWorker struct {
	repository ImageAuditRepository
	reviewer   ImageAuditReviewer
	moderator  provider.ImageModerator
	lease      ImageAuditWorkerLease
	options    ImageAuditWorkerOptions
}

func NewImageAuditWorker(
	repository ImageAuditRepository,
	reviewer ImageAuditReviewer,
	moderator provider.ImageModerator,
	options ImageAuditWorkerOptions,
) *ImageAuditWorker {
	if options.PollInterval <= 0 {
		options.PollInterval = 2 * time.Second
	}
	if options.LockTimeout < time.Minute {
		options.LockTimeout = 2 * time.Minute
	}
	if options.RecoveryInterval <= 0 {
		options.RecoveryInterval = time.Minute
	}
	if options.LeaseTTL <= 0 {
		options.LeaseTTL = 45 * time.Second
	}
	if options.LeaseRenewInterval <= 0 ||
		options.LeaseRenewInterval >= options.LeaseTTL {
		options.LeaseRenewInterval = options.LeaseTTL / 3
	}
	if options.LeaseRetryInterval <= 0 {
		options.LeaseRetryInterval = 10 * time.Second
	}
	if options.BatchSize <= 0 {
		options.BatchSize = 10
	}
	if options.BatchSize > 100 {
		options.BatchSize = 100
	}
	if options.Workers <= 0 {
		options.Workers = 2
	}
	if options.Workers > 8 {
		options.Workers = 8
	}
	return &ImageAuditWorker{
		repository: repository,
		reviewer:   reviewer,
		moderator:  moderator,
		options:    options,
	}
}

func (w *ImageAuditWorker) SetLease(lease ImageAuditWorkerLease) {
	w.lease = lease
}

func (w *ImageAuditWorker) Start(ctx context.Context) {
	go w.run(ctx)
}

func (w *ImageAuditWorker) run(ctx context.Context) {
	if w.lease == nil {
		w.runLeader(ctx)
		return
	}

	waitingLogged := false
	for {
		if ctx.Err() != nil {
			return
		}
		acquired, err := w.lease.TryAcquire(ctx, w.options.LeaseTTL)
		if err != nil {
			logger.Errorf("活动图片审核 Worker 租约获取失败: %v", err)
		} else if acquired {
			waitingLogged = false
			logger.Infof("活动图片审核 Worker 已取得主节点租约")
			w.runLeaseSession(ctx)
			if ctx.Err() != nil {
				return
			}
			logger.Warnf("活动图片审核 Worker 租约已失效，停止领取任务并等待重新接管")
		} else if !waitingLogged {
			logger.Infof("活动图片审核 Worker 当前为备用节点，不执行任务扫描")
			waitingLogged = true
		}

		timer := time.NewTimer(w.options.LeaseRetryInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (w *ImageAuditWorker) runLeaseSession(ctx context.Context) {
	leaderCtx, cancel := context.WithCancel(ctx)
	renewDone := make(chan struct{})
	go func() {
		defer close(renewDone)
		ticker := time.NewTicker(w.options.LeaseRenewInterval)
		defer ticker.Stop()
		for {
			select {
			case <-leaderCtx.Done():
				return
			case <-ticker.C:
				acquired, err := w.lease.TryAcquire(
					leaderCtx,
					w.options.LeaseTTL,
				)
				if err != nil {
					logger.Errorf("活动图片审核 Worker 租约续期失败: %v", err)
					cancel()
					return
				}
				if !acquired {
					cancel()
					return
				}
			}
		}
	}()

	w.runLeader(leaderCtx)
	cancel()
	<-renewDone

	releaseCtx, releaseCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer releaseCancel()
	if err := w.lease.Release(releaseCtx); err != nil {
		logger.Errorf("活动图片审核 Worker 租约释放失败: %v", err)
	}
}

func (w *ImageAuditWorker) runLeader(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	w.ensurePendingTasks(ctx)
	w.recoverStaleTasks(ctx)
	w.process(ctx)
	pollTicker := time.NewTicker(w.options.PollInterval)
	recoveryTicker := time.NewTicker(w.options.RecoveryInterval)
	defer pollTicker.Stop()
	defer recoveryTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pollTicker.C:
			w.process(ctx)
		case <-recoveryTicker.C:
			w.ensurePendingTasks(ctx)
			w.recoverStaleTasks(ctx)
		}
	}
}

func (w *ImageAuditWorker) process(ctx context.Context) {
	tasks, err := w.repository.ClaimImageAuditTasks(ctx, time.Now(), w.options.BatchSize)
	if err != nil {
		logger.Errorf("活动图片审核任务领取失败: %v", err)
		return
	}

	semaphore := make(chan struct{}, w.options.Workers)
	var wait sync.WaitGroup
	for _, item := range tasks {
		task := item
		wait.Add(1)
		semaphore <- struct{}{}
		go func() {
			defer wait.Done()
			defer func() { <-semaphore }()
			defer func() {
				if recovered := recover(); recovered != nil {
					w.retry(
						ctx,
						task,
						task.Attempts+1,
						fmt.Errorf("活动图片审核任务异常: %v", recovered),
					)
				}
			}()
			w.processTask(ctx, task)
		}()
	}
	wait.Wait()
}

func (w *ImageAuditWorker) ensurePendingTasks(ctx context.Context) {
	created, err := w.repository.EnsurePendingImageAuditTasks(ctx, time.Now(), 100)
	if err != nil {
		logger.Errorf("待审核活动任务补建失败: %v", err)
		return
	}
	if created > 0 {
		logger.Infof("待审核活动任务补建完成: count=%d", created)
	}
}

func (w *ImageAuditWorker) recoverStaleTasks(ctx context.Context) {
	if err := w.repository.RecoverStaleImageAuditTasks(
		ctx,
		time.Now().Add(-w.options.LockTimeout),
	); err != nil {
		logger.Errorf("活动图片审核任务恢复失败: %v", err)
	}
}

func (w *ImageAuditWorker) processTask(ctx context.Context, task ImageAuditTask) {
	attempts := task.Attempts + 1
	if task.Decision == "" {
		if !w.activityNeedsAudit(ctx, task, attempts) {
			return
		}
		result, err := w.requestAudit(ctx, task)
		if err != nil {
			w.retry(ctx, task, attempts, err)
			return
		}
		rejectReason := ""
		if result.Suggestion == provider.ImageAuditSuggestionBlock {
			rejectReason = truncateImageAuditError(result.RejectReason())
		}
		if err := w.repository.SaveImageAuditDecision(
			ctx,
			task.ID,
			attempts,
			result.Suggestion,
			result.RawJSON,
			rejectReason,
		); err != nil {
			w.retry(ctx, task, attempts, err)
			return
		}
		task.Decision = result.Suggestion
		task.ResultJSON = result.RawJSON
		task.RejectReason = rejectReason
		task.Attempts = attempts
	}

	switch task.Decision {
	case provider.ImageAuditSuggestionPass:
		w.applyReview(ctx, task, attempts, true, ImageAuditTaskPassed)
	case provider.ImageAuditSuggestionBlock:
		w.applyReview(ctx, task, attempts, false, ImageAuditTaskRejected)
	case provider.ImageAuditSuggestionReview:
		w.complete(ctx, task, attempts, ImageAuditTaskManualReview)
	default:
		w.retry(ctx, task, attempts, fmt.Errorf("图片审核任务存在未知决策: %s", task.Decision))
	}
}

func (w *ImageAuditWorker) activityNeedsAudit(
	ctx context.Context,
	task ImageAuditTask,
	attempts int,
) bool {
	item, err := w.repository.FindByID(ctx, task.ActivityID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			w.complete(ctx, task, attempts, ImageAuditTaskSkipped)
			return false
		}
		w.retry(ctx, task, attempts, err)
		return false
	}
	if item.Status != StatusPending {
		w.complete(ctx, task, attempts, ImageAuditTaskSkipped)
		return false
	}
	return true
}

func (w *ImageAuditWorker) requestAudit(
	ctx context.Context,
	task ImageAuditTask,
) (provider.ImageAuditResult, error) {
	if w.moderator == nil {
		return provider.ImageAuditResult{}, errors.New("活动图片审核服务未初始化")
	}
	var imageURLs []string
	if err := json.Unmarshal([]byte(task.ImageURLsJSON), &imageURLs); err != nil {
		return provider.ImageAuditResult{}, fmt.Errorf("活动图片列表解析失败: %w", err)
	}
	return w.moderator.Audit(ctx, imageURLs)
}

func (w *ImageAuditWorker) applyReview(
	ctx context.Context,
	task ImageAuditTask,
	attempts int,
	approve bool,
	completedStatus string,
) {
	if w.reviewer == nil {
		w.retry(ctx, task, attempts, errors.New("活动审核服务未初始化"))
		return
	}

	item, err := w.repository.FindByID(ctx, task.ActivityID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			w.complete(ctx, task, attempts, ImageAuditTaskSkipped)
			return
		}
		w.retry(ctx, task, attempts, err)
		return
	}

	expectedStatus := StatusRejected
	if approve {
		expectedStatus = StatusOngoing
	}
	idempotencyKey := fmt.Sprintf("activity-image-audit:%d", task.ID)

	if item.Status == StatusPending {
		result, reviewErr := w.reviewer.ReviewActivity(
			ctx,
			item.ID,
			approve,
			task.RejectReason,
			ReviewSourceImageAudit,
			idempotencyKey,
		)
		if reviewErr != nil {
			w.retry(ctx, task, attempts, reviewErr)
			return
		}
		if result.notificationErr != nil {
			w.retry(ctx, task, attempts, result.notificationErr)
			return
		}
		if err := w.repository.MarkImageAuditNotificationDone(ctx, task.ID); err != nil {
			w.retry(ctx, task, attempts, err)
			return
		}
		task.NotificationDone = true
	} else {
		if item.Status != expectedStatus || item.ReviewSource != ReviewSourceImageAudit {
			w.complete(ctx, task, attempts, ImageAuditTaskSkipped)
			return
		}
		if !task.NotificationDone {
			if err := w.reviewer.RetryActivityReviewNotification(
				ctx,
				item.ID,
				approve,
				task.RejectReason,
				idempotencyKey,
			); err != nil {
				w.retry(ctx, task, attempts, err)
				return
			}
			if err := w.repository.MarkImageAuditNotificationDone(ctx, task.ID); err != nil {
				w.retry(ctx, task, attempts, err)
				return
			}
		}
	}

	w.complete(ctx, task, attempts, completedStatus)
}

func (w *ImageAuditWorker) retry(
	ctx context.Context,
	task ImageAuditTask,
	attempts int,
	err error,
) {
	nextRetryAt := time.Now().Add(imageAuditRetryDelay(attempts))
	if updateErr := w.repository.MarkImageAuditRetry(
		ctx,
		task.ID,
		attempts,
		nextRetryAt,
		err.Error(),
	); updateErr != nil {
		logger.Errorf(
			"活动图片审核重试状态保存失败: task_id=%d, activity_id=%d, error=%v",
			task.ID,
			task.ActivityID,
			updateErr,
		)
		return
	}
	logger.Warnf(
		"活动图片审核将在稍后重试: task_id=%d, activity_id=%d, attempts=%d, error=%v",
		task.ID,
		task.ActivityID,
		attempts,
		err,
	)
}

func (w *ImageAuditWorker) complete(
	ctx context.Context,
	task ImageAuditTask,
	attempts int,
	status string,
) {
	if err := w.repository.MarkImageAuditCompleted(ctx, task.ID, status, attempts); err != nil {
		logger.Errorf(
			"活动图片审核完成状态保存失败: task_id=%d, activity_id=%d, status=%s, error=%v",
			task.ID,
			task.ActivityID,
			status,
			err,
		)
	}
}

func imageAuditRetryDelay(attempts int) time.Duration {
	delays := []time.Duration{
		5 * time.Second,
		15 * time.Second,
		time.Minute,
		5 * time.Minute,
		15 * time.Minute,
		30 * time.Minute,
		time.Hour,
	}
	if attempts <= 0 {
		return delays[0]
	}
	if attempts > len(delays) {
		return delays[len(delays)-1]
	}
	return delays[attempts-1]
}
