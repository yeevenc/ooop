package activity

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"ooop-admin-api/internal/user"
)

// 登记按用户与城市去重，地图坐标记录偏好位置，通知按城市和类别匹配。
type ActivityIntent struct {
	ID            int64     `gorm:"primaryKey;autoIncrement" json:"id,string"`
	UserID        int64     `gorm:"not null;uniqueIndex:uniq_intent_user_city" json:"userId,string"`
	City          string    `gorm:"size:64;not null;uniqueIndex:uniq_intent_user_city;index" json:"city"`
	Latitude      float64   `gorm:"not null" json:"latitude"`
	Longitude     float64   `gorm:"not null" json:"longitude"`
	LocationText  string    `gorm:"size:255;not null" json:"locationText"`
	Note          string    `gorm:"size:200;not null" json:"note"`
	TimeSlotsJSON string    `gorm:"type:text" json:"-"`
	NotifyEnabled bool      `gorm:"not null;default:false" json:"notifyEnabled"`
	Active        bool      `gorm:"not null;default:true;index" json:"active"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}
type ActivityIntentCategory struct {
	IntentID   int64 `gorm:"primaryKey"`
	CategoryID int64 `gorm:"primaryKey;index"`
	CreatedAt  time.Time
}
type IntentInput struct {
	City          string   `json:"city"`
	Latitude      *float64 `json:"latitude"`
	Longitude     *float64 `json:"longitude"`
	LocationText  string   `json:"locationText"`
	CategoryIDs   []int64  `json:"categoryIds"`
	TimeSlots     []string `json:"timeSlots"`
	Note          string   `json:"note"`
	NotifyEnabled bool     `json:"notifyEnabled"`
}
type PublicIntent struct {
	ActivityIntent
	CategoryIDs []int64  `json:"categoryIds"`
	TimeSlots   []string `json:"timeSlots"`
}

var ErrInvalidIntent = errors.New("请选择有效城市、地图位置、类别和时段，补充说明最多200字")
var ErrIntentUnavailable = errors.New("意向登记服务暂不可用")

type IntentStore struct{ db *gorm.DB }

func NewIntentStore(db *gorm.DB) *IntentStore    { return &IntentStore{db: db} }
func (s *Service) SetIntents(store *IntentStore) { s.intents = store }
func normalizeIntent(input IntentInput) (IntentInput, error) {
	input.City = normalizeCity(input.City)
	input.LocationText = strings.TrimSpace(input.LocationText)
	input.Note = strings.TrimSpace(input.Note)
	if input.City == "" || utf8.RuneCountInString(input.City) > 30 || input.LocationText == "" || utf8.RuneCountInString(input.LocationText) > 255 || utf8.RuneCountInString(input.Note) > 200 || input.Latitude == nil || input.Longitude == nil || math.IsNaN(*input.Latitude) || math.IsNaN(*input.Longitude) || math.IsInf(*input.Latitude, 0) || math.IsInf(*input.Longitude, 0) || *input.Latitude < -90 || *input.Latitude > 90 || *input.Longitude < -180 || *input.Longitude > 180 || len(input.CategoryIDs) == 0 || len(input.CategoryIDs) > 20 || len(input.TimeSlots) == 0 || len(input.TimeSlots) > 5 {
		return input, ErrInvalidIntent
	}
	ids := []int64{}
	seen := map[int64]bool{}
	for _, id := range input.CategoryIDs {
		if id <= 0 {
			return input, ErrInvalidIntent
		}
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	input.CategoryIDs = ids
	times := []string{}
	slots := map[string]bool{}
	for _, slot := range input.TimeSlots {
		switch slot {
		case "工作日白天", "工作日晚上", "周末白天", "周末晚上", "时间灵活":
		default:
			return input, ErrInvalidIntent
		}
		if !slots[slot] {
			times = append(times, slot)
			slots[slot] = true
		}
	}
	input.TimeSlots = times
	return input, nil
}
func (s *Service) SaveIntent(ctx context.Context, uid int64, input IntentInput) (PublicIntent, error) {
	if s.intents == nil {
		return PublicIntent{}, ErrIntentUnavailable
	}
	input, err := normalizeIntent(input)
	if err != nil {
		return PublicIntent{}, err
	}
	if err = s.checkActivityContent(ctx, input.City, input.LocationText, input.Note); err != nil {
		return PublicIntent{}, err
	}
	for _, id := range input.CategoryIDs {
		if _, err = s.activities.FindCategory(ctx, id); err != nil {
			return PublicIntent{}, ErrInvalidCategory
		}
	}
	var item ActivityIntent
	err = s.intents.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var owner user.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&owner, uid).Error; err != nil {
			return err
		}
		times, _ := json.Marshal(input.TimeSlots)
		item = ActivityIntent{UserID: uid, City: input.City, Latitude: *input.Latitude, Longitude: *input.Longitude, LocationText: input.LocationText, Note: input.Note, TimeSlotsJSON: string(times), NotifyEnabled: input.NotifyEnabled, Active: true}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}, {Name: "city"}}, DoUpdates: clause.AssignmentColumns([]string{"latitude", "longitude", "location_text", "note", "time_slots_json", "notify_enabled", "active", "updated_at"})}).Create(&item).Error; err != nil {
			return err
		}
		item.ID = 0
		if err := tx.Where("user_id = ? AND city = ?", uid, input.City).First(&item).Error; err != nil {
			return err
		}
		// 仅新增类别重置登记时间，避免编辑说明后重新接收旧活动通知。
		if err := tx.Where("intent_id = ? AND category_id NOT IN ?", item.ID, input.CategoryIDs).Delete(&ActivityIntentCategory{}).Error; err != nil {
			return err
		}
		for _, id := range input.CategoryIDs {
			row := ActivityIntentCategory{IntentID: item.ID, CategoryID: id}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return PublicIntent{ActivityIntent: item, CategoryIDs: input.CategoryIDs, TimeSlots: input.TimeSlots}, err
}
func (s *Service) ListIntents(ctx context.Context, uid int64, city string, page int) ([]PublicIntent, error) {
	if s.intents == nil {
		return nil, ErrIntentUnavailable
	}
	if page < 1 {
		page = 1
	}
	db := s.intents.db.WithContext(ctx).Where("active = ?", true)
	if uid > 0 {
		db = db.Where("user_id = ?", uid)
	}
	if city != "" {
		db = db.Where("city = ?", normalizeCity(city))
	}
	var items []ActivityIntent
	if err := db.Order("updated_at DESC, id DESC").Limit(50).Offset((page - 1) * 50).Find(&items).Error; err != nil {
		return nil, err
	}
	out := make([]PublicIntent, 0, len(items))
	for _, item := range items {
		ids := []int64{}
		slots := []string{}
		if err := s.intents.db.WithContext(ctx).Model(&ActivityIntentCategory{}).Where("intent_id = ?", item.ID).Order("category_id").Pluck("category_id", &ids).Error; err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(item.TimeSlotsJSON), &slots)
		out = append(out, PublicIntent{ActivityIntent: item, CategoryIDs: ids, TimeSlots: slots})
	}
	return out, nil
}
func (s *Service) CancelIntent(ctx context.Context, uid, id int64) error {
	if s.intents == nil {
		return ErrIntentUnavailable
	}
	return s.intents.db.WithContext(ctx).Model(&ActivityIntent{}).Where("id = ? AND user_id = ?", id, uid).Update("active", false).Error
}
