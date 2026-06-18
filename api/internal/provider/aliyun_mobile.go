package provider

import (
	"context"
	"errors"
	"strings"

	"ooop-admin-api/internal/config"
)

type MobileVerifier interface {
	Verify(ctx context.Context, accessToken string) (string, error)
}

type AliyunMobileVerifier struct {
	client *AliyunRPCClient
	cfg    config.AliyunMobileConfig
}

func NewAliyunMobileVerifier(client *AliyunRPCClient, cfg config.AliyunMobileConfig) *AliyunMobileVerifier {
	return &AliyunMobileVerifier{client: client, cfg: cfg}
}

func (v *AliyunMobileVerifier) Verify(ctx context.Context, accessToken string) (string, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return "", errors.New("一键登录凭证不能为空")
	}

	result, err := v.client.Call(ctx, v.cfg.Endpoint, map[string]string{
		"Action":      "GetMobile",
		"Version":     "2017-05-25",
		"AccessToken": accessToken,
	})
	if err != nil {
		return "", err
	}

	phone := findString(result, "Mobile")
	if phone == "" {
		return "", errors.New("阿里云未返回手机号")
	}
	return phone, nil
}

func findString(value interface{}, key string) string {
	switch current := value.(type) {
	case map[string]interface{}:
		if raw, ok := current[key]; ok {
			if text, ok := raw.(string); ok {
				return text
			}
		}
		for _, item := range current {
			if text := findString(item, key); text != "" {
				return text
			}
		}
	case []interface{}:
		for _, item := range current {
			if text := findString(item, key); text != "" {
				return text
			}
		}
	}
	return ""
}
