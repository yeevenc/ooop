package activity

import (
	"testing"
	"time"
)

func TestAdminDisplayStatus(t *testing.T) {
	now := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.Local)
	past := now.Add(-time.Minute)
	future := now.Add(time.Minute)

	tests := []struct {
		name     string
		activity Activity
		expected string
	}{
		{
			name:     "进行中的过期活动展示为已结束",
			activity: Activity{Status: StatusOngoing, ActivityDate: &past},
			expected: StatusEnded,
		},
		{
			name:     "进行中的未来活动保持原状态",
			activity: Activity{Status: StatusOngoing, ActivityDate: &future},
			expected: StatusOngoing,
		},
		{
			name:     "未设置活动时间时保持原状态",
			activity: Activity{Status: StatusOngoing},
			expected: StatusOngoing,
		},
		{
			name:     "已下架活动不派生为已结束",
			activity: Activity{Status: StatusTakenDown, ActivityDate: &past},
			expected: StatusTakenDown,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if result := adminDisplayStatus(test.activity, now); result != test.expected {
				t.Fatalf("adminDisplayStatus() = %s, want %s", result, test.expected)
			}
		})
	}
}
