package upload

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ooop-admin-api/internal/auth"
	"ooop-admin-api/internal/httpx"
)

const (
	imageFieldName = "file"
	imageMaxSize   = 5 << 20
	imageURLPrefix = "/uploads/images"
	imageSaveDir   = "uploads/images"
)

var (
	ErrFileRequired      = errors.New("请选择上传图片")
	ErrFileTooLarge      = errors.New("图片大小不能超过 5MB")
	ErrUnsupportedFormat = errors.New("仅支持 jpg、jpeg、png、webp 图片")
)

type Handler struct {
	tokenManager *auth.TokenManager
}

type ImageResult struct {
	URL  string `json:"url"`
	Path string `json:"path"`
}

func NewHandler(tokenManager *auth.TokenManager) *Handler {
	return &Handler{tokenManager: tokenManager}
}

func (h *Handler) Register(api *gin.RouterGroup) {
	group := api.Group("/upload")
	group.Use(auth.Middleware(h.tokenManager))
	group.POST("/image", h.image)
}

func (h *Handler) image(c *gin.Context) {
	file, err := c.FormFile(imageFieldName)
	if err != nil {
		writeError(c, ErrFileRequired)
		return
	}

	if err := validateImage(file); err != nil {
		writeError(c, err)
		return
	}

	if err := os.MkdirAll(imageSaveDir, 0755); err != nil {
		httpx.Fail(c, http.StatusInternalServerError, 500001, err.Error())
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	savePath := filepath.Join(imageSaveDir, filename)
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		httpx.Fail(c, http.StatusInternalServerError, 500001, err.Error())
		return
	}

	publicPath := imageURLPrefix + "/" + filename
	httpx.OK(c, ImageResult{
		URL:  absoluteURL(c, publicPath),
		Path: publicPath,
	})
}

func validateImage(file *multipart.FileHeader) error {
	if file == nil {
		return ErrFileRequired
	}
	if file.Size <= 0 {
		return ErrFileRequired
	}
	if file.Size > imageMaxSize {
		return ErrFileTooLarge
	}

	switch strings.ToLower(filepath.Ext(file.Filename)) {
	case ".jpg", ".jpeg", ".png", ".webp":
		return nil
	default:
		return ErrUnsupportedFormat
	}
}

func absoluteURL(c *gin.Context, path string) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if forwardedProto := c.GetHeader("X-Forwarded-Proto"); forwardedProto != "" {
		scheme = forwardedProto
	}

	return scheme + "://" + c.Request.Host + path
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrFileRequired),
		errors.Is(err, ErrFileTooLarge),
		errors.Is(err, ErrUnsupportedFormat):
		httpx.Fail(c, http.StatusBadRequest, 400003, err.Error())
	default:
		httpx.Fail(c, http.StatusInternalServerError, 500001, err.Error())
	}
}
