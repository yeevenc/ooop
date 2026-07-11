package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"ooop-admin-api/internal/httpx"
)

const UserIDKey = "user_id"

// CodeAccountBanned APP 用户被封禁时的业务码。
// 备注：HTTP 403 + code=403002，与通用禁止 403001 区分，客户端可跳转封禁页。
const CodeAccountBanned = 403002

// AccessChecker 校验 APP 用户是否允许访问（封禁拦截等）。
type AccessChecker interface {
	CheckAppUserAccess(ctx context.Context, userID int64) error
}

// ErrAccountBanned 账号被封禁；外层 message 可能附带解封时间/原因备注。
var ErrAccountBanned = errors.New("账号已被封禁")

// Middleware 校验 APP access token。
// 备注：传入 AccessChecker 后，已登录用户每次请求也会校验封禁状态（含限时自动解封）。
func Middleware(tokenManager *TokenManager, checkers ...AccessChecker) gin.HandlerFunc {
	var checker AccessChecker
	if len(checkers) > 0 {
		checker = checkers[0]
	}

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

		if checker != nil {
			if err := checker.CheckAppUserAccess(c.Request.Context(), claims.UserID); err != nil {
				if errors.Is(err, ErrAccountBanned) {
					httpx.Fail(c, http.StatusForbidden, CodeAccountBanned, err.Error())
					c.Abort()
					return
				}
				httpx.Fail(c, http.StatusForbidden, 403001, err.Error())
				c.Abort()
				return
			}
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
