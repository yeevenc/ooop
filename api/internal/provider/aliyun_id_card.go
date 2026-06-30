package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ooop-admin-api/internal/config"
)

type RealNameVerifier interface {
	Verify(ctx context.Context, name string, idCard string) (RealNameVerifyResult, error)
}

type RealNameVerifyResult struct {
	Passed  bool
	Message string
}

type AliyunIDCardVerifier struct {
	cfg    config.AliyunIDCardConfig
	client *http.Client
}

func NewAliyunIDCardVerifier(cfg config.AliyunIDCardConfig) *AliyunIDCardVerifier {
	return &AliyunIDCardVerifier{
		cfg: cfg,
		client: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

func (v *AliyunIDCardVerifier) Verify(ctx context.Context, name string, idCard string) (RealNameVerifyResult, error) {
	endpoint := strings.TrimSpace(v.cfg.Endpoint)
	appCode := strings.TrimSpace(v.cfg.AppCode)
	if endpoint == "" || appCode == "" {
		return RealNameVerifyResult{}, errors.New("实名认证服务未配置")
	}

	form := url.Values{}
	form.Set("name", strings.TrimSpace(name))
	form.Set("idcard", strings.TrimSpace(idCard))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return RealNameVerifyResult{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "APPCODE "+appCode)

	resp, err := v.client.Do(req)
	if err != nil {
		return RealNameVerifyResult{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return RealNameVerifyResult{}, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return RealNameVerifyResult{}, fmt.Errorf("实名认证服务请求失败: %s", resp.Status)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return RealNameVerifyResult{}, errors.New("实名认证服务返回格式不正确")
	}

	message := aliyunIDCardMessage(payload)
	return RealNameVerifyResult{
		Passed:  aliyunIDCardPassed(payload),
		Message: message,
	}, nil
}

func aliyunIDCardPassed(payload map[string]interface{}) bool {
	if code, ok := findNumberByDirectKey(payload, "code"); ok {
		if code != 200 {
			return false
		}
		data, ok := findMapByDirectKey(payload, "data")
		if !ok {
			return false
		}
		if result, ok := findNumberByDirectKey(data, "result"); ok {
			return result == 0
		}
		desc := strings.TrimSpace(findTextByDirectKey(data, "desc"))
		return desc == "一致"
	}

	if value, ok := findBoolByKeys(payload, "passed", "match", "matched"); ok && value {
		return true
	}

	text := strings.ToLower(strings.TrimSpace(strings.Join([]string{
		findTextByKeys(payload, "status", "result", "res", "state"),
		findTextByKeys(payload, "msg", "message", "desc", "result_msg", "reason"),
	}, " ")))

	if text == "" {
		return false
	}

	successTokens := []string{"200", "0", "00", "true", "pass", "passed", "match", "matched", "一致", "匹配", "认证成功", "核验成功"}
	failTokens := []string{"false", "fail", "failed", "mismatch", "不一致", "不匹配", "失败", "错误"}
	for _, token := range failTokens {
		if strings.Contains(text, token) {
			return false
		}
	}
	for _, token := range successTokens {
		if strings.Contains(text, token) {
			return true
		}
	}
	return false
}

func aliyunIDCardMessage(payload map[string]interface{}) string {
	if data, ok := findMapByDirectKey(payload, "data"); ok {
		if text := findTextByDirectKey(data, "desc"); text != "" {
			return text
		}
	}
	if text := findTextByKeys(payload, "msg", "message", "desc", "result_msg", "reason"); text != "" {
		return text
	}
	if text := findTextByKeys(payload, "code", "status", "result"); text != "" {
		return text
	}
	return "实名认证未通过"
}

func findTextByKeys(value interface{}, keys ...string) string {
	switch current := value.(type) {
	case map[string]interface{}:
		for _, key := range keys {
			if raw, ok := current[key]; ok {
				switch item := raw.(type) {
				case string:
					return strings.TrimSpace(item)
				case float64:
					return fmt.Sprintf("%.0f", item)
				case bool:
					if item {
						return "true"
					}
					return "false"
				}
			}
		}
		for _, item := range current {
			if text := findTextByKeys(item, keys...); text != "" {
				return text
			}
		}
	case []interface{}:
		for _, item := range current {
			if text := findTextByKeys(item, keys...); text != "" {
				return text
			}
		}
	}
	return ""
}

func findTextByDirectKey(value map[string]interface{}, key string) string {
	raw, ok := value[key]
	if !ok {
		return ""
	}
	switch item := raw.(type) {
	case string:
		return strings.TrimSpace(item)
	case float64:
		return fmt.Sprintf("%.0f", item)
	case bool:
		if item {
			return "true"
		}
		return "false"
	}
	return ""
}

func findNumberByDirectKey(value map[string]interface{}, key string) (int, bool) {
	raw, ok := value[key]
	if !ok {
		return 0, false
	}
	switch item := raw.(type) {
	case float64:
		return int(item), true
	case string:
		text := strings.TrimSpace(item)
		if text == "" {
			return 0, false
		}
		var number int
		if _, err := fmt.Sscanf(text, "%d", &number); err != nil {
			return 0, false
		}
		return number, true
	}
	return 0, false
}

func findMapByDirectKey(value map[string]interface{}, key string) (map[string]interface{}, bool) {
	raw, ok := value[key]
	if !ok {
		return nil, false
	}
	item, ok := raw.(map[string]interface{})
	return item, ok
}

func findBoolByKeys(value interface{}, keys ...string) (bool, bool) {
	switch current := value.(type) {
	case map[string]interface{}:
		for _, key := range keys {
			if raw, ok := current[key]; ok {
				if item, ok := raw.(bool); ok {
					return item, true
				}
			}
		}
		for _, item := range current {
			if value, ok := findBoolByKeys(item, keys...); ok {
				return value, true
			}
		}
	case []interface{}:
		for _, item := range current {
			if value, ok := findBoolByKeys(item, keys...); ok {
				return value, true
			}
		}
	}
	return false, false
}
