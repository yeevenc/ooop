package admin

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ooop-admin-api/internal/activity"
	"ooop-admin-api/internal/httpx"
)

// ===== 活动分类管理 =====

func (h *Handler) categoryList(c *gin.Context) {
	result, err := h.activities.AdminListCategories(c.Request.Context())
	writeResult(c, result, err)
}

func (h *Handler) categoryCreate(c *gin.Context) {
	var req struct {
		ID     string `json:"id"`
		Label  string `json:"label"`
		Icon   string `json:"icon"`
		Sort   int    `json:"sort"`
		Status int    `json:"status"`
	}
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.activities.CreateCategory(c.Request.Context(), activity.CategoryInput{
		ID:     req.ID,
		Label:  req.Label,
		Icon:   req.Icon,
		Sort:   req.Sort,
		Status: req.Status,
	})
	writeResult(c, result, err)
}

func (h *Handler) categoryUpdate(c *gin.Context) {
	var req struct {
		Label  string `json:"label"`
		Icon   string `json:"icon"`
		Sort   int    `json:"sort"`
		Status int    `json:"status"`
	}
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.activities.UpdateCategory(c.Request.Context(), c.Param("id"), activity.CategoryInput{
		Label:  req.Label,
		Icon:   req.Icon,
		Sort:   req.Sort,
		Status: req.Status,
	})
	writeResult(c, result, err)
}

func (h *Handler) categoryDelete(c *gin.Context) {
	id := c.Param("id")
	err := h.activities.DeleteCategory(c.Request.Context(), id)
	writeResult(c, gin.H{"id": id}, err)
}

// ===== 活动管理 =====

type activityUpdateRequest struct {
	Title             string `json:"title"`
	CategoryID        string `json:"category_id"`
	ActivityTime      string `json:"activity_time"`
	LocationText      string `json:"location_text"`
	City              string `json:"city"`
	TotalCount        int    `json:"total_count"`
	CostType          string `json:"cost_type"`
	FeeDetail         string `json:"fee_detail"`
	GenderRequirement string `json:"gender_requirement"`
	Intro             string `json:"intro"`
	Notice            string `json:"notice"`
}

func (r activityUpdateRequest) toUpdate() activity.AdminActivityUpdate {
	return activity.AdminActivityUpdate{
		Title:             r.Title,
		CategoryID:        r.CategoryID,
		ActivityTime:      r.ActivityTime,
		LocationText:      r.LocationText,
		City:              r.City,
		TotalCount:        r.TotalCount,
		CostType:          r.CostType,
		FeeDetail:         r.FeeDetail,
		GenderRequirement: r.GenderRequirement,
		Intro:             r.Intro,
		Notice:            r.Notice,
	}
}

func (h *Handler) activityList(c *gin.Context) {
	result, err := h.activities.AdminListActivities(c.Request.Context(), activity.AdminActivityQuery{
		Keyword:    c.Query("keyword"),
		Status:     c.Query("status"),
		CategoryID: c.Query("category_id"),
		Page:       queryInt(c, "page", 1),
		PageSize:   queryInt(c, "page_size", 10),
	})
	writeResult(c, result, err)
}

func (h *Handler) activityDetail(c *gin.Context) {
	id, ok := parseActivityID(c)
	if !ok {
		return
	}
	result, err := h.activities.GetActivityByID(c.Request.Context(), id)
	writeResult(c, result, err)
}

func (h *Handler) activityUpdate(c *gin.Context) {
	id, ok := parseActivityID(c)
	if !ok {
		return
	}
	var req activityUpdateRequest
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.activities.UpdateActivity(c.Request.Context(), id, req.toUpdate())
	writeResult(c, result, err)
}

func (h *Handler) activityDelete(c *gin.Context) {
	id, ok := parseActivityID(c)
	if !ok {
		return
	}
	err := h.activities.DeleteActivity(c.Request.Context(), id)
	writeResult(c, gin.H{"id": id}, err)
}

func (h *Handler) activityReview(c *gin.Context) {
	id, ok := parseActivityID(c)
	if !ok {
		return
	}
	var req struct {
		Action string `json:"action"`
	}
	if !bindJSON(c, &req) {
		return
	}
	if req.Action != "approve" && req.Action != "reject" {
		httpx.Fail(c, http.StatusBadRequest, 400004, "审核动作不合法")
		return
	}
	result, err := h.activities.ReviewActivity(c.Request.Context(), id, req.Action == "approve")
	writeResult(c, result, err)
}

func (h *Handler) activityStatus(c *gin.Context) {
	id, ok := parseActivityID(c)
	if !ok {
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.activities.SetActivityStatus(c.Request.Context(), id, req.Status)
	writeResult(c, result, err)
}

func parseActivityID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Fail(c, http.StatusBadRequest, 400001, "活动 ID 格式不正确")
		return 0, false
	}
	return id, true
}
