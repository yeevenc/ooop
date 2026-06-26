package feedback

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"ooop-admin-api/internal/user"
)

var (
	ErrInvalidType    = errors.New("请选择正确的问题类型")
	ErrInvalidContent = errors.New("请填写 10 到 1000 字的问题描述")
	ErrTooManyImages  = errors.New("最多上传 6 张截图")
)

type UserProvider interface {
	Profile(ctx context.Context, userID int64) (user.PublicUser, error)
}

type CreateInput struct {
	Type           string   `json:"type"`
	Content        string   `json:"content"`
	ImageURLs      []string `json:"imageUrls"`
	DevicePlatform string   `json:"devicePlatform"`
	DeviceVersion  string   `json:"deviceVersion"`
	AppVersion     string   `json:"appVersion"`
}

type ListResult struct {
	List     []PublicFeedback `json:"list"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

type Service struct {
	feedbacks Repository
	users     UserProvider
}

func NewService(feedbacks Repository, users UserProvider) *Service {
	return &Service{
		feedbacks: feedbacks,
		users:     users,
	}
}

func (s *Service) Create(ctx context.Context, userID int64, input CreateInput) (PublicFeedback, error) {
	normalized, err := normalizeCreateInput(input)
	if err != nil {
		return PublicFeedback{}, err
	}

	profile, err := s.users.Profile(ctx, userID)
	if err != nil {
		return PublicFeedback{}, err
	}

	imageData, err := json.Marshal(normalized.ImageURLs)
	if err != nil {
		return PublicFeedback{}, err
	}

	item := Feedback{
		UserID:         userID,
		UserPhone:      profile.Phone,
		UserNickname:   profile.Nickname,
		Type:           normalized.Type,
		Content:        normalized.Content,
		ImageURLs:      string(imageData),
		DevicePlatform: normalized.DevicePlatform,
		DeviceVersion:  normalized.DeviceVersion,
		AppVersion:     normalized.AppVersion,
	}
	if err := s.feedbacks.Create(ctx, &item); err != nil {
		return PublicFeedback{}, err
	}

	return toPublicFeedback(item), nil
}

func (s *Service) List(ctx context.Context, query ListQuery) (ListResult, error) {
	query.Type = strings.TrimSpace(query.Type)
	query.Keyword = strings.TrimSpace(query.Keyword)
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 10
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}
	if query.Type != "" && !isValidType(query.Type) {
		return ListResult{}, ErrInvalidType
	}

	items, total, err := s.feedbacks.List(ctx, query)
	if err != nil {
		return ListResult{}, err
	}

	list := make([]PublicFeedback, 0, len(items))
	for _, item := range items {
		list = append(list, toPublicFeedback(item))
	}

	return ListResult{
		List:     list,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

func normalizeCreateInput(input CreateInput) (CreateInput, error) {
	input.Type = strings.TrimSpace(input.Type)
	input.Content = strings.TrimSpace(input.Content)
	input.DevicePlatform = trimMax(input.DevicePlatform, 32)
	input.DeviceVersion = trimMax(input.DeviceVersion, 64)
	input.AppVersion = trimMax(input.AppVersion, 32)

	if !isValidType(input.Type) {
		return CreateInput{}, ErrInvalidType
	}
	contentLength := utf8.RuneCountInString(input.Content)
	if contentLength < 10 || contentLength > 1000 {
		return CreateInput{}, ErrInvalidContent
	}
	if len(input.ImageURLs) > 6 {
		return CreateInput{}, ErrTooManyImages
	}

	imageURLs := make([]string, 0, len(input.ImageURLs))
	for _, item := range input.ImageURLs {
		url := trimMax(item, 500)
		if url != "" {
			imageURLs = append(imageURLs, url)
		}
	}
	input.ImageURLs = imageURLs
	return input, nil
}

func isValidType(value string) bool {
	switch value {
	case TypeProduct, TypeAccount, TypeActivity:
		return true
	default:
		return false
	}
}

func trimMax(value string, max int) string {
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) <= max {
		return value
	}

	runes := []rune(value)
	return string(runes[:max])
}
