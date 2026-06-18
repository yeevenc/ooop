package user

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("数据不存在")

type UserRepository interface {
	FindByID(ctx context.Context, id int64) (User, error)
	FindByPhone(ctx context.Context, phone string) (User, error)
	FindByUsernameOrPhone(ctx context.Context, account string) (User, error)
	List(ctx context.Context, query UserListQuery) ([]User, int64, error)
	Create(ctx context.Context, item *User) error
	UpdatePassword(ctx context.Context, id int64, username string, passwordHash string) error
	TouchLastLogin(ctx context.Context, id int64, loginAt time.Time, meta ClientMeta) error
}

type LoginCodeRepository interface {
	Create(ctx context.Context, item *LoginCode) error
	FindValid(ctx context.Context, phone string, scene string, codeHash string, now time.Time) (LoginCode, error)
	MarkUsed(ctx context.Context, id int64, usedAt time.Time) error
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, item *RefreshToken) error
	FindValid(ctx context.Context, tokenHash string, now time.Time) (RefreshToken, error)
	Revoke(ctx context.Context, tokenHash string, revokedAt time.Time) error
}

type GormUserRepository struct {
	db *gorm.DB
}

func NewGormUserRepository(db *gorm.DB) *GormUserRepository {
	return &GormUserRepository{db: db}
}

func (r *GormUserRepository) FindByID(ctx context.Context, id int64) (User, error) {
	var item User
	err := r.db.WithContext(ctx).First(&item, id).Error
	return item, normalizeNotFound(err)
}

func (r *GormUserRepository) FindByPhone(ctx context.Context, phone string) (User, error) {
	var item User
	err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&item).Error
	return item, normalizeNotFound(err)
}

func (r *GormUserRepository) FindByUsernameOrPhone(ctx context.Context, account string) (User, error) {
	var item User
	err := r.db.WithContext(ctx).
		Where("phone = ? OR username = ?", account, account).
		First(&item).Error
	return item, normalizeNotFound(err)
}

func (r *GormUserRepository) List(ctx context.Context, query UserListQuery) ([]User, int64, error) {
	var items []User
	var total int64
	db := r.db.WithContext(ctx).Model(&User{})
	db = db.Where("username IS NULL OR username <> ?", ReservedAdminUsername)

	if query.Keyword != "" {
		keyword := "%" + query.Keyword + "%"
		db = db.Where("phone LIKE ? OR username LIKE ? OR nickname LIKE ?", keyword, keyword, keyword)
	}
	if query.Status != nil {
		db = db.Where("status = ?", *query.Status)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Order("id DESC").
		Offset((query.Page - 1) * query.PageSize).
		Limit(query.PageSize).
		Find(&items).Error
	return items, total, err
}

func (r *GormUserRepository) Create(ctx context.Context, item *User) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *GormUserRepository) UpdatePassword(ctx context.Context, id int64, username string, passwordHash string) error {
	updates := map[string]interface{}{
		"password_hash": passwordHash,
	}
	if username != "" {
		updates["username"] = &username
	}
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", id).Updates(updates).Error
}

func (r *GormUserRepository) TouchLastLogin(ctx context.Context, id int64, loginAt time.Time, meta ClientMeta) error {
	updates := map[string]interface{}{
		"last_login_at": loginAt,
	}
	if meta.Platform != "" {
		updates["platform"] = meta.Platform
	}
	if meta.DeviceNo != "" {
		updates["device_no"] = meta.DeviceNo
	}
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", id).Updates(updates).Error
}

type GormLoginCodeRepository struct {
	db *gorm.DB
}

func NewGormLoginCodeRepository(db *gorm.DB) *GormLoginCodeRepository {
	return &GormLoginCodeRepository{db: db}
}

func (r *GormLoginCodeRepository) Create(ctx context.Context, item *LoginCode) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *GormLoginCodeRepository) FindValid(ctx context.Context, phone string, scene string, codeHash string, now time.Time) (LoginCode, error) {
	var item LoginCode
	err := r.db.WithContext(ctx).
		Where("phone = ? AND scene = ? AND code_hash = ? AND used_at IS NULL AND expires_at > ?", phone, scene, codeHash, now).
		Order("id DESC").
		First(&item).Error
	return item, normalizeNotFound(err)
}

func (r *GormLoginCodeRepository) MarkUsed(ctx context.Context, id int64, usedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&LoginCode{}).Where("id = ?", id).Update("used_at", usedAt).Error
}

type GormRefreshTokenRepository struct {
	db *gorm.DB
}

func NewGormRefreshTokenRepository(db *gorm.DB) *GormRefreshTokenRepository {
	return &GormRefreshTokenRepository{db: db}
}

func (r *GormRefreshTokenRepository) Create(ctx context.Context, item *RefreshToken) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *GormRefreshTokenRepository) FindValid(ctx context.Context, tokenHash string, now time.Time) (RefreshToken, error) {
	var item RefreshToken
	err := r.db.WithContext(ctx).
		Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", tokenHash, now).
		First(&item).Error
	return item, normalizeNotFound(err)
}

func (r *GormRefreshTokenRepository) Revoke(ctx context.Context, tokenHash string, revokedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&RefreshToken{}).Where("token_hash = ?", tokenHash).Update("revoked_at", revokedAt).Error
}

func normalizeNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}
