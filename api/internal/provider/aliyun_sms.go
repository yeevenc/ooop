package provider

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"ooop-admin-api/internal/config"
)

type SMSSender interface {
	SendCode(ctx context.Context, phone string, code string) error
}

type AliyunSMSSender struct {
	client *AliyunRPCClient
	cfg    config.AliyunSMSConfig
}

func NewAliyunSMSSender(client *AliyunRPCClient, cfg config.AliyunSMSConfig) *AliyunSMSSender {
	return &AliyunSMSSender{client: client, cfg: cfg}
}

func (s *AliyunSMSSender) SendCode(ctx context.Context, phone string, code string) error {
	if strings.TrimSpace(s.cfg.SignName) == "" || strings.TrimSpace(s.cfg.TemplateCode) == "" {
		return errors.New("阿里云短信签名或模板未配置")
	}

	templateParam, err := json.Marshal(map[string]string{"code": code})
	if err != nil {
		return err
	}

	_, err = s.client.Call(ctx, s.cfg.Endpoint, map[string]string{
		"Action":        "SendSms",
		"Version":       "2017-05-25",
		"PhoneNumbers":  phone,
		"SignName":      s.cfg.SignName,
		"TemplateCode":  s.cfg.TemplateCode,
		"TemplateParam": string(templateParam),
	})
	return err
}
