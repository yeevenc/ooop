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
	FindByIDs(ctx context.Context, ids []int64) ([]User, error)
	FindByPhone(ctx context.Context, phone string) (User, error)
	FindByUsernameOrPhone(ctx context.Context, account string) (User, error)
	List(ctx context.Context, query UserListQuery) ([]User, int64, error)
	Create(ctx context.Context, item *User) error
	UpdatePassword(ctx context.Context, id int64, username string, passwordHash string) error
	UpdatePhone(ctx context.Context, id int64, phone string) error
	UpdateProfile(ctx context.Context, id int64, update ProfileUpdate) error
	UpdatePrivacySettings(ctx context.Context, id int64, update PrivacySettingsUpdate) error
	UpdateNotificationSettings(ctx context.Context, id int64, update NotificationSettingsUpdate) error
	UpdatePushRegistration(ctx context.Context, id int64, platform string, registrationID string) error
	UpdateRealNameVerification(ctx context.Context, id int64, realName string, idCardMask string, gender string, verifiedAt time.Time) error
	TouchLastLogin(ctx context.Context, id int64, loginAt time.Time, meta ClientMeta) error
	CancelAccount(ctx context.Context, id int64) error
}

type LoginCodeRepository interface {
	Create(ctx context.Context, item *LoginCode) error
	FindValid(ctx context.Context, phone string, scene string, codeHash string, now time.Time) (LoginCode, error)
	MarkUsed(ctx context.Context, id int64, usedAt time.Time) error
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

// FindByIDs 批量按 id 取用户（用于参加者/申请人列表，避免逐个查询的 N+1）。
func (r *GormUserRepository) FindByIDs(ctx context.Context, ids []int64) ([]User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var items []User
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&items).Error
	return items, err
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

func (r *GormUserRepository) UpdatePhone(ctx context.Context, id int64, phone string) error {
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", id).Update("phone", phone).Error
}

// UpdateProfile 仅更新调用方显式传入(非 nil)的资料字段，未传入的字段保持不变。
func (r *GormUserRepository) UpdateProfile(ctx context.Context, id int64, update ProfileUpdate) error {
	updates := profileUpdateColumns(update)
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", id).Updates(updates).Error
}

func (r *GormUserRepository) UpdatePrivacySettings(ctx context.Context, id int64, update PrivacySettingsUpdate) error {
	updates := privacySettingsUpdateColumns(update)
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", id).Updates(updates).Error
}

func (r *GormUserRepository) UpdateNotificationSettings(ctx context.Context, id int64, update NotificationSettingsUpdate) error {
	updates := notificationSettingsUpdateColumns(update)
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", id).Updates(updates).Error
}

func (r *GormUserRepository) UpdatePushRegistration(ctx context.Context, id int64, platform string, registrationID string) error {
	updates := map[string]interface{}{
		"push_platform":   normalizeMetaValue(platform),
		"registration_id": normalizeMetaValue(registrationID),
	}
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", id).Updates(updates).Error
}

func (r *GormUserRepository) UpdateRealNameVerification(ctx context.Context, id int64, realName string, idCardMask string, gender string, verifiedAt time.Time) error {
	updates := map[string]interface{}{
		"real_name":             realName,
		"id_card_mask":          idCardMask,
		"is_real_name_verified": true,
		"real_name_verified_at": verifiedAt,
	}
	if gender != "" {
		updates["gender"] = gender
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

func (r *GormUserRepository) CancelAccount(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item User
		if err := tx.First(&item, id).Error; err != nil {
			return normalizeNotFound(err)
		}

		var activityIDs []int64
		if err := tx.Table("activities").Where("user_id = ?", id).Pluck("id", &activityIDs).Error; err != nil {
			return err
		}

		// 注销账号需要删除与该用户直接关联的数据，避免旧手机号再次登录时继承历史关系。
		if len(activityIDs) > 0 {
			if err := tx.Exec("DELETE FROM activity_participants WHERE activity_id IN ?", activityIDs).Error; err != nil {
				return err
			}
			if err := tx.Exec("DELETE FROM activity_favorites WHERE activity_id IN ?", activityIDs).Error; err != nil {
				return err
			}
		}
		if err := tx.Exec("DELETE FROM activities WHERE user_id = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM activity_participants WHERE user_id = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM activity_favorites WHERE user_id = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM user_messages WHERE user_id = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM feedbacks WHERE user_id = ?", id).Error; err != nil {
			return err
		}
		if item.Phone != "" {
			if err := tx.Exec("DELETE FROM login_codes WHERE phone = ?", item.Phone).Error; err != nil {
				return err
			}
		}

		return tx.Delete(&User{}, id).Error
	})
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

func normalizeNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}
