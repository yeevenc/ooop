package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"ooop-admin-api/internal/logger"
	"ooop-admin-api/internal/provider"
	"ooop-admin-api/internal/user"
)

// 推送失败会持续重试到消息接近过期，避免短时服务商故障形成永久断点。
const maxPushAttempts = 64

type ChannelPushSender interface {
	PushChannel(ctx context.Context, channel string, payload provider.PushPayload) (provider.PushChannelResult, error)
}

type WorkerOptions struct {
	PushInterval    time.Duration
	CleanupInterval time.Duration
	BatchSize       int
	Workers         int
	Retention       time.Duration
	PushCategory    string
}

type Worker struct {
	repository PushRepository
	users      PushUserReader
	pusher     ChannelPushSender
	options    WorkerOptions
}

type PushUserReader interface {
	FindByID(ctx context.Context, id int64) (user.User, error)
}

type realtimeMessage struct {
	MessageID       string    `json:"messageId"`
	ConversationID  string    `json:"conversationId"`
	SenderID        string    `json:"senderId"`
	RecipientID     string    `json:"recipientId"`
	ClientMessageID string    `json:"clientMessageId"`
	Type            string    `json:"type"`
	MessageType     string    `json:"messageType"`
	Content         string    `json:"content"`
	CreatedAt       time.Time `json:"createdAt"`
}

func NewWorker(repository PushRepository, users PushUserReader, pusher ChannelPushSender, options WorkerOptions) *Worker {
	if options.PushInterval <= 0 {
		options.PushInterval = time.Second
	}
	if options.CleanupInterval <= 0 {
		options.CleanupInterval = time.Hour
	}
	options.BatchSize = normalizePageSize(options.BatchSize, 100, 500)
	if options.Workers <= 0 {
		options.Workers = 4
	}
	if options.Workers > 16 {
		options.Workers = 16
	}
	options.Retention = normalizeRetention(options.Retention)
	if options.PushCategory == "" {
		options.PushCategory = provider.HarmonyCategoryWork
	}
	return &Worker{repository: repository, users: users, pusher: pusher, options: options}
}

func (w *Worker) Start(ctx context.Context) {
	go w.run(ctx)
}

func (w *Worker) run(ctx context.Context) {
	if err := w.repository.RecoverStalePushTasks(ctx, time.Now().Add(-time.Minute)); err != nil {
		logger.Errorf("聊天推送任务恢复失败: %v", err)
	}
	w.processPushTasks(ctx)
	w.cleanup(ctx)

	pushTicker := time.NewTicker(w.options.PushInterval)
	cleanupTicker := time.NewTicker(w.options.CleanupInterval)
	defer pushTicker.Stop()
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pushTicker.C:
			w.processPushTasks(ctx)
		case <-cleanupTicker.C:
			w.cleanup(ctx)
		}
	}
}

func (w *Worker) processPushTasks(ctx context.Context) {
	if err := w.repository.RecoverStalePushTasks(ctx, time.Now().Add(-time.Minute)); err != nil {
		logger.Errorf("聊天推送任务解锁失败: %v", err)
		return
	}
	tasks, err := w.repository.ClaimPushTasks(ctx, time.Now(), w.options.BatchSize)
	if err != nil {
		logger.Errorf("聊天推送任务领取失败: %v", err)
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
			w.processPushTask(ctx, task)
		}()
	}
	wait.Wait()
}

func (w *Worker) processPushTask(ctx context.Context, task PushTask) {
	attempts := task.Attempts + 1
	message, err := w.repository.FindPushMessage(ctx, task.MessageID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			_ = w.repository.MarkPushSkipped(ctx, task.ID, attempts, "聊天消息已过期")
			return
		}
		w.retryPushTask(ctx, task, attempts, err)
		return
	}
	if !message.ExpiresAt.After(time.Now()) {
		_ = w.repository.MarkPushSkipped(ctx, task.ID, attempts, "聊天消息已过期")
		return
	}

	pushUser, err := w.users.FindByID(ctx, task.RecipientID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			_ = w.repository.MarkPushSkipped(ctx, task.ID, attempts, "接收用户不存在")
			return
		}
		w.retryPushTask(ctx, task, attempts, err)
		return
	}
	if task.Channel == provider.PushChannelHarmony && !pushUser.IsNotificationPermissionEnabled() {
		_ = w.repository.MarkPushSkipped(ctx, task.ID, attempts, "用户系统通知权限关闭")
		return
	}
	if w.pusher == nil {
		w.retryPushTask(ctx, task, attempts, errors.New("推送服务未初始化"))
		return
	}

	payload, err := w.buildPushPayload(message, pushUser)
	if err != nil {
		w.retryPushTask(ctx, task, attempts, err)
		return
	}
	result, err := w.pusher.PushChannel(ctx, task.Channel, payload)
	if err == nil && result.Success {
		if err := w.repository.MarkPushSucceeded(ctx, task.ID, attempts); err != nil {
			logger.Errorf("聊天推送任务完成状态保存失败: task_id=%d, error=%v", task.ID, err)
		}
		return
	}
	if err == nil && !result.Triggered {
		if err := w.repository.MarkPushSkipped(ctx, task.ID, attempts, result.Message); err != nil {
			logger.Errorf("聊天推送任务跳过状态保存失败: task_id=%d, error=%v", task.ID, err)
		}
		return
	}
	if err == nil {
		err = errors.New(result.Message)
	}
	w.retryPushTask(ctx, task, attempts, err)
}

func (w *Worker) buildPushPayload(message Message, pushUser user.User) (provider.PushPayload, error) {
	realtime := realtimeMessage{
		MessageID:       formatID(message.ID),
		ConversationID:  formatID(message.ConversationID),
		SenderID:        formatID(message.SenderID),
		RecipientID:     formatID(message.RecipientID),
		ClientMessageID: message.ClientMessageID,
		Type:            PushMessageType,
		MessageType:     message.Type,
		Content:         message.Content,
		CreatedAt:       message.CreatedAt,
	}
	customContent, err := json.Marshal(realtime)
	if err != nil {
		return provider.PushPayload{}, err
	}

	return provider.PushPayload{
		Alias:            strconv.FormatInt(pushUser.ID, 10),
		RegistrationID:   pushUser.RegistrationID,
		HarmonyPushToken: pushUser.HarmonyPushToken,
		Title:            "新会话",
		Alert:            "您有新会话",
		CustomContent:    string(customContent),
		MessageType:      PushMessageType,
		Category:         w.options.PushCategory,
		MessageID:        message.ID,
		Extras: map[string]string{
			"type":           PushMessageType,
			"messageId":      formatID(message.ID),
			"conversationId": formatID(message.ConversationID),
			"senderId":       formatID(message.SenderID),
			"messageType":    message.Type,
		},
	}, nil
}

func (w *Worker) retryPushTask(ctx context.Context, task PushTask, attempts int, err error) {
	dead := attempts >= maxPushAttempts
	nextRetryAt := time.Now().Add(pushRetryDelay(attempts))
	if updateErr := w.repository.MarkPushRetry(ctx, task.ID, attempts, nextRetryAt, err.Error(), dead); updateErr != nil {
		logger.Errorf("聊天推送任务重试状态保存失败: task_id=%d, error=%v", task.ID, updateErr)
		return
	}
	if dead {
		logger.Errorf("聊天推送任务超过最大重试次数: task_id=%d, channel=%s, error=%v", task.ID, task.Channel, err)
		return
	}
	logger.Warnf("聊天推送将在稍后重试: task_id=%d, channel=%s, attempts=%d, error=%v", task.ID, task.Channel, attempts, err)
}

func (w *Worker) cleanup(ctx context.Context) {
	const cleanupBatchSize = 1000
	const maxCleanupBatches = 20
	now := time.Now()
	for index := 0; index < maxCleanupBatches; index++ {
		count, err := w.repository.DeleteExpiredMessages(ctx, now, cleanupBatchSize)
		if err != nil {
			logger.Errorf("聊天过期消息清理失败: %v", err)
			return
		}
		if count < cleanupBatchSize {
			break
		}
	}
	if err := w.repository.DeleteEmptyConversations(ctx); err != nil {
		logger.Errorf("聊天空会话清理失败: %v", err)
	}
	for index := 0; index < maxCleanupBatches; index++ {
		count, err := w.repository.DeleteExpiredPushTasks(ctx, now.Add(-w.options.Retention), cleanupBatchSize)
		if err != nil {
			logger.Errorf("聊天过期推送任务清理失败: %v", err)
			return
		}
		if count < cleanupBatchSize {
			break
		}
	}
}

func pushRetryDelay(attempts int) time.Duration {
	delays := []time.Duration{
		5 * time.Second,
		15 * time.Second,
		time.Minute,
		5 * time.Minute,
		15 * time.Minute,
		30 * time.Minute,
		time.Hour,
		3 * time.Hour,
	}
	if attempts <= 0 {
		return delays[0]
	}
	if attempts > len(delays) {
		return delays[len(delays)-1]
	}
	return delays[attempts-1]
}

func (w WorkerOptions) String() string {
	return fmt.Sprintf("interval=%s, workers=%d, batch=%d, retention=%s", w.PushInterval, w.Workers, w.BatchSize, w.Retention)
}
