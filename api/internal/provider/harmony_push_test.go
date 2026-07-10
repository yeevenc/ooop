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
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ooop-admin-api/internal/config"
)

func TestSignHarmonyJWTUsesServiceAccountClaims(t *testing.T) {
	account, privateKey := newHarmonyTestAccount(t)
	token, err := signHarmonyJWT(account, 100, 200)
	if err != nil {
		t.Fatalf("signHarmonyJWT() error = %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("JWT parts = %d, want 3", len(parts))
	}
	header := decodeJWTPart(t, parts[0])
	claims := decodeJWTPart(t, parts[1])
	if header["alg"] != "PS256" || header["kid"] != account.KeyID {
		t.Fatalf("JWT header = %+v", header)
	}
	if claims["iss"] != account.SubAccount || claims["aud"] != account.TokenURI {
		t.Fatalf("JWT claims = %+v", claims)
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatalf("decode signature error = %v", err)
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPSS(&privateKey.PublicKey, crypto.SHA256, digest[:], signature, &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthEqualsHash,
		Hash:       crypto.SHA256,
	}); err != nil {
		t.Fatalf("VerifyPSS() error = %v", err)
	}
}

func TestHarmonyPusherSendsNotification(t *testing.T) {
	account, _ := newHarmonyTestAccount(t)
	accountFile := filepath.Join(t.TempDir(), "service_account.json")
	accountBody, err := json.Marshal(account)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(accountFile, accountBody, 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/project-test/messages:send" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Errorf("Authorization = %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("push-type") != harmonyPushTypeNotification {
			t.Errorf("push-type = %s", r.Header.Get("push-type"))
		}
		body, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			t.Errorf("io.ReadAll() error = %v", readErr)
		}
		var request harmonyPushRequest
		if err := json.Unmarshal(body, &request); err != nil {
			t.Errorf("json.Unmarshal() error = %v", err)
		}
		if request.Payload.Notification.ClickAction.Data["messageId"] != "88" {
			t.Errorf("messageId = %s", request.Payload.Notification.ClickAction.Data["messageId"])
		}
		if !bytes.Contains(body, []byte(`"foregroundShow":false`)) {
			t.Errorf("request does not disable foreground display: %s", string(body))
		}
		if len(request.Target.Token) != 1 || request.Target.Token[0] != "harmony-token" {
			t.Errorf("target = %+v", request.Target.Token)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"80000000","msg":"success","requestId":"request-1"}`))
	}))
	defer server.Close()

	pusher := NewHarmonyPusher(config.HarmonyPushConfig{
		ServiceAccountFile: accountFile,
		PushURL:            server.URL,
	})
	result, err := pusher.Push(context.Background(), PushPayload{
		HarmonyPushToken: "harmony-token",
		Title:            "活动通知",
		Alert:            "活动已通过审核",
		MessageType:      "activity_review",
		Category:         HarmonyCategorySystemReminder,
		MessageID:        88,
		ActivityID:       99,
	})
	if err != nil {
		t.Fatalf("Push() error = %v", err)
	}
	if !result.Triggered || !result.Success {
		t.Fatalf("result = %+v", result)
	}
}

func newHarmonyTestAccount(t *testing.T) (harmonyServiceAccount, *rsa.PrivateKey) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() error = %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("x509.MarshalPKCS8PrivateKey() error = %v", err)
	}
	return harmonyServiceAccount{
		ProjectID:  "project-test",
		KeyID:      "key-test",
		PrivateKey: string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})),
		SubAccount: "sub-account-test",
		TokenURI:   "https://oauth-login.cloud.huawei.com/oauth2/v3/token",
	}, privateKey
}

func decodeJWTPart(t *testing.T, value string) map[string]interface{} {
	t.Helper()
	body, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		t.Fatalf("DecodeString() error = %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	return result
}
