package activity

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestIntentMapPayload(t *testing.T) {
	var input IntentInput
	err := json.Unmarshal([]byte(`{"city":" 杭州市 ","latitude":30.2,"longitude":120.1,"locationText":"杭州市西湖区","categoryIds":[1,1,2],"timeSlots":["周末白天","周末白天"],"note":" 新手 ","notifyEnabled":true}`), &input)
	if err != nil {
		t.Fatal(err)
	}
	got, err := normalizeIntent(input)
	if err != nil {
		t.Fatal(err)
	}
	if got.City != "杭州市" || *got.Latitude != 30.2 || *got.Longitude != 120.1 || len(got.CategoryIDs) != 2 || len(got.TimeSlots) != 1 || !got.NotifyEnabled || got.Note != "新手" {
		t.Fatalf("地图登记字段未正确保留: %+v", got)
	}
}
func TestIntentRejectsMissingOrInvalidMapPoint(t *testing.T) {
	for _, body := range []string{
		`{"city":"杭州市","locationText":"西湖区","categoryIds":[1],"timeSlots":["周末白天"]}`,
		`{"city":"杭州市","latitude":91,"longitude":120,"locationText":"西湖区","categoryIds":[1],"timeSlots":["周末白天"]}`,
		`{"city":"杭州市","latitude":30,"longitude":181,"locationText":"西湖区","categoryIds":[1],"timeSlots":["周末白天"]}`,
		`{"city":"杭州市","latitude":30,"longitude":120,"locationText":"西湖区","categoryIds":[1],"timeSlots":["错误时段"]}`,
	} {
		var input IntentInput
		if err := json.Unmarshal([]byte(body), &input); err != nil {
			t.Fatal(err)
		}
		if _, err := normalizeIntent(input); !errors.Is(err, ErrInvalidIntent) {
			t.Fatalf("应拒绝无效登记: %s", body)
		}
	}
}
