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

const defaultHarmonyOSDistribution = "jpush"
const defaultPushTimeToLive = 300

func NewJiguangPusher(cfg config.JiguangConfig) *JiguangPusher {
	return &JiguangPusher{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *JiguangPusher) Push(ctx context.Context, payload PushPayload) (PushChannelResult, error) {
	alias := strings.TrimSpace(payload.Alias)
	result := PushChannelResult{
		Channel: PushChannelJiguang,
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
		"message": map[string]interface{}{
			"msg_content":  payload.Alert,
			"title":        payload.Title,
			"content_type": "text",
			"extras": map[string]interface{}{
				"messageId":  fmt.Sprintf("%d", payload.MessageID),
				"activityId": fmt.Sprintf("%d", payload.ActivityID),
				"type":       payload.MessageType,
			},
		},
		"options": map[string]interface{}{
			"time_to_live": defaultPushTimeToLive,
			"third_party_channel": map[string]interface{}{
				"hmos": map[string]interface{}{
					// 极光只承担进程存活时的实时消息，系统通知统一由 Push Kit 下发。
					"distribution": defaultHarmonyOSDistribution,
				},
			},
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
	result.Triggered = true

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
	result.Message = "极光自定义消息发送成功"
	return result, nil
}
