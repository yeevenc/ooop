package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ooop-admin-api/internal/httpx"
	"ooop-admin-api/internal/message"
)

func (h *Handler) systemBroadcastList(c *gin.Context) {
	if h.messages == nil {
		writeResult(c, nil, message.ErrBroadcastUnavailable)
		return
	}
	result, err := h.messages.ListSystemBroadcasts(c.Request.Context(), message.BroadcastQuery{
		Page:     queryInt(c, "page", 1),
		PageSize: queryInt(c, "page_size", 10),
		Status:   c.Query("status"),
	})
	writeResult(c, result, err)
}

func (h *Handler) systemBroadcastCreate(c *gin.Context) {
	if h.messages == nil {
		writeResult(c, nil, message.ErrBroadcastUnavailable)
		return
	}
	adminID, exists := c.Get(AdminIDKey)
	if !exists {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录后台")
		return
	}

	var req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if !bindJSON(c, &req) {
		return
	}

	id, _ := adminID.(int64)
	result, err := h.messages.CreateSystemBroadcast(c.Request.Context(), id, req.Title, req.Content)
	writeResult(c, result, err)
}
