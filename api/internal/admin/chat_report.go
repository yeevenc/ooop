package admin

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"ooop-admin-api/internal/chat"
	"ooop-admin-api/internal/httpx"
)

func (h *Handler) chatReportList(c *gin.Context) {
	result, err := h.chatReports.ListAdmin(c.Request.Context(), chat.AdminReportQuery{
		Page:     queryInt(c, "page", 1),
		PageSize: queryInt(c, "page_size", 20),
		Status:   c.Query("status"),
		Keyword:  c.Query("keyword"),
	})
	writeResult(c, result, err)
}

func (h *Handler) chatReportDetail(c *gin.Context) {
	id, ok := chatReportID(c)
	if !ok {
		return
	}
	result, err := h.chatReports.DetailAdmin(c.Request.Context(), id)
	writeResult(c, result, err)
}

func (h *Handler) resolveChatReport(c *gin.Context) {
	id, ok := chatReportID(c)
	if !ok {
		return
	}
	adminID, exists := c.Get(AdminIDKey)
	if !exists {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录后台")
		return
	}

	var req struct {
		Status           string     `json:"status"`
		Result           string     `json:"result"`
		RestrictionUntil *time.Time `json:"restriction_until"`
	}
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.chatReports.Resolve(c.Request.Context(), id, adminID.(int64), chat.ResolveReportInput{
		Status:           req.Status,
		Result:           req.Result,
		RestrictionUntil: req.RestrictionUntil,
	})
	writeResult(c, result, err)
}

func chatReportID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Fail(c, http.StatusBadRequest, 400001, "举报 ID 格式不正确")
		return 0, false
	}
	return id, true
}
