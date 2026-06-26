package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"ooop-admin-api/internal/config"
)

type SMSScene string

const (
	SMSSceneLogin           SMSScene = "login"
	SMSSceneChangePhone     SMSScene = "change_phone"
	SMSSceneResetPassword   SMSScene = "reset_password"
	SMSSceneBindNewPhone    SMSScene = "bind_new_phone"
	SMSSceneVerifyBindPhone SMSScene = "verify_bind_phone"
)

type SMSSender interface {
	SendCode(ctx context.Context, phone string, scene SMSScene) error
	CheckCode(ctx context.Context, phone string, scene SMSScene, code string) (bool, error)
}

type AliyunSMSSender struct {
	client *AliyunRPCClient
	cfg    config.AliyunSMSConfig
}

func NewAliyunSMSSender(client *AliyunRPCClient, cfg config.AliyunSMSConfig) *AliyunSMSSender {
	return &AliyunSMSSender{client: client, cfg: cfg}
}

func (s *AliyunSMSSender) SendCode(ctx context.Context, phone string, scene SMSScene) error {
	if strings.TrimSpace(s.cfg.SignName) == "" {
		return errors.New("阿里云短信签名未配置")
	}
	templateCode, err := s.templateCode(scene)
	if err != nil {
		return err
	}

	templateParam, err := json.Marshal(map[string]string{
		"code": "##code##",
		"min":  strconv.Itoa(validMinutes(s.cfg.ValidSeconds)),
	})
	if err != nil {
		return err
	}

	params := map[string]string{
		"Action":          "SendSmsVerifyCode",
		"Version":         "2017-05-25",
		"CountryCode":     "86",
		"PhoneNumber":     phone,
		"SignName":        s.cfg.SignName,
		"TemplateCode":    templateCode,
		"TemplateParam":   string(templateParam),
		"CodeLength":      strconv.Itoa(s.cfg.CodeLength),
		"ValidTime":       strconv.Itoa(s.cfg.ValidSeconds),
		"DuplicatePolicy": strconv.Itoa(s.cfg.DuplicatePolicy),
		"Interval":        strconv.Itoa(s.cfg.IntervalSeconds),
		"CodeType":        "1",
	}
	if s.cfg.SchemeName != "" {
		params["SchemeName"] = s.cfg.SchemeName
	}

	result, err := s.client.Call(ctx, s.cfg.Endpoint, params)
	if err != nil {
		return err
	}
	if ok, _ := result["Success"].(bool); !ok {
		return errors.New("阿里云短信验证码发送失败")
	}
	return nil
}

func (s *AliyunSMSSender) CheckCode(ctx context.Context, phone string, scene SMSScene, code string) (bool, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return false, nil
	}
	if _, err := s.templateCode(scene); err != nil {
		return false, err
	}

	params := map[string]string{
		"Action":         "CheckSmsVerifyCode",
		"Version":        "2017-05-25",
		"CountryCode":    "86",
		"PhoneNumber":    phone,
		"VerifyCode":     code,
		"CaseAuthPolicy": "1",
	}
	if s.cfg.SchemeName != "" {
		params["SchemeName"] = s.cfg.SchemeName
	}

	result, err := s.client.Call(ctx, s.cfg.Endpoint, params)
	if err != nil {
		return false, err
	}
	model, ok := result["Model"].(map[string]interface{})
	if !ok {
		return false, errors.New("阿里云短信验证码核验结果为空")
	}
	return strings.EqualFold(findString(model, "VerifyResult"), "PASS"), nil
}

func (s *AliyunSMSSender) templateCode(scene SMSScene) (string, error) {
	switch scene {
	case SMSSceneLogin:
		return requiredTemplateCode(s.cfg.LoginTemplateCode, scene)
	case SMSSceneChangePhone:
		return requiredTemplateCode(s.cfg.ChangePhoneTemplateCode, scene)
	case SMSSceneResetPassword:
		return requiredTemplateCode(s.cfg.ResetPasswordTemplateCode, scene)
	case SMSSceneBindNewPhone:
		return requiredTemplateCode(s.cfg.BindNewPhoneTemplateCode, scene)
	case SMSSceneVerifyBindPhone:
		return requiredTemplateCode(s.cfg.VerifyBindPhoneTemplateCode, scene)
	default:
		return "", fmt.Errorf("短信验证码场景不支持: %s", scene)
	}
}

func requiredTemplateCode(templateCode string, scene SMSScene) (string, error) {
	templateCode = strings.TrimSpace(templateCode)
	if templateCode == "" {
		return "", fmt.Errorf("短信验证码模板未配置: %s", scene)
	}
	return templateCode, nil
}

func validMinutes(seconds int) int {
	if seconds <= 0 {
		return 5
	}
	minutes := seconds / 60
	if minutes <= 0 {
		return 1
	}
	return minutes
}
