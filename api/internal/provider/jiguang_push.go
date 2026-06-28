package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ooop-admin-api/internal/config"
	"ooop-admin-api/internal/logger"
)

type JiguangPusher struct {
	cfg        config.JiguangConfig
	httpClient *http.Client
}

const defaultHarmonyOSCategory = "MARKETING"
const defaultHarmonyOSIntent = "action.system.home"

type JiguangPushPayload struct {
	Alias      string
	Title      string
	Alert      string
	ActivityID int64
}

type JiguangPushResult struct {
	Triggered bool   `json:"triggered"`
	Success   bool   `json:"success"`
	Alias     string `json:"alias"`
	Message   string `json:"message"`
	Response  string `json:"response,omitempty"`
}

func NewJiguangPusher(cfg config.JiguangConfig) *JiguangPusher {
	return &JiguangPusher{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *JiguangPusher) Push(ctx context.Context, payload JiguangPushPayload) (JiguangPushResult, error) {
	alias := strings.TrimSpace(payload.Alias)
	result := JiguangPushResult{
		Triggered: true,
		Alias:     alias,
	}
	if alias == "" {
		err := errors.New("极光推送别名不能为空")
		result.Message = err.Error()
		return result, err
	}
	if strings.TrimSpace(payload.Title) == "" || strings.TrimSpace(payload.Alert) == "" {
		err := errors.New("极光推送标题或内容不能为空")
		result.Message = err.Error()
		return result, err
	}
	if strings.TrimSpace(p.cfg.PushURL) == "" {
		err := errors.New("极光推送地址未配置")
		result.Message = err.Error()
		return result, err
	}
	if strings.TrimSpace(p.cfg.AppKey) == "" || strings.TrimSpace(p.cfg.MasterSecret) == "" {
		err := errors.New("极光推送鉴权配置缺失")
		result.Message = err.Error()
		return result, err
	}

	requestBody := map[string]interface{}{
		"platform": []string{"hmos"},
		"audience": map[string]interface{}{
			"alias": []string{alias},
		},
		"notification": map[string]interface{}{
			"alert": payload.Alert,
			"hmos": map[string]interface{}{
				"alert":    payload.Alert,
				"title":    payload.Title,
				"category": defaultHarmonyOSCategory,
				"intent": map[string]interface{}{
					"url": defaultHarmonyOSIntent,
				},
				"badge_add_num": 1,
				"push_type":     0,
				"test_message":  false,
				"extras": map[string]interface{}{
					"activityId": fmt.Sprintf("%d", payload.ActivityID),
				},
			},
		},
		"options": map[string]interface{}{
			"time_to_live":   0,
			"classification": 1, // 消息分类，0：代表运营消息。1：代表系统消息。
		},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		result.Message = err.Error()
		return result, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.PushURL, bytes.NewReader(body))
	if err != nil {
		result.Message = err.Error()
		return result, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(p.cfg.AppKey, p.cfg.MasterSecret)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		result.Message = err.Error()
		return result, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Message = err.Error()
		return result, err
	}
	result.Response = string(respBody)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		err := fmt.Errorf("极光推送请求失败: %s, response=%s", resp.Status, string(respBody))
		result.Message = err.Error()
		return result, err
	}

	var resultData map[string]interface{}
	if err := json.Unmarshal(respBody, &resultData); err != nil {
		result.Message = err.Error()
		return result, err
	}
	if rawError, ok := resultData["error"]; ok {
		errorText, _ := json.Marshal(rawError)
		err := fmt.Errorf("极光推送返回失败: %s", string(errorText))
		result.Message = err.Error()
		return result, err
	}

	logger.Infof("极光推送发送成功: alias=%s, title=%s, response=%s", alias, payload.Title, string(respBody))
	result.Success = true
	result.Message = "极光推送发送成功"
	return result, nil
}
