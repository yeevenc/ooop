package user

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"ooop-admin-api/internal/auth"
	"ooop-admin-api/internal/httpx"
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
	authGroup.POST("/register", h.register)
	authGroup.POST("/send-code", h.sendCode)
	authGroup.POST("/mobile-code-login", h.mobileCodeLogin)
	authGroup.POST("/password-login", h.passwordLogin)
	authGroup.POST("/refresh-token", h.refreshToken)
	authGroup.POST("/set-password", auth.Middleware(h.tokenManager), h.setPassword)

	userGroup := api.Group("/user")
	userGroup.Use(auth.Middleware(h.tokenManager))
	userGroup.GET("/profile", h.profile)
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

func (h *Handler) sendCode(c *gin.Context) {
	var req struct {
		Phone string `json:"phone"`
	}
	if !bindJSON(c, &req) {
		return
	}
	err := h.service.SendLoginCode(c.Request.Context(), req.Phone)
	writeServiceResult(c, gin.H{"sent": true}, err)
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

func (h *Handler) refreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.service.RefreshToken(c.Request.Context(), req.RefreshToken)
	writeServiceResult(c, result, err)
}

func (h *Handler) setPassword(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !bindJSON(c, &req) {
		return
	}
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, 401001, "请先登录")
		return
	}
	result, err := h.service.SetPassword(c.Request.Context(), userID, req.Username, req.Password)
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
		errors.Is(err, ErrInvalidCode),
		errors.Is(err, ErrPhoneExists),
		errors.Is(err, ErrReservedUsername):
		httpx.Fail(c, http.StatusBadRequest, 400002, err.Error())
	case errors.Is(err, ErrInvalidAccount):
		httpx.Fail(c, http.StatusUnauthorized, 401003, err.Error())
	case errors.Is(err, ErrDisabledUser):
		httpx.Fail(c, http.StatusForbidden, 403001, err.Error())
	case errors.Is(err, auth.ErrInvalidToken),
		errors.Is(err, auth.ErrExpiredToken):
		httpx.Fail(c, http.StatusUnauthorized, 401002, err.Error())
	default:
		httpx.Fail(c, http.StatusInternalServerError, 500001, err.Error())
	}
}
