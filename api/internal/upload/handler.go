package upload

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/storage"

	"ooop-admin-api/internal/config"
	"ooop-admin-api/internal/httpx"
)

const (
	imageFieldName = "file"
	imageMaxSize   = 5 << 20
	imageKeyPrefix = "images"
)

var (
	ErrFileRequired      = errors.New("请选择上传图片")
	ErrFileTooLarge      = errors.New("图片大小不能超过 5MB")
	ErrUnsupportedFormat = errors.New("仅支持 jpg、jpeg、png、webp 图片")
	ErrQiniuNotReady     = errors.New("七牛云上传配置未完成")
)

type Handler struct {
	cfg      config.QiniuConfig
	uploader *storage.FormUploader
}

type ImageResult struct {
	URL  string `json:"url"`
	Path string `json:"path"`
}

func NewHandler() *Handler {
	return NewHandlerWithConfig(config.QiniuConfig{})
}

func NewHandlerWithConfig(cfg config.QiniuConfig) *Handler {
	return &Handler{
		cfg: cfg,
		uploader: storage.NewFormUploader(&storage.Config{
			UseHTTPS:      true,
			UseCdnDomains: false,
		}),
	}
}

func (h *Handler) Register(api *gin.RouterGroup) {
	group := api.Group("/upload")
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

	ext := strings.ToLower(filepath.Ext(file.Filename))
	key := buildImageKey(ext)
	if err := h.uploadToQiniu(c.Request.Context(), file, key); err != nil {
		httpx.Fail(c, http.StatusInternalServerError, 500001, err.Error())
		return
	}

	publicPath := "/" + key
	httpx.OK(c, ImageResult{
		URL:  h.publicURL(publicPath),
		Path: publicPath,
	})
}

func (h *Handler) uploadToQiniu(ctx context.Context, fileHeader *multipart.FileHeader, key string) error {
	if strings.TrimSpace(h.cfg.AccessKey) == "" ||
		strings.TrimSpace(h.cfg.SecretKey) == "" ||
		strings.TrimSpace(h.cfg.Bucket) == "" ||
		h.uploader == nil {
		return ErrQiniuNotReady
	}

	file, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	putPolicy := storage.PutPolicy{
		Scope: h.cfg.Bucket,
	}
	token := putPolicy.UploadToken(qbox.NewMac(h.cfg.AccessKey, h.cfg.SecretKey))
	ret := storage.PutRet{}
	putExtra := storage.PutExtra{}
	return h.uploader.Put(ctx, &ret, token, key, file, fileHeader.Size, &putExtra)
}

func (h *Handler) publicURL(path string) string {
	domain := strings.TrimRight(strings.TrimSpace(h.cfg.Domain), "/")
	if domain == "" {
		domain = "https://source.ooopai.cn"
	}
	return domain + path
}

func buildImageKey(ext string) string {
	now := time.Now()
	return fmt.Sprintf(
		"%s/%s/%d%s",
		imageKeyPrefix,
		now.Format("2006/01/02"),
		now.UnixNano(),
		ext,
	)
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
