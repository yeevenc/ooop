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

const harmonyPushTypeNotification = "0"

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

type harmonyNotification struct {
	Category       string             `json:"category"`
	Title          string             `json:"title"`
	Body           string             `json:"body"`
	ClickAction    harmonyClickAction `json:"clickAction"`
	ForegroundShow bool               `json:"foregroundShow"`
	Badge          harmonyBadge       `json:"badge"`
}

type harmonyClickAction struct {
	ActionType int               `json:"actionType"`
	Data       map[string]string `json:"data"`
}

type harmonyBadge struct {
	AddNum int `json:"addNum"`
}

type harmonyTarget struct {
	Token []string `json:"token"`
}

type harmonyPushOptions struct {
	TTL         int  `json:"ttl"`
	TestMessage bool `json:"testMessage"`
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
	if strings.TrimSpace(p.cfg.ServiceAccountFile) == "" {
		err := errors.New("鸿蒙 Service Account 文件未配置")
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
	requestBody := harmonyPushRequest{
		Payload: harmonyPayload{
			Notification: harmonyNotification{
				Category:       payload.Category,
				Title:          payload.Title,
				Body:           payload.Alert,
				ForegroundShow: false,
				ClickAction: harmonyClickAction{
					ActionType: 0,
					Data: map[string]string{
						"messageId":  fmt.Sprintf("%d", payload.MessageID),
						"activityId": fmt.Sprintf("%d", payload.ActivityID),
						"type":       payload.MessageType,
					},
				},
				Badge: harmonyBadge{AddNum: 1},
			},
		},
		Target: harmonyTarget{Token: []string{token}},
		PushOptions: harmonyPushOptions{
			TTL:         defaultPushTimeToLive,
			TestMessage: p.cfg.TestMessage,
		},
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		result.Message = err.Error()
		return result, err
	}

	pushURL := fmt.Sprintf("%s/v3/%s/messages:send", strings.TrimRight(p.cfg.PushURL, "/"), p.account.ProjectID)
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
		body, err := os.ReadFile(p.cfg.ServiceAccountFile)
		if err != nil {
			p.accountErr = fmt.Errorf("读取鸿蒙 Service Account 失败: %w", err)
			return
		}
		if err := json.Unmarshal(body, &p.account); err != nil {
			p.accountErr = fmt.Errorf("解析鸿蒙 Service Account 失败: %w", err)
			return
		}
		if strings.TrimSpace(p.account.ProjectID) == "" || strings.TrimSpace(p.account.KeyID) == "" ||
			strings.TrimSpace(p.account.PrivateKey) == "" || strings.TrimSpace(p.account.SubAccount) == "" ||
			strings.TrimSpace(p.account.TokenURI) == "" {
			p.accountErr = errors.New("鸿蒙 Service Account 配置不完整")
		}
	})
	return p.account, p.accountErr
}

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
	return code == "0" || code == "80000000"
}
