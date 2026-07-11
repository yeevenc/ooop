package contentmoderation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	sensitive "github.com/zmexing/go-sensitive-word"
)

var (
	ErrRejected    = errors.New("内容包含敏感词，请修改后重试")
	ErrUnavailable = errors.New("内容审核服务暂时不可用，请稍后重试")
)

const (
	SceneNickname = "nickname"
	SceneContent  = "content"

	// 词库经 channel 异步写入 DFA，用探针词确认就绪（不会与正常用户内容重合）
	readyProbeWord = "__ooop_sensitive_ready_probe__"
)

type Field struct {
	Name    string
	Content string
}

// Checker 本地敏感词检测（免费开源词库 + 自定义禁用词），不依赖任何收费第三方。
type Checker struct {
	filter *sensitive.Manager
}

// NewChecker 初始化本地过滤器：加载内置词库，并合并自定义禁用词。
func NewChecker(extraWords []string) (*Checker, error) {
	filter, err := sensitive.NewFilter(
		sensitive.StoreOption{Type: sensitive.StoreMemory},
		sensitive.FilterOption{Type: sensitive.FilterDfa},
	)
	if err != nil {
		return nil, fmt.Errorf("初始化敏感词过滤器失败: %w", err)
	}

	// 内置词库覆盖政治/暴恐/色情/民生等，全部本地运行、零费用。
	if err := filter.LoadDictEmbed(
		sensitive.DictReactionary,
		sensitive.DictViolence,
		sensitive.DictPornography,
		sensitive.DictSexual,
		sensitive.DictPolitical,
		sensitive.DictGunExplosion,
		sensitive.DictPeopleLife,
		sensitive.DictCorruption,
		sensitive.DictAdditional,
		sensitive.DictOther,
		sensitive.DictTemporaryTencent,
		sensitive.DictGFWAdditional,
		sensitive.DictNeteaseFE,
	); err != nil {
		return nil, fmt.Errorf("加载敏感词词库失败: %w", err)
	}

	words := make([]string, 0, len(extraWords)+1)
	for _, word := range extraWords {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}
		words = append(words, word)
		// 额外写入去空白形态，覆盖「插空格」配置写法
		if normalized := normalize(word); normalized != "" && normalized != strings.ToLower(word) {
			words = append(words, normalized)
		}
	}
	// 探针词放在最后，就绪后即可认为此前入队词均已写入 DFA
	words = append(words, readyProbeWord)
	if err := filter.AddWord(words...); err != nil {
		return nil, fmt.Errorf("添加自定义禁用词失败: %w", err)
	}
	if err := waitFilterReady(filter, readyProbeWord, 3*time.Second); err != nil {
		return nil, err
	}

	return &Checker{filter: filter}, nil
}

// waitFilterReady 等待 DFA 异步消费完成，避免启动后立刻检测漏过。
func waitFilterReady(filter *sensitive.Manager, probe string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if filter.IsSensitive(probe) {
			return nil
		}
		if time.Now().After(deadline) {
			return errors.New("敏感词过滤器就绪超时")
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// Check 检测多个字段；任一命中敏感词则返回 ErrRejected，业务侧映射为敏感词提示。
func (c *Checker) Check(ctx context.Context, scene string, fields ...Field) error {
	_ = ctx
	_ = scene
	if c == nil || c.filter == nil {
		return nil
	}

	for _, field := range fields {
		content := strings.TrimSpace(field.Content)
		if content == "" {
			continue
		}
		// 原文检测（库内会转小写）+ 去空白/噪声检测，防止插空格、符号绕过
		if c.filter.IsSensitive(content) || c.filter.IsSensitive(normalize(content)) {
			return fmt.Errorf("%s: %w", field.Name, ErrRejected)
		}
	}
	return nil
}

// normalize 统一大小写并移除空白与常见干扰符，减少绕过本地词库的情况。
func normalize(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		switch r {
		case '*', '@', '_', '-', '·', '.', '`', '~', '|', '/', '\\', '#',
			'!', '！', ',', '，', '。', '、', '　', '\u200b', '\u200c', '\u200d', '\ufeff':
			return -1
		}
		return unicode.ToLower(r)
	}, strings.TrimSpace(value))
}
