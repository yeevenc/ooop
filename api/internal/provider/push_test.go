package provider

import (
	"context"
	"errors"
	"testing"
)

type stubChannelPusher struct {
	result PushChannelResult
	err    error
}

func (s stubChannelPusher) Push(context.Context, PushPayload) (PushChannelResult, error) {
	return s.result, s.err
}

func TestDualChannelPusherAllowsPartialSuccess(t *testing.T) {
	pusher := NewDualChannelPusher(
		stubChannelPusher{result: PushChannelResult{
			Channel:   PushChannelJiguang,
			Triggered: true,
			Success:   true,
			Message:   "极光发送成功",
		}},
		stubChannelPusher{result: PushChannelResult{
			Channel:   PushChannelHarmony,
			Triggered: true,
			Message:   "鸿蒙发送失败",
		}, err: errors.New("鸿蒙发送失败")},
	)

	result, err := pusher.Push(context.Background(), PushPayload{Alias: "3000"})
	if err != nil {
		t.Fatalf("Push() error = %v", err)
	}
	if !result.Success || result.Message != "部分通道发送成功" {
		t.Fatalf("result = %+v", result)
	}
	if len(result.Channels) != 2 {
		t.Fatalf("channels = %d", len(result.Channels))
	}
}
