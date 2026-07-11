package provider

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type AliyunRPCClient struct {
	accessKeyID     string
	accessKeySecret string
	httpClient      *http.Client
}

func NewAliyunRPCClient(accessKeyID string, accessKeySecret string) *AliyunRPCClient {
	return &AliyunRPCClient{
		accessKeyID:     accessKeyID,
		accessKeySecret: accessKeySecret,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *AliyunRPCClient) Call(ctx context.Context, endpoint string, params map[string]string) (map[string]interface{}, error) {
	return c.call(ctx, http.MethodGet, endpoint, params)
}

func (c *AliyunRPCClient) CallPOST(ctx context.Context, endpoint string, params map[string]string) (map[string]interface{}, error) {
	return c.call(ctx, http.MethodPost, endpoint, params)
}

func (c *AliyunRPCClient) call(ctx context.Context, method string, endpoint string, params map[string]string) (map[string]interface{}, error) {
	if c.accessKeyID == "" || c.accessKeySecret == "" {
		return nil, errors.New("阿里云访问密钥未配置")
	}

	values := map[string]string{
		"Format":           "JSON",
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureNonce":   nonce(),
		"SignatureVersion": "1.0",
		"AccessKeyId":      c.accessKeyID,
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
	}
	for key, value := range params {
		values[key] = value
	}
	values["Signature"] = c.signatureWithMethod(method, values)

	requestURL := "https://" + endpoint + "/?" + encodeQuery(values)
	req, err := http.NewRequestWithContext(ctx, method, requestURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("阿里云接口请求失败: %s", resp.Status)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if code, ok := result["Code"].(string); ok && code != "" && code != "OK" {
		message, _ := result["Message"].(string)
		return nil, fmt.Errorf("阿里云接口返回失败: %s %s", code, message)
	}

	return result, nil
}

func (c *AliyunRPCClient) signatureWithMethod(method string, values map[string]string) string {
	canonicalized := encodeQuery(values)
	stringToSign := method + "&%2F&" + percentEncode(canonicalized)
	mac := hmac.New(sha1.New, []byte(c.accessKeySecret+"&"))
	mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func encodeQuery(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, percentEncode(key)+"="+percentEncode(values[key]))
	}
	return strings.Join(parts, "&")
}

func percentEncode(value string) string {
	escaped := url.QueryEscape(value)
	escaped = strings.ReplaceAll(escaped, "+", "%20")
	escaped = strings.ReplaceAll(escaped, "*", "%2A")
	escaped = strings.ReplaceAll(escaped, "%7E", "~")
	return escaped
}

func nonce() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return base64.RawURLEncoding.EncodeToString(bytes)
}
