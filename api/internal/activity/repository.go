package activity

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ListQuery struct {
	City        string
	CategoryID  string
	Keyword     string
	Latitude    float64
	Longitude   float64
	HasLocation bool
	Page        int
	PageSize    int
}

type UserActivityQuery struct {
	UserID   int64
	Page     int
	PageSize int
	// Statuses 指定要查询的活动状态；为空时默认仅 ongoing（对外可见）。
	Statuses []string
}

// AdminActivityQuery 后台活动列表查询：状态/分类/标题关键词/分页，不强制只看 ongoing。
type AdminActivityQuery struct {
	Keyword    string
	Status     string
	CategoryID string
	Page       int
	PageSize   int
}

type Repository interface {
	Create(ctx context.Context, item *Activity) error
	List(ctx context.Context, query ListQuery) ([]Activity, error)
	ListByUser(ctx context.Context, query UserActivityQuery) ([]Activity, error)
	FindByID(ctx context.Context, id int64) (Activity, error)
	Save(ctx context.Context, item *Activity) error
	Delete(ctx context.Context, id int64) error
	UpdateStatus(ctx context.Context, id int64, status string) error
	AdjustCurrentCount(ctx context.Context, id int64, delta int) error
	FindActivitiesByIDs(ctx context.Context, ids []int64) ([]Activity, error)
	FindFavorite(ctx context.Context, userID, activityID int64) (ActivityFavorite, error)
	CreateFavorite(ctx context.Context, item *ActivityFavorite) error
	DeleteFavorite(ctx context.Context, userID, activityID int64) error
	ListFavoritesByUser(ctx context.Context, userID int64, page, pageSize int) ([]ActivityFavorite, error)
	// 参加(报名)关联：参加人↔活动 = (activity_id, user_id)
	FindParticipant(ctx context.Context, activityID, userID int64) (ActivityParticipant, error)
	CreateParticipant(ctx context.Context, item *ActivityParticipant) error
	SaveParticipant(ctx context.Context, item *ActivityParticipant) error
	ListParticipantsByActivity(ctx context.Context, activityID int64, statuses []string, limit int) ([]ActivityParticipant, error)
	ListParticipantsByUser(ctx context.Context, userID int64, statuses []string) ([]ActivityParticipant, error)
	CountByActivityIDsAndStatus(ctx context.Context, ids []int64, status string) (map[int64]int, error)
	AdminList(ctx context.Context, query AdminActivityQuery) ([]Activity, int64, error)
	FindCategory(ctx context.Context, id string) (ActivityCategory, error)
	ListCategories(ctx context.Context) ([]ActivityCategory, error)
	SaveCategories(ctx context.Context, items []ActivityCategory) error
	AdminListCategories(ctx context.Context) ([]ActivityCategory, error)
	FindCategoryByID(ctx context.Context, id string) (ActivityCategory, error)
	CreateCategory(ctx context.Context, item *ActivityCategory) error
	UpdateCategory(ctx context.Context, id string, fields map[string]interface{}) error
	DeleteCategory(ctx context.Context, id string) error
}

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) Create(ctx context.Context, item *Activity) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *GormRepository) List(ctx context.Context, query ListQuery) ([]Activity, error) {
	var items []Activity
	db := r.db.WithContext(ctx).
		Model(&Activity{}).
		Where("status = ?", StatusOngoing).
		Where("activity_date IS NULL OR activity_date >= ?", time.Now())

	if query.City != "" {
		db = db.Where("city = ?", query.City)
	}
	if query.CategoryID != "" && query.CategoryID != "all" {
		db = db.Where("category_id = ?", query.CategoryID)
	}
	if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where(
			"title LIKE ? OR intro LIKE ? OR location_text LIKE ? OR category_label LIKE ? OR city LIKE ?",
			like,
			like,
			like,
			like,
			like,
		)
	}

	page := query.Page
	if page <= 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	orderBy := "created_at DESC"
	if query.HasLocation {
		orderBy = fmt.Sprintf(
			"6371000 * ACOS(LEAST(1, GREATEST(-1, COS(RADIANS(%f)) * COS(RADIANS(latitude)) * COS(RADIANS(longitude) - RADIANS(%f)) + SIN(RADIANS(%f)) * SIN(RADIANS(latitude))))) ASC, created_at DESC",
			query.Latitude,
			query.Longitude,
			query.Latitude,
		)
	}

	err := db.Order(orderBy).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items).Error
	return items, err
}

func (r *GormRepository) ListByUser(ctx context.Context, query UserActivityQuery) ([]Activity, error) {
	statuses := query.Statuses
	if len(statuses) == 0 {
		statuses = []string{StatusOngoing}
	}

	var items []Activity
	db := r.db.WithContext(ctx).
		Model(&Activity{}).
		Where("user_id = ? AND status IN ?", query.UserID, statuses)

	err := paginate(db, query.Page, query.PageSize).
		Order("created_at DESC").
		Find(&items).Error
	return items, err
}

func (r *GormRepository) FindCategory(ctx context.Context, id string) (ActivityCategory, error) {
	var item ActivityCategory
	err := r.db.WithContext(ctx).
		Where("id = ? AND status = ?", id, CategoryEnabled).
		First(&item).Error
	return item, err
}

func (r *GormRepository) ListCategories(ctx context.Context) ([]ActivityCategory, error) {
	var items []ActivityCategory
	err := r.db.WithContext(ctx).
		Where("status = ?", CategoryEnabled).
		Order("sort ASC, created_at ASC").
		Find(&items).Error
	return items, err
}

func (r *GormRepository) SaveCategories(ctx context.Context, items []ActivityCategory) error {
	if len(items) == 0 {
		return nil
	}

	// 仅补种缺失的默认分类，不覆盖管理员对已有分类的编辑（名称/图标/排序/状态）。
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoNothing: true,
		}).
		Create(&items).Error
}

func (r *GormRepository) FindByID(ctx context.Context, id int64) (Activity, error) {
	var item Activity
	err := r.db.WithContext(ctx).First(&item, id).Error
	return item, err
}

func (r *GormRepository) Save(ctx context.Context, item *Activity) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *GormRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("activity_id = ?", id).Delete(&ActivityFavorite{}).Error; err != nil {
			return err
		}
		return tx.Delete(&Activity{}, id).Error
	})
}

func (r *GormRepository) FindFavorite(ctx context.Context, userID, activityID int64) (ActivityFavorite, error) {
	var item ActivityFavorite
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND activity_id = ?", userID, activityID).
		First(&item).Error
	return item, err
}

func (r *GormRepository) CreateFavorite(ctx context.Context, item *ActivityFavorite) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "activity_id"}},
			DoNothing: true,
		}).
		Create(item).Error
}

func (r *GormRepository) DeleteFavorite(ctx context.Context, userID, activityID int64) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND activity_id = ?", userID, activityID).
		Delete(&ActivityFavorite{}).Error
}

func (r *GormRepository) ListFavoritesByUser(ctx context.Context, userID int64, page, pageSize int) ([]ActivityFavorite, error) {
	var items []ActivityFavorite
	err := paginate(
		r.db.WithContext(ctx).
			Model(&ActivityFavorite{}).
			Where("user_id = ?", userID),
		page,
		pageSize,
	).
		Order("created_at DESC").
		Find(&items).Error
	return items, err
}

func (r *GormRepository) UpdateStatus(ctx context.Context, id int64, status string) error {
	return r.db.WithContext(ctx).Model(&Activity{}).Where("id = ?", id).Update("status", status).Error
}

func (r *GormRepository) AdminList(ctx context.Context, query AdminActivityQuery) ([]Activity, int64, error) {
	var items []Activity
	var total int64

	db := r.db.WithContext(ctx).Model(&Activity{})
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.CategoryID != "" && query.CategoryID != "all" {
		db = db.Where("category_id = ?", query.CategoryID)
	}
	if query.Keyword != "" {
		db = db.Where("title LIKE ?", "%"+query.Keyword+"%")
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := paginate(db, query.Page, query.PageSize).
		Order("created_at DESC").
		Find(&items).Error
	return items, total, err
}

func (r *GormRepository) AdminListCategories(ctx context.Context) ([]ActivityCategory, error) {
	var items []ActivityCategory
	err := r.db.WithContext(ctx).
		Order("sort ASC, created_at ASC").
		Find(&items).Error
	return items, err
}

func (r *GormRepository) FindCategoryByID(ctx context.Context, id string) (ActivityCategory, error) {
	var item ActivityCategory
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error
	return item, err
}

func (r *GormRepository) CreateCategory(ctx context.Context, item *ActivityCategory) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *GormRepository) UpdateCategory(ctx context.Context, id string, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&ActivityCategory{}).Where("id = ?", id).Updates(fields).Error
}

func (r *GormRepository) DeleteCategory(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&ActivityCategory{}).Error
}

func paginate(db *gorm.DB, page int, pageSize int) *gorm.DB {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	return db.Offset((page - 1) * pageSize).Limit(pageSize)
}
