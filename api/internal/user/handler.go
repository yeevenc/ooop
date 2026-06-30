package user

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ooop-admin-api/internal/auth"
	"ooop-admin-api/internal/httpx"
	"ooop-admin-api/internal/logger"
	"ooop-admin-api/internal/provider"
)

type Handler struct {
	service      *AuthService
	tokenManager *auth.TokenManager
}

func NewHandler(service *AuthService, tokenManager *auth.TokenManager) *Handler {
	return &Handler{service: service, tokenManager: tokenManager}
}

func (h *Handler) Register(api *gin.RouterGroup) {
	authGroup := api.Group("/auth")
	authGroup.POST("/aliyun-mobile-login", h.aliyunMobileLogin)
	authGroup.POST("/jverification-login", h.jverificationLogin)
	authGroup.POST("/register", h.register)
	authGroup.POST("/send-code", h.sendCode)
	authGroup.POST("/check-code", h.checkCode)
	authGroup.POST("/mobile-code-login", h.mobileCodeLogin)
	authGroup.POST("/password-login", h.passwordLogin)
	authGroup.POST("/set-password", auth.Middleware(h.tokenManager), h.setPassword)

	userGroup := api.Group("/user")
	userGroup.Use(auth.Middleware(h.tokenManager))
	userGroup.GET("/profile", h.profile)
	userGroup.PUT("/profile", h.updateProfile)
	userGroup.PUT("/phone", h.changePhone)
	userGroup.PUT("/push-registration", h.bindPushRegistration)
	userGroup.POST("/real-name-verify", h.realNameVerify)

	// 公开的他人用户资料（安全子集），用于用户主页展示。
	api.GET("/users/:id", h.publicProfile)
}

func (h *Handler) aliyunMobileLogin(c *gin.Context) {
	var req struct {
		AccessToken string `json:"access_token"`
		Platform    string `json:"platform"`
		DeviceNo    string `json:"device_no"`
	}
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.service.AliyunMobileLogin(c.Request.Context(), req.AccessToken, clientMeta(req.Platform, req.DeviceNo))
	writeServiceResult(c, result, err)
}

func (h *Handler) jverificationLogin(c *gin.Context) {
	var req struct {
		LoginToken  string `json:"login_token"`
		AccessToken string `json:"access_token"`
		Operator    string `json:"operator"`
		Platform    string `json:"platform"`
		DeviceNo    string `json:"device_no"`
	}
	if !bindJSON(c, &req) {
		return
	}
	loginToken := req.LoginToken
	if loginToken == "" {
		loginToken = req.AccessToken
	}
	result, err := h.service.JiguangMobileLogin(c.Request.Context(), loginToken, req.Operator, clientMeta(req.Platform, req.DeviceNo))
	writeServiceResult(c, result, err)
}

func (h *Handler) sendCode(c *gin.Context) {
	var req struct {
		Phone string `json:"phone"`
		Scene string `json:"scene"`
	}
	if !bindJSON(c, &req) {
		return
	}
	err := h.service.SendLoginCode(c.Request.Context(), req.Phone, provider.SMSScene(req.Scene))
	writeServiceResult(c, gin.H{"sent": true}, err)
}

func (h *Handler) checkCode(c *gin.Context) {
	var req struct {
		Phone string `json:"phone"`
		Scene string `json:"scene"`
		Code  string `json:"code"`
	}
	if !bindJSON(c, &req) {
		return
	}
	err := h.service.CheckSMSCode(c.Request.Context(), req.Phone, provider.SMSScene(req.Scene), req.Code)
	writeServiceResult(c, gin.H{"verified": true}, err)
}

func (h *Handler) mobileCodeLogin(c *gin.Context) {
	var req struct {
		Phone    string `json:"phone"`
		Code     string `json:"code"`
		Platform string `json:"platform"`
		DeviceNo string `json:"device_no"`
	}
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.service.MobileCodeLogin(c.Request.Context(), req.Phone, req.Code, clientMeta(req.Platform, req.DeviceNo))
	writeServiceResult(c, result, err)
}

func (h *Handler) register(c *gin.Context) {
	var req struct {
		Phone    string `json:"phone"`
		Username string `json:"username"`
		Password string `json:"password"`
		Platform string `json:"platform"`
		DeviceNo string `json:"device_no"`
	}
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.service.RegisterByPassword(c.Request.Context(), req.Phone, req.Username, req.Password, clientMeta(req.Platform, req.DeviceNo))
	writeServiceResult(c, result, err)
}

func (h *Handler) passwordLogin(c *gin.Context) {
	var req struct {
		Account  string `json:"account"`
		Password string `json:"password"`
		Platform string `json:"platform"`
		DeviceNo string `json:"device_no"`
	}
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.service.PasswordLogin(c.Request.Context(), req.Account, req.Password, clientMeta(req.Platform, req.DeviceNo))
	writeServiceResult(c, result, err)
}

func (h *Handler) setPassword(c *gin.Context) {
	var req struct {
		Username    string `json:"username"`
		OldPassword string `json:"old_password"`
		Password    string `json:"password"`
	}
	if !bindJSON(c, &req) {
		return
	}
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
		return
	}
	result, err := h.service.SetPassword(c.Request.Context(), userID, req.Username, req.OldPassword, req.Password)
	writeServiceResult(c, result, err)
}

func (h *Handler) profile(c *gin.Context) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
		return
	}
	result, err := h.service.Profile(c.Request.Context(), userID)
	writeServiceResult(c, result, err)
}

// publicProfile 公开的他人用户资料（安全子集），用于用户主页展示。
func (h *Handler) publicProfile(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Fail(c, http.StatusBadRequest, 400001, "用户 ID 格式不正确")
		return
	}
	result, err := h.service.PublicProfile(c.Request.Context(), id)
	writeServiceResult(c, result, err)
}

func (h *Handler) updateProfile(c *gin.Context) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
		return
	}
	var req ProfileUpdateInput
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.service.UpdateProfile(c.Request.Context(), userID, req.ToProfileUpdate())
	writeServiceResult(c, result, err)
}

func (h *Handler) changePhone(c *gin.Context) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
		return
	}
	var req struct {
		NewPhone string `json:"new_phone"`
		Code     string `json:"code"`
	}
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.service.ChangePhone(c.Request.Context(), userID, req.NewPhone, req.Code)
	writeServiceResult(c, result, err)
}

func (h *Handler) bindPushRegistration(c *gin.Context) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
		return
	}
	var req struct {
		Platform       string `json:"platform"`
		RegistrationID string `json:"registration_id"`
	}
	if !bindJSON(c, &req) {
		return
	}
	logger.Infof(
		"收到 push registration 绑定请求: user_id=%d, platform=%s, registration_id=%s",
		userID,
		req.Platform,
		req.RegistrationID,
	)
	err := h.service.BindPushRegistration(c.Request.Context(), userID, req.Platform, req.RegistrationID)
	writeServiceResult(c, gin.H{"bound": true}, err)
}

func (h *Handler) realNameVerify(c *gin.Context) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
		return
	}
	var req struct {
		Name   string `json:"name"`
		IDCard string `json:"id_card"`
	}
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.service.VerifyRealName(c.Request.Context(), userID, req.Name, req.IDCard)
	writeServiceResult(c, result, err)
}

func bindJSON(c *gin.Context, target interface{}) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		httpx.Fail(c, http.StatusBadRequest, 400001, "请求参数格式不正确")
		return false
	}
	return true
}

func clientMeta(platform string, deviceNo string) ClientMeta {
	return ClientMeta{
		Platform: platform,
		DeviceNo: deviceNo,
	}
}

func writeServiceResult(c *gin.Context, data interface{}, err error) {
	if err == nil {
		httpx.OK(c, data)
		return
	}

	switch {
	case errors.Is(err, ErrInvalidPhone),
		errors.Is(err, ErrInvalidPassword),
		errors.Is(err, ErrInvalidOldPass),
		errors.Is(err, ErrInvalidCode),
		errors.Is(err, ErrPhoneExists),
		errors.Is(err, ErrReservedUsername),
		errors.Is(err, ErrInvalidProfile),
		errors.Is(err, ErrInvalidRealName),
		errors.Is(err, ErrRealNameMismatch):
		httpx.Fail(c, http.StatusBadRequest, 400002, err.Error())
	case errors.Is(err, ErrInvalidAccount):
		httpx.Fail(c, http.StatusUnauthorized, 401003, err.Error())
	case errors.Is(err, ErrDisabledUser):
		httpx.Fail(c, http.StatusForbidden, 403001, err.Error())
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, http.StatusNotFound, 404001, "用户不存在")
	case errors.Is(err, auth.ErrInvalidToken),
		errors.Is(err, auth.ErrExpiredToken):
		httpx.Fail(c, http.StatusUnauthorized, 401002, err.Error())
	default:
		httpx.Fail(c, http.StatusInternalServerError, 500001, err.Error())
	}
}
