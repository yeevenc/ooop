package admin

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"ooop-admin-api/internal/auth"
	"ooop-admin-api/internal/httpx"
)

const AdminIDKey = "admin_id"

func Middleware(tokenManager *auth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		if token == "" || token == header {
			httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录后台")
			c.Abort()
			return
		}

		claims, err := tokenManager.Parse(token, auth.TokenTypeAccess)
		if err != nil {
			httpx.Fail(c, http.StatusUnauthorized, 401002, err.Error())
			c.Abort()
			return
		}

		c.Set(AdminIDKey, claims.UserID)
		c.Next()
	}
}
