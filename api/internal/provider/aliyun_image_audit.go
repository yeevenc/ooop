package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"ooop-admin-api/internal/config"
)

const (
	ImageAuditSuggestionPass   = "pass"
	ImageAuditSuggestionReview = "review"
	ImageAuditSuggestionBlock  = "block"

	aliyunImageAuditVersion  = "2019-12-30"
	aliyunImageAuditMaxTasks = 5
)

type ImageAuditHit struct {
	ImageIndex int     `json:"image_index"`
	Scene      string  `json:"scene"`
	Label      string  `json:"label"`
	Suggestion string  `json:"suggestion"`
	Rate       float64 `json:"rate"`
}

type ImageAuditResult struct {
	Suggestion string          `json:"suggestion"`
	Hits       []ImageAuditHit `json:"hits"`
	RawJSON    string          `json:"-"`
}

func (r ImageAuditResult) RejectReason() string {
	if len(r.Hits) == 0 {
		return "活动图片命中平台内容安全规则"
	}

	details := make([]string, 0, len(r.Hits))
	for _, hit := range r.Hits {
		if hit.Suggestion != ImageAuditSuggestionBlock {
			continue
		}
		detail := fmt.Sprintf(
			"第%d张图片命中%s（%s，置信度%.2f%%）",
			hit.ImageIndex,
			imageAuditSceneText(hit.Scene),
			imageAuditLabelText(hit.Label),
			hit.Rate,
		)
		details = append(details, detail)
		if len(details) >= 3 {
			break
		}
	}
	if len(details) == 0 {
		return "活动图片命中平台内容安全规则"
	}
	return "图片内容审核未通过：" + strings.Join(details, "；")
}

type ImageModerator interface {
	Audit(ctx context.Context, imageURLs []string) (ImageAuditResult, error)
}

type AliyunImageModerator struct {
	client *AliyunRPCClient
	cfg    config.AliyunImageAuditConfig
}

func NewAliyunImageModerator(client *AliyunRPCClient, cfg config.AliyunImageAuditConfig) *AliyunImageModerator {
	return &AliyunImageModerator{client: client, cfg: cfg}
}

func (m *AliyunImageModerator) Audit(ctx context.Context, imageURLs []string) (ImageAuditResult, error) {
	if len(imageURLs) == 0 {
		return ImageAuditResult{
			Suggestion: ImageAuditSuggestionPass,
			Hits:       []ImageAuditHit{},
			RawJSON:    `{"suggestion":"pass","hits":[]}`,
		}, nil
	}
	if m.client == nil {
		return ImageAuditResult{}, errors.New("阿里云图片审核客户端未初始化")
	}
	if strings.TrimSpace(m.cfg.Endpoint) == "" {
		return ImageAuditResult{}, errors.New("阿里云图片审核地址未配置")
	}

	combined := ImageAuditResult{
		Suggestion: ImageAuditSuggestionPass,
		Hits:       make([]ImageAuditHit, 0),
	}
	rawResponses := make([]map[string]interface{}, 0)
	for start := 0; start < len(imageURLs); start += aliyunImageAuditMaxTasks {
		end := start + aliyunImageAuditMaxTasks
		if end > len(imageURLs) {
			end = len(imageURLs)
		}
		result, raw, err := m.auditBatch(ctx, imageURLs[start:end], start)
		if err != nil {
			return ImageAuditResult{}, err
		}
		combined.Suggestion = strongerImageAuditSuggestion(combined.Suggestion, result.Suggestion)
		combined.Hits = append(combined.Hits, result.Hits...)
		rawResponses = append(rawResponses, raw)
	}

	rawJSON, err := json.Marshal(rawResponses)
	if err != nil {
		return ImageAuditResult{}, err
	}
	combined.RawJSON = string(rawJSON)
	return combined, nil
}

func (m *AliyunImageModerator) auditBatch(
	ctx context.Context,
	imageURLs []string,
	offset int,
) (ImageAuditResult, map[string]interface{}, error) {
	params, err := buildAliyunImageAuditParams(
		imageURLs,
		offset,
		m.cfg.RegionID,
		m.cfg.Scenes,
	)
	if err != nil {
		return ImageAuditResult{}, nil, err
	}

	response, err := m.client.Call(ctx, m.cfg.Endpoint, params)
	if err != nil {
		return ImageAuditResult{}, nil, err
	}
	result, err := parseAliyunImageAuditResponse(response, offset)
	return result, response, err
}

func buildAliyunImageAuditParams(
	imageURLs []string,
	offset int,
	regionID string,
	scenes []string,
) (map[string]string, error) {
	params := map[string]string{
		"Action":   "ScanImage",
		"Version":  aliyunImageAuditVersion,
		"RegionId": strings.TrimSpace(regionID),
	}
	for index, imageURL := range imageURLs {
		imageURL = strings.TrimSpace(imageURL)
		if imageURL == "" {
			return nil, fmt.Errorf("第%d张活动图片地址为空", offset+index+1)
		}
		position := index + 1
		params[fmt.Sprintf("Task.%d.ImageURL", position)] = imageURL
		params[fmt.Sprintf("Task.%d.DataId", position)] = fmt.Sprintf(
			"image-%d",
			offset+position,
		)
		params[fmt.Sprintf("Task.%d.MaxFrames", position)] = "1"
	}
	for index, scene := range normalizeImageAuditScenes(scenes) {
		params[fmt.Sprintf("Scene.%d", index+1)] = scene
	}
	return params, nil
}

func parseAliyunImageAuditResponse(payload map[string]interface{}, offset int) (ImageAuditResult, error) {
	data, ok := payload["Data"].(map[string]interface{})
	if !ok {
		return ImageAuditResult{}, errors.New("阿里云图片审核未返回 Data")
	}
	results, ok := data["Results"].([]interface{})
	if !ok || len(results) == 0 {
		return ImageAuditResult{}, errors.New("阿里云图片审核未返回检测结果")
	}

	result := ImageAuditResult{
		Suggestion: ImageAuditSuggestionPass,
		Hits:       make([]ImageAuditHit, 0),
	}
	for resultIndex, rawResult := range results {
		item, ok := rawResult.(map[string]interface{})
		if !ok {
			return ImageAuditResult{}, errors.New("阿里云图片审核结果格式不正确")
		}
		if rawCode, exists := item["Code"]; exists {
			code, ok := numberValue(rawCode)
			if !ok || int(code) != 200 {
				message, _ := item["Message"].(string)
				return ImageAuditResult{}, fmt.Errorf(
					"第%d张图片审核失败: %s",
					offset+resultIndex+1,
					strings.TrimSpace(message),
				)
			}
		}
		subResults, ok := item["SubResults"].([]interface{})
		if !ok || len(subResults) == 0 {
			return ImageAuditResult{}, fmt.Errorf("第%d张图片未返回场景结果", offset+resultIndex+1)
		}
		for _, rawSubResult := range subResults {
			subResult, ok := rawSubResult.(map[string]interface{})
			if !ok {
				return ImageAuditResult{}, errors.New("阿里云图片审核场景结果格式不正确")
			}
			suggestion, _ := subResult["Suggestion"].(string)
			suggestion = strings.ToLower(strings.TrimSpace(suggestion))
			if !validImageAuditSuggestion(suggestion) {
				return ImageAuditResult{}, fmt.Errorf("阿里云图片审核返回未知建议: %s", suggestion)
			}
			result.Suggestion = strongerImageAuditSuggestion(result.Suggestion, suggestion)
			if suggestion == ImageAuditSuggestionPass {
				continue
			}
			rate, _ := numberValue(subResult["Rate"])
			scene, _ := subResult["Scene"].(string)
			label, _ := subResult["Label"].(string)
			result.Hits = append(result.Hits, ImageAuditHit{
				ImageIndex: offset + resultIndex + 1,
				Scene:      strings.TrimSpace(scene),
				Label:      strings.TrimSpace(label),
				Suggestion: suggestion,
				Rate:       rate,
			})
		}
	}
	return result, nil
}

func normalizeImageAuditScenes(values []string) []string {
	allowed := map[string]struct{}{
		"porn":      {},
		"terrorism": {},
		"ad":        {},
		"live":      {},
		"logo":      {},
	}
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		scene := strings.ToLower(strings.TrimSpace(value))
		if _, ok := allowed[scene]; !ok {
			continue
		}
		if _, ok := seen[scene]; ok {
			continue
		}
		seen[scene] = struct{}{}
		result = append(result, scene)
	}
	if len(result) == 0 {
		return []string{"porn", "terrorism", "ad", "live"}
	}
	return result
}

func validImageAuditSuggestion(value string) bool {
	return value == ImageAuditSuggestionPass ||
		value == ImageAuditSuggestionReview ||
		value == ImageAuditSuggestionBlock
}

func strongerImageAuditSuggestion(current string, next string) string {
	rank := map[string]int{
		ImageAuditSuggestionPass:   1,
		ImageAuditSuggestionReview: 2,
		ImageAuditSuggestionBlock:  3,
	}
	if rank[next] > rank[current] {
		return next
	}
	return current
}

func numberValue(value interface{}) (float64, bool) {
	switch number := value.(type) {
	case float64:
		return number, true
	case float32:
		return float64(number), true
	case int:
		return float64(number), true
	case int64:
		return float64(number), true
	case json.Number:
		result, err := number.Float64()
		return result, err == nil
	case string:
		result, err := strconv.ParseFloat(strings.TrimSpace(number), 64)
		return result, err == nil
	default:
		return 0, false
	}
}

func imageAuditSceneText(scene string) string {
	switch scene {
	case "porn":
		return "色情低俗内容"
	case "terrorism":
		return "暴恐或敏感内容"
	case "ad":
		return "广告或图片文字风险"
	case "live":
		return "不良场景"
	case "logo":
		return "受管控标识"
	default:
		return "内容安全风险"
	}
}

func imageAuditLabelText(label string) string {
	labels := map[string]string{
		"sexy":        "性感低俗",
		"porn":        "色情内容",
		"bloody":      "血腥内容",
		"explosion":   "爆炸场景",
		"weapon":      "武器",
		"politics":    "敏感内容",
		"violence":    "暴力打斗",
		"drug":        "涉毒",
		"gamble":      "赌博",
		"abuse":       "辱骂内容",
		"terrorism":   "涉恐内容",
		"contraband":  "违禁内容",
		"spam":        "垃圾内容",
		"npx":         "违规广告",
		"qrcode":      "二维码",
		"programCode": "小程序码",
		"ad":          "广告内容",
		"meaningless": "无意义图片",
		"PIP":         "画中画",
		"smoking":     "吸烟场景",
		"drivelive":   "车内直播",
		"TV":          "管控台标",
		"trademark":   "商标",
	}
	if text := labels[label]; text != "" {
		return text
	}
	if strings.TrimSpace(label) == "" {
		return "风险内容"
	}
	return label
}
