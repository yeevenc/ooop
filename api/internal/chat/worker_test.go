package chat

import (
	"encoding/json"
	"testing"
	"time"

	"ooop-admin-api/internal/provider"
	"ooop-admin-api/internal/user"
)

func TestBuildPushPayloadSeparatesRealtimeDataAndBackgroundCopy(t *testing.T) {
	worker := NewWorker(nil, nil, nil, WorkerOptions{PushCategory: provider.HarmonyCategoryWork})
	createdAt := time.Date(2026, 7, 17, 12, 0, 0, 0, time.Local)
	payload, err := worker.buildPushPayload(Message{
		ID:              88,
		ConversationID:  66,
		SenderID:        3000,
		RecipientID:     3001,
		ClientMessageID: "0190f25d-6b71-7b68",
		Type:            MessageTypeText,
		Content:         "你好 😊",
		CreatedAt:       createdAt,
	}, user.User{
		ID:               3001,
		RegistrationID:   "registration-id",
		HarmonyPushToken: "push-token",
	})
	if err != nil {
		t.Fatalf("buildPushPayload() error = %v", err)
	}
	if payload.Title != "新会话" || payload.Alert != "您有新会话" {
		t.Fatalf("background copy = %s / %s", payload.Title, payload.Alert)
	}
	if payload.Extras["conversationId"] != "66" || payload.Extras["senderId"] != "3000" {
		t.Fatalf("extras = %+v", payload.Extras)
	}

	var realtime realtimeMessage
	if err := json.Unmarshal([]byte(payload.CustomContent), &realtime); err != nil {
		t.Fatalf("custom content is not JSON: %v", err)
	}
	if realtime.MessageID != "88" || realtime.Content != "你好 😊" || realtime.Type != PushMessageType {
		t.Fatalf("realtime = %+v", realtime)
	}
}

func TestChatModelsUseDedicatedTables(t *testing.T) {
	if (Conversation{}).TableName() != "chat_conversations" {
		t.Fatal("Conversation table name is incorrect")
	}
	if (Message{}).TableName() != "chat_messages" {
		t.Fatal("Message table name is incorrect")
	}
	if (PushTask{}).TableName() != "chat_push_tasks" {
		t.Fatal("PushTask table name is incorrect")
	}
}
