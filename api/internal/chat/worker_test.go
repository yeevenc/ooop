package chat

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"ooop-admin-api/internal/provider"
	"ooop-admin-api/internal/user"
)

type workerTestPusher struct {
	pushCalls        int
	pushChannelCalls int
}

func (p *workerTestPusher) Push(context.Context, provider.PushPayload) (provider.PushResult, error) {
	p.pushCalls++
	return provider.PushResult{Triggered: true, Success: true, Message: "双通道发送成功"}, nil
}

func (p *workerTestPusher) PushChannel(context.Context, string, provider.PushPayload) (provider.PushChannelResult, error) {
	p.pushChannelCalls++
	return provider.PushChannelResult{Triggered: true, Success: true}, nil
}

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
	if payload.Extras["type"] != PushMessageType || payload.Extras["messageId"] != "88" ||
		payload.Extras["conversationId"] != "66" || payload.Extras["senderId"] != "3000" ||
		payload.Extras["messageType"] != MessageTypeText {
		t.Fatalf("extras = %+v", payload.Extras)
	}

	var realtime realtimeMessage
	if err := json.Unmarshal([]byte(payload.CustomContent), &realtime); err != nil {
		t.Fatalf("custom content is not JSON: %v", err)
	}
	if realtime.MessageID != "88" || realtime.Content != "你好 😊" || realtime.Type != PushMessageType ||
		realtime.MessageType != MessageTypeText {
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

func TestWorkerUsesSharedDualChannelPushForNewTasks(t *testing.T) {
	pusher := &workerTestPusher{}
	worker := NewWorker(nil, nil, pusher, WorkerOptions{})

	result, err := worker.sendPush(context.Background(), PushTaskChannelDual, provider.PushPayload{})
	if err != nil {
		t.Fatalf("sendPush() error = %v", err)
	}
	if !result.Success || pusher.pushCalls != 1 || pusher.pushChannelCalls != 0 {
		t.Fatalf("result=%+v, pushCalls=%d, pushChannelCalls=%d", result, pusher.pushCalls, pusher.pushChannelCalls)
	}
}

func TestWorkerKeepsLegacySingleChannelTasksCompatible(t *testing.T) {
	pusher := &workerTestPusher{}
	worker := NewWorker(nil, nil, pusher, WorkerOptions{})

	result, err := worker.sendPush(context.Background(), provider.PushChannelJiguang, provider.PushPayload{})
	if err != nil {
		t.Fatalf("sendPush() error = %v", err)
	}
	if !result.Success || pusher.pushCalls != 0 || pusher.pushChannelCalls != 1 {
		t.Fatalf("result=%+v, pushCalls=%d, pushChannelCalls=%d", result, pusher.pushCalls, pusher.pushChannelCalls)
	}
}
