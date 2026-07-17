package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ooop-admin-api/internal/config"
)

func TestJiguangPusherSendsCustomMessageOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Decode() error = %v", err)
		}
		if _, exists := body["notification"]; exists {
			t.Error("request should not contain notification")
		}
		audience := body["audience"].(map[string]interface{})
		registrationIDs := audience["registration_id"].([]interface{})
		if len(registrationIDs) != 1 || registrationIDs[0] != "registration-3000" {
			t.Errorf("audience = %+v", audience)
		}
		if _, exists := audience["alias"]; exists {
			t.Errorf("registration_id 存在时不应再使用 alias: %+v", audience)
		}
		message, ok := body["message"].(map[string]interface{})
		if !ok || message["msg_content"] != "活动已通过审核" {
			t.Errorf("message = %+v", message)
		}
		options := body["options"].(map[string]interface{})
		thirdParty := options["third_party_channel"].(map[string]interface{})
		hmos := thirdParty["hmos"].(map[string]interface{})
		if hmos["distribution"] != "jpush" {
			t.Errorf("distribution = %v", hmos["distribution"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"msg_id":"1"}`))
	}))
	defer server.Close()

	pusher := NewJiguangPusher(config.JiguangConfig{
		AppKey:       "app-key",
		MasterSecret: "master-secret",
		PushURL:      server.URL,
	})
	result, err := pusher.Push(context.Background(), PushPayload{
		Alias:          "3000",
		RegistrationID: "registration-3000",
		Title:          "活动通知",
		Alert:          "活动已通过审核",
		MessageType:    "activity_review",
		MessageID:      88,
		ActivityID:     99,
	})
	if err != nil {
		t.Fatalf("Push() error = %v", err)
	}
	if !result.Success {
		t.Fatalf("result = %+v", result)
	}
}

func TestJiguangPusherUsesChatContentAndRoutingExtras(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Decode() error = %v", err)
		}
		message := body["message"].(map[string]interface{})
		if message["msg_content"] != `{"messageId":"88","content":"你好"}` {
			t.Errorf("msg_content = %v", message["msg_content"])
		}
		extras := message["extras"].(map[string]interface{})
		if extras["messageId"] != "88" || extras["conversationId"] != "66" || extras["senderId"] != "3000" {
			t.Errorf("extras = %+v", extras)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"msg_id":"2"}`))
	}))
	defer server.Close()

	pusher := NewJiguangPusher(config.JiguangConfig{
		AppKey:       "app-key",
		MasterSecret: "master-secret",
		PushURL:      server.URL,
	})
	result, err := pusher.Push(context.Background(), PushPayload{
		Alias:         "3001",
		Title:         "新会话",
		Alert:         "您有新会话",
		CustomContent: `{"messageId":"88","content":"你好"}`,
		MessageType:   "chat_message",
		MessageID:     88,
		Extras: map[string]string{
			"conversationId": "66",
			"senderId":       "3000",
		},
	})
	if err != nil {
		t.Fatalf("Push() error = %v", err)
	}
	if !result.Success {
		t.Fatalf("result = %+v", result)
	}
}
