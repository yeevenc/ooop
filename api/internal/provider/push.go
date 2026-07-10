package provider

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"ooop-admin-api/internal/logger"
)

const (
	PushChannelJiguang = "jiguang"
	PushChannelHarmony = "harmony"

	// 华为 Push Kit 官方 category（大写），见场景化消息分类标准。
	// 未开通对应自分类权益时，云端会回落为资讯营销（MARKETING）频控。
	HarmonyCategoryMarketing    = "MARKETING"
	HarmonyCategoryWork         = "WORK"         // 审核进度、待办、系统业务提醒
	HarmonyCategorySubscription = "SUBSCRIPTION" // 用户相关互动/订阅提醒
	HarmonyCategoryAccount      = "ACCOUNT"      // 账号动态

)

type PushPayload struct {
	Alias            string
	HarmonyPushToken string
	Title            string
	Alert            string
	MessageType      string
	Category         string
	MessageID        int64
	ActivityID       int64
}

type PushChannelResult struct {
	Channel   string `json:"channel"`
	Triggered bool   `json:"triggered"`
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Response  string `json:"response,omitempty"`
}

type PushResult struct {
	Triggered bool                `json:"triggered"`
	Success   bool                `json:"success"`
	Alias     string              `json:"alias"`
	Message   string              `json:"message"`
	Channels  []PushChannelResult `json:"channels"`
}

type ChannelPusher interface {
	Push(ctx context.Context, payload PushPayload) (PushChannelResult, error)
}

type DualChannelPusher struct {
	jiguang ChannelPusher
	harmony ChannelPusher
}

func NewDualChannelPusher(jiguang ChannelPusher, harmony ChannelPusher) *DualChannelPusher {
	return &DualChannelPusher{
		jiguang: jiguang,
		harmony: harmony,
	}
}

func (p *DualChannelPusher) Push(ctx context.Context, payload PushPayload) (PushResult, error) {
	channels := []PushChannelResult{
		{Channel: PushChannelJiguang, Message: "极光推送未初始化"},
		{Channel: PushChannelHarmony, Message: "鸿蒙推送未初始化"},
	}
	errs := make([]error, len(channels))
	tasks := []ChannelPusher{p.jiguang, p.harmony}

	var wg sync.WaitGroup
	for index, sender := range tasks {
		if sender == nil {
			continue
		}
		wg.Add(1)
		go func(i int, channelPusher ChannelPusher) {
			defer wg.Done()
			channels[i], errs[i] = channelPusher.Push(ctx, payload)
		}(index, sender)
	}
	wg.Wait()

	result := PushResult{
		Alias:    payload.Alias,
		Channels: channels,
	}
	var failures []error
	for index, channel := range channels {
		result.Triggered = result.Triggered || channel.Triggered
		result.Success = result.Success || channel.Success
		if errs[index] != nil {
			failures = append(failures, fmt.Errorf("%s: %w", channel.Channel, errs[index]))
		}
	}

	switch {
	case channels[0].Success && channels[1].Success:
		result.Message = "双通道发送成功"
	case result.Success:
		result.Message = "部分通道发送成功"
	case result.Triggered:
		result.Message = "推送发送失败"
	default:
		result.Message = "没有可用的推送通道"
	}

	if result.Success {
		for _, err := range failures {
			logger.Warnf("双通道部分发送失败: %v", err)
		}
		return result, nil
	}
	if len(failures) > 0 {
		return result, errors.Join(failures...)
	}
	return result, nil
}
