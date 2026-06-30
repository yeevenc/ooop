package user

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"ooop-admin-api/internal/auth"
	"ooop-admin-api/internal/logger"
	"ooop-admin-api/internal/provider"
)

const LoginCodeSceneLogin = "login"

var (
	ErrInvalidPhone     = errors.New("手机号格式不正确")
	ErrInvalidPassword  = errors.New("密码长度不能少于 8 位")
	ErrInvalidAccount   = errors.New("账号或密码错误")
	ErrInvalidOldPass   = errors.New("当前密码不正确")
	ErrInvalidCode      = errors.New("验证码错误或已过期")
	ErrDisabledUser     = errors.New("账号已被禁用")
	ErrPhoneExists      = errors.New("手机号已注册")
	ErrReservedUsername = errors.New("该用户名不可使用")
	ErrInvalidProfile   = errors.New("资料字段不合法")
	ErrInvalidRealName  = errors.New("实名信息格式不正确")
	ErrRealNameMismatch = errors.New("实名认证未通过")
)

var phonePattern = regexp.MustCompile(`^1[3-9]\d{9}$`)
var idCardPattern = regexp.MustCompile(`^\d{17}[\dXx]$`)

type AuthServiceOptions struct {
	Users            UserRepository
	Stats            UserStatsRepository
	LoginCodes       LoginCodeRepository
	PasswordHasher   auth.PasswordHasher
	TokenManager     *auth.TokenManager
	MobileVerifier   provider.MobileVerifier
	SMSSender        provider.SMSSender
	RealNameVerifier provider.RealNameVerifier
	CodeSecret       string
}

type AuthService struct {
	users            UserRepository
	stats            UserStatsRepository
	loginCodes       LoginCodeRepository
	passwordHasher   auth.PasswordHasher
	tokenManager     *auth.TokenManager
	mobileVerifier   provider.MobileVerifier
	smsSender        provider.SMSSender
	realNameVerifier provider.RealNameVerifier
	codeSecret       string
}

type LoginResult struct {
	User        PublicUser `json:"user"`
	Tokens      auth.Token `json:"tokens"`
	MaskedPhone string     `json:"masked_phone,omitempty"`
	Operator    string     `json:"operator,omitempty"`
}

type UserListQuery struct {
	Page     int
	PageSize int
	Keyword  string
	Status   *int
}

type UserListResult struct {
	List     []PublicUser `json:"list"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
}

type UserStatsRepository interface {
	CountPublishedByUser(ctx context.Context, userID int64, statuses []string) (int64, error)
	CountJoinedByUser(ctx context.Context, userID int64, statuses []string) (int64, error)
}

type ClientMeta struct {
	Platform string
	DeviceNo string
}

// ProfileUpdate 描述一次资料更新：仅非 nil 字段会被写入，便于做部分更新。
type ProfileUpdate struct {
	Nickname *string
	Gender   *string
	Region   *string
	Bio      *string
	Avatar   *string
}

// ProfileUpdateInput 是资料更新接口的请求体，APP 改自己与后台改指定用户共用同一组可改字段。
type ProfileUpdateInput struct {
	Nickname *string `json:"nickname"`
	Gender   *string `json:"gender"`
	Region   *string `json:"region"`
	Bio      *string `json:"bio"`
	Avatar   *string `json:"avatar"`
}

func (i ProfileUpdateInput) ToProfileUpdate() ProfileUpdate {
	return ProfileUpdate(i)
}

func NewAuthService(opts AuthServiceOptions) *AuthService {
	return &AuthService{
		users:            opts.Users,
		stats:            opts.Stats,
		loginCodes:       opts.LoginCodes,
		passwordHasher:   opts.PasswordHasher,
		tokenManager:     opts.TokenManager,
		mobileVerifier:   opts.MobileVerifier,
		smsSender:        opts.SMSSender,
		realNameVerifier: opts.RealNameVerifier,
		codeSecret:       opts.CodeSecret,
	}
}

func (s *AuthService) AliyunMobileLogin(ctx context.Context, accessToken string, meta ClientMeta) (LoginResult, error) {
	return s.mobileTokenLogin(ctx, accessToken, RegisterSourceAliyunMobile, "", meta)
}

func (s *AuthService) JiguangMobileLogin(ctx context.Context, loginToken string, operator string, meta ClientMeta) (LoginResult, error) {
	return s.mobileTokenLogin(ctx, loginToken, RegisterSourceJiguangMobile, operator, meta)
}

func (s *AuthService) mobileTokenLogin(ctx context.Context, token string, source string, operator string, meta ClientMeta) (LoginResult, error) {
	verifyResult, err := s.mobileVerifier.Verify(ctx, token)
	if err != nil {
		return LoginResult{}, err
	}
	phone := verifyResult.Phone
	if !isValidPhone(phone) {
		return LoginResult{}, ErrInvalidPhone
	}
	if verifyResult.Operator != "" {
		operator = verifyResult.Operator
	}
	result, err := s.loginOrCreateByPhone(ctx, phone, source, meta)
	if err != nil {
		return LoginResult{}, err
	}
	result.MaskedPhone = maskPhone(phone)
	result.Operator = normalizeOperator(operator)
	return result, nil
}

func (s *AuthService) SendLoginCode(ctx context.Context, phone string, scene provider.SMSScene) error {
	phone = normalizePhone(phone)
	if !isValidPhone(phone) {
		return ErrInvalidPhone
	}
	return s.smsSender.SendCode(ctx, phone, normalizeSMSScene(scene))
}

func (s *AuthService) MobileCodeLogin(ctx context.Context, phone string, code string, meta ClientMeta) (LoginResult, error) {
	phone = normalizePhone(phone)
	code = strings.TrimSpace(code)
	if !isValidPhone(phone) {
		return LoginResult{}, ErrInvalidPhone
	}
	if code == "" {
		return LoginResult{}, ErrInvalidCode
	}

	passed, err := s.smsSender.CheckCode(ctx, phone, provider.SMSSceneLogin, code)
	if err != nil {
		return LoginResult{}, err
	}
	if !passed {
		return LoginResult{}, ErrInvalidCode
	}
	return s.loginOrCreateByPhone(ctx, phone, RegisterSourceMobileCode, meta)
}

func (s *AuthService) CheckSMSCode(ctx context.Context, phone string, scene provider.SMSScene, code string) error {
	phone = normalizePhone(phone)
	code = strings.TrimSpace(code)
	if !isValidPhone(phone) {
		return ErrInvalidPhone
	}
	if code == "" {
		return ErrInvalidCode
	}
	passed, err := s.smsSender.CheckCode(ctx, phone, normalizeSMSScene(scene), code)
	if err != nil {
		return err
	}
	if !passed {
		return ErrInvalidCode
	}
	return nil
}

func (s *AuthService) RegisterByPassword(ctx context.Context, phone string, username string, password string, meta ClientMeta) (LoginResult, error) {
	phone = normalizePhone(phone)
	username = strings.TrimSpace(username)
	if !isValidPhone(phone) {
		return LoginResult{}, ErrInvalidPhone
	}
	if len(password) < 8 {
		return LoginResult{}, ErrInvalidPassword
	}
	if isReservedUsername(username) {
		return LoginResult{}, ErrReservedUsername
	}

	_, err := s.users.FindByPhone(ctx, phone)
	if err == nil {
		return LoginResult{}, ErrPhoneExists
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		return LoginResult{}, err
	}

	passwordHash, err := s.passwordHasher.Hash(password)
	if err != nil {
		return LoginResult{}, err
	}

	item := User{
		Phone:          phone,
		Nickname:       username,
		Status:         UserStatusEnabled,
		RegisterSource: RegisterSourcePassword,
		PasswordHash:   passwordHash,
		Platform:       normalizeMetaValue(meta.Platform),
		DeviceNo:       normalizeMetaValue(meta.DeviceNo),
	}
	if username != "" {
		item.Username = &username
	}

	if err := s.users.Create(ctx, &item); err != nil {
		return LoginResult{}, err
	}
	return s.issueLoginResult(ctx, item, meta)
}

func (s *AuthService) PasswordLogin(ctx context.Context, account string, password string, meta ClientMeta) (LoginResult, error) {
	account = strings.TrimSpace(account)
	if account == "" || password == "" {
		return LoginResult{}, ErrInvalidAccount
	}

	item, err := s.users.FindByUsernameOrPhone(ctx, account)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return LoginResult{}, ErrInvalidAccount
		}
		return LoginResult{}, err
	}
	if item.Status != UserStatusEnabled {
		return LoginResult{}, ErrDisabledUser
	}
	if item.PasswordHash == "" || !s.passwordHasher.Compare(item.PasswordHash, password) {
		return LoginResult{}, ErrInvalidAccount
	}
	return s.issueLoginResult(ctx, item, meta)
}

func (s *AuthService) SetPassword(ctx context.Context, userID int64, username string, oldPassword string, password string) (PublicUser, error) {
	username = strings.TrimSpace(username)
	oldPassword = strings.TrimSpace(oldPassword)
	if len(password) < 8 {
		return PublicUser{}, ErrInvalidPassword
	}
	if isReservedUsername(username) {
		return PublicUser{}, ErrReservedUsername
	}
	item, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return PublicUser{}, err
	}
	if item.PasswordHash != "" {
		if oldPassword == "" || !s.passwordHasher.Compare(item.PasswordHash, oldPassword) {
			return PublicUser{}, ErrInvalidOldPass
		}
	}
	hash, err := s.passwordHasher.Hash(password)
	if err != nil {
		return PublicUser{}, err
	}
	if err := s.users.UpdatePassword(ctx, userID, username, hash); err != nil {
		return PublicUser{}, err
	}
	item, err = s.users.FindByID(ctx, userID)
	if err != nil {
		return PublicUser{}, err
	}
	return ToPublicUser(item), nil
}

func (s *AuthService) ChangePhone(ctx context.Context, userID int64, newPhone string, code string) (PublicUser, error) {
	newPhone = normalizePhone(newPhone)
	code = strings.TrimSpace(code)
	if !isValidPhone(newPhone) {
		return PublicUser{}, ErrInvalidPhone
	}
	if code == "" {
		return PublicUser{}, ErrInvalidCode
	}
	current, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return PublicUser{}, err
	}
	if current.Phone == newPhone {
		return ToPublicUser(current), nil
	}
	if _, err := s.users.FindByPhone(ctx, newPhone); err == nil {
		return PublicUser{}, ErrPhoneExists
	} else if !errors.Is(err, ErrNotFound) {
		return PublicUser{}, err
	}
	passed, err := s.smsSender.CheckCode(ctx, newPhone, provider.SMSSceneBindNewPhone, code)
	if err != nil {
		return PublicUser{}, err
	}
	if !passed {
		return PublicUser{}, ErrInvalidCode
	}
	if err := s.users.UpdatePhone(ctx, userID, newPhone); err != nil {
		return PublicUser{}, err
	}
	current.Phone = newPhone
	return ToPublicUser(current), nil
}

func (s *AuthService) Profile(ctx context.Context, userID int64) (PublicUser, error) {
	item, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return PublicUser{}, err
	}
	stats, err := s.userStats(ctx, userID, false)
	if err != nil {
		return PublicUser{}, err
	}
	return ToPublicUserWithStats(item, stats), nil
}

// PublicProfile 返回他人可见的用户资料安全子集（用于用户主页展示）。
func (s *AuthService) PublicProfile(ctx context.Context, userID int64) (UserPublicProfile, error) {
	item, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return UserPublicProfile{}, err
	}
	stats, err := s.userStats(ctx, userID, true)
	if err != nil {
		return UserPublicProfile{}, err
	}
	return ToUserPublicProfileWithStats(item, stats), nil
}

func (s *AuthService) userStats(ctx context.Context, userID int64, publicOnly bool) (UserStats, error) {
	if s.stats == nil {
		return UserStats{}, nil
	}

	var publishedStatuses []string
	joinedStatuses := []string{"joined", "approved", "rejected"}
	if publicOnly {
		publishedStatuses = []string{"ongoing"}
		joinedStatuses = []string{"approved"}
	}

	publishedCount, err := s.stats.CountPublishedByUser(ctx, userID, publishedStatuses)
	if err != nil {
		return UserStats{}, err
	}
	joinedCount, err := s.stats.CountJoinedByUser(ctx, userID, joinedStatuses)
	if err != nil {
		return UserStats{}, err
	}

	return UserStats{
		PublishedCount: int(publishedCount),
		JoinedCount:    int(joinedCount),
		// 当前没有点赞表或获赞字段，先随用户信息返回 0，后续接点赞体系时只需替换这里的来源。
		LikedCount: 0,
	}, nil
}

// UpdateProfile 更新指定用户的资料字段（昵称/性别/地区/简介），APP 端改自己、后台改指定用户共用。
func (s *AuthService) UpdateProfile(ctx context.Context, userID int64, update ProfileUpdate) (PublicUser, error) {
	normalized, err := normalizeProfileUpdate(update)
	if err != nil {
		return PublicUser{}, err
	}
	if err := s.users.UpdateProfile(ctx, userID, normalized); err != nil {
		return PublicUser{}, err
	}
	item, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return PublicUser{}, err
	}
	stats, err := s.userStats(ctx, userID, false)
	if err != nil {
		return PublicUser{}, err
	}
	return ToPublicUserWithStats(item, stats), nil
}

func (s *AuthService) BindPushRegistration(ctx context.Context, userID int64, platform string, registrationID string) error {
	registrationID = strings.TrimSpace(registrationID)
	platform = normalizeMetaValue(platform)

	if registrationID == "" {
		logger.Warnf("绑定 push registration 失败: user_id=%d, registration_id 为空", userID)
		return ErrInvalidProfile
	}

	_, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := s.users.UpdatePushRegistration(ctx, userID, platform, registrationID); err != nil {
		return err
	}
	logger.Infof(
		"绑定 push registration 成功: user_id=%d, platform=%s, registration_id=%s",
		userID,
		platform,
		registrationID,
	)
	return nil
}

func (s *AuthService) VerifyRealName(ctx context.Context, userID int64, name string, idCard string) (PublicUser, error) {
	name = strings.TrimSpace(name)
	idCard = strings.ToUpper(strings.TrimSpace(idCard))
	if !isValidRealName(name) || !idCardPattern.MatchString(idCard) {
		return PublicUser{}, ErrInvalidRealName
	}

	item, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return PublicUser{}, err
	}
	if item.RealNameVerified {
		stats, err := s.userStats(ctx, userID, false)
		if err != nil {
			return PublicUser{}, err
		}
		return ToPublicUserWithStats(item, stats), nil
	}
	if s.realNameVerifier == nil {
		return PublicUser{}, errors.New("实名认证服务未配置")
	}

	result, err := s.realNameVerifier.Verify(ctx, name, idCard)
	if err != nil {
		return PublicUser{}, err
	}
	if !result.Passed {
		message := strings.TrimSpace(result.Message)
		if message == "" {
			message = "姓名和身份证号不匹配"
		}
		return PublicUser{}, fmt.Errorf("%w: %s", ErrRealNameMismatch, message)
	}

	now := time.Now()
	if err := s.users.UpdateRealNameVerification(ctx, userID, name, maskIDCard(idCard), now); err != nil {
		return PublicUser{}, err
	}
	item, err = s.users.FindByID(ctx, userID)
	if err != nil {
		return PublicUser{}, err
	}
	stats, err := s.userStats(ctx, userID, false)
	if err != nil {
		return PublicUser{}, err
	}
	return ToPublicUserWithStats(item, stats), nil
}

func (s *AuthService) ListUsers(ctx context.Context, query UserListQuery) (UserListResult, error) {
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

	items, total, err := s.users.List(ctx, query)
	if err != nil {
		return UserListResult{}, err
	}

	list := make([]PublicUser, 0, len(items))
	for _, item := range items {
		list = append(list, ToPublicUser(item))
	}

	return UserListResult{
		List:     list,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

func (s *AuthService) loginOrCreateByPhone(ctx context.Context, phone string, source string, meta ClientMeta) (LoginResult, error) {
	item, err := s.users.FindByPhone(ctx, phone)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return LoginResult{}, err
	}
	if errors.Is(err, ErrNotFound) {
		item = User{
			Phone:          phone,
			Status:         UserStatusEnabled,
			RegisterSource: source,
			Platform:       normalizeMetaValue(meta.Platform),
			DeviceNo:       normalizeMetaValue(meta.DeviceNo),
		}
		if err := s.users.Create(ctx, &item); err != nil {
			return LoginResult{}, err
		}
	}
	return s.issueLoginResult(ctx, item, meta)
}

func (s *AuthService) issueLoginResult(ctx context.Context, item User, meta ClientMeta) (LoginResult, error) {
	if item.Status != UserStatusEnabled {
		return LoginResult{}, ErrDisabledUser
	}

	now := time.Now()
	if err := s.users.TouchLastLogin(ctx, item.ID, now, meta); err != nil {
		return LoginResult{}, err
	}
	item.LastLoginAt = &now
	if meta.Platform != "" {
		item.Platform = normalizeMetaValue(meta.Platform)
	}
	if meta.DeviceNo != "" {
		item.DeviceNo = normalizeMetaValue(meta.DeviceNo)
	}

	tokens, err := s.tokenManager.NewToken(item.ID)
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{
		User:   ToPublicUser(item),
		Tokens: tokens,
	}, nil
}

func (s *AuthService) hashCode(phone string, code string) string {
	sum := sha256.Sum256([]byte(phone + ":" + code + ":" + s.codeSecret))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func normalizePhone(phone string) string {
	return strings.TrimSpace(phone)
}

func normalizeMetaValue(value string) string {
	return strings.TrimSpace(value)
}

func maskPhone(phone string) string {
	phone = strings.TrimSpace(phone)
	if len(phone) < 7 {
		return phone
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

func normalizeOperator(operator string) string {
	operator = strings.ToUpper(strings.TrimSpace(operator))
	switch operator {
	case "CM", "CU", "CT", "CMHK":
		return operator
	default:
		return ""
	}
}

func normalizeSMSScene(scene provider.SMSScene) provider.SMSScene {
	switch scene {
	case provider.SMSSceneChangePhone,
		provider.SMSSceneResetPassword,
		provider.SMSSceneBindNewPhone,
		provider.SMSSceneVerifyBindPhone:
		return scene
	default:
		return provider.SMSSceneLogin
	}
}

// normalizeProfileUpdate 对传入的资料字段去空白并做长度校验，长度上限与数据表列宽一致。
func normalizeProfileUpdate(update ProfileUpdate) (ProfileUpdate, error) {
	if update.Nickname != nil {
		nickname := strings.TrimSpace(*update.Nickname)
		if nickname == "" || utf8.RuneCountInString(nickname) > 32 {
			return ProfileUpdate{}, ErrInvalidProfile
		}
		update.Nickname = &nickname
	}
	if update.Gender != nil {
		gender := strings.TrimSpace(*update.Gender)
		if utf8.RuneCountInString(gender) > 16 {
			return ProfileUpdate{}, ErrInvalidProfile
		}
		update.Gender = &gender
	}
	if update.Region != nil {
		region := strings.TrimSpace(*update.Region)
		if utf8.RuneCountInString(region) > 64 {
			return ProfileUpdate{}, ErrInvalidProfile
		}
		update.Region = &region
	}
	if update.Bio != nil {
		bio := strings.TrimSpace(*update.Bio)
		if utf8.RuneCountInString(bio) > 200 {
			return ProfileUpdate{}, ErrInvalidProfile
		}
		update.Bio = &bio
	}
	if update.Avatar != nil {
		avatar := strings.TrimSpace(*update.Avatar)
		if utf8.RuneCountInString(avatar) > 255 {
			return ProfileUpdate{}, ErrInvalidProfile
		}
		update.Avatar = &avatar
	}
	return update, nil
}

// profileUpdateColumns 把资料更新转成仅含非 nil 字段的列映射，供 GORM 部分更新使用。
func profileUpdateColumns(update ProfileUpdate) map[string]interface{} {
	columns := map[string]interface{}{}
	if update.Nickname != nil {
		columns["nickname"] = *update.Nickname
	}
	if update.Gender != nil {
		columns["gender"] = *update.Gender
	}
	if update.Region != nil {
		columns["region"] = *update.Region
	}
	if update.Bio != nil {
		columns["bio"] = *update.Bio
	}
	if update.Avatar != nil {
		columns["avatar"] = *update.Avatar
	}
	return columns
}

func isReservedUsername(username string) bool {
	return strings.EqualFold(strings.TrimSpace(username), ReservedAdminUsername)
}

func isValidPhone(phone string) bool {
	return phonePattern.MatchString(phone)
}

func isValidRealName(name string) bool {
	count := utf8.RuneCountInString(name)
	return count >= 2 && count <= 32
}

func maskIDCard(value string) string {
	value = strings.TrimSpace(value)
	if len(value) < 8 {
		return value
	}
	return value[:6] + "********" + value[len(value)-4:]
}

func randomCode() string {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	}
	value := int(bytes[0])<<24 | int(bytes[1])<<16 | int(bytes[2])<<8 | int(bytes[3])
	if value < 0 {
		value = -value
	}
	return fmt.Sprintf("%06d", value%1000000)
}
