package provider

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ooop-admin-api/internal/config"
)

type JiguangMobileVerifier struct {
	cfg        config.JiguangConfig
	httpClient *http.Client
}

func NewJiguangMobileVerifier(cfg config.JiguangConfig) *JiguangMobileVerifier {
	return &JiguangMobileVerifier{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (v *JiguangMobileVerifier) Verify(ctx context.Context, loginToken string) (MobileVerifyResult, error) {
	loginToken = strings.TrimSpace(loginToken)
	if loginToken == "" {
		return MobileVerifyResult{}, errors.New("一键登录凭证不能为空")
	}
	if strings.TrimSpace(v.cfg.VerifyURL) == "" {
		return MobileVerifyResult{}, errors.New("极光号码认证校验地址未配置")
	}
	if strings.TrimSpace(v.cfg.PrivateKey) == "" {
		return MobileVerifyResult{}, errors.New("极光号码认证私钥未配置")
	}

	body, err := json.Marshal(map[string]string{"loginToken": loginToken})
	if err != nil {
		return MobileVerifyResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.cfg.VerifyURL, bytes.NewReader(body))
	if err != nil {
		return MobileVerifyResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if v.cfg.AppKey != "" && v.cfg.MasterSecret != "" {
		req.SetBasicAuth(v.cfg.AppKey, v.cfg.MasterSecret)
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return MobileVerifyResult{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return MobileVerifyResult{}, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return MobileVerifyResult{}, fmt.Errorf("极光号码认证请求失败: %s", resp.Status)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return MobileVerifyResult{}, err
	}
	if err := jiguangBusinessError(result); err != nil {
		return MobileVerifyResult{}, err
	}

	encryptedPhone := findString(result, "phone")
	if encryptedPhone == "" {
		encryptedPhone = findString(result, "mobile")
	}
	if encryptedPhone == "" {
		return MobileVerifyResult{}, errors.New("极光号码认证未返回手机号")
	}
	phone, err := decryptJiguangPhone(encryptedPhone, v.cfg.PrivateKey)
	if err != nil {
		return MobileVerifyResult{}, fmt.Errorf("极光手机号解密失败: %w", err)
	}
	return MobileVerifyResult{Phone: phone}, nil
}

func jiguangBusinessError(result map[string]interface{}) error {
	for _, key := range []string{"code", "errorCode"} {
		raw, ok := result[key]
		if !ok {
			continue
		}
		code := fmt.Sprint(raw)
		if code == "" || code == "0" || code == "200" || code == "8000" {
			return nil
		}
		message := findString(result, "message")
		if message == "" {
			message = findString(result, "msg")
		}
		if message == "" {
			message = findString(result, "content")
		}
		if message == "" {
			message = "极光号码认证返回失败"
		}
		return fmt.Errorf("%s: %s", message, code)
	}
	return nil
}

func decryptJiguangPhone(encryptedPhone string, privateKey string) (string, error) {
	cipherText, err := base64.StdEncoding.DecodeString(encryptedPhone)
	if err != nil {
		return "", err
	}

	block, _ := pem.Decode([]byte(privateKey))
	if block == nil {
		return "", errors.New("私钥格式不正确")
	}

	parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		rsaKey, pkcs1Err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if pkcs1Err != nil {
			return "", err
		}
		parsedKey = rsaKey
	}

	rsaKey, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return "", errors.New("私钥类型不正确")
	}

	plainText, err := rsa.DecryptPKCS1v15(rand.Reader, rsaKey, cipherText)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(plainText)), nil
}
