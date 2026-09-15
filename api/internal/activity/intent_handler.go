package activity

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"ooop-admin-api/internal/auth"
	"ooop-admin-api/internal/httpx"
)

func (h *Handler) saveIntent(c *gin.Context) {
	uid, ok := auth.CurrentUserID(c)
	if !ok {
		return
	}
	var input IntentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Fail(c, http.StatusBadRequest, 400001, "登记参数格式不正确")
		return
	}
	item, err := h.service.SaveIntent(c.Request.Context(), uid, input)
	if errors.Is(err, ErrInvalidIntent) {
		httpx.Fail(c, http.StatusBadRequest, 400001, err.Error())
		return
	}
	writeResult(c, item, err)
}
func (h *Handler) listIntents(c *gin.Context) {
	uid, ok := auth.CurrentUserID(c)
	if !ok {
		return
	}
	items, err := h.service.ListIntents(c.Request.Context(), uid, c.Query("city"), queryInt(c, "page", 1))
	writeResult(c, items, err)
}
func (h *Handler) cancelIntent(c *gin.Context) {
	uid, ok := auth.CurrentUserID(c)
	if !ok {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	err := h.service.CancelIntent(c.Request.Context(), uid, id)
	writeResult(c, gin.H{"id": id}, err)
}
