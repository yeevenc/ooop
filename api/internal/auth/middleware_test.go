package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"ooop-admin-api/internal/config"
	"ooop-admin-api/internal/httpx"
)

func TestMiddlewarePreventsProtectedResponseCaching(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenManager := NewTokenManager(config.JWTConfig{
		Secret:         "middleware-test-secret",
		AccessTokenTTL: time.Hour,
		Issuer:         "middleware-test",
	})
	token, err := tokenManager.NewToken(3000)
	if err != nil {
		t.Fatalf("NewToken() error = %v", err)
	}

	tests := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{name: "未登录响应", wantStatus: http.StatusUnauthorized},
		{
			name:       "已登录响应",
			token:      token.AccessToken,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := gin.New()
			engine.Use(httpx.CORS([]string{"https://admin.ooopai.cn"}))
			engine.GET("/protected", Middleware(tokenManager), func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "private"})
			})

			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			request.Header.Set("Origin", "https://admin.ooopai.cn")
			if tt.token != "" {
				request.Header.Set("Authorization", "Bearer "+tt.token)
			}
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if cacheControl := recorder.Header().Get("Cache-Control"); !strings.Contains(cacheControl, "no-store") {
				t.Fatalf("Cache-Control = %q, want no-store", cacheControl)
			}
			if pragma := recorder.Header().Get("Pragma"); pragma != "no-cache" {
				t.Fatalf("Pragma = %q, want no-cache", pragma)
			}
			vary := strings.Join(recorder.Header().Values("Vary"), ",")
			if !strings.Contains(vary, "Authorization") {
				t.Fatalf("Vary = %q, want Authorization", vary)
			}
			if !strings.Contains(vary, "Origin") {
				t.Fatalf("Vary = %q, want Origin", vary)
			}
			if allowOrigin := recorder.Header().Get("Access-Control-Allow-Origin"); allowOrigin != "https://admin.ooopai.cn" {
				t.Fatalf("Access-Control-Allow-Origin = %q", allowOrigin)
			}
		})
	}
}
