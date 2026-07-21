package chat

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"ooop-admin-api/internal/logger"
	"ooop-admin-api/internal/user"
)

var (
	ErrReportNotFound            = errors.New("举报记录不存在")
	ErrReportReasonInvalid       = errors.New("请选择有效的举报原因")
	ErrReportDescription         = errors.New("请填写举报补充说明")
	ErrReportTooLong             = errors.New("举报补充说明不能超过 500 个字符")
	ErrReportPending             = errors.New("该会话已有待处理举报，请勿重复提交")
	ErrReportStatusInvalid       = errors.New("举报处理状态不正确")
	ErrReportResultRequired      = errors.New("请填写处理结果")
	ErrReportResultTooLong       = errors.New("处理结果不能超过 500 个字符")
	ErrReportRestrictionRequired = errors.New("举报成立时请选择聊天限制解除时间")
	ErrReportRestrictionInvalid  = errors.New("聊天限制解除时间必须晚于当前时间")
	ErrReportProcessed           = errors.New("该举报已处理")
)

type ReportNotificationPusher interface {
	PushChatReportResult(ctx context.Context, userID int64, messageID int64, title string, content string) error
}

type ReportService struct {
	reports  ReportRepository
	messages MessageRepository
	users    UserReader
	pusher   ReportNotificationPusher
}

func NewReportService(reports ReportRepository, messages MessageRepository, users UserReader, pusher ReportNotificationPusher) *ReportService {
	return &ReportService{reports: reports, messages: messages, users: users, pusher: pusher}
}

func (s *ReportService) Submit(ctx context.Context, reporterID int64, conversationID int64, input SubmitReportInput) (ReportReceipt, error) {
	reason := strings.TrimSpace(input.Reason)
	description := strings.TrimSpace(input.Description)
	if !validReportReason(reason) {
		return ReportReceipt{}, ErrReportReasonInvalid
	}
	if reason == ReportReasonOther && description == "" {
		return ReportReceipt{}, ErrReportDescription
	}
	if utf8.RuneCountInString(description) > 500 {
		return ReportReceipt{}, ErrReportTooLong
	}

	conversation, err := s.messages.FindConversationForUser(ctx, conversationID, reporterID)
	if err != nil {
		return ReportReceipt{}, err
	}
	messages, err := s.messages.ListMessages(ctx, MessageQuery{
		ConversationID:  conversationID,
		DeletedBeforeID: conversationDeletedBeforeID(conversation, reporterID),
		PageSize:        50,
	})
	if err != nil {
		return ReportReceipt{}, err
	}

	evidence := make([]ReportEvidenceMessage, 0, len(messages))
	for index := len(messages) - 1; index >= 0; index-- {
		item := messages[index]
		evidence = append(evidence, ReportEvidenceMessage{
			ID:        formatID(item.ID),
			SenderID:  formatID(item.SenderID),
			Type:      item.Type,
			Content:   item.Content,
			CreatedAt: item.CreatedAt,
		})
	}
	evidenceJSON, err := json.Marshal(evidence)
	if err != nil {
		return ReportReceipt{}, err
	}

	now := time.Now()
	item := &ChatReport{
		ConversationID: conversationID,
		ReporterID:     reporterID,
		ReportedUserID: otherUserID(conversation, reporterID),
		Reason:         reason,
		Description:    description,
		EvidenceJSON:   string(evidenceJSON),
		Status:         ReportStatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.reports.CreateReport(ctx, item); err != nil {
		return ReportReceipt{}, err
	}
	return ReportReceipt{ID: formatID(item.ID), Status: item.Status, CreatedAt: item.CreatedAt}, nil
}

func (s *ReportService) ListAdmin(ctx context.Context, query AdminReportQuery) (AdminReportListResult, error) {
	query.Page = normalizePage(query.Page)
	query.PageSize = normalizePageSize(query.PageSize, 20, 100)
	if query.Status != "" && !validReportStatus(query.Status, true) {
		return AdminReportListResult{}, ErrReportStatusInvalid
	}
	items, total, err := s.reports.ListReports(ctx, query)
	if err != nil {
		return AdminReportListResult{}, err
	}
	users, err := s.reportUsers(ctx, items)
	if err != nil {
		return AdminReportListResult{}, err
	}
	list := make([]AdminChatReport, 0, len(items))
	for _, item := range items {
		list = append(list, toAdminChatReport(item, users, false))
	}
	return AdminReportListResult{List: list, Total: total, Page: query.Page, PageSize: query.PageSize}, nil
}

func (s *ReportService) DetailAdmin(ctx context.Context, id int64) (AdminChatReport, error) {
	item, err := s.reports.FindReport(ctx, id)
	if err != nil {
		return AdminChatReport{}, err
	}
	users, err := s.reportUsers(ctx, []ChatReport{item})
	if err != nil {
		return AdminChatReport{}, err
	}
	return toAdminChatReport(item, users, true), nil
}

func (s *ReportService) Resolve(ctx context.Context, id int64, adminID int64, input ResolveReportInput) (AdminChatReport, error) {
	status := strings.TrimSpace(input.Status)
	result := strings.TrimSpace(input.Result)
	if !validReportStatus(status, false) {
		return AdminChatReport{}, ErrReportStatusInvalid
	}
	if result == "" {
		return AdminChatReport{}, ErrReportResultRequired
	}
	if utf8.RuneCountInString(result) > 500 {
		return AdminChatReport{}, ErrReportResultTooLong
	}

	now := time.Now()
	restrictionUntil := input.RestrictionUntil
	if status == ReportStatusResolved {
		if restrictionUntil == nil {
			return AdminChatReport{}, ErrReportRestrictionRequired
		}
		if !restrictionUntil.After(now) {
			return AdminChatReport{}, ErrReportRestrictionInvalid
		}
	} else {
		restrictionUntil = nil
	}

	resolution, err := s.reports.ResolveReport(ctx, id, adminID, status, result, restrictionUntil, now)
	if err != nil {
		return AdminChatReport{}, err
	}
	if s.pusher != nil {
		if err := s.pusher.PushChatReportResult(ctx, resolution.Report.ReporterID, resolution.MessageID, resolution.MessageTitle, resolution.MessageContent); err != nil {
			logger.Errorf("举报处理结果 Push 发送失败: report_id=%d, user_id=%d, error=%v", id, resolution.Report.ReporterID, err)
		}
	}
	users, err := s.reportUsers(ctx, []ChatReport{resolution.Report})
	if err != nil {
		return AdminChatReport{}, err
	}
	return toAdminChatReport(resolution.Report, users, true), nil
}

func (s *ReportService) reportUsers(ctx context.Context, items []ChatReport) (map[int64]user.User, error) {
	ids := make([]int64, 0, len(items)*2)
	seen := make(map[int64]struct{}, len(items)*2)
	for _, item := range items {
		for _, id := range []int64{item.ReporterID, item.ReportedUserID} {
			if _, exists := seen[id]; exists {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	users, err := s.users.FindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := make(map[int64]user.User, len(users))
	for _, item := range users {
		result[item.ID] = item
	}
	return result, nil
}

func toAdminChatReport(item ChatReport, users map[int64]user.User, includeEvidence bool) AdminChatReport {
	evidence := parseReportEvidence(item.EvidenceJSON)
	result := AdminChatReport{
		ID:               formatID(item.ID),
		ConversationID:   formatID(item.ConversationID),
		Reporter:         toAdminReportUser(users[item.ReporterID], item.ReporterID),
		ReportedUser:     toAdminReportUser(users[item.ReportedUserID], item.ReportedUserID),
		Reason:           item.Reason,
		Description:      item.Description,
		EvidenceCount:    len(evidence),
		Status:           item.Status,
		HandleResult:     item.HandleResult,
		HandledAt:        item.HandledAt,
		RestrictionUntil: item.RestrictionUntil,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
	}
	if item.HandlerAdminID != nil {
		result.HandlerAdminID = formatID(*item.HandlerAdminID)
	}
	if includeEvidence {
		result.Evidence = evidence
	}
	return result
}

func toAdminReportUser(item user.User, fallbackID int64) AdminReportUser {
	return AdminReportUser{
		ID:       formatID(fallbackID),
		Nickname: item.Nickname,
		Phone:    item.Phone,
		Avatar:   user.AvatarURL(item.Avatar),
	}
}

func parseReportEvidence(value string) []ReportEvidenceMessage {
	var items []ReportEvidenceMessage
	if err := json.Unmarshal([]byte(value), &items); err != nil {
		return []ReportEvidenceMessage{}
	}
	return items
}

func validReportReason(value string) bool {
	switch value {
	case ReportReasonSpam, ReportReasonHarassment, ReportReasonPornography, ReportReasonFraud, ReportReasonIllegal, ReportReasonOther:
		return true
	default:
		return false
	}
}

func validReportStatus(value string, allowPending bool) bool {
	if allowPending && value == ReportStatusPending {
		return true
	}
	return value == ReportStatusResolved || value == ReportStatusDismissed
}

func normalizePage(value int) int {
	if value <= 0 {
		return 1
	}
	return value
}
