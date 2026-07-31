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

func TestChatRoutesRequireAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	NewHandler(nil, nil, nil, nil).Register(engine.Group("/api/v1"))

	tests := []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/chat/messages"},
		{method: http.MethodGet, path: "/api/v1/chat/conversations"},
		{method: http.MethodGet, path: "/api/v1/chat/conversations/1/messages"},
		{method: http.MethodPut, path: "/api/v1/chat/conversations/1/read"},
		{method: http.MethodDelete, path: "/api/v1/chat/conversations/1"},
		{method: http.MethodPost, path: "/api/v1/chat/conversations/1/reports"},
		{method: http.MethodGet, path: "/api/v1/chat/unread-count"},
		{method: http.MethodGet, path: "/api/v1/chat/access-status"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.path, nil)
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
			}
			var response struct {
				Code int `json:"code"`
			}
			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatalf("decode response error = %v", err)
			}
			if response.Code != 401001 {
				t.Fatalf("code = %d, want 401001", response.Code)
			}
		})
	}
}
