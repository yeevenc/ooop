package activity

import (
	"context"

	"gorm.io/gorm"
)

// ===== 活动批量查询 & 计数维护 =====

// FindActivitiesByIDs 批量按 id 取活动（用于「我参加/Ta 参加的活动」列表）。
func (r *GormRepository) FindActivitiesByIDs(ctx context.Context, ids []int64) ([]Activity, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var items []Activity
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&items).Error
	return items, err
}

// AdjustCurrentCount 原子增减活动已参加人数（审核通过 +count / 取消通过 -count）。
// 用 SQL 自增避免读改写丢更新，并兜底不小于 0。
func (r *GormRepository) AdjustCurrentCount(ctx context.Context, id int64, delta int) error {
	return r.db.WithContext(ctx).
		Model(&Activity{}).
		Where("id = ?", id).
		Update("current_count", gorm.Expr("GREATEST(current_count + ?, 0)", delta)).
		Error
}

// ===== 参加(报名)关联 =====

// FindParticipant 取某用户对某活动的报名记录；无则返回 gorm.ErrRecordNotFound。
func (r *GormRepository) FindParticipant(ctx context.Context, activityID, userID int64) (ActivityParticipant, error) {
	var item ActivityParticipant
	err := r.db.WithContext(ctx).
		Where("activity_id = ? AND user_id = ?", activityID, userID).
		First(&item).Error
	return item, err
}

func (r *GormRepository) CreateParticipant(ctx context.Context, item *ActivityParticipant) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *GormRepository) SaveParticipant(ctx context.Context, item *ActivityParticipant) error {
	return r.db.WithContext(ctx).Save(item).Error
}

// ListParticipantsByActivity 列出某活动指定状态的参加者，按报名时间倒序；limit<=0 表示不限。
func (r *GormRepository) ListParticipantsByActivity(ctx context.Context, activityID int64, statuses []string, limit int) ([]ActivityParticipant, error) {
	var items []ActivityParticipant
	db := r.db.WithContext(ctx).
		Where("activity_id = ?", activityID)
	if len(statuses) > 0 {
		db = db.Where("status IN ?", statuses)
	}
	db = db.Order("created_at DESC")
	if limit > 0 {
		db = db.Limit(limit)
	}
	err := db.Find(&items).Error
	return items, err
}

// ListParticipantsByUser 列出某用户指定状态的参加记录（用于「我/Ta 参加的活动」）。
func (r *GormRepository) ListParticipantsByUser(ctx context.Context, userID int64, statuses []string) ([]ActivityParticipant, error) {
	var items []ActivityParticipant
	db := r.db.WithContext(ctx).
		Where("user_id = ?", userID)
	if len(statuses) > 0 {
		db = db.Where("status IN ?", statuses)
	}
	err := db.Order("created_at DESC").Find(&items).Error
	return items, err
}

// CountByActivityIDsAndStatus 批量统计多个活动在指定状态下的报名条数（用于「我的发布」待审核人数）。
func (r *GormRepository) CountByActivityIDsAndStatus(ctx context.Context, ids []int64, status string) (map[int64]int, error) {
	result := make(map[int64]int)
	if len(ids) == 0 {
		return result, nil
	}

	type row struct {
		ActivityID int64
		Total      int
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Model(&ActivityParticipant{}).
		Select("activity_id, COALESCE(SUM(count), 0) AS total").
		Where("activity_id IN ? AND status = ?", ids, status).
		Group("activity_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, item := range rows {
		result[item.ActivityID] = item.Total
	}
	return result, nil
}

func (r *GormRepository) CountPublishedByUser(ctx context.Context, userID int64, statuses []string) (int64, error) {
	var total int64
	db := r.db.WithContext(ctx).
		Model(&Activity{}).
		Where("user_id = ?", userID)
	if len(statuses) > 0 {
		db = db.Where("status IN ?", statuses)
	}
	err := db.Count(&total).Error
	return total, err
}

func (r *GormRepository) CountJoinedByUser(ctx context.Context, userID int64, statuses []string) (int64, error) {
	var total int64
	db := r.db.WithContext(ctx).
		Model(&ActivityParticipant{}).
		Where("user_id = ?", userID)
	if len(statuses) > 0 {
		db = db.Where("status IN ?", statuses)
	}
	err := db.Count(&total).Error
	return total, err
}
