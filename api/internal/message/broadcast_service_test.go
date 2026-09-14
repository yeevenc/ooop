package message

import (
	"context"
	"sort"
	"sync"
	"testing"
	"time"

	"ooop-admin-api/internal/provider"
	"ooop-admin-api/internal/user"
)

type broadcastTestMessages struct {
	pushTestRepository
	mu     sync.Mutex
	nextID int64
	items  map[int64]UserMessage
	keys   map[string]int64
}

func newBroadcastTestMessages() *broadcastTestMessages {
	return &broadcastTestMessages{
		nextID: 1,
		items:  map[int64]UserMessage{},
		keys:   map[string]int64{},
	}
}

func (r *broadcastTestMessages) Create(_ context.Context, item *UserMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item.ID = r.nextID
	r.nextID++
	r.items[item.ID] = *item
	if item.IdempotencyKey != nil {
		r.keys[*item.IdempotencyKey] = item.ID
	}
	return nil
}

func (r *broadcastTestMessages) CreateIdempotent(ctx context.Context, item *UserMessage) error {
	r.mu.Lock()
	if item.IdempotencyKey != nil {
		if id, ok := r.keys[*item.IdempotencyKey]; ok {
			existing := r.items[id]
			r.mu.Unlock()
			*item = existing
			return nil
		}
	}
	r.mu.Unlock()
	return r.Create(ctx, item)
}

func (r *broadcastTestMessages) list() []UserMessage {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := make([]UserMessage, 0, len(r.items))
	for _, item := range r.items {
		items = append(items, item)
	}
	return items
}

type broadcastTestUsers struct {
	user.UserRepository
	items map[int64]user.User
}

func (r *broadcastTestUsers) FindByID(_ context.Context, id int64) (user.User, error) {
	item, ok := r.items[id]
	if !ok {
		return user.User{}, user.ErrNotFound
	}
	return item, nil
}

func (r *broadcastTestUsers) CountEnabledForBroadcast(ctx context.Context) (int64, error) {
	items, err := r.ListEnabledForBroadcast(ctx, 0, 1<<20)
	if err != nil {
		return 0, err
	}
	return int64(len(items)), nil
}

func (r *broadcastTestUsers) ListEnabledForBroadcast(_ context.Context, afterID int64, limit int) ([]user.User, error) {
	list := make([]user.User, 0, len(r.items))
	for _, item := range r.items {
		if item.ID <= afterID || item.Status != user.UserStatusEnabled {
			continue
		}
		if item.Username != nil && *item.Username == user.ReservedAdminUsername {
			continue
		}
		list = append(list, item)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].ID < list[j].ID
	})
	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list, nil
}

type memoryBroadcasts struct {
	mu     sync.Mutex
	nextID int64
	items  map[int64]SystemBroadcast
}

func newMemoryBroadcasts() *memoryBroadcasts {
	return &memoryBroadcasts{
		nextID: 1,
		items:  map[int64]SystemBroadcast{},
	}
}

func (r *memoryBroadcasts) Create(_ context.Context, item *SystemBroadcast) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item.ID = r.nextID
	r.nextID++
	item.CreatedAt = time.Now()
	item.UpdatedAt = item.CreatedAt
	r.items[item.ID] = *item
	return nil
}

func (r *memoryBroadcasts) ClaimNext(_ context.Context, now time.Time) (*SystemBroadcast, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var found *SystemBroadcast
	for id := int64(1); id < r.nextID; id++ {
		item, ok := r.items[id]
		if !ok {
			continue
		}
		if item.Status != BroadcastStatusPending && item.Status != BroadcastStatusRunning {
			continue
		}
		if item.Status == BroadcastStatusPending {
			item.Status = BroadcastStatusRunning
			startedAt := now
			item.StartedAt = &startedAt
			r.items[id] = item
		}
		copyItem := r.items[id]
		found = &copyItem
		break
	}
	return found, nil
}

func (r *memoryBroadcasts) SaveProgress(_ context.Context, item *SystemBroadcast) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item.UpdatedAt = time.Now()
	r.items[item.ID] = *item
	return nil
}

func (r *memoryBroadcasts) List(_ context.Context, query BroadcastQuery) ([]SystemBroadcast, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var list []SystemBroadcast
	for id := r.nextID - 1; id >= 1; id-- {
		item, ok := r.items[id]
		if !ok {
			continue
		}
		if query.Status != "" && item.Status != query.Status {
			continue
		}
		list = append(list, item)
	}
	total := int64(len(list))
	return list, total, nil
}

func (r *memoryBroadcasts) get(id int64) SystemBroadcast {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.items[id]
}

type recordingPusher struct {
	mu       sync.Mutex
	payloads []provider.PushPayload
}

func (s *recordingPusher) Push(_ context.Context, payload provider.PushPayload) (provider.PushResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.payloads = append(s.payloads, payload)
	return provider.PushResult{Triggered: true, Success: true, Alias: payload.Alias}, nil
}

func newBroadcastService(users *broadcastTestUsers, pusher PushSender) (*Service, *broadcastTestMessages, *memoryBroadcasts) {
	messages := newBroadcastTestMessages()
	broadcasts := newMemoryBroadcasts()
	service := NewService(messages, pusher, users)
	service.SetBroadcasts(broadcasts, 50)
	return service, messages, broadcasts
}

func TestCreateSystemBroadcastRejectsEmptyTitle(t *testing.T) {
	service, _, _ := newBroadcastService(&broadcastTestUsers{items: map[int64]user.User{}}, nil)
	_, err := service.CreateSystemBroadcast(context.Background(), 1, "  ", "内容")
	if err != ErrInvalidBroadcastTitle {
		t.Fatalf("error = %v, want %v", err, ErrInvalidBroadcastTitle)
	}
}

func TestCreateSystemBroadcastRejectsEmptyContent(t *testing.T) {
	service, _, _ := newBroadcastService(&broadcastTestUsers{items: map[int64]user.User{}}, nil)
	_, err := service.CreateSystemBroadcast(context.Background(), 1, "标题", "   ")
	if err != ErrInvalidBroadcastContent {
		t.Fatalf("error = %v, want %v", err, ErrInvalidBroadcastContent)
	}
}

func TestCreateSystemBroadcastCountsEnabledUsers(t *testing.T) {
	adminName := user.ReservedAdminUsername
	users := &broadcastTestUsers{items: map[int64]user.User{
		3001: {ID: 3001, Status: user.UserStatusEnabled},
		3002: {ID: 3002, Status: user.UserStatusDisabled},
		3003: {ID: 3003, Status: user.UserStatusEnabled, Username: &adminName},
		3004: {ID: 3004, Status: user.UserStatusEnabled},
	}}
	service, _, _ := newBroadcastService(users, nil)

	result, err := service.CreateSystemBroadcast(context.Background(), 8, "系统维护", "今晚 23:00 进行维护")
	if err != nil {
		t.Fatalf("CreateSystemBroadcast() error = %v", err)
	}
	if result.TargetCount != 2 {
		t.Fatalf("targetCount = %d, want 2", result.TargetCount)
	}
	if result.Status != BroadcastStatusPending {
		t.Fatalf("status = %s, want %s", result.Status, BroadcastStatusPending)
	}
}

func TestProcessNextBroadcastWritesMessagesAndPushes(t *testing.T) {
	users := &broadcastTestUsers{items: map[int64]user.User{
		3001: {ID: 3001, Status: user.UserStatusEnabled, RegistrationID: "rid-1", HarmonyPushToken: "token-1"},
		3002: {ID: 3002, Status: user.UserStatusEnabled, NotificationDisabled: true, HarmonyPushToken: "token-2"},
		3003: {ID: 3003, Status: user.UserStatusDisabled, HarmonyPushToken: "token-3"},
	}}
	pusher := &recordingPusher{}
	service, messages, broadcasts := newBroadcastService(users, pusher)

	created, err := service.CreateSystemBroadcast(context.Background(), 8, "活动上线", "新版本已发布")
	if err != nil {
		t.Fatalf("CreateSystemBroadcast() error = %v", err)
	}

	service.ProcessNextBroadcast(context.Background())
	service.ProcessNextBroadcast(context.Background())

	item := broadcasts.get(1)
	if item.Status != BroadcastStatusCompleted {
		t.Fatalf("status = %s, want %s", item.Status, BroadcastStatusCompleted)
	}
	if item.MessageCount != 2 {
		t.Fatalf("messageCount = %d, want 2", item.MessageCount)
	}
	if item.PushSuccess != 1 || item.PushSkipped != 1 || item.PushFailed != 0 {
		t.Fatalf("push counters success=%d skipped=%d failed=%d", item.PushSuccess, item.PushSkipped, item.PushFailed)
	}

	stored := messages.list()
	if len(stored) != 2 {
		t.Fatalf("messages = %d, want 2", len(stored))
	}
	for _, msg := range stored {
		if msg.Type != TypeSystem {
			t.Fatalf("message type = %s, want %s", msg.Type, TypeSystem)
		}
		if msg.Title != "活动上线" {
			t.Fatalf("title = %s", msg.Title)
		}
	}
	if len(pusher.payloads) != 1 {
		t.Fatalf("push count = %d, want 1", len(pusher.payloads))
	}
	if pusher.payloads[0].Alias != "3001" || pusher.payloads[0].MessageType != TypeSystem {
		t.Fatalf("payload = %+v", pusher.payloads[0])
	}
	if pusher.payloads[0].ActivityID != 0 {
		t.Fatalf("activityId should be empty for system broadcast")
	}
	if created.ID != "1" {
		t.Fatalf("created id = %s", created.ID)
	}
}

func TestProcessNextBroadcastIsIdempotent(t *testing.T) {
	users := &broadcastTestUsers{items: map[int64]user.User{
		3001: {ID: 3001, Status: user.UserStatusEnabled},
	}}
	pusher := &recordingPusher{}
	service, messages, broadcasts := newBroadcastService(users, pusher)
	if _, err := service.CreateSystemBroadcast(context.Background(), 8, "重复发送", "只应写入一次"); err != nil {
		t.Fatalf("CreateSystemBroadcast() error = %v", err)
	}

	service.ProcessNextBroadcast(context.Background())
	item := broadcasts.get(1)
	item.Status = BroadcastStatusPending
	item.LastUserID = 0
	item.MessageCount = 0
	item.PushSuccess = 0
	_ = broadcasts.SaveProgress(context.Background(), &item)

	service.ProcessNextBroadcast(context.Background())
	if len(messages.list()) != 1 {
		t.Fatalf("messages = %d, want 1", len(messages.list()))
	}
}

func TestListSystemBroadcastsReturnsCreatedItems(t *testing.T) {
	service, _, _ := newBroadcastService(&broadcastTestUsers{items: map[int64]user.User{
		3001: {ID: 3001, Status: user.UserStatusEnabled},
	}}, nil)
	if _, err := service.CreateSystemBroadcast(context.Background(), 8, "标题", "内容"); err != nil {
		t.Fatalf("CreateSystemBroadcast() error = %v", err)
	}

	result, err := service.ListSystemBroadcasts(context.Background(), BroadcastQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListSystemBroadcasts() error = %v", err)
	}
	if result.Total != 1 || len(result.List) != 1 {
		t.Fatalf("result = %+v", result)
	}
	if result.List[0].Title != "标题" {
		t.Fatalf("title = %s", result.List[0].Title)
	}
}
