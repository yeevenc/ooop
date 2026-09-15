package contentmoderation

import (
	"context"
	"errors"
	"testing"
)

func TestCheckerRejectsBlockedWordWithSpaces(t *testing.T) {
	checker, err := NewChecker([]string{"禁用词"})
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}
	err = checker.Check(context.Background(), SceneContent, Field{Name: "活动标题", Content: "包含禁 用词"})
	if !errors.Is(err, ErrRejected) {
		t.Fatalf("期望命中禁用词，实际错误: %v", err)
	}
}

func TestCheckerRejectsBlockedWordCaseInsensitive(t *testing.T) {
	checker, err := NewChecker([]string{"BadWord"})
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}
	err = checker.Check(context.Background(), SceneNickname, Field{Name: "昵称", Content: "xxBADwordxx"})
	if !errors.Is(err, ErrRejected) {
		t.Fatalf("期望忽略大小写命中，实际错误: %v", err)
	}
}

func TestCheckerAllowsCleanContent(t *testing.T) {
	checker, err := NewChecker([]string{"禁用词"})
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}
	err = checker.Check(context.Background(), SceneContent,
		Field{Name: "活动标题", Content: "周末羽毛球局"},
		Field{Name: "活动简介", Content: "欢迎一起打球"},
	)
	if err != nil {
		t.Fatalf("期望正常内容通过，实际错误: %v", err)
	}
}

func TestCheckerAllowsGreetingAndCityNames(t *testing.T) {
	checker, err := NewChecker(nil)
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}
	err = checker.Check(context.Background(), SceneContent,
		Field{Name: "补充说明", Content: ""},
		Field{Name: "补充说明", Content: "你好"},
		Field{Name: "补充说明", Content: "hello"},
		Field{Name: "城市", Content: "杭州市"},
	)
	if err != nil {
		t.Fatalf("日常问候和城市名应通过，实际错误: %v", err)
	}
}

func TestCheckerRejectsNoiseBypass(t *testing.T) {
	checker, err := NewChecker([]string{"违禁词"})
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}
	err = checker.Check(context.Background(), SceneContent, Field{Name: "个性签名", Content: "含违*禁_词内容"})
	if !errors.Is(err, ErrRejected) {
		t.Fatalf("期望命中插符号绕过，实际错误: %v", err)
	}
}
