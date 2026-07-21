package chat

import (
	"context"
	"errors"
	"testing"
	"time"

	"ooop-admin-api/internal/user"
)

type reportTestRepository struct {
	created          ChatReport
	resolution       ReportResolution
	restrictionUntil *time.Time
}

func (r *reportTestRepository) CreateReport(_ context.Context, item *ChatReport) error {
	item.ID = 12
	r.created = *item
	return nil
}

func (*reportTestRepository) ListReports(context.Context, AdminReportQuery) ([]ChatReport, int64, error) {
	return nil, 0, nil
}

func (*reportTestRepository) FindReport(context.Context, int64) (ChatReport, error) {
	return ChatReport{}, nil
}

func (r *reportTestRepository) ResolveReport(_ context.Context, _ int64, _ int64, _ string, _ string, restrictionUntil *time.Time, _ time.Time) (ReportResolution, error) {
	r.restrictionUntil = restrictionUntil
	return r.resolution, nil
}

type reportTestPusher struct {
	userID    int64
	messageID int64
}

func (p *reportTestPusher) PushChatReportResult(_ context.Context, userID int64, messageID int64, _ string, _ string) error {
	p.userID = userID
	p.messageID = messageID
	return nil
}

func TestSubmitReportCapturesConversationEvidence(t *testing.T) {
	reports := &reportTestRepository{}
	messages := &serviceTestRepository{
		conversation: Conversation{ID: 9, UserAID: 3000, UserBID: 3001},
		messageItems: []Message{{
			ID:             10,
			ConversationID: 9,
			SenderID:       3001,
			Type:           MessageTypeText,
			Content:        "测试聊天内容",
			CreatedAt:      time.Now(),
		}},
	}
	service := NewReportService(reports, messages, serviceTestUsers{}, nil)

	result, err := service.Submit(context.Background(), 3000, 9, SubmitReportInput{
		Reason:      ReportReasonHarassment,
		Description: "对方持续骚扰",
	})
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	if result.ID != "12" || reports.created.ReportedUserID != 3001 {
		t.Fatalf("result = %+v, report = %+v", result, reports.created)
	}
	evidence := parseReportEvidence(reports.created.EvidenceJSON)
	if len(evidence) != 1 || evidence[0].Content != "测试聊天内容" {
		t.Fatalf("evidence = %+v", evidence)
	}
}

func TestSubmitOtherReportRequiresDescription(t *testing.T) {
	service := NewReportService(&reportTestRepository{}, &serviceTestRepository{}, serviceTestUsers{}, nil)
	_, err := service.Submit(context.Background(), 3000, 9, SubmitReportInput{Reason: ReportReasonOther})
	if err != ErrReportDescription {
		t.Fatalf("error = %v, want ErrReportDescription", err)
	}
}

func TestResolveReportPushesStoredResult(t *testing.T) {
	reports := &reportTestRepository{resolution: ReportResolution{
		Report:         ChatReport{ID: 12, ReporterID: 3000, ReportedUserID: 3001, Status: ReportStatusResolved},
		MessageID:      88,
		MessageTitle:   "举报处理结果",
		MessageContent: "处理完成",
	}}
	pusher := &reportTestPusher{}
	service := NewReportService(reports, &serviceTestRepository{}, serviceTestUsers{items: map[int64]user.User{
		3000: {ID: 3000},
		3001: {ID: 3001},
	}}, pusher)

	_, err := service.Resolve(context.Background(), 12, 1, ResolveReportInput{
		Status:           ReportStatusResolved,
		Result:           "举报成立，已处理",
		RestrictionUntil: timePointer(time.Now().Add(24 * time.Hour)),
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if pusher.userID != 3000 || pusher.messageID != 88 {
		t.Fatalf("push user = %d, message = %d", pusher.userID, pusher.messageID)
	}
	if reports.restrictionUntil == nil {
		t.Fatal("restrictionUntil is nil")
	}
}

func TestResolveReportRequiresFutureRestrictionTime(t *testing.T) {
	service := NewReportService(&reportTestRepository{}, &serviceTestRepository{}, serviceTestUsers{}, nil)
	_, err := service.Resolve(context.Background(), 12, 1, ResolveReportInput{
		Status: ReportStatusResolved,
		Result: "举报成立",
	})
	if !errors.Is(err, ErrReportRestrictionRequired) {
		t.Fatalf("error = %v, want ErrReportRestrictionRequired", err)
	}
}

func timePointer(value time.Time) *time.Time {
	return &value
}
