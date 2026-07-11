package contentmoderation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

var (
	ErrRejected    = errors.New("内容包含不适宜信息，请修改后重试")
	ErrUnavailable = errors.New("内容审核服务暂时不可用，请稍后重试")
)

const (
	SceneNickname = "nickname"
	SceneContent  = "content"
)

type Reviewer interface {
	Review(ctx context.Context, scene string, content string) (bool, error)
}

type Field struct {
	Name    string
	Content string
}

type Checker struct {
	reviewer     Reviewer
	blockedWords []string
}

func NewChecker(reviewer Reviewer, blockedWords []string) *Checker {
	words := make([]string, 0, len(blockedWords))
	for _, word := range blockedWords {
		word = normalize(word)
		if word != "" {
			words = append(words, word)
		}
	}
	return &Checker{reviewer: reviewer, blockedWords: words}
}

func (c *Checker) Check(ctx context.Context, scene string, fields ...Field) error {
	if c == nil {
		return nil
	}

	validFields := make([]Field, 0, len(fields))
	for _, field := range fields {
		content := strings.TrimSpace(field.Content)
		if content == "" {
			continue
		}
		normalized := normalize(content)
		for _, word := range c.blockedWords {
			if strings.Contains(normalized, word) {
				return fmt.Errorf("%s包含禁用词: %w", field.Name, ErrRejected)
			}
		}
		validFields = append(validFields, Field{Name: field.Name, Content: content})
	}

	if len(validFields) == 0 || c.reviewer == nil {
		return nil
	}
	for _, field := range validFields {
		passed, err := c.reviewer.Review(ctx, scene, field.Content)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrUnavailable, err)
		}
		if !passed {
			return fmt.Errorf("%s: %w", field.Name, ErrRejected)
		}
	}
	return nil
}

// normalize 统一大小写并移除空白，减少通过插入空格绕过本地词库的情况。
func normalize(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return unicode.ToLower(r)
	}, strings.TrimSpace(value))
}
