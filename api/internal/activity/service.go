package activity

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"ooop-admin-api/internal/user"
)

var (
	ErrInvalidTitle    = errors.New("请填写活动标题")
	ErrInvalidCategory = errors.New("请选择活动分类")
	ErrInvalidLocation = errors.New("请选择活动地点")
	ErrInvalidCity     = errors.New("请选择活动城市")
	ErrInvalidIntro    = errors.New("请填写活动简介")
	ErrInvalidCount    = errors.New("活动人数不能少于 2 人")
	ErrNotFound        = errors.New("活动不存在")
	ErrInvalidStatus   = errors.New("状态不合法")
	ErrCategoryExists  = errors.New("分类标识已存在")
	ErrCategoryMissing = errors.New("分类不存在")

	// 报名(参加)相关
	ErrAlreadyJoined       = errors.New("你已报名，请勿重复报名")
	ErrJoinOwnActivity     = errors.New("不能报名自己发起的活动")
	ErrActivityFull        = errors.New("活动名额已满")
	ErrActivityNotJoinable = errors.New("活动未开放报名")
	ErrNotOrganizer        = errors.New("无权操作该活动")
	ErrParticipantNotFound = errors.New("报名记录不存在")
	ErrRejectReasonMissing = errors.New("请填写拒绝原因")
	ErrActivityStarted     = errors.New("活动已开始，不能进行该操作")
)

// 详情页「已报名成员」头像最多展示数量。
const maxDetailParticipants = 20

// PublicParticipant 已报名成员（详情页头像行用）。
type PublicParticipant struct {
	ID          string `json:"id"`
	UserID      string `json:"userId"`
	Name        string `json:"name"`
	Avatar      string `json:"avatar"`
	AvatarColor string `json:"avatarColor"`
	IsOnline    bool   `json:"isOnline"`
}

// PublicApplicant 待审核申请人（发起人「申请人审核」用）。
type PublicApplicant struct {
	ID           string `json:"id"`     // 参加行 id
	UserID       string `json:"userId"` // 申请用户 id
	Name         string `json:"name"`
	Gender       string `json:"gender"`
	Phone        string `json:"phone"`
	Avatar       string `json:"avatar"`
	AvatarColor  string `json:"avatarColor"`
	ApplyTime    string `json:"applyTime"`    // 相对时间，如 "2小时前"
	Count        int    `json:"count"`        // 报名人数
	ApplyContent string `json:"applyContent"` // 报名备注
}

type PublicJoinInfo struct {
	Status       string `json:"status"`
	Count        int    `json:"count"`
	Remark       string `json:"remark"`
	RejectReason string `json:"rejectReason"`
	EntryCode    string `json:"entryCode"`
	ApplyTime    string `json:"applyTime"`
}

type Service struct {
	activities     Repository
	users          user.UserRepository
	reviewNotifier ReviewNotifier
}

type ReviewNotifier interface {
	CreateActivityReviewMessage(ctx context.Context, userID int64, activityID int64, activityTitle string, approved bool) error
	CreateActivityRegistrationMessage(ctx context.Context, userID int64, activityID int64, activityTitle string, applicantName string) error
}

type CreateInput struct {
	Title             string     `json:"title"`
	CategoryID        string     `json:"category_id"`
	CategoryLabel     string     `json:"category_label"`
	ActivityDate      *time.Time `json:"activity_date"`
	ActivityTime      string     `json:"activity_time"`
	DeadlineAt        *time.Time `json:"deadline_at"`
	LocationText      string     `json:"location_text"`
	City              string     `json:"city"`
	Latitude          float64    `json:"latitude"`
	Longitude         float64    `json:"longitude"`
	TotalCount        int        `json:"total_count"`
	CostType          string     `json:"cost_type"`
	FeeDetail         string     `json:"fee_detail"`
	GenderRequirement string     `json:"gender_requirement"`
	Intro             string     `json:"intro"`
	Notice            string     `json:"notice"`
	ImageURLs         []string   `json:"image_urls"`
}

var DefaultCategories = []ActivityCategory{
	{ID: "outdoor", Label: "户外", Sort: 10, Status: CategoryEnabled},
	{ID: "games", Label: "游戏", Sort: 20, Status: CategoryEnabled},
	{ID: "food", Label: "美食", Sort: 30, Status: CategoryEnabled},
	{ID: "sports", Label: "运动", Sort: 40, Status: CategoryEnabled},
	{ID: "music", Label: "音乐", Sort: 50, Status: CategoryEnabled},
	{ID: "photo", Label: "摄影", Sort: 60, Status: CategoryEnabled},
	{ID: "art", Label: "艺术", Sort: 70, Status: CategoryEnabled},
	{ID: "hiking", Label: "登山", Sort: 80, Status: CategoryEnabled},
	{ID: "citywalk", Label: "城市漫步", Sort: 90, Status: CategoryEnabled},
	{ID: "movie", Label: "电影", Sort: 100, Status: CategoryEnabled},
}

func NewService(activities Repository, users user.UserRepository) *Service {
	return &Service{
		activities: activities,
		users:      users,
	}
}

func (s *Service) SetReviewNotifier(notifier ReviewNotifier) {
	s.reviewNotifier = notifier
}

func (s *Service) Create(ctx context.Context, userID int64, input CreateInput) (PublicActivity, error) {
	item, err := input.toModel(userID)
	if err != nil {
		return PublicActivity{}, err
	}

	category, err := s.activities.FindCategory(ctx, item.CategoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return PublicActivity{}, ErrInvalidCategory
		}
		return PublicActivity{}, err
	}
	item.CategoryLabel = category.Label

	if err := s.activities.Create(ctx, &item); err != nil {
		return PublicActivity{}, err
	}
	return s.toPublic(ctx, item), nil
}

func (s *Service) List(ctx context.Context, query ListQuery) ([]PublicActivity, error) {
	query.City = normalizeCity(query.City)
	query.CategoryID = strings.TrimSpace(query.CategoryID)
	query.Keyword = strings.TrimSpace(query.Keyword)

	items, err := s.activities.List(ctx, query)
	if err != nil {
		return nil, err
	}

	list := make([]PublicActivity, 0, len(items))
	for _, item := range items {
		list = append(list, s.toPublic(ctx, item))
	}
	return list, nil
}

func (s *Service) ListCategories(ctx context.Context) ([]PublicActivityCategory, error) {
	items, err := s.activities.ListCategories(ctx)
	if err != nil {
		return nil, err
	}

	list := make([]PublicActivityCategory, 0, len(items))
	for _, item := range items {
		list = append(list, toPublicCategory(item))
	}
	return list, nil
}

func (s *Service) EnsureDefaultCategories(ctx context.Context) error {
	return EnsureDefaultCategories(ctx, s.activities)
}

func EnsureDefaultCategories(ctx context.Context, activities Repository) error {
	return activities.SaveCategories(ctx, DefaultCategories)
}

// ===== 后台管理（admin）方法 =====

type AdminActivityListResult struct {
	List     []PublicActivity `json:"list"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

// AdminActivityUpdate 后台编辑活动可改字段（仅文本类；日期/截止/坐标/图片/发起人/状态保持不变）。
type AdminActivityUpdate struct {
	Title             string
	CategoryID        string
	ActivityTime      string
	LocationText      string
	City              string
	TotalCount        int
	CostType          string
	FeeDetail         string
	GenderRequirement string
	Intro             string
	Notice            string
}

type AdminCategory struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Icon   string `json:"icon"`
	Sort   int    `json:"sort"`
	Status int    `json:"status"`
}

type CategoryInput struct {
	ID     string
	Label  string
	Icon   string
	Sort   int
	Status int
}

func (s *Service) AdminListActivities(ctx context.Context, query AdminActivityQuery) (AdminActivityListResult, error) {
	query.Keyword = strings.TrimSpace(query.Keyword)
	query.Status = strings.TrimSpace(query.Status)
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 10
	}

	items, total, err := s.activities.AdminList(ctx, query)
	if err != nil {
		return AdminActivityListResult{}, err
	}

	list := make([]PublicActivity, 0, len(items))
	for _, item := range items {
		list = append(list, s.toPublic(ctx, item))
	}

	return AdminActivityListResult{
		List:     list,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

func (s *Service) GetActivityByID(ctx context.Context, id int64) (PublicActivity, error) {
	item, err := s.findActivity(ctx, id)
	if err != nil {
		return PublicActivity{}, err
	}
	return s.toPublic(ctx, item), nil
}

// GetPublicActivityByID 返回 App 端可见（已审核通过 ongoing）的活动详情。
// 待审核/已拒绝/已下架的活动一律按「不存在」处理，避免未过审内容被 App 直接访问。
func (s *Service) GetPublicActivityByID(ctx context.Context, id int64) (PublicActivity, error) {
	item, err := s.findActivity(ctx, id)
	if err != nil {
		return PublicActivity{}, err
	}
	if item.Status != StatusOngoing {
		return PublicActivity{}, ErrNotFound
	}
	pub := s.toPublic(ctx, item)
	// 详情页挂上「已报名成员」（已通过审核者）。
	pub.Participants = s.approvedParticipants(ctx, id)
	return pub, nil
}

// ListPublicUserActivities 返回某用户对外可见（ongoing）的发布活动，用于「他人主页」。
func (s *Service) ListPublicUserActivities(ctx context.Context, userID int64, page, pageSize int) ([]PublicActivity, error) {
	return s.listUserActivities(ctx, userID, page, pageSize, []string{StatusOngoing})
}

// ListMyActivities 返回当前登录用户自己的发布活动（含审核中 pending），用于「我的主页」。
func (s *Service) ListMyActivities(ctx context.Context, userID int64, page, pageSize int) ([]PublicActivity, error) {
	list, err := s.listUserActivities(ctx, userID, page, pageSize, []string{
		StatusOngoing,
		StatusPending,
		StatusRejected,
		StatusTakenDown,
		StatusCancelled,
	})
	if err != nil {
		return nil, err
	}
	// 批量填充每个活动的「待审核报名数」，供「我的发布」卡片展示。
	ids := make([]int64, 0, len(list))
	for _, a := range list {
		if n, e := strconv.ParseInt(a.ID, 10, 64); e == nil {
			ids = append(ids, n)
		}
	}
	if counts, e := s.activities.CountByActivityIDsAndStatus(ctx, ids, ParticipantStatusJoined); e == nil {
		for i := range list {
			if n, pe := strconv.ParseInt(list[i].ID, 10, 64); pe == nil {
				list[i].PendingCount = counts[n]
			}
		}
	}
	return list, nil
}

func (s *Service) listUserActivities(ctx context.Context, userID int64, page, pageSize int, statuses []string) ([]PublicActivity, error) {
	items, err := s.activities.ListByUser(ctx, UserActivityQuery{
		UserID:   userID,
		Page:     page,
		PageSize: pageSize,
		Statuses: statuses,
	})
	if err != nil {
		return nil, err
	}
	list := make([]PublicActivity, 0, len(items))
	for _, item := range items {
		list = append(list, s.toPublic(ctx, item))
	}
	return list, nil
}

// ===== 报名(参加) =====

// JoinActivity 报名参加活动：生成 joined（待发起人审核）记录，不立即计入正式人数。
// 校验：活动可见(ongoing)、非本人发起、尚有空位、未重复报名（已拒绝/取消的可重新报名）。
func (s *Service) JoinActivity(ctx context.Context, userID, activityID int64, count int, remark string) error {
	if count < 1 {
		count = 1
	}

	item, err := s.findActivity(ctx, activityID)
	if err != nil {
		return err
	}
	if item.Status != StatusOngoing {
		return ErrActivityNotJoinable
	}
	if item.UserID == userID {
		return ErrJoinOwnActivity
	}
	if item.CurrentCount+count > item.TotalCount {
		return ErrActivityFull
	}

	remark = strings.TrimSpace(remark)
	existing, err := s.activities.FindParticipant(ctx, activityID, userID)
	if err == nil {
		// 已有记录：joined/approved 视为重复；rejected/cancelled 允许复用同一行重新报名。
		if existing.Status == ParticipantStatusJoined || existing.Status == ParticipantStatusApproved {
			return ErrAlreadyJoined
		}
		existing.Count = count
		existing.Remark = remark
		existing.RejectReason = ""
		existing.Status = ParticipantStatusJoined
		if err := s.activities.SaveParticipant(ctx, &existing); err != nil {
			return err
		}
		s.notifyOrganizerRegistration(ctx, item, userID)
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if err := s.activities.CreateParticipant(ctx, &ActivityParticipant{
		ActivityID:   activityID,
		UserID:       userID,
		Count:        count,
		Remark:       remark,
		RejectReason: "",
		Status:       ParticipantStatusJoined,
	}); err != nil {
		return err
	}
	s.notifyOrganizerRegistration(ctx, item, userID)
	return nil
}

// CancelParticipation 参加人取消参加活动；活动开始前可取消，已审核通过时同步扣减人数。
func (s *Service) CancelParticipation(ctx context.Context, userID, activityID int64) error {
	item, err := s.findActivity(ctx, activityID)
	if err != nil {
		return err
	}
	if hasActivityStarted(item) {
		return ErrActivityStarted
	}

	p, err := s.activities.FindParticipant(ctx, activityID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrParticipantNotFound
		}
		return err
	}
	if p.Status != ParticipantStatusJoined && p.Status != ParticipantStatusApproved {
		return ErrParticipantNotFound
	}

	shouldAdjustCount := p.Status == ParticipantStatusApproved
	p.Status = ParticipantStatusCancelled
	p.RejectReason = ""
	if err := s.activities.SaveParticipant(ctx, &p); err != nil {
		return err
	}
	if shouldAdjustCount {
		return s.activities.AdjustCurrentCount(ctx, activityID, -p.Count)
	}
	return nil
}

// ListApplicants 发起人查看某活动的待审核报名（joined）。
func (s *Service) ListApplicants(ctx context.Context, organizerID, activityID int64) ([]PublicApplicant, error) {
	item, err := s.findActivity(ctx, activityID)
	if err != nil {
		return nil, err
	}
	if item.UserID != organizerID {
		return nil, ErrNotOrganizer
	}

	parts, err := s.activities.ListParticipantsByActivity(ctx, activityID, []string{ParticipantStatusJoined}, 0)
	if err != nil {
		return nil, err
	}
	users := s.usersByParticipants(ctx, parts)

	list := make([]PublicApplicant, 0, len(parts))
	for _, p := range parts {
		u := users[p.UserID]
		list = append(list, PublicApplicant{
			ID:           strconv.FormatInt(p.ID, 10),
			UserID:       strconv.FormatInt(p.UserID, 10),
			Name:         displayName(u, p.UserID),
			Gender:       u.Gender,
			Phone:        u.Phone,
			Avatar:       user.AvatarURL(u.Avatar),
			AvatarColor:  "#8fa061",
			ApplyTime:    relativeTime(p.CreatedAt),
			Count:        p.Count,
			ApplyContent: defaultText(p.Remark, "申请参加这个活动"),
		})
	}
	return list, nil
}

// ReviewApplicant 发起人审核某报名：approve→approved 并把人数计入；reject→rejected。仅处理 joined。
func (s *Service) ReviewApplicant(ctx context.Context, organizerID, activityID, targetUserID int64, approve bool, rejectReason string) error {
	item, err := s.findActivity(ctx, activityID)
	if err != nil {
		return err
	}
	if item.UserID != organizerID {
		return ErrNotOrganizer
	}

	p, err := s.activities.FindParticipant(ctx, activityID, targetUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrParticipantNotFound
		}
		return err
	}
	if p.Status != ParticipantStatusJoined {
		// 已处理过（approved/rejected）或非待审核状态，视为记录不存在，避免重复计数。
		return ErrParticipantNotFound
	}

	if !approve {
		rejectReason = strings.TrimSpace(rejectReason)
		if rejectReason == "" {
			return ErrRejectReasonMissing
		}
		p.Status = ParticipantStatusRejected
		p.RejectReason = rejectReason
		return s.activities.SaveParticipant(ctx, &p)
	}

	// 通过前重读活动校验名额，避免超员。
	fresh, err := s.findActivity(ctx, activityID)
	if err != nil {
		return err
	}
	if fresh.CurrentCount+p.Count > fresh.TotalCount {
		return ErrActivityFull
	}
	p.Status = ParticipantStatusApproved
	p.RejectReason = ""
	if p.EntryCode == "" {
		// 仅在审核通过时生成参加编号，避免给「待审核/被拒」的报名浪费编号。
		p.EntryCode = generateEntryCode()
	}
	if err := s.activities.SaveParticipant(ctx, &p); err != nil {
		return err
	}
	return s.activities.AdjustCurrentCount(ctx, activityID, p.Count)
}

// MyParticipation 返回当前用户对某活动的报名状态（用于详情页按钮状态切换）；未报名返回 nil。
func (s *Service) MyParticipation(ctx context.Context, userID, activityID int64) (*PublicJoinInfo, error) {
	p, err := s.activities.FindParticipant(ctx, activityID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toPublicJoinInfo(p), nil
}

// ListMyJoinedActivities 我参加的活动（joined+approved），用于「我的主页」参与数。
func (s *Service) ListMyJoinedActivities(ctx context.Context, userID int64) ([]PublicActivity, error) {
	return s.listJoinedActivities(ctx, userID, []string{
		ParticipantStatusJoined,
		ParticipantStatusApproved,
		ParticipantStatusRejected,
	})
}

// ListUserJoinedActivities Ta 参加的活动（仅 approved），用于「他人主页」参与数。
func (s *Service) ListUserJoinedActivities(ctx context.Context, userID int64) ([]PublicActivity, error) {
	return s.listJoinedActivities(ctx, userID, []string{ParticipantStatusApproved})
}

func (s *Service) listJoinedActivities(ctx context.Context, userID int64, statuses []string) ([]PublicActivity, error) {
	parts, err := s.activities.ListParticipantsByUser(ctx, userID, statuses)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(parts))
	for _, p := range parts {
		ids = append(ids, p.ActivityID)
	}
	if len(ids) == 0 {
		return []PublicActivity{}, nil
	}

	items, err := s.activities.FindActivitiesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	list := make([]PublicActivity, 0, len(items))
	participantByActivityID := make(map[int64]ActivityParticipant, len(parts))
	for _, part := range parts {
		participantByActivityID[part.ActivityID] = part
	}
	for _, item := range items {
		// 只展示仍对外可见(ongoing)的活动，避免泄露未过审/已下架内容。
		if item.Status != StatusOngoing {
			continue
		}
		publicItem := s.toPublic(ctx, item)
		if part, ok := participantByActivityID[item.ID]; ok {
			publicItem.JoinInfo = toPublicJoinInfo(part)
		}
		list = append(list, publicItem)
	}
	return list, nil
}

func toPublicJoinInfo(item ActivityParticipant) *PublicJoinInfo {
	return &PublicJoinInfo{
		Status:       item.Status,
		Count:        item.Count,
		Remark:       item.Remark,
		RejectReason: item.RejectReason,
		EntryCode:    item.EntryCode,
		ApplyTime:    relativeTime(item.CreatedAt),
	}
}

func hasActivityStarted(item Activity) bool {
	if item.ActivityDate == nil {
		return false
	}
	return !item.ActivityDate.After(time.Now())
}

// approvedParticipants 取活动已通过审核的成员（限量），转为详情页可消费的对象列表。
func (s *Service) approvedParticipants(ctx context.Context, activityID int64) []any {
	parts, err := s.activities.ListParticipantsByActivity(ctx, activityID, []string{ParticipantStatusApproved}, maxDetailParticipants)
	if err != nil || len(parts) == 0 {
		return []any{}
	}
	users := s.usersByParticipants(ctx, parts)

	list := make([]any, 0, len(parts))
	for _, p := range parts {
		u := users[p.UserID]
		list = append(list, PublicParticipant{
			ID:          strconv.FormatInt(p.ID, 10),
			UserID:      strconv.FormatInt(p.UserID, 10),
			Name:        displayName(u, p.UserID),
			Avatar:      user.AvatarURL(u.Avatar),
			AvatarColor: "#8fa061",
			IsOnline:    false,
		})
	}
	return list
}

// usersByParticipants 批量取参加者的用户资料，返回 id→User 映射（避免 N+1）。
func (s *Service) usersByParticipants(ctx context.Context, parts []ActivityParticipant) map[int64]user.User {
	ids := make([]int64, 0, len(parts))
	for _, p := range parts {
		ids = append(ids, p.UserID)
	}
	users, _ := s.users.FindByIDs(ctx, ids)
	m := make(map[int64]user.User, len(users))
	for _, u := range users {
		m[u.ID] = u
	}
	return m
}

// displayName 取用户展示名：昵称 → 用户名 → 手机号 → 兜底「用户<id>」。
func displayName(u user.User, fallbackID int64) string {
	name := u.Nickname
	if name == "" && u.Username != nil {
		name = *u.Username
	}
	if name == "" {
		name = u.Phone
	}
	if name == "" {
		name = "用户" + strconv.FormatInt(fallbackID, 10)
	}
	return name
}

func (s *Service) notifyOrganizerRegistration(ctx context.Context, item Activity, applicantUserID int64) {
	if s.reviewNotifier == nil {
		return
	}

	applicantName := ""
	applicant, err := s.users.FindByID(ctx, applicantUserID)
	if err == nil {
		applicantName = displayName(applicant, applicantUserID)
	}

	_ = s.reviewNotifier.CreateActivityRegistrationMessage(
		ctx,
		item.UserID,
		item.ID,
		item.Title,
		applicantName,
	)
}

// relativeTime 把时间转成相对文案（刚刚/x分钟前/x小时前/x天前）。
func relativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "刚刚"
	case d < time.Hour:
		return fmt.Sprintf("%d分钟前", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d小时前", int(d.Hours()))
	default:
		return fmt.Sprintf("%d天前", int(d.Hours()/24))
	}
}

// 参加编号字母表：大写字母 + 数字，去掉易混淆的 I/O/0/1。
const entryCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// generateEntryCode 生成 8 位「参加编号」（数字+字母）。仅审核通过时调用。
// 31^8 ≈ 8.5e11，当前规模冲突概率可忽略；rand 失败时退化用时间戳兜底。
func generateEntryCode() string {
	const n = 8
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("OO%06d", time.Now().UnixNano()%1000000)
	}
	out := make([]byte, n)
	for i, b := range buf {
		out[i] = entryCodeAlphabet[int(b)%len(entryCodeAlphabet)]
	}
	return string(out)
}

func (s *Service) UpdateActivity(ctx context.Context, id int64, input AdminActivityUpdate) (PublicActivity, error) {
	item, err := s.findActivity(ctx, id)
	if err != nil {
		return PublicActivity{}, err
	}

	title := strings.TrimSpace(input.Title)
	if title == "" {
		return PublicActivity{}, ErrInvalidTitle
	}
	categoryID := strings.TrimSpace(input.CategoryID)
	if categoryID == "" {
		return PublicActivity{}, ErrInvalidCategory
	}
	locationText := strings.TrimSpace(input.LocationText)
	if locationText == "" {
		return PublicActivity{}, ErrInvalidLocation
	}
	city := normalizeCity(input.City)
	if city == "" {
		return PublicActivity{}, ErrInvalidCity
	}
	intro := strings.TrimSpace(input.Intro)
	if intro == "" {
		return PublicActivity{}, ErrInvalidIntro
	}
	if input.TotalCount < 2 {
		return PublicActivity{}, ErrInvalidCount
	}

	category, err := s.activities.FindCategory(ctx, categoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return PublicActivity{}, ErrInvalidCategory
		}
		return PublicActivity{}, err
	}

	// 仅更新文本类字段；日期/截止/坐标/图片/发起人/状态保持不变。
	item.Title = title
	item.CategoryID = categoryID
	item.CategoryLabel = category.Label
	item.ActivityTime = strings.TrimSpace(input.ActivityTime)
	item.LocationText = locationText
	item.City = city
	item.TotalCount = input.TotalCount
	item.CostType = strings.TrimSpace(input.CostType)
	item.FeeDetail = strings.TrimSpace(input.FeeDetail)
	item.GenderRequirement = strings.TrimSpace(input.GenderRequirement)
	item.Intro = intro
	item.Notice = strings.TrimSpace(input.Notice)

	if err := s.activities.Save(ctx, &item); err != nil {
		return PublicActivity{}, err
	}
	return s.toPublic(ctx, item), nil
}

func (s *Service) DeleteActivity(ctx context.Context, id int64) error {
	if _, err := s.findActivity(ctx, id); err != nil {
		return err
	}
	return s.activities.Delete(ctx, id)
}

// CancelOwnedActivity 发起人取消自己发布的活动；仅活动开始前允许取消。
func (s *Service) CancelOwnedActivity(ctx context.Context, ownerID, id int64) (PublicActivity, error) {
	item, err := s.findActivity(ctx, id)
	if err != nil {
		return PublicActivity{}, err
	}
	if item.UserID != ownerID {
		return PublicActivity{}, ErrNotOrganizer
	}
	if hasActivityStarted(item) {
		return PublicActivity{}, ErrActivityStarted
	}
	if err := s.activities.UpdateStatus(ctx, id, StatusCancelled); err != nil {
		return PublicActivity{}, err
	}
	item.Status = StatusCancelled
	return s.toPublic(ctx, item), nil
}

// TakeDownOwnedActivity 发起人下架自己发布的活动；仅活动开始前允许下架。
func (s *Service) TakeDownOwnedActivity(ctx context.Context, ownerID, id int64) (PublicActivity, error) {
	item, err := s.findActivity(ctx, id)
	if err != nil {
		return PublicActivity{}, err
	}
	if item.UserID != ownerID {
		return PublicActivity{}, ErrNotOrganizer
	}
	if hasActivityStarted(item) {
		return PublicActivity{}, ErrActivityStarted
	}
	if err := s.activities.UpdateStatus(ctx, id, StatusTakenDown); err != nil {
		return PublicActivity{}, err
	}
	item.Status = StatusTakenDown
	return s.toPublic(ctx, item), nil
}

// DeleteOwnedActivity 发起人删除自己发布的活动；删除后从我的发布中移除。
func (s *Service) DeleteOwnedActivity(ctx context.Context, ownerID, id int64) error {
	item, err := s.findActivity(ctx, id)
	if err != nil {
		return err
	}
	if item.UserID != ownerID {
		return ErrNotOrganizer
	}
	return s.activities.Delete(ctx, id)
}

// ReviewActivity 审核：仅对「待审核」活动生效，通过→ongoing，拒绝→rejected。
func (s *Service) ReviewActivity(ctx context.Context, id int64, approve bool) (PublicActivity, error) {
	item, err := s.findActivity(ctx, id)
	if err != nil {
		return PublicActivity{}, err
	}
	if item.Status != StatusPending {
		return PublicActivity{}, ErrInvalidStatus
	}

	next := StatusRejected
	if approve {
		next = StatusOngoing
	}
	if err := s.activities.UpdateStatus(ctx, id, next); err != nil {
		return PublicActivity{}, err
	}
	item.Status = next
	if s.reviewNotifier != nil {
		// 审核状态以活动更新为准，站内消息失败不阻断后台审核流程。
		_ = s.reviewNotifier.CreateActivityReviewMessage(ctx, item.UserID, item.ID, item.Title, approve)
	}
	return s.toPublic(ctx, item), nil
}

// SetActivityStatus 上下架：仅允许 ongoing 与 taken_down 互转。
func (s *Service) SetActivityStatus(ctx context.Context, id int64, status string) (PublicActivity, error) {
	if status != StatusOngoing && status != StatusTakenDown {
		return PublicActivity{}, ErrInvalidStatus
	}
	item, err := s.findActivity(ctx, id)
	if err != nil {
		return PublicActivity{}, err
	}
	if err := s.activities.UpdateStatus(ctx, id, status); err != nil {
		return PublicActivity{}, err
	}
	item.Status = status
	return s.toPublic(ctx, item), nil
}

func (s *Service) AdminListCategories(ctx context.Context) ([]AdminCategory, error) {
	items, err := s.activities.AdminListCategories(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]AdminCategory, 0, len(items))
	for _, item := range items {
		list = append(list, toAdminCategory(item))
	}
	return list, nil
}

func (s *Service) CreateCategory(ctx context.Context, input CategoryInput) (AdminCategory, error) {
	id := strings.TrimSpace(input.ID)
	label := strings.TrimSpace(input.Label)
	if id == "" || label == "" {
		return AdminCategory{}, ErrInvalidCategory
	}

	_, err := s.activities.FindCategoryByID(ctx, id)
	if err == nil {
		return AdminCategory{}, ErrCategoryExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return AdminCategory{}, err
	}

	item := ActivityCategory{
		ID:     id,
		Label:  label,
		Icon:   strings.TrimSpace(input.Icon),
		Sort:   input.Sort,
		Status: normalizeCategoryStatus(input.Status),
	}
	if err := s.activities.CreateCategory(ctx, &item); err != nil {
		return AdminCategory{}, err
	}
	return toAdminCategory(item), nil
}

func (s *Service) UpdateCategory(ctx context.Context, id string, input CategoryInput) (AdminCategory, error) {
	id = strings.TrimSpace(id)
	existing, err := s.activities.FindCategoryByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AdminCategory{}, ErrCategoryMissing
		}
		return AdminCategory{}, err
	}

	label := strings.TrimSpace(input.Label)
	if label == "" {
		return AdminCategory{}, ErrInvalidCategory
	}
	status := normalizeCategoryStatus(input.Status)
	icon := strings.TrimSpace(input.Icon)
	fields := map[string]interface{}{
		"label":  label,
		"icon":   icon,
		"sort":   input.Sort,
		"status": status,
	}
	if err := s.activities.UpdateCategory(ctx, id, fields); err != nil {
		return AdminCategory{}, err
	}

	existing.Label = label
	existing.Icon = icon
	existing.Sort = input.Sort
	existing.Status = status
	return toAdminCategory(existing), nil
}

func (s *Service) DeleteCategory(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if _, err := s.activities.FindCategoryByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCategoryMissing
		}
		return err
	}
	return s.activities.DeleteCategory(ctx, id)
}

func (s *Service) findActivity(ctx context.Context, id int64) (Activity, error) {
	item, err := s.activities.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Activity{}, ErrNotFound
		}
		return Activity{}, err
	}
	return item, nil
}

func toAdminCategory(item ActivityCategory) AdminCategory {
	return AdminCategory{
		ID:     item.ID,
		Label:  item.Label,
		Icon:   item.Icon,
		Sort:   item.Sort,
		Status: item.Status,
	}
}

func normalizeCategoryStatus(status int) int {
	if status == CategoryEnabled {
		return CategoryEnabled
	}
	return 0
}

func (input CreateInput) toModel(userID int64) (Activity, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return Activity{}, ErrInvalidTitle
	}

	categoryID := strings.TrimSpace(input.CategoryID)
	if categoryID == "" {
		return Activity{}, ErrInvalidCategory
	}

	locationText := strings.TrimSpace(input.LocationText)
	if locationText == "" || input.Latitude == 0 || input.Longitude == 0 {
		return Activity{}, ErrInvalidLocation
	}

	city := normalizeCity(input.City)
	if city == "" {
		return Activity{}, ErrInvalidCity
	}

	intro := strings.TrimSpace(input.Intro)
	if intro == "" {
		return Activity{}, ErrInvalidIntro
	}

	totalCount := input.TotalCount
	if totalCount < 2 {
		return Activity{}, ErrInvalidCount
	}

	gallery := normalizeImageURLs(input.ImageURLs)
	galleryJSON, _ := json.Marshal(gallery)
	imageURL := ""
	if len(gallery) > 0 {
		imageURL = gallery[0]
	}

	return Activity{
		UserID:            userID,
		Title:             title,
		CategoryID:        categoryID,
		CategoryLabel:     strings.TrimSpace(input.CategoryLabel),
		ActivityDate:      input.ActivityDate,
		ActivityTime:      strings.TrimSpace(input.ActivityTime),
		DeadlineAt:        input.DeadlineAt,
		LocationText:      locationText,
		City:              city,
		Latitude:          input.Latitude,
		Longitude:         input.Longitude,
		TotalCount:        totalCount,
		CurrentCount:      1,
		CostType:          strings.TrimSpace(input.CostType),
		FeeDetail:         strings.TrimSpace(input.FeeDetail),
		GenderRequirement: strings.TrimSpace(input.GenderRequirement),
		Intro:             intro,
		Notice:            strings.TrimSpace(input.Notice),
		ImageURL:          imageURL,
		GalleryJSON:       string(galleryJSON),
		Status:            StatusPending,
	}, nil
}

func (s *Service) toPublic(ctx context.Context, item Activity) PublicActivity {
	gallery := decodeGallery(item.GalleryJSON, item.ImageURL)
	organizer := s.organizer(ctx, item.UserID)
	needCount := item.TotalCount - item.CurrentCount
	if needCount < 0 {
		needCount = 0
	}

	return PublicActivity{
		ID:                strconv.FormatInt(item.ID, 10),
		Title:             item.Title,
		CategoryID:        item.CategoryID,
		CategoryLabel:     item.CategoryLabel,
		ImageURL:          firstImage(gallery),
		Gallery:           gallery,
		Status:            item.Status,
		CostLabel:         defaultText(item.CostType, "AA制"),
		CostType:          item.CostType,
		Time:              formatCardTime(item.ActivityDate, item.ActivityTime),
		CurrentCount:      item.CurrentCount,
		TotalCount:        item.TotalCount,
		NeedCount:         needCount,
		DeadlineText:      formatDeadlineText(item.DeadlineAt),
		DateText:          formatDateText(item.ActivityDate),
		TimeRange:         formatTimeRange(item.ActivityTime),
		ActivityTime:      item.ActivityTime,
		ActivityDate:      item.ActivityDate,
		LocationText:      item.LocationText,
		City:              item.City,
		Latitude:          item.Latitude,
		Longitude:         item.Longitude,
		FeeDetail:         item.FeeDetail,
		GenderRequirement: item.GenderRequirement,
		Intro:             item.Intro,
		Notice:            item.Notice,
		Organizer:         organizer,
		OrganizerProfile:  organizer,
		Participants:      []any{},
		ActionType:        actionType(item.CurrentCount, item.TotalCount),
		PendingCount:      0,
		CreatedAt:         item.CreatedAt,
	}
}

func (s *Service) organizer(ctx context.Context, userID int64) Organizer {
	item, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return Organizer{
			ID:               strconv.FormatInt(userID, 10),
			Name:             "活动发起人",
			Avatar:           user.DefaultAvatarURL(),
			AvatarColor:      "#8fa061",
			CreditLabel:      "信用良好",
			Rating:           5,
			PersonalityLabel: "",
			CompletionRate:   0,
			Verified:         false,
		}
	}

	name := item.Nickname
	if name == "" && item.Username != nil {
		name = *item.Username
	}
	if name == "" {
		name = item.Phone
	}

	return Organizer{
		ID:               strconv.FormatInt(item.ID, 10),
		Name:             name,
		Avatar:           user.AvatarURL(item.Avatar),
		Gender:           item.Gender,
		AvatarColor:      "#8fa061",
		CreditLabel:      "信用良好",
		Rating:           5,
		PersonalityLabel: "",
		CompletionRate:   100,
		Verified:         true,
	}
}

func normalizeCity(value string) string {
	result := strings.TrimSpace(value)
	result = strings.ReplaceAll(result, " ", "")
	result = strings.ReplaceAll(result, "·", "")
	return result
}

func normalizeImageURLs(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		item := strings.TrimSpace(value)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func decodeGallery(value string, fallback string) []string {
	var gallery []string
	if value != "" {
		_ = json.Unmarshal([]byte(value), &gallery)
	}
	if len(gallery) == 0 && fallback != "" {
		gallery = append(gallery, fallback)
	}
	if len(gallery) == 0 {
		gallery = append(gallery, "https://picsum.photos/seed/ooop-activity/1200/1600")
	}
	return gallery
}

func firstImage(gallery []string) string {
	if len(gallery) == 0 {
		return ""
	}
	return gallery[0]
}

func toPublicCategory(item ActivityCategory) PublicActivityCategory {
	return PublicActivityCategory{
		ID:    item.ID,
		Label: item.Label,
		Icon:  item.Icon,
		Sort:  item.Sort,
	}
}

func formatCardTime(date *time.Time, timeText string) string {
	if date == nil {
		return timeText
	}
	if timeText == "" {
		return fmt.Sprintf("%d月%d日", int(date.Month()), date.Day())
	}
	return fmt.Sprintf("%d月%d日 %s", int(date.Month()), date.Day(), timeText)
}

func formatDateText(date *time.Time) string {
	if date == nil {
		return ""
	}
	return fmt.Sprintf("%d月%d日", int(date.Month()), date.Day())
}

func formatTimeRange(timeText string) string {
	if timeText == "" {
		return "时间待定"
	}
	return timeText
}

func formatDeadlineText(deadline *time.Time) string {
	if deadline == nil {
		return "报名截止待定"
	}
	return fmt.Sprintf("报名截止 %d月%d日", int(deadline.Month()), deadline.Day())
}

func defaultText(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func actionType(currentCount int, totalCount int) string {
	if currentCount >= totalCount {
		return "full"
	}
	return "signup"
}
