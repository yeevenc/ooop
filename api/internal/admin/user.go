package admin

import "ooop-admin-api/internal/user"

// AdminUserResponse 后台管理系统的App用户响应结构，时间字段使用 AdminTime 格式化为 YYYY-MM-DD HH:mm:ss。
type AdminUserResponse struct {
	ID                 int64      `json:"id"`
	Phone              string     `json:"phone"`
	Username           string     `json:"username"`
	Nickname           string     `json:"nickname"`
	Avatar             string     `json:"avatar"`
	Gender             string     `json:"gender"`
	Region             string     `json:"region"`
	Bio                string     `json:"bio"`
	Platform           string     `json:"platform"`
	DeviceNo           string     `json:"device_no"`
	PushPlatform       string     `json:"push_platform"`
	RegistrationID     string     `json:"registration_id"`
	RealName           string     `json:"real_name"`
	IDCardMask         string     `json:"id_card_mask"`
	RealNameVerified   bool       `json:"is_real_name_verified"`
	RealNameVerifiedAt *AdminTime `json:"real_name_verified_at"`
	Status int `json:"status"` // 1 正常 / 0 封禁
	// BannedUntil 限时解封时间；永久封禁时为 null
	BannedUntil *AdminTime `json:"banned_until"`
	// BanReason 封禁原因备注
	BanReason string `json:"ban_reason"`
	CreditScore        int        `json:"credit_score"`
	RegisterSource     string     `json:"register_source"`
	HasPassword        bool       `json:"has_password"`
	LastLoginAt        *AdminTime `json:"last_login_at"`
	CreatedAt          AdminTime  `json:"created_at"`
	PublishedCount     int        `json:"published_count"`
	JoinedCount        int        `json:"joined_count"`
	LikedCount         int        `json:"liked_count"`
}

// ToAdminUserResponse 将 user.PublicUser 转换为后台管理的 AdminUserResponse 格式。
func ToAdminUserResponse(u user.PublicUser) AdminUserResponse {
	return AdminUserResponse{
		ID:                 u.ID,
		Phone:              u.Phone,
		Username:           u.Username,
		Nickname:           u.Nickname,
		Avatar:             u.Avatar,
		Gender:             u.Gender,
		Region:             u.Region,
		Bio:                u.Bio,
		Platform:           u.Platform,
		DeviceNo:           u.DeviceNo,
		PushPlatform:       u.PushPlatform,
		RegistrationID:     u.RegistrationID,
		RealName:           u.RealName,
		IDCardMask:         u.IDCardMask,
		RealNameVerified:   u.RealNameVerified,
		RealNameVerifiedAt: ToAdminTime(u.RealNameVerifiedAt),
		Status:             u.Status,
		BannedUntil:        ToAdminTime(u.BannedUntil),
		BanReason:          u.BanReason,
		CreditScore:        u.CreditScore,
		RegisterSource:     u.RegisterSource,
		HasPassword:        u.HasPassword,
		LastLoginAt:        ToAdminTime(u.LastLoginAt),
		CreatedAt:          ToAdminTimeRequired(u.CreatedAt),
		PublishedCount:     u.PublishedCount,
		JoinedCount:        u.JoinedCount,
		LikedCount:         u.LikedCount,
	}
}

// AdminUserListResult 后台管理的用户列表响应。
type AdminUserListResult struct {
	List     []AdminUserResponse `json:"list"`
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
}
