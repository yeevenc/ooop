package chat

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ooop-admin-api/internal/httpx"
)

func (h *Handler) submitReport(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	conversationID, ok := pathID(c, "id", "会话 ID 格式不正确")
	if !ok {
		return
	}
	if h.reports == nil {
		httpx.Fail(c, http.StatusServiceUnavailable, 503001, "举报服务暂不可用")
		return
	}

	var req struct {
		Reason      string `json:"reason"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, 400001, "请求参数格式不正确")
		return
	}

	result, err := h.reports.Submit(c.Request.Context(), userID, conversationID, SubmitReportInput{
		Reason:      req.Reason,
		Description: req.Description,
	})
	writeChatResult(c, result, err)
}
