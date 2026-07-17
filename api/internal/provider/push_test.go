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

type countingChannelPusher struct {
	count   int
	channel string
}

func (s *countingChannelPusher) Push(context.Context, PushPayload) (PushChannelResult, error) {
	s.count++
	return PushChannelResult{Channel: s.channel, Triggered: true, Success: true}, nil
}

func (s stubChannelPusher) Push(context.Context, PushPayload) (PushChannelResult, error) {
	return s.result, s.err
}

func TestDualChannelPusherCanRetrySingleChannel(t *testing.T) {
	jiguang := &countingChannelPusher{channel: PushChannelJiguang}
	harmony := &countingChannelPusher{channel: PushChannelHarmony}
	pusher := NewDualChannelPusher(jiguang, harmony)

	result, err := pusher.PushChannel(context.Background(), PushChannelHarmony, PushPayload{})
	if err != nil {
		t.Fatalf("PushChannel() error = %v", err)
	}
	if !result.Success || harmony.count != 1 || jiguang.count != 0 {
		t.Fatalf("result=%+v, jiguang=%d, harmony=%d", result, jiguang.count, harmony.count)
	}
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
