package provider

import (
	"strings"
	"testing"
)

func TestAliyunRPCResponseErrorKeepsProviderDetails(t *testing.T) {
	err := aliyunRPCResponseError(
		"400 Bad Request",
		[]byte(`{"Code":"InvalidAction.NotFound","Message":"Specified action is not found.","RequestId":"request-123"}`),
	)
	message := err.Error()
	if !strings.Contains(message, "InvalidAction.NotFound") ||
		!strings.Contains(message, "Specified action is not found.") ||
		!strings.Contains(message, "request-123") {
		t.Fatalf("error = %s", message)
	}
}

func TestAliyunRPCResponseErrorDoesNotExposeUnknownBody(t *testing.T) {
	err := aliyunRPCResponseError(
		"502 Bad Gateway",
		[]byte(`<html>gateway error</html>`),
	)
	if err.Error() != "阿里云接口请求失败: 502 Bad Gateway" {
		t.Fatalf("error = %s", err)
	}
}
