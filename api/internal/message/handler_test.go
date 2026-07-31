package message

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMessageRoutesRequireAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	NewHandler(nil, nil, nil).Register(engine.Group("/api/v1"))

	tests := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/v1/messages"},
		{method: http.MethodPut, path: "/api/v1/messages/read-all"},
		{method: http.MethodPut, path: "/api/v1/messages/1/read"},
		{method: http.MethodDelete, path: "/api/v1/messages"},
		{method: http.MethodDelete, path: "/api/v1/messages/1"},
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
