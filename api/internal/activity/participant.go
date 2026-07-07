package activity

import "time"

// 报名(参加)状态
const (
	ParticipantStatusJoined    = "joined"    // 已报名/参加中
	ParticipantStatusApproved  = "approved"  // 发布人已通过
	ParticipantStatusRejected  = "rejected"  // 发布人已拒绝
	ParticipantStatusCancelled = "cancelled" // 用户已取消
)

// ActivityParticipant 活动参加关联表：用「活动 id + 用户 id」把用户与其参加的活动绑定起来。
//   - uniq_activity_user：(activity_id, user_id) 复合唯一，防止同一用户对同一活动重复报名；
//     该复合索引的最左前缀同时服务「某活动的参加者」(按 activity_id) 查询。
//   - user_id 单列索引服务「我参加的活动」(按 user_id) 反向查询。
//
// 活动本身与发布人的绑定见 Activity.UserID（activities.user_id ↔ users.id）。
type ActivityParticipant struct {
	ID           int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	ActivityID   int64  `gorm:"not null;uniqueIndex:uniq_activity_user" json:"activity_id"`
	UserID       int64  `gorm:"not null;uniqueIndex:uniq_activity_user;index" json:"user_id"`
	Count        int    `gorm:"not null;default:1" json:"count"` // 报名人数（本人 + 同行）
	Remark       string `gorm:"size:255;not null;default:''" json:"remark"`
	ContactInfo  string `gorm:"size:64;not null;default:''" json:"contact_info"` // 线下联系用的报名联系方式
	RejectReason string `gorm:"size:255;not null;default:''" json:"reject_reason"`
	// EntryCode 参加编号（数字+字母）：审核「通过」时一次性生成，仅已通过的报名才有，用于线下核对参加身份。
	EntryCode string    `gorm:"size:16;not null;default:''" json:"entry_code"`
	Status    string    `gorm:"size:32;not null;default:'joined';index" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
