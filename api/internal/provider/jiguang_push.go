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
)

type JiguangPusher struct {
	cfg        config.JiguangConfig
	httpClient *http.Client
}

const defaultHarmonyOSCategory = "CATEGORY_RECOMMENDATION"
const defaultHarmonyOSIntent = "action.system.home"

type JiguangPushPayload struct {
	Alias      string
	Title      string
	Alert      string
	ActivityID int64
}

func NewJiguangPusher(cfg config.JiguangConfig) *JiguangPusher {
	return &JiguangPusher{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *JiguangPusher) Push(ctx context.Context, payload JiguangPushPayload) error {
	alias := strings.TrimSpace(payload.Alias)
	if alias == "" {
		return errors.New("极光推送别名不能为空")
	}
	if strings.TrimSpace(payload.Title) == "" || strings.TrimSpace(payload.Alert) == "" {
		return errors.New("极光推送标题或内容不能为空")
	}
	if strings.TrimSpace(p.cfg.PushURL) == "" {
		return errors.New("极光推送地址未配置")
	}
	if strings.TrimSpace(p.cfg.AppKey) == "" || strings.TrimSpace(p.cfg.MasterSecret) == "" {
		return errors.New("极光推送鉴权配置缺失")
	}

	requestBody := map[string]interface{}{
		"platform": "all",
		"audience": map[string]interface{}{
			"alias": []string{alias},
		},
		"notification": map[string]interface{}{
			"alert": payload.Alert,
			"hmos": map[string]interface{}{
				"alert":    payload.Alert,
				"title":    payload.Title,
				"category": defaultHarmonyOSCategory,
				"intent":   defaultHarmonyOSIntent,
				"extras": map[string]interface{}{
					"activityId": fmt.Sprintf("%d", payload.ActivityID),
				},
			},
		},
		"options": map[string]interface{}{
			"time_to_live":   86400,
			"classification": 1, // 消息分类，0：代表运营消息。1：代表系统消息。
			"active_push":    true,
		},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.PushURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(p.cfg.AppKey, p.cfg.MasterSecret)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("极光推送请求失败: %s", resp.Status)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return err
	}
	if rawError, ok := result["error"]; ok {
		errorText, _ := json.Marshal(rawError)
		return fmt.Errorf("极光推送返回失败: %s", string(errorText))
	}

	return nil
}
