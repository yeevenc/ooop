package chat

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"ooop-admin-api/internal/contentmoderation"
)

func TestWriteChatResultUsesSensitiveContentCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	writeChatResult(
		context,
		nil,
		fmt.Errorf("消息内容: %w", contentmoderation.ErrRejected),
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	var response struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response error = %v", err)
	}
	if response.Code != chatContentRejectedCode {
		t.Fatalf("code = %d, want %d", response.Code, chatContentRejectedCode)
	}
}

func TestHandlerRegistersDeleteConversation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	NewHandler(nil, nil, nil, nil).Register(engine.Group("/api/v1"))

	for _, route := range engine.Routes() {
		if route.Method == http.MethodDelete && route.Path == "/api/v1/chat/conversations/:id" {
			return
		}
	}
	t.Fatal("delete conversation route was not registered")
}

func TestHandlerRegistersSubmitReport(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	NewHandler(nil, nil, nil, nil).Register(engine.Group("/api/v1"))

	for _, route := range engine.Routes() {
		if route.Method == http.MethodPost && route.Path == "/api/v1/chat/conversations/:id/reports" {
			return
		}
	}
	t.Fatal("submit report route was not registered")
}
