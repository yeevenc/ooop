package activity

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ooop-admin-api/internal/auth"
	"ooop-admin-api/internal/contentmoderation"
	"ooop-admin-api/internal/httpx"
)

type Handler struct {
	service      *Service
	tokenManager *auth.TokenManager
}

type createRequest struct {
	Title             string   `json:"title"`
	CategoryID        string   `json:"category_id"`
	CategoryLabel     string   `json:"category_label"`
	ActivityDate      string   `json:"activity_date"`
	ActivityTime      string   `json:"activity_time"`
	DeadlineAt        string   `json:"deadline_at"`
	LocationText      string   `json:"location_text"`
	City              string   `json:"city"`
	Latitude          float64  `json:"latitude"`
	Longitude         float64  `json:"longitude"`
	TotalCount        int      `json:"total_count"`
	CostType          string   `json:"cost_type"`
	FeeDetail         string   `json:"fee_detail"`
	GenderRequirement string   `json:"gender_requirement"`
	Intro             string   `json:"intro"`
	Notice            string   `json:"notice"`
	ImageURLs         []string `json:"image_urls"`
}

func NewHandler(service *Service, tokenManager *auth.TokenManager) *Handler {
	return &Handler{service: service, tokenManager: tokenManager}
}

func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/activity-categories", h.listCategories)

	group := api.Group("/activities")
	group.GET("", h.list)
	group.GET("/:id", h.detail)
	group.POST("", auth.Middleware(h.tokenManager), h.create)
	group.PUT("/:id/favorite", auth.Middleware(h.tokenManager), h.favorite)
	group.DELETE("/:id/favorite", auth.Middleware(h.tokenManager), h.unfavorite)

	// 用户维度的发布活动：他人主页（公开，仅 ongoing）、我的主页（鉴权，含审核中）
	api.GET("/users/:id/activities", h.userActivities)
	api.GET("/user/activities", auth.Middleware(h.tokenManager), h.myActivities)
	api.GET("/user/favorite-activities", auth.Middleware(h.tokenManager), h.myFavorites)

	// 报名(参加)：报名 / 发起人查看 & 审核申请人
	group.POST("/:id/join", auth.Middleware(h.tokenManager), h.join)
	group.PUT("/:id/participation/cancel", auth.Middleware(h.tokenManager), h.cancelParticipation)
	group.PUT("/:id/cancel", auth.Middleware(h.tokenManager), h.cancelOwnedActivity)
	group.PUT("/:id/take-down", auth.Middleware(h.tokenManager), h.takeDownOwnedActivity)
	group.DELETE("/:id", auth.Middleware(h.tokenManager), h.deleteOwnedActivity)
	group.GET("/:id/my-participation", auth.Middleware(h.tokenManager), h.myParticipation)
	group.GET("/:id/applicants", auth.Middleware(h.tokenManager), h.applicants)
	group.PUT("/:id/applicants/:uid", auth.Middleware(h.tokenManager), h.reviewApplicant)

	// 我参加的 / Ta 参加的活动
	api.GET("/user/joined-activities", auth.Middleware(h.tokenManager), h.myJoined)
	api.GET("/users/:id/joined-activities", h.userJoined)
}

func (h *Handler) create(c *gin.Context) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
		return
	}

	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, 400001, "请求参数格式不正确")
		return
	}

	result, err := h.service.Create(c.Request.Context(), userID, req.toInput())
	writeResult(c, result, err)
}

func (h *Handler) list(c *gin.Context) {
	latitude, hasLatitude := queryFloat(c, "latitude")
	longitude, hasLongitude := queryFloat(c, "longitude")

	result, err := h.service.List(c.Request.Context(), ListQuery{
		City:        c.Query("city"),
		CategoryID:  c.Query("category_id"),
		Keyword:     c.Query("keyword"),
		Latitude:    latitude,
		Longitude:   longitude,
		HasLocation: hasLatitude && hasLongitude && validCoordinate(latitude, longitude),
		Page:        queryInt(c, "page", 1),
		PageSize:    queryInt(c, "page_size", 20),
	})
	writeResult(c, result, err)
}

// detail 活动详情（公开）：仅返回 App 可见（ongoing）的活动，未过审/已下架/不存在均按 404 处理。
func (h *Handler) detail(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Fail(c, http.StatusBadRequest, 400001, "活动 ID 格式不正确")
		return
	}
	userID := h.optionalUserID(c)
	result, err := h.service.GetPublicActivityByIDForUser(c.Request.Context(), id, userID)
	writeResult(c, result, err)
}

func (h *Handler) favorite(c *gin.Context) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	result, err := h.service.FavoriteActivity(c.Request.Context(), userID, id)
	writeResult(c, result, err)
}

func (h *Handler) unfavorite(c *gin.Context) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	result, err := h.service.UnfavoriteActivity(c.Request.Context(), userID, id)
	writeResult(c, result, err)
}

func (h *Handler) listCategories(c *gin.Context) {
	result, err := h.service.ListCategories(c.Request.Context())
	writeResult(c, result, err)
}

// userActivities 某用户对外可见（ongoing）的发布活动，用于他人主页。
func (h *Handler) userActivities(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Fail(c, http.StatusBadRequest, 400001, "用户 ID 格式不正确")
		return
	}
	result, err := h.service.ListPublicUserActivities(
		c.Request.Context(), id, queryInt(c, "page", 1), queryInt(c, "page_size", 20),
	)
	writeResult(c, result, err)
}

// myActivities 当前登录用户自己的发布活动（含审核中 pending），用于我的主页。
func (h *Handler) myActivities(c *gin.Context) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
		return
	}
	result, err := h.service.ListMyActivities(
		c.Request.Context(), userID, queryInt(c, "page", 1), queryInt(c, "page_size", 20),
	)
	writeResult(c, result, err)
}

// myFavorites 当前登录用户收藏的活动，数据来源为 activity_favorites 独立表。
func (h *Handler) myFavorites(c *gin.Context) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
		return
	}
	result, err := h.service.ListMyFavoriteActivities(
		c.Request.Context(), userID, queryInt(c, "page", 1), queryInt(c, "page_size", 20),
	)
	writeResult(c, result, err)
}

// join 报名参加活动（登录用户）：生成待审核记录。
func (h *Handler) join(c *gin.Context) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req struct {
		Count       int    `json:"count"`
		Remark      string `json:"remark"`
		ContactInfo string `json:"contact_info"`
		ContactText string `json:"contactInfo"`
		Contact     string `json:"contact"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, 400001, "请求参数格式不正确")
		return
	}
	contactInfo := req.ContactInfo
	if contactInfo == "" {
		contactInfo = req.ContactText
	}
	if contactInfo == "" {
		contactInfo = req.Contact
	}
	err := h.service.JoinActivity(c.Request.Context(), userID, id, req.Count, req.Remark, contactInfo)
	writeResult(c, gin.H{"joined": true}, err)
}

// cancelParticipation 参加人在活动开始前取消参加。
func (h *Handler) cancelParticipation(c *gin.Context) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	err := h.service.CancelParticipation(c.Request.Context(), userID, id)
	writeResult(c, gin.H{"cancelled": true}, err)
}

// cancelOwnedActivity 发起人在活动开始前取消自己发布的活动。
func (h *Handler) cancelOwnedActivity(c *gin.Context) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	result, err := h.service.CancelOwnedActivity(c.Request.Context(), userID, id)
	writeResult(c, result, err)
}

// takeDownOwnedActivity 发起人在活动开始前下架自己发布的活动。
func (h *Handler) takeDownOwnedActivity(c *gin.Context) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	result, err := h.service.TakeDownOwnedActivity(c.Request.Context(), userID, id)
	writeResult(c, result, err)
}

// deleteOwnedActivity 发起人删除自己发布的活动。
func (h *Handler) deleteOwnedActivity(c *gin.Context) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	err := h.service.DeleteOwnedActivity(c.Request.Context(), userID, id)
	writeResult(c, gin.H{"id": id}, err)
}

// myParticipation 当前用户对某活动的报名状态（详情页按钮用）；未报名返回 data:null。
func (h *Handler) myParticipation(c *gin.Context) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	result, err := h.service.MyParticipation(c.Request.Context(), userID, id)
	writeResult(c, result, err)
}

// applicants 发起人查看某活动的待审核报名（joined）。
func (h *Handler) applicants(c *gin.Context) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	result, err := h.service.ListApplicants(c.Request.Context(), userID, id)
	writeResult(c, result, err)
}

// reviewApplicant 发起人通过/拒绝某报名。
func (h *Handler) reviewApplicant(c *gin.Context) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	uid, ok := parseID(c, "uid")
	if !ok {
		return
	}
	var req struct {
		Action       string `json:"action"`
		RejectReason string `json:"reject_reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, 400001, "请求参数格式不正确")
		return
	}
	if req.Action != "approve" && req.Action != "reject" {
		httpx.Fail(c, http.StatusBadRequest, 400004, "审核动作不合法")
		return
	}
	err := h.service.ReviewApplicant(c.Request.Context(), userID, id, uid, req.Action == "approve", req.RejectReason)
	writeResult(c, gin.H{"ok": true}, err)
}

// myJoined 我参加的活动（joined+approved）。
func (h *Handler) myJoined(c *gin.Context) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
		return
	}
	result, err := h.service.ListMyJoinedActivities(c.Request.Context(), userID)
	writeResult(c, result, err)
}

// userJoined Ta 参加的活动（公开，仅已通过 approved）。
func (h *Handler) userJoined(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	result, err := h.service.ListUserJoinedActivities(c.Request.Context(), id)
	writeResult(c, result, err)
}

func (r createRequest) toInput() CreateInput {
	return CreateInput{
		Title:             r.Title,
		CategoryID:        r.CategoryID,
		CategoryLabel:     r.CategoryLabel,
		ActivityDate:      parseOptionalTime(r.ActivityDate),
		ActivityTime:      r.ActivityTime,
		DeadlineAt:        parseOptionalTime(r.DeadlineAt),
		LocationText:      r.LocationText,
		City:              r.City,
		Latitude:          r.Latitude,
		Longitude:         r.Longitude,
		TotalCount:        r.TotalCount,
		CostType:          r.CostType,
		FeeDetail:         r.FeeDetail,
		GenderRequirement: r.GenderRequirement,
		Intro:             r.Intro,
		Notice:            r.Notice,
		ImageURLs:         r.ImageURLs,
	}
}

func parseOptionalTime(value string) *time.Time {
	if value == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return &parsed
		}
	}
	return nil
}

// parseID 解析路径参数为正整数 id；非法时直接写 400 并返回 false。
func parseID(c *gin.Context, param string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(param), 10, 64)
	if err != nil || id <= 0 {
		httpx.Fail(c, http.StatusBadRequest, 400001, "ID 格式不正确")
		return 0, false
	}
	return id, true
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

func (h *Handler) optionalUserID(c *gin.Context) int64 {
	header := c.GetHeader("Authorization")
	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if token == "" || token == header {
		return 0
	}
	claims, err := h.tokenManager.Parse(token, auth.TokenTypeAccess)
	if err != nil {
		return 0
	}
	return claims.UserID
}

func queryFloat(c *gin.Context, key string) (float64, bool) {
	value := c.Query(key)
	if value == "" {
		return 0, false
	}
	result, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	if math.IsNaN(result) || math.IsInf(result, 0) {
		return 0, false
	}
	return result, true
}

func validCoordinate(latitude, longitude float64) bool {
	return latitude >= -90 && latitude <= 90 && longitude >= -180 && longitude <= 180
}

func writeResult(c *gin.Context, data interface{}, err error) {
	if err == nil {
		httpx.OK(c, data)
		return
	}

	switch {
	case errors.Is(err, ErrNotFound),
		errors.Is(err, ErrParticipantNotFound):
		httpx.Fail(c, http.StatusNotFound, 404001, err.Error())
	case errors.Is(err, ErrNotOrganizer):
		httpx.Fail(c, http.StatusForbidden, 403001, err.Error())
	case errors.Is(err, contentmoderation.ErrRejected):
		httpx.Fail(c, http.StatusUnprocessableEntity, 422001, err.Error())
	case errors.Is(err, contentmoderation.ErrUnavailable):
		httpx.Fail(c, http.StatusServiceUnavailable, 503001, contentmoderation.ErrUnavailable.Error())
	case errors.Is(err, ErrInvalidTitle),
		errors.Is(err, ErrInvalidCategory),
		errors.Is(err, ErrInvalidLocation),
		errors.Is(err, ErrInvalidCity),
		errors.Is(err, ErrInvalidIntro),
		errors.Is(err, ErrInvalidCount),
		errors.Is(err, ErrAlreadyJoined),
		errors.Is(err, ErrJoinOwnActivity),
		errors.Is(err, ErrActivityFull),
		errors.Is(err, ErrActivityNotJoinable),
		errors.Is(err, ErrRejectReasonMissing),
		errors.Is(err, ErrInvalidContactInfo),
		errors.Is(err, ErrActivityStarted):
		httpx.Fail(c, http.StatusBadRequest, 400004, err.Error())
	default:
		httpx.Fail(c, http.StatusInternalServerError, 500001, err.Error())
	}
}
