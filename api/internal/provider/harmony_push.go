package provider

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"ooop-admin-api/internal/config"
	"ooop-admin-api/internal/logger"
)

// push-type: 0 = Alert 通知消息（见 push-send-alert）
const harmonyPushTypeNotification = "0"

// 通知消息离线缓存：文档示例常用 86400 秒（1 天），覆盖杀进程后回连场景
const harmonyPushTimeToLive = 86400

// notifyId 合法范围 [0, 2147483647]
const harmonyMaxNotifyID = 2147483647

type HarmonyPusher struct {
	cfg         config.HarmonyPushConfig
	httpClient  *http.Client
	account     harmonyServiceAccount
	accountErr  error
	accountOnce sync.Once
	tokenMu     sync.Mutex
	token       string
	tokenExpiry time.Time
}

type harmonyServiceAccount struct {
	ProjectID  string `json:"project_id"`
	KeyID      string `json:"key_id"`
	PrivateKey string `json:"private_key"`
	SubAccount string `json:"sub_account"`
	TokenURI   string `json:"token_uri"`
}

type harmonyPushRequest struct {
	Payload     harmonyPayload     `json:"payload"`
	Target      harmonyTarget      `json:"target"`
	PushOptions harmonyPushOptions `json:"pushOptions"`
}

type harmonyPayload struct {
	Notification harmonyNotification `json:"notification"`
}

// 对齐华为 V3 通知消息体：category/title/body/clickAction 必填语义，
// foregroundShow=false 时前台不展示通知栏，由客户端 receiveMessage 处理。
type harmonyNotification struct {
	Category       string             `json:"category"`
	Title          string             `json:"title"`
	Body           string             `json:"body"`
	ClickAction    harmonyClickAction `json:"clickAction"`
	ForegroundShow bool               `json:"foregroundShow"`
	Badge          harmonyBadge       `json:"badge"`
	NotifyID       *int               `json:"notifyId,omitempty"`
}

// actionType: 0 打开应用首页；data 在客户端 onCreate/onNewWant 的 want.parameters 中读取
type harmonyClickAction struct {
	ActionType int               `json:"actionType"`
	Data       map[string]string `json:"data,omitempty"`
}

type harmonyBadge struct {
	AddNum int `json:"addNum"`
}

type harmonyTarget struct {
	Token []string `json:"token"`
}

type harmonyPushOptions struct {
	TTL         int  `json:"ttl"`
	TestMessage bool `json:"testMessage,omitempty"`
}

type harmonyPushResponse struct {
	Code      string `json:"code"`
	Message   string `json:"msg"`
	RequestID string `json:"requestId"`
}

func NewHarmonyPusher(cfg config.HarmonyPushConfig) *HarmonyPusher {
	return &HarmonyPusher{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *HarmonyPusher) Push(ctx context.Context, payload PushPayload) (PushChannelResult, error) {
	result := PushChannelResult{Channel: PushChannelHarmony}
	token := strings.TrimSpace(payload.HarmonyPushToken)
	if token == "" {
		result.Message = "鸿蒙 Push Token 未绑定"
		return result, nil
	}
	logger.Infof("鸿蒙推送准备发送: message_id=%d, token_len=%d, token=%s", payload.MessageID, len(token), token)
	if strings.TrimSpace(p.cfg.ServiceAccountFile) == "" {
		err := errors.New("鸿蒙 Service Account 文件未配置，请设置环境变量 HARMONY_PUSH_SERVICE_ACCOUNT_FILE 指向 AGC 服务账号 JSON")
		result.Message = err.Error()
		return result, err
	}
	if strings.TrimSpace(payload.Title) == "" || strings.TrimSpace(payload.Alert) == "" {
		err := errors.New("鸿蒙推送标题或内容不能为空")
		result.Message = err.Error()
		return result, err
	}

	jwt, err := p.authorizationToken()
	if err != nil {
		result.Message = err.Error()
		return result, err
	}

	// 确保 project_id 已加载（authorizationToken 会触发 serviceAccount）
	account, err := p.serviceAccount()
	if err != nil {
		result.Message = err.Error()
		return result, err
	}

	requestBody := harmonyPushRequest{
		Payload: harmonyPayload{
			Notification: harmonyNotification{
				// category 必填；非法/空值回落 MARKETING，避免因自定义字符串发送失败
				Category:       normalizeHarmonyCategory(payload.Category),
				Title:          strings.TrimSpace(payload.Title),
				Body:           strings.TrimSpace(payload.Alert),
				ForegroundShow: false,
				ClickAction: harmonyClickAction{
					ActionType: 0,
					Data:       buildHarmonyClickData(payload),
				},
				Badge:    harmonyBadge{AddNum: 1},
				NotifyID: buildHarmonyNotifyID(payload.MessageID),
			},
		},
		Target: harmonyTarget{Token: []string{token}},
		PushOptions: harmonyPushOptions{
			TTL:         harmonyPushTimeToLive,
			TestMessage: p.cfg.TestMessage,
		},
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		result.Message = err.Error()
		return result, err
	}

	// POST https://push-api.cloud.huawei.com/v3/{projectId}/messages:send
	pushURL := fmt.Sprintf("%s/v3/%s/messages:send", strings.TrimRight(p.cfg.PushURL, "/"), account.ProjectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, pushURL, bytes.NewReader(body))
	if err != nil {
		result.Message = err.Error()
		return result, err
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("push-type", harmonyPushTypeNotification)
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
		err := fmt.Errorf("鸿蒙推送请求失败: %s, response=%s", resp.Status, string(respBody))
		result.Message = err.Error()
		return result, err
	}

	var response harmonyPushResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		result.Message = err.Error()
		return result, err
	}
	if !isHarmonyPushSuccess(response.Code) {
		err := fmt.Errorf("鸿蒙推送返回失败: code=%s, message=%s", response.Code, response.Message)
		result.Message = err.Error()
		return result, err
	}

	logger.Infof("鸿蒙推送发送成功: message_id=%d, request_id=%s", payload.MessageID, response.RequestID)
	result.Success = true
	result.Message = "鸿蒙通知发送成功"
	return result, nil
}

func (p *HarmonyPusher) authorizationToken() (string, error) {
	p.tokenMu.Lock()
	defer p.tokenMu.Unlock()

	now := time.Now()
	if p.token != "" && now.Add(10*time.Second).Before(p.tokenExpiry) {
		return p.token, nil
	}
	account, err := p.serviceAccount()
	if err != nil {
		return "", err
	}
	expiresAt := now.Add(time.Hour)
	token, err := signHarmonyJWT(account, now.Unix(), expiresAt.Unix())
	if err != nil {
		return "", err
	}
	p.token = token
	p.tokenExpiry = expiresAt
	return token, nil
}

func (p *HarmonyPusher) serviceAccount() (harmonyServiceAccount, error) {
	p.accountOnce.Do(func() {
		// 对齐文档：从 AGC 服务账号 JSON 读取 project_id / key_id / private_key / sub_account / token_uri
		// https://developer.huawei.com/consumer/cn/doc/harmonyos-guides/push-jwt-token
		if strings.TrimSpace(p.cfg.ServiceAccountFile) == "" {
			p.accountErr = errors.New("鸿蒙 Service Account 文件未配置，请设置 HARMONY_PUSH_SERVICE_ACCOUNT_FILE")
			return
		}
		body, err := os.ReadFile(p.cfg.ServiceAccountFile)
		if err != nil {
			p.accountErr = fmt.Errorf("读取鸿蒙 Service Account 失败(%s): %w", p.cfg.ServiceAccountFile, err)
			return
		}
		if err := json.Unmarshal(body, &p.account); err != nil {
			p.accountErr = fmt.Errorf("解析鸿蒙 Service Account 失败: %w", err)
			return
		}
		if strings.TrimSpace(p.account.ProjectID) == "" || strings.TrimSpace(p.account.KeyID) == "" ||
			strings.TrimSpace(p.account.PrivateKey) == "" || strings.TrimSpace(p.account.SubAccount) == "" ||
			strings.TrimSpace(p.account.TokenURI) == "" {
			p.accountErr = errors.New("鸿蒙 Service Account 配置不完整，需包含 project_id/key_id/private_key/sub_account/token_uri")
			return
		}
		logger.Infof(
			"鸿蒙 Service Account 已加载: project_id=%s, key_id=%s, sub_account=%s, token_uri=%s",
			p.account.ProjectID,
			p.account.KeyID,
			p.account.SubAccount,
			p.account.TokenURI,
		)
	})
	return p.account, p.accountErr
}

// 按华为文档签发 JWT：
// Header: alg=PS256, kid=key_id, typ=JWT
// Payload: iss=sub_account, aud=token_uri, iat, exp（有效期 1 小时）
// Signature: SHA256withRSA/PSS
func signHarmonyJWT(account harmonyServiceAccount, issuedAt int64, expiresAt int64) (string, error) {
	header, err := json.Marshal(map[string]interface{}{
		"alg": "PS256",
		"kid": account.KeyID,
		"typ": "JWT",
	})
	if err != nil {
		return "", err
	}
	claims, err := json.Marshal(map[string]interface{}{
		"aud": account.TokenURI,
		"exp": expiresAt,
		"iat": issuedAt,
		"iss": account.SubAccount,
	})
	if err != nil {
		return "", err
	}
	encode := base64.RawURLEncoding.EncodeToString
	signingInput := encode(header) + "." + encode(claims)
	digest := sha256.Sum256([]byte(signingInput))
	privateKey, err := parseHarmonyPrivateKey(account.PrivateKey)
	if err != nil {
		return "", err
	}
	signature, err := rsa.SignPSS(rand.Reader, privateKey, crypto.SHA256, digest[:], &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthEqualsHash,
		Hash:       crypto.SHA256,
	})
	if err != nil {
		return "", fmt.Errorf("签发鸿蒙 JWT 失败: %w", err)
	}
	return signingInput + "." + encode(signature), nil
}

// ValidateServiceAccount 启动时预检：文件可读、字段完整、私钥可解析
func (p *HarmonyPusher) ValidateServiceAccount() error {
	account, err := p.serviceAccount()
	if err != nil {
		return err
	}
	if _, err := parseHarmonyPrivateKey(account.PrivateKey); err != nil {
		return err
	}
	// 试签一次，确认 PS256 可用
	now := time.Now().Unix()
	if _, err := signHarmonyJWT(account, now, now+3600); err != nil {
		return err
	}
	return nil
}

func parseHarmonyPrivateKey(value string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(value))
	if block == nil {
		return nil, errors.New("鸿蒙 Service Account 私钥格式不正确")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("解析鸿蒙 Service Account 私钥失败: %w", err)
	}
	privateKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("鸿蒙 Service Account 私钥不是 RSA 私钥")
	}
	return privateKey, nil
}

func isHarmonyPushSuccess(code string) bool {
	// 文档/现网常见成功码：0、80000000
	return code == "0" || code == "80000000"
}

// 官方允许的 category 白名单（未申请权益时云端会当 MARKETING 处理，但必须是合法枚举）
var harmonyAllowedCategories = map[string]struct{}{
	HarmonyCategoryMarketing:    {},
	HarmonyCategoryWork:         {},
	HarmonyCategorySubscription: {},
	HarmonyCategoryAccount:      {},
	"IM":                        {},
	"VOIP":                      {},
	"MISS_CALL":                 {},
	"TRAVEL":                    {},
	"HEALTH":                    {},
	"EXPRESS":                   {},
	"FINANCE":                   {},
	"DEVICE_REMINDER":           {},
	"MAIL":                      {},
	"PLAY_VOICE":                {},
}

func normalizeHarmonyCategory(category string) string {
	value := strings.ToUpper(strings.TrimSpace(category))
	if value == "" {
		return HarmonyCategoryMarketing
	}
	// 纠正历史错误取值
	switch value {
	case "SYSTEM_REMINDER":
		return HarmonyCategoryWork
	case "SOCIAL_DYNAMICS":
		return HarmonyCategorySubscription
	}
	if _, ok := harmonyAllowedCategories[value]; ok {
		return value
	}
	return HarmonyCategoryMarketing
}

func buildHarmonyClickData(payload PushPayload) map[string]string {
	data := map[string]string{}
	if payload.MessageID > 0 {
		data["messageId"] = fmt.Sprintf("%d", payload.MessageID)
	}
	// activityId=0 时不下发，避免客户端误跳转到无效详情
	if payload.ActivityID > 0 {
		data["activityId"] = fmt.Sprintf("%d", payload.ActivityID)
	}
	if messageType := strings.TrimSpace(payload.MessageType); messageType != "" {
		data["type"] = messageType
	}
	if len(data) == 0 {
		return nil
	}
	return data
}

func buildHarmonyNotifyID(messageID int64) *int {
	if messageID <= 0 || messageID > harmonyMaxNotifyID {
		return nil
	}
	value := int(messageID)
	return &value
}
