package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"ooop-admin-api/internal/config"
	"ooop-admin-api/internal/contentmoderation"
)

type AliyunContentModerator struct {
	client *AliyunRPCClient
	config config.AliyunContentModerationConfig
}

func NewAliyunContentModerator(client *AliyunRPCClient, cfg config.AliyunContentModerationConfig) *AliyunContentModerator {
	return &AliyunContentModerator{client: client, config: cfg}
}

func (m *AliyunContentModerator) Review(ctx context.Context, scene string, content string) (bool, error) {
	service := m.config.ContentService
	if scene == contentmoderation.SceneNickname {
		service = m.config.NicknameService
	}
	if service == "" {
		return false, errors.New("阿里云内容审核服务类型未配置")
	}
	for _, chunk := range splitText(content, 600) {
		passed, err := m.reviewChunk(ctx, service, chunk)
		if err != nil || !passed {
			return passed, err
		}
	}
	return true, nil
}

func (m *AliyunContentModerator) reviewChunk(ctx context.Context, service string, content string) (bool, error) {
	parameters, err := json.Marshal(map[string]string{"content": content})
	if err != nil {
		return false, err
	}
	result, err := m.client.CallPOST(ctx, m.config.Endpoint, map[string]string{
		"Version":           "2022-03-02",
		"Action":            "TextModeration",
		"Service":           service,
		"ServiceParameters": string(parameters),
	})
	if err != nil {
		return false, err
	}
	if code := numberValue(result["Code"]); code != 200 {
		return false, fmt.Errorf("阿里云内容审核失败: code=%d message=%v", code, result["Message"])
	}
	data, ok := result["Data"].(map[string]interface{})
	if !ok {
		return false, errors.New("阿里云内容审核返回数据格式不正确")
	}
	labels, _ := data["labels"].(string)
	if labels == "" {
		labels, _ = data["Labels"].(string)
	}
	return strings.TrimSpace(labels) == "", nil
}

func splitText(content string, limit int) []string {
	runes := []rune(content)
	if len(runes) == 0 || limit <= 0 {
		return nil
	}
	chunks := make([]string, 0, (len(runes)+limit-1)/limit)
	for start := 0; start < len(runes); start += limit {
		end := start + limit
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
	}
	return chunks
}

func numberValue(value interface{}) int {
	switch current := value.(type) {
	case float64:
		return int(current)
	case json.Number:
		var result int
		_, _ = fmt.Sscan(current.String(), &result)
		return result
	case string:
		var result int
		_, _ = fmt.Sscan(current, &result)
		return result
	default:
		return 0
	}
}
