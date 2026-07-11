package provider

import (
	"strings"
	"testing"
)

func TestSplitTextUsesRuneLimit(t *testing.T) {
	content := strings.Repeat("活动", 401)
	chunks := splitText(content, 600)
	if len(chunks) != 2 {
		t.Fatalf("期望拆成 2 段，实际为 %d 段", len(chunks))
	}
	if len([]rune(chunks[0])) != 600 || len([]rune(chunks[1])) != 202 {
		t.Fatalf("分段字符数不正确: %d, %d", len([]rune(chunks[0])), len([]rune(chunks[1])))
	}
}
