package contentmoderation

import (
	"context"
	"errors"
	"testing"
)

type reviewerStub struct {
	passed bool
	err    error
}

func (r reviewerStub) Review(context.Context, string, string) (bool, error) {
	return r.passed, r.err
}

func TestCheckerRejectsBlockedWordWithSpaces(t *testing.T) {
	checker := NewChecker(nil, []string{"禁用词"})
	err := checker.Check(context.Background(), SceneContent, Field{Name: "活动标题", Content: "包含禁 用词"})
	if !errors.Is(err, ErrRejected) {
		t.Fatalf("期望命中禁用词，实际错误: %v", err)
	}
}

func TestCheckerRejectsThirdPartyRisk(t *testing.T) {
	checker := NewChecker(reviewerStub{passed: false}, nil)
	err := checker.Check(context.Background(), SceneContent, Field{Name: "活动简介", Content: "待检测内容"})
	if !errors.Is(err, ErrRejected) {
		t.Fatalf("期望第三方审核拒绝，实际错误: %v", err)
	}
}

func TestCheckerReturnsUnavailable(t *testing.T) {
	checker := NewChecker(reviewerStub{err: errors.New("timeout")}, nil)
	err := checker.Check(context.Background(), SceneNickname, Field{Name: "昵称", Content: "测试"})
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("期望审核服务不可用，实际错误: %v", err)
	}
}
