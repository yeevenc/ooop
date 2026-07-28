package provider

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"ooop-admin-api/internal/config"
)

type imageAuditRoundTripFunc func(*http.Request) (*http.Response, error)

func (f imageAuditRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestAliyunImageModeratorUsesPostRequest(t *testing.T) {
	client := NewAliyunRPCClient("access-key-id", "access-key-secret")
	client.httpClient = &http.Client{
		Transport: imageAuditRoundTripFunc(func(request *http.Request) (*http.Response, error) {
			if request.Method != http.MethodPost {
				t.Fatalf("request method = %s, want POST", request.Method)
			}
			if request.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
				t.Fatalf("Content-Type = %s", request.Header.Get("Content-Type"))
			}
			if err := request.ParseForm(); err != nil {
				t.Fatalf("ParseForm() error = %v", err)
			}
			if request.PostForm.Get("Action") != "ScanImage" {
				t.Fatalf("Action = %s", request.PostForm.Get("Action"))
			}
			if request.PostForm.Get("Scene.1") != "porn" {
				t.Fatalf("Scene.1 = %s", request.PostForm.Get("Scene.1"))
			}
			if request.PostForm.Get("Task.1.ImageURL") != "https://source.example.com/activity.jpg" {
				t.Fatalf("Task.1.ImageURL = %s", request.PostForm.Get("Task.1.ImageURL"))
			}
			if request.PostForm.Get("Signature") == "" {
				t.Fatal("Signature is empty")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(`{
					"Data": {
						"Results": [{
							"SubResults": [{
								"Suggestion": "pass",
								"Label": "normal",
								"Scene": "porn",
								"Rate": "99.00"
							}]
						}]
					}
				}`)),
			}, nil
		}),
	}
	moderator := NewAliyunImageModerator(client, config.AliyunImageAuditConfig{
		Endpoint: "imageaudit.cn-shanghai.aliyuncs.com",
		RegionID: "cn-shanghai",
		Scenes:   []string{"porn"},
	})

	result, err := moderator.Audit(
		context.Background(),
		[]string{"https://source.example.com/activity.jpg"},
	)
	if err != nil {
		t.Fatalf("Audit() error = %v", err)
	}
	if result.Suggestion != ImageAuditSuggestionPass {
		t.Fatalf("Suggestion = %s, want pass", result.Suggestion)
	}
}

func TestBuildAliyunImageAuditParamsUsesIndexedFields(t *testing.T) {
	params, err := buildAliyunImageAuditParams(
		[]string{
			"https://source.example.com/first.jpg",
			"https://source.example.com/second.jpg",
		},
		5,
		"cn-shanghai",
		[]string{"porn", "terrorism"},
	)
	if err != nil {
		t.Fatalf("buildAliyunImageAuditParams() error = %v", err)
	}

	expected := map[string]string{
		"Action":           "ScanImage",
		"Version":          aliyunImageAuditVersion,
		"RegionId":         "cn-shanghai",
		"Scene.1":          "porn",
		"Scene.2":          "terrorism",
		"Task.1.ImageURL":  "https://source.example.com/first.jpg",
		"Task.1.DataId":    "image-6",
		"Task.1.MaxFrames": "1",
		"Task.2.ImageURL":  "https://source.example.com/second.jpg",
		"Task.2.DataId":    "image-7",
		"Task.2.MaxFrames": "1",
	}
	if !reflect.DeepEqual(params, expected) {
		t.Fatalf("params = %#v, want %#v", params, expected)
	}
}

func TestBuildAliyunImageAuditParamsKeepsSingleSceneCompatible(t *testing.T) {
	params, err := buildAliyunImageAuditParams(
		[]string{"https://source.example.com/activity.jpg"},
		0,
		"cn-shanghai",
		[]string{"porn"},
	)
	if err != nil {
		t.Fatalf("buildAliyunImageAuditParams() error = %v", err)
	}
	if params["Scene.1"] != "porn" {
		t.Fatalf("Scene.1 = %s, want porn", params["Scene.1"])
	}
	if _, exists := params["Scene"]; exists {
		t.Fatal("params contains unsupported Scene array field")
	}
	if _, exists := params["Task"]; exists {
		t.Fatal("params contains unsupported Task array field")
	}
}

func TestParseAliyunImageAuditResponseUsesStrongestSuggestion(t *testing.T) {
	payload := map[string]interface{}{
		"Data": map[string]interface{}{
			"Results": []interface{}{
				map[string]interface{}{
					"Code": float64(200),
					"SubResults": []interface{}{
						map[string]interface{}{
							"Suggestion": "pass",
							"Scene":      "porn",
							"Label":      "normal",
							"Rate":       float64(99),
						},
						map[string]interface{}{
							"Suggestion": "review",
							"Scene":      "ad",
							"Label":      "qrcode",
							"Rate":       float64(76.5),
						},
					},
				},
				map[string]interface{}{
					"Code": float64(200),
					"SubResults": []interface{}{
						map[string]interface{}{
							"Suggestion": "block",
							"Scene":      "terrorism",
							"Label":      "weapon",
							"Rate":       float64(98.25),
						},
					},
				},
			},
		},
	}

	result, err := parseAliyunImageAuditResponse(payload, 10)
	if err != nil {
		t.Fatalf("parseAliyunImageAuditResponse() error = %v", err)
	}
	if result.Suggestion != ImageAuditSuggestionBlock {
		t.Fatalf("Suggestion = %s, want block", result.Suggestion)
	}
	if len(result.Hits) != 1 {
		t.Fatalf("Hits length = %d, want 1", len(result.Hits))
	}
	if result.Hits[0].ImageIndex != 12 {
		t.Fatalf("ImageIndex = %d, want 12", result.Hits[0].ImageIndex)
	}
	reason := result.RejectReason()
	if !strings.Contains(reason, "第12张图片") ||
		!strings.Contains(reason, "暴恐或敏感内容") ||
		!strings.Contains(reason, "武器") {
		t.Fatalf("RejectReason = %s", reason)
	}
}

func TestParseAliyunImageAuditResponseSupportsDebugToolResponse(t *testing.T) {
	payload := map[string]interface{}{
		"Data": map[string]interface{}{
			"Results": []interface{}{
				map[string]interface{}{
					"ImageURL": "https://source.example.com/activity.jpg",
					"SubResults": []interface{}{
						map[string]interface{}{
							"Suggestion": "review",
							"Rate":       "91.06",
							"Label":      "sexy",
							"Scene":      "porn",
						},
					},
				},
			},
		},
	}

	result, err := parseAliyunImageAuditResponse(payload, 0)
	if err != nil {
		t.Fatalf("parseAliyunImageAuditResponse() error = %v", err)
	}
	if result.Suggestion != ImageAuditSuggestionBlock {
		t.Fatalf("Suggestion = %s, want block", result.Suggestion)
	}
	if len(result.Hits) != 1 {
		t.Fatalf("Hits length = %d, want 1", len(result.Hits))
	}
	if result.Hits[0].Suggestion != ImageAuditSuggestionBlock {
		t.Fatalf("Hit suggestion = %s, want block", result.Hits[0].Suggestion)
	}
	if result.Hits[0].Rate != 91.06 {
		t.Fatalf("Rate = %.2f, want 91.06", result.Hits[0].Rate)
	}
	reason := result.RejectReason()
	if !strings.Contains(reason, "活动图片未通过内容审核") ||
		!strings.Contains(reason, "性感低俗") {
		t.Fatalf("RejectReason = %s", reason)
	}
	for _, forbidden := range []string{"命中", "置信度", "%"} {
		if strings.Contains(reason, forbidden) {
			t.Fatalf("RejectReason contains %q: %s", forbidden, reason)
		}
	}
}

func TestParseAliyunImageAuditResponseAllowsConfiguredAdLabels(t *testing.T) {
	labels := []string{"spam", "npx", "qrcode", "programCode", "ad"}
	for _, label := range labels {
		label := label
		t.Run(label, func(t *testing.T) {
			payload := map[string]interface{}{
				"Data": map[string]interface{}{
					"Results": []interface{}{
						map[string]interface{}{
							"SubResults": []interface{}{
								map[string]interface{}{
									"Suggestion": "block",
									"Rate":       "99.00",
									"Label":      " " + label + " ",
									"Scene":      " AD ",
								},
							},
						},
					},
				},
			}

			result, err := parseAliyunImageAuditResponse(payload, 0)
			if err != nil {
				t.Fatalf("parseAliyunImageAuditResponse() error = %v", err)
			}
			if result.Suggestion != ImageAuditSuggestionPass {
				t.Fatalf("Suggestion = %s, want pass", result.Suggestion)
			}
			if len(result.Hits) != 0 {
				t.Fatalf("Hits length = %d, want 0", len(result.Hits))
			}
		})
	}
}

func TestParseAliyunImageAuditResponseKeepsOtherAdLabelsBlocked(t *testing.T) {
	payload := map[string]interface{}{
		"Data": map[string]interface{}{
			"Results": []interface{}{
				map[string]interface{}{
					"SubResults": []interface{}{
						map[string]interface{}{
							"Suggestion": "review",
							"Rate":       "80.00",
							"Label":      "illegal",
							"Scene":      "ad",
						},
					},
				},
			},
		},
	}

	result, err := parseAliyunImageAuditResponse(payload, 0)
	if err != nil {
		t.Fatalf("parseAliyunImageAuditResponse() error = %v", err)
	}
	if result.Suggestion != ImageAuditSuggestionBlock {
		t.Fatalf("Suggestion = %s, want block", result.Suggestion)
	}
	if len(result.Hits) != 1 {
		t.Fatalf("Hits length = %d, want 1", len(result.Hits))
	}
}

func TestParseAliyunImageAuditResponseKeepsNormalReview(t *testing.T) {
	payload := map[string]interface{}{
		"Data": map[string]interface{}{
			"Results": []interface{}{
				map[string]interface{}{
					"SubResults": []interface{}{
						map[string]interface{}{
							"Suggestion": "review",
							"Rate":       "60.00",
							"Label":      " NORMAL ",
							"Scene":      "porn",
						},
					},
				},
			},
		},
	}

	result, err := parseAliyunImageAuditResponse(payload, 0)
	if err != nil {
		t.Fatalf("parseAliyunImageAuditResponse() error = %v", err)
	}
	if result.Suggestion != ImageAuditSuggestionReview {
		t.Fatalf("Suggestion = %s, want review", result.Suggestion)
	}
	if len(result.Hits) != 1 {
		t.Fatalf("Hits length = %d, want 1", len(result.Hits))
	}
	if result.Hits[0].Suggestion != ImageAuditSuggestionReview {
		t.Fatalf("Hit suggestion = %s, want review", result.Hits[0].Suggestion)
	}
}

func TestParseAliyunImageAuditResponseRejectsIncompleteResult(t *testing.T) {
	payload := map[string]interface{}{
		"Data": map[string]interface{}{
			"Results": []interface{}{
				map[string]interface{}{
					"Code":       float64(200),
					"SubResults": []interface{}{},
				},
			},
		},
	}

	if _, err := parseAliyunImageAuditResponse(payload, 0); err == nil {
		t.Fatal("parseAliyunImageAuditResponse() error = nil, want error")
	}
}

func TestNormalizeImageAuditScenesFallsBackToSafeDefaults(t *testing.T) {
	scenes := normalizeImageAuditScenes([]string{"unknown"})
	if strings.Join(scenes, ",") != "porn,terrorism,ad,live" {
		t.Fatalf("scenes = %v", scenes)
	}
}
