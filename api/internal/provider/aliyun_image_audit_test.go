package provider

import (
	"strings"
	"testing"
)

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
	if len(result.Hits) != 2 {
		t.Fatalf("Hits length = %d, want 2", len(result.Hits))
	}
	if result.Hits[1].ImageIndex != 12 {
		t.Fatalf("ImageIndex = %d, want 12", result.Hits[1].ImageIndex)
	}
	reason := result.RejectReason()
	if !strings.Contains(reason, "第12张图片") ||
		!strings.Contains(reason, "暴恐或敏感内容") ||
		!strings.Contains(reason, "武器") {
		t.Fatalf("RejectReason = %s", reason)
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
