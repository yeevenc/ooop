package chat

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ooop-admin-api/internal/auth"
	"ooop-admin-api/internal/contentmoderation"
	"ooop-admin-api/internal/httpx"
)

const (
	chatContentRejectedCode = 400101
	chatRestrictedCode      = 403003
)

type Handler struct {
	service      *Service
	reports      *ReportService
	tokenManager *auth.TokenManager
	access       auth.AccessChecker
}

func NewHandler(service *Service, reports *ReportService, tokenManager *auth.TokenManager, access auth.AccessChecker) *Handler {
	return &Handler{service: service, reports: reports, tokenManager: tokenManager, access: access}
}

func (h *Handler) Register(api *gin.RouterGroup) {
	group := api.Group("/chat", auth.Middleware(h.tokenManager, h.access))
	group.POST("/messages", h.sendMessage)
	group.GET("/conversations", h.listConversations)
	group.GET("/conversations/:id/messages", h.listMessages)
	group.PUT("/conversations/:id/read", h.markRead)
	group.DELETE("/conversations/:id", h.deleteConversation)
	group.POST("/conversations/:id/reports", h.submitReport)
	group.GET("/unread-count", h.unreadCount)
	group.GET("/access-status", h.accessStatus)
}

func (h *Handler) sendMessage(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req struct {
		RecipientID     int64  `json:"recipient_id"`
		ClientMessageID string `json:"client_message_id"`
		Type            string `json:"type"`
		Content         string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, 400001, "请求参数格式不正确")
		return
	}

	result, err := h.service.SendMessage(c.Request.Context(), userID, SendMessageInput{
		RecipientID:     req.RecipientID,
		ClientMessageID: req.ClientMessageID,
		Type:            req.Type,
		Content:         req.Content,
	})
	writeChatResult(c, result, err)
}

func (h *Handler) listConversations(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	result, err := h.service.ListConversations(
		c.Request.Context(),
		userID,
		queryInt(c, "page", 1),
		queryInt(c, "page_size", 20),
	)
	writeChatResult(c, result, err)
}

func (h *Handler) listMessages(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	conversationID, ok := pathID(c, "id", "会话 ID 格式不正确")
	if !ok {
		return
	}
	beforeID, valid := optionalQueryID(c, "before_id")
	if !valid {
		return
	}
	afterID, valid := optionalQueryID(c, "after_id")
	if !valid {
		return
	}

	result, err := h.service.ListMessages(c.Request.Context(), userID, MessageQuery{
		ConversationID: conversationID,
		BeforeID:       beforeID,
		AfterID:        afterID,
		PageSize:       queryInt(c, "page_size", 50),
	})
	writeChatResult(c, result, err)
}

func (h *Handler) markRead(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	conversationID, ok := pathID(c, "id", "会话 ID 格式不正确")
	if !ok {
		return
	}
	var req struct {
		LastMessageID int64 `json:"last_message_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, 400001, "请求参数格式不正确")
		return
	}

	err := h.service.MarkRead(c.Request.Context(), userID, conversationID, req.LastMessageID)
	writeChatResult(c, gin.H{
		"conversationId": formatID(conversationID),
		"lastMessageId":  formatID(req.LastMessageID),
	}, err)
}

func (h *Handler) deleteConversation(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	conversationID, ok := pathID(c, "id", "会话 ID 格式不正确")
	if !ok {
		return
	}

	err := h.service.DeleteConversation(c.Request.Context(), userID, conversationID)
	writeChatResult(c, gin.H{"id": formatID(conversationID)}, err)
}

func (h *Handler) unreadCount(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	count, err := h.service.CountUnread(c.Request.Context(), userID)
	writeChatResult(c, gin.H{"count": count}, err)
}

func (h *Handler) accessStatus(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	result, err := h.service.GetChatAccessStatus(c.Request.Context(), userID)
	writeChatResult(c, result, err)
}

func currentUserID(c *gin.Context) (int64, bool) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
		return 0, false
	}
	return userID, true
}

func pathID(c *gin.Context, key string, message string) (int64, bool) {
	value, err := strconv.ParseInt(c.Param(key), 10, 64)
	if err != nil || value <= 0 {
		httpx.Fail(c, http.StatusBadRequest, 400001, message)
		return 0, false
	}
	return value, true
}

func optionalQueryID(c *gin.Context, key string) (int64, bool) {
	raw := c.Query(key)
	if raw == "" {
		return 0, true
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		httpx.Fail(c, http.StatusBadRequest, 400001, key+" 格式不正确")
		return 0, false
	}
	return value, true
}

func queryInt(c *gin.Context, key string, fallback int) int {
	value, err := strconv.Atoi(c.Query(key))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func writeChatResult(c *gin.Context, data interface{}, err error) {
	if err == nil {
		httpx.OK(c, data)
		return
	}

	switch {
	case errors.Is(err, ErrChatRestricted):
		httpx.Fail(c, http.StatusForbidden, chatRestrictedCode, err.Error())
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrRecipientNotFound):
		httpx.Fail(c, http.StatusNotFound, 404001, err.Error())
	case errors.Is(err, ErrReportNotFound):
		httpx.Fail(c, http.StatusNotFound, 404001, err.Error())
	case errors.Is(err, contentmoderation.ErrRejected):
		httpx.Fail(c, http.StatusBadRequest, chatContentRejectedCode, err.Error())
	case errors.Is(err, ErrSendToSelf),
		errors.Is(err, ErrContentRequired),
		errors.Is(err, ErrContentTooLong),
		errors.Is(err, ErrMessageTypeInvalid),
		errors.Is(err, ErrImageURLInvalid),
		errors.Is(err, ErrClientMessageInvalid),
		errors.Is(err, ErrClientMessageConflict),
		errors.Is(err, ErrCursorConflict),
		errors.Is(err, ErrReportReasonInvalid),
		errors.Is(err, ErrReportDescription),
		errors.Is(err, ErrReportTooLong),
		errors.Is(err, ErrReportStatusInvalid),
		errors.Is(err, ErrReportResultRequired),
		errors.Is(err, ErrReportResultTooLong),
		errors.Is(err, ErrReportRestrictionRequired),
		errors.Is(err, ErrReportRestrictionInvalid):
		httpx.Fail(c, http.StatusBadRequest, 400001, err.Error())
	case errors.Is(err, ErrReportPending), errors.Is(err, ErrReportProcessed):
		httpx.Fail(c, http.StatusConflict, 409001, err.Error())
	case errors.Is(err, ErrRateLimited):
		httpx.Fail(c, http.StatusTooManyRequests, 429001, err.Error())
	default:
		httpx.Fail(c, http.StatusInternalServerError, 500001, err.Error())
	}
}
