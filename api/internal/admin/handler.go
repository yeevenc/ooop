package admin

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ooop-admin-api/internal/auth"
	"ooop-admin-api/internal/httpx"
	"ooop-admin-api/internal/user"
)

type Handler struct {
	service      *Service
	appUsers     *user.AuthService
	tokenManager *auth.TokenManager
}

func NewHandler(service *Service, appUsers *user.AuthService, tokenManager *auth.TokenManager) *Handler {
	return &Handler{
		service:      service,
		appUsers:     appUsers,
		tokenManager: tokenManager,
	}
}

func (h *Handler) Register(api *gin.RouterGroup) {
	adminGroup := api.Group("/admin")
	adminGroup.POST("/auth/login", h.login)

	protected := adminGroup.Group("")
	protected.Use(Middleware(h.tokenManager))
	protected.GET("/users", h.userList)
}

func (h *Handler) login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !bindJSON(c, &req) {
		return
	}

	result, err := h.service.Login(c.Request.Context(), req.Username, req.Password)
	writeResult(c, result, err)
}

func (h *Handler) userList(c *gin.Context) {
	query := user.UserListQuery{
		Page:     queryInt(c, "page", 1),
		PageSize: queryInt(c, "page_size", 10),
		Keyword:  c.Query("keyword"),
	}
	if status := c.Query("status"); status != "" {
		value, err := strconv.Atoi(status)
		if err != nil {
			httpx.Fail(c, http.StatusBadRequest, 400001, "用户状态格式不正确")
			return
		}
		query.Status = &value
	}

	result, err := h.appUsers.ListUsers(c.Request.Context(), query)
	writeResult(c, result, err)
}

func bindJSON(c *gin.Context, target interface{}) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		httpx.Fail(c, http.StatusBadRequest, 400001, "请求参数格式不正确")
		return false
	}
	return true
}

func queryInt(c *gin.Context, key string, fallback int) int {
	value := c.Query(key)
	if value == "" {
		return fallback
	}

	result, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return result
}

func writeResult(c *gin.Context, data interface{}, err error) {
	if err == nil {
		httpx.OK(c, data)
		return
	}

	switch {
	case errors.Is(err, ErrInvalidAccount):
		httpx.Fail(c, http.StatusUnauthorized, 401003, err.Error())
	case errors.Is(err, ErrDisabledAdmin):
		httpx.Fail(c, http.StatusForbidden, 403001, err.Error())
	case errors.Is(err, auth.ErrInvalidToken),
		errors.Is(err, auth.ErrExpiredToken):
		httpx.Fail(c, http.StatusUnauthorized, 401002, err.Error())
	default:
		httpx.Fail(c, http.StatusInternalServerError, 500001, err.Error())
	}
}
