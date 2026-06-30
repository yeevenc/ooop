package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"ooop-admin-api/internal/config"
)

func TestAliyunIDCardVerifierPassesOnlyWhenResultIsZero(t *testing.T) {
	verifier := newTestAliyunIDCardVerifier(t, `{
		"msg": "成功",
		"success": true,
		"code": 200,
		"data": {
			"result": 0,
			"sex": "男",
			"desc": "一致"
		}
	}`)

	result, err := verifier.Verify(context.Background(), "张三", "11010119900307451X")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !result.Passed {
		t.Fatalf("Passed = false, want true")
	}
	if result.Gender != "男" {
		t.Fatalf("Gender = %q, want 男", result.Gender)
	}
}

func TestAliyunIDCardVerifierRejectsMismatchResult(t *testing.T) {
	verifier := newTestAliyunIDCardVerifier(t, `{
		"msg": "成功",
		"success": true,
		"code": 200,
		"data": {
			"result": 1,
			"desc": "不一致"
		}
	}`)

	result, err := verifier.Verify(context.Background(), "张三", "11010119900307451X")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if result.Passed {
		t.Fatalf("Passed = true, want false")
	}
	if result.Message != "不一致" {
		t.Fatalf("Message = %q, want 不一致", result.Message)
	}
}

func TestAliyunIDCardVerifierRejectsInvalidIDCardResponse(t *testing.T) {
	verifier := newTestAliyunIDCardVerifier(t, `{
		"msg": "身份证号不合法",
		"success": true,
		"code": 400,
		"data": {
			"orderNo": "202606301224326819738"
		}
	}`)

	result, err := verifier.Verify(context.Background(), "张三", "bad-id-card")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if result.Passed {
		t.Fatalf("Passed = true, want false")
	}
	if result.Message != "身份证号不合法" {
		t.Fatalf("Message = %q, want 身份证号不合法", result.Message)
	}
}

func newTestAliyunIDCardVerifier(t *testing.T, response string) *AliyunIDCardVerifier {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.Header.Get("Authorization") != "APPCODE test-app-code" {
			t.Fatalf("Authorization = %q, want APPCODE test-app-code", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(server.Close)

	return NewAliyunIDCardVerifier(config.AliyunIDCardConfig{
		Endpoint: server.URL,
		AppCode:  "test-app-code",
	})
}
