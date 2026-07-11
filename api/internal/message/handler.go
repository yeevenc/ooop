package message

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ooop-admin-api/internal/auth"
	"ooop-admin-api/internal/httpx"
)

type Handler struct {
	service      *Service
	tokenManager *auth.TokenManager
	access       auth.AccessChecker
}

func NewHandler(service *Service, tokenManager *auth.TokenManager, access auth.AccessChecker) *Handler {
	return &Handler{
		service:      service,
		tokenManager: tokenManager,
		access:       access,
	}
}

func (h *Handler) Register(api *gin.RouterGroup) {
	group := api.Group("/messages", auth.Middleware(h.tokenManager, h.access))
	group.GET("", h.list)
	group.PUT("/read-all", h.markAllRead)
	group.PUT("/:id/read", h.markRead)
	group.DELETE("", h.clear)
	group.DELETE("/:id", h.delete)
}

func (h *Handler) list(c *gin.Context) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
		return
	}

	result, err := h.service.ListUserMessages(
		c.Request.Context(),
		userID,
		queryInt(c, "page", 1),
		queryInt(c, "page_size", 20),
	)
	writeResult(c, result, err)
}

func (h *Handler) markRead(c *gin.Context) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Fail(c, http.StatusBadRequest, 400001, "消息 ID 格式不正确")
		return
	}

	err = h.service.MarkRead(c.Request.Context(), userID, id)
	writeResult(c, gin.H{"id": c.Param("id")}, err)
}

func (h *Handler) markAllRead(c *gin.Context) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
		return
	}

	count, err := h.service.MarkAllRead(c.Request.Context(), userID)
	writeResult(c, gin.H{"count": count}, err)
}

func (h *Handler) delete(c *gin.Context) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Fail(c, http.StatusBadRequest, 400001, "消息 ID 格式不正确")
		return
	}

	err = h.service.DeleteMessage(c.Request.Context(), userID, id)
	writeResult(c, gin.H{"id": c.Param("id")}, err)
}

func (h *Handler) clear(c *gin.Context) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
		return
	}

	count, err := h.service.ClearMessages(c.Request.Context(), userID)
	writeResult(c, gin.H{"count": count}, err)
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

	if errors.Is(err, ErrNotFound) {
		httpx.Fail(c, http.StatusNotFound, 404001, err.Error())
		return
	}
	httpx.Fail(c, http.StatusInternalServerError, 500001, err.Error())
}
