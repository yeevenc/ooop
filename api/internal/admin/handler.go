package admin

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ooop-admin-api/internal/activity"
	"ooop-admin-api/internal/auth"
	"ooop-admin-api/internal/chat"
	"ooop-admin-api/internal/contentmoderation"
	"ooop-admin-api/internal/httpx"
	"ooop-admin-api/internal/user"
)

type Handler struct {
	service      *Service
	appUsers     *user.AuthService
	activities   *activity.Service
	chatReports  *chat.ReportService
	tokenManager *auth.TokenManager
}

func NewHandler(service *Service, appUsers *user.AuthService, activities *activity.Service, chatReports *chat.ReportService, tokenManager *auth.TokenManager) *Handler {
	return &Handler{
		service:      service,
		appUsers:     appUsers,
		activities:   activities,
		chatReports:  chatReports,
		tokenManager: tokenManager,
	}
}

func (h *Handler) Register(api *gin.RouterGroup) {
	adminGroup := api.Group("/admin")
	adminGroup.POST("/auth/login", h.login)

	protected := adminGroup.Group("")
	protected.Use(Middleware(h.tokenManager))
	protected.GET("/users", h.userList)
	protected.GET("/users/:id", h.userDetail)
	protected.PUT("/users/:id", h.updateUser)
	// APP 用户封禁/解封（仅操作 users 表，与 admin_users 无关）
	// 封禁后 APP 登录与鉴权返回 code=403002
	protected.PUT("/users/:id/ban", h.banUser)
	protected.PUT("/users/:id/unban", h.unbanUser)

	// 活动分类管理
	protected.GET("/activity-categories", h.categoryList)
	protected.POST("/activity-categories", h.categoryCreate)
	protected.PUT("/activity-categories/:id", h.categoryUpdate)
	protected.DELETE("/activity-categories/:id", h.categoryDelete)

	// 活动管理（审核/编辑/上下架/删除）
	protected.GET("/activities", h.activityList)
	protected.GET("/activities/:id", h.activityDetail)
	protected.PUT("/activities/:id", h.activityUpdate)
	protected.DELETE("/activities/:id", h.activityDelete)
	protected.PUT("/activities/:id/review", h.activityReview)
	protected.PUT("/activities/:id/status", h.activityStatus)

	// 聊天举报管理
	protected.GET("/chat-reports", h.chatReportList)
	protected.GET("/chat-reports/:id", h.chatReportDetail)
	protected.PUT("/chat-reports/:id/resolve", h.resolveChatReport)
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
	if err != nil {
		writeResult(c, nil, err)
		return
	}

	// 转换为后台管理格式（时间字段格式化为 YYYY-MM-DD HH:mm:ss）
	adminUsers := make([]AdminUserResponse, 0, len(result.List))
	for _, u := range result.List {
		adminUsers = append(adminUsers, ToAdminUserResponse(u))
	}

	writeResult(c, AdminUserListResult{
		List:     adminUsers,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	}, nil)
}

func (h *Handler) updateUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Fail(c, http.StatusBadRequest, 400001, "用户 ID 格式不正确")
		return
	}

	var req user.ProfileUpdateInput
	if !bindJSON(c, &req) {
		return
	}

	result, err := h.appUsers.UpdateProfile(c.Request.Context(), id, req.ToProfileUpdate())
	if err != nil {
		writeResult(c, nil, err)
		return
	}
	writeResult(c, ToAdminUserResponse(result), nil)
}

// banUser 封禁 APP 用户。
// 备注：body.type=permanent|temporary；限时需 duration_hours（前端按时间区间换算）。
func (h *Handler) banUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Fail(c, http.StatusBadRequest, 400001, "用户 ID 格式不正确")
		return
	}

	var req user.BanUserInput
	if !bindJSON(c, &req) {
		return
	}

	result, err := h.appUsers.BanUser(c.Request.Context(), id, req)
	if err != nil {
		writeResult(c, nil, err)
		return
	}
	writeResult(c, ToAdminUserResponse(result), nil)
}

// unbanUser 解封 APP 用户。
func (h *Handler) unbanUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Fail(c, http.StatusBadRequest, 400001, "用户 ID 格式不正确")
		return
	}

	result, err := h.appUsers.UnbanUser(c.Request.Context(), id)
	if err != nil {
		writeResult(c, nil, err)
		return
	}
	writeResult(c, ToAdminUserResponse(result), nil)
}

func (h *Handler) userDetail(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Fail(c, http.StatusBadRequest, 400001, "用户 ID 格式不正确")
		return
	}

	result, err := h.appUsers.Profile(c.Request.Context(), id)
	if err != nil {
		writeResult(c, nil, err)
		return
	}

	// 转换为后台管理格式（时间字段格式化为 YYYY-MM-DD HH:mm:ss）
	writeResult(c, ToAdminUserResponse(result), nil)
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
	case errors.Is(err, contentmoderation.ErrRejected):
		httpx.Fail(c, http.StatusUnprocessableEntity, 422001, err.Error())
	case errors.Is(err, contentmoderation.ErrUnavailable):
		httpx.Fail(c, http.StatusServiceUnavailable, 503001, contentmoderation.ErrUnavailable.Error())
	case errors.Is(err, user.ErrInvalidProfile), errors.Is(err, user.ErrInvalidBan):
		httpx.Fail(c, http.StatusBadRequest, 400002, err.Error())
	case errors.Is(err, user.ErrNotFound):
		httpx.Fail(c, http.StatusNotFound, 404001, "用户不存在")
	case errors.Is(err, activity.ErrNotFound),
		errors.Is(err, activity.ErrCategoryMissing):
		httpx.Fail(c, http.StatusNotFound, 404001, err.Error())
	case errors.Is(err, activity.ErrInvalidTitle),
		errors.Is(err, activity.ErrInvalidCategory),
		errors.Is(err, activity.ErrInvalidLocation),
		errors.Is(err, activity.ErrInvalidCity),
		errors.Is(err, activity.ErrInvalidIntro),
		errors.Is(err, activity.ErrInvalidCount),
		errors.Is(err, activity.ErrInvalidStatus),
		errors.Is(err, activity.ErrCategoryExists):
		httpx.Fail(c, http.StatusBadRequest, 400004, err.Error())
	case errors.Is(err, chat.ErrReportNotFound):
		httpx.Fail(c, http.StatusNotFound, 404001, err.Error())
	case errors.Is(err, chat.ErrReportStatusInvalid),
		errors.Is(err, chat.ErrReportResultRequired),
		errors.Is(err, chat.ErrReportTooLong),
		errors.Is(err, chat.ErrReportResultTooLong),
		errors.Is(err, chat.ErrReportRestrictionRequired),
		errors.Is(err, chat.ErrReportRestrictionInvalid):
		httpx.Fail(c, http.StatusBadRequest, 400001, err.Error())
	case errors.Is(err, chat.ErrReportProcessed):
		httpx.Fail(c, http.StatusConflict, 409001, err.Error())
	case errors.Is(err, auth.ErrInvalidToken),
		errors.Is(err, auth.ErrExpiredToken):
		httpx.Fail(c, http.StatusUnauthorized, 401002, err.Error())
	default:
		httpx.Fail(c, http.StatusInternalServerError, 500001, err.Error())
	}
}
