package contentmoderation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"

	sensitive "github.com/zmexing/go-sensitive-word"
	sensitivefilter "github.com/zmexing/go-sensitive-word/filter"
)

var (
	ErrRejected    = errors.New("内容包含敏感词，请修改后重试")
	ErrUnavailable = errors.New("内容审核服务暂时不可用，请稍后重试")
)

const (
	SceneNickname = "nickname"
	SceneContent  = "content"
)

type Field struct {
	Name    string
	Content string
}

// Checker 本地敏感词检测（免费开源词库 + 自定义禁用词），不依赖任何收费第三方。
type Checker struct {
	filter sensitivefilter.Filter
}

// NewChecker 初始化本地过滤器：加载内置词库，并合并自定义禁用词。
func NewChecker(extraWords []string) (*Checker, error) {
	words := appendSensitiveDictionaryWords(
		nil,
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
	)
	for _, word := range extraWords {
		word = strings.ToLower(strings.TrimSpace(word))
		if word == "" {
			continue
		}
		words = append(words, word)
		// 额外写入去空白形态，覆盖「插空格」配置写法
		if normalized := normalize(word); normalized != "" && normalized != strings.ToLower(word) {
			words = append(words, normalized)
		}
	}
	filter := sensitivefilter.NewDfaModel()
	filter.AddWords(words...)
	return &Checker{filter: filter}, nil
}

func appendSensitiveDictionaryWords(words []string, contents ...string) []string {
	for _, content := range contents {
		for _, line := range strings.Split(content, "\n") {
			word := strings.ToLower(strings.TrimSpace(line))
			if word != "" {
				words = append(words, word)
			}
		}
	}
	return words
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
