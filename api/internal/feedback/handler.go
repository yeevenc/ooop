package feedback

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ooop-admin-api/internal/admin"
	"ooop-admin-api/internal/auth"
	"ooop-admin-api/internal/httpx"
)

type Handler struct {
	service           *Service
	tokenManager      *auth.TokenManager
	adminTokenManager *auth.TokenManager
}

func NewHandler(service *Service, tokenManager *auth.TokenManager, adminTokenManager *auth.TokenManager) *Handler {
	return &Handler{
		service:           service,
		tokenManager:      tokenManager,
		adminTokenManager: adminTokenManager,
	}
}

func (h *Handler) Register(api *gin.RouterGroup) {
	group := api.Group("/feedbacks", auth.Middleware(h.tokenManager))
	group.POST("", h.create)

	adminGroup := api.Group("/admin/feedbacks", admin.Middleware(h.adminTokenManager))
	adminGroup.GET("", h.adminList)
}

func (h *Handler) create(c *gin.Context) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
		return
	}

	var req CreateInput
	if !bindJSON(c, &req) {
		return
	}

	result, err := h.service.Create(c.Request.Context(), userID, req)
	writeResult(c, result, err)
}

func (h *Handler) adminList(c *gin.Context) {
	query := ListQuery{
		Page:     queryInt(c, "page", 1),
		PageSize: queryInt(c, "page_size", 10),
		Type:     c.Query("type"),
		Keyword:  c.Query("keyword"),
	}

	result, err := h.service.List(c.Request.Context(), query)
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
	case errors.Is(err, ErrInvalidType),
		errors.Is(err, ErrInvalidContent),
		errors.Is(err, ErrTooManyImages):
		httpx.Fail(c, http.StatusBadRequest, 400001, err.Error())
	default:
		httpx.Fail(c, http.StatusInternalServerError, 500001, err.Error())
	}
}
