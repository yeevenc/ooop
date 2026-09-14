package message

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"ooop-admin-api/internal/logger"
	"ooop-admin-api/internal/provider"
	"ooop-admin-api/internal/user"
)

var (
	ErrInvalidBroadcastTitle   = errors.New("通知标题不能为空，且不能超过 80 字")
	ErrInvalidBroadcastContent = errors.New("通知内容不能为空，且不能超过 500 字")
	ErrBroadcastUnavailable    = errors.New("系统通知服务未初始化")
)

func (s *Service) SetBroadcasts(repo BroadcastRepository, batchSize int) {
	s.broadcasts = repo
	if batchSize <= 0 {
		batchSize = defaultBroadcastBatchSize
	}
	if batchSize > maxBroadcastBatchSize {
		batchSize = maxBroadcastBatchSize
	}
	s.broadcastBatchSize = batchSize
}

func (s *Service) CreateSystemBroadcast(ctx context.Context, adminID int64, title string, content string) (AdminSystemBroadcast, error) {
	if s.broadcasts == nil || s.users == nil {
		return AdminSystemBroadcast{}, ErrBroadcastUnavailable
	}

	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	if title == "" || utf8.RuneCountInString(title) > maxBroadcastTitleLength {
		return AdminSystemBroadcast{}, ErrInvalidBroadcastTitle
	}
	if content == "" || utf8.RuneCountInString(content) > maxBroadcastContentLength {
		return AdminSystemBroadcast{}, ErrInvalidBroadcastContent
	}

	targetCount, err := s.users.CountEnabledForBroadcast(ctx)
	if err != nil {
		return AdminSystemBroadcast{}, err
	}

	item := &SystemBroadcast{
		AdminID:     adminID,
		Title:       title,
		Content:     content,
		Status:      BroadcastStatusPending,
		TargetCount: int(targetCount),
	}
	if err := s.broadcasts.Create(ctx, item); err != nil {
		return AdminSystemBroadcast{}, err
	}
	logger.Infof("系统广播已创建: id=%d, admin_id=%d, target_count=%d", item.ID, adminID, item.TargetCount)
	return toAdminSystemBroadcast(*item), nil
}

func (s *Service) ListSystemBroadcasts(ctx context.Context, query BroadcastQuery) (AdminSystemBroadcastList, error) {
	if s.broadcasts == nil {
		return AdminSystemBroadcastList{}, ErrBroadcastUnavailable
	}

	items, total, err := s.broadcasts.List(ctx, query)
	if err != nil {
		return AdminSystemBroadcastList{}, err
	}

	list := make([]AdminSystemBroadcast, 0, len(items))
	for _, item := range items {
		list = append(list, toAdminSystemBroadcast(item))
	}
	page := query.Page
	if page <= 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	return AdminSystemBroadcastList{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Service) ProcessNextBroadcast(ctx context.Context) {
	if s.broadcasts == nil || s.users == nil {
		return
	}

	item, err := s.broadcasts.ClaimNext(ctx, time.Now())
	if err != nil {
		logger.Errorf("系统广播领取失败: %v", err)
		return
	}
	if item == nil {
		return
	}
	if err := s.processBroadcastBatch(ctx, item); err != nil {
		logger.Errorf("系统广播处理失败: id=%d, error=%v", item.ID, err)
	}
}

func (s *Service) processBroadcastBatch(ctx context.Context, item *SystemBroadcast) error {
	users, err := s.users.ListEnabledForBroadcast(ctx, item.LastUserID, s.broadcastBatchSize)
	if err != nil {
		return err
	}
	if len(users) == 0 {
		now := time.Now()
		item.Status = BroadcastStatusCompleted
		item.FinishedAt = &now
		item.ErrorMessage = ""
		return s.broadcasts.SaveProgress(ctx, item)
	}

	for _, target := range users {
		if err := s.deliverSystemBroadcast(ctx, item, target); err != nil {
			item.ErrorMessage = truncateMessageContent(err.Error(), 500)
			if saveErr := s.broadcasts.SaveProgress(ctx, item); saveErr != nil {
				logger.Errorf("系统广播进度保存失败: id=%d, error=%v", item.ID, saveErr)
			}
			return err
		}
		item.LastUserID = target.ID
		item.ErrorMessage = ""
		if err := s.broadcasts.SaveProgress(ctx, item); err != nil {
			return err
		}
	}

	if len(users) < s.broadcastBatchSize {
		now := time.Now()
		item.Status = BroadcastStatusCompleted
		item.FinishedAt = &now
		item.ErrorMessage = ""
		return s.broadcasts.SaveProgress(ctx, item)
	}
	return nil
}

func (s *Service) deliverSystemBroadcast(ctx context.Context, item *SystemBroadcast, target user.User) error {
	key := fmt.Sprintf("system-broadcast:%d:%d", item.ID, target.ID)
	msg := &UserMessage{
		UserID:         target.ID,
		Type:           TypeSystem,
		Title:          item.Title,
		Content:        item.Content,
		IdempotencyKey: &key,
	}
	if err := s.messages.CreateIdempotent(ctx, msg); err != nil {
		return err
	}
	item.MessageCount++

	result, err := s.pushToUser(ctx, target.ID, TypeSystem, item.Title, item.Content, msg.ID, 0)
	switch classifyBroadcastPush(result, err) {
	case "success":
		item.PushSuccess++
	case "skipped":
		item.PushSkipped++
	default:
		item.PushFailed++
	}
	return nil
}

func classifyBroadcastPush(result provider.PushResult, err error) string {
	if result.Success {
		return "success"
	}
	if !result.Triggered {
		return "skipped"
	}
	if err != nil {
		return "failed"
	}
	return "failed"
}
