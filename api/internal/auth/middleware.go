package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"ooop-admin-api/internal/httpx"
)

const UserIDKey = "user_id"

func Middleware(tokenManager *TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		if token == "" || token == header {
			httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
			c.Abort()
			return
		}

		claims, err := tokenManager.Parse(token, TokenTypeAccess)
		if err != nil {
			httpx.Fail(c, http.StatusUnauthorized, 401002, err.Error())
			c.Abort()
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Next()
	}
}

func CurrentUserID(c *gin.Context) (int64, bool) {
	value, exists := c.Get(UserIDKey)
	if !exists {
		return 0, false
	}
	userID, ok := value.(int64)
	return userID, ok
}
