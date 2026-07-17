package chat

import (
	"sync"
	"time"
)

const (
	perUserMessagesPerSecond = 5.0
	perUserMessageBurst      = 10.0
	globalMessagesPerSecond  = 200.0
	globalMessageBurst       = 400.0
)

type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
}

type messageRateLimiter struct {
	mu     sync.Mutex
	users  map[int64]tokenBucket
	global tokenBucket
}

func newMessageRateLimiter() *messageRateLimiter {
	now := time.Now()
	return &messageRateLimiter{
		users: make(map[int64]tokenBucket),
		global: tokenBucket{
			tokens:     globalMessageBurst,
			lastRefill: now,
		},
	}
}

func (l *messageRateLimiter) Allow(userID int64, now time.Time) bool {
	if l == nil {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	userBucket, exists := l.users[userID]
	if !exists {
		userBucket = tokenBucket{tokens: perUserMessageBurst, lastRefill: now}
	}
	refillBucket(&userBucket, now, perUserMessagesPerSecond, perUserMessageBurst)
	refillBucket(&l.global, now, globalMessagesPerSecond, globalMessageBurst)
	if userBucket.tokens < 1 || l.global.tokens < 1 {
		l.users[userID] = userBucket
		return false
	}

	userBucket.tokens--
	l.global.tokens--
	l.users[userID] = userBucket
	return true
}

func refillBucket(bucket *tokenBucket, now time.Time, rate float64, capacity float64) {
	if !now.After(bucket.lastRefill) {
		return
	}
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens += elapsed * rate
	if bucket.tokens > capacity {
		bucket.tokens = capacity
	}
	bucket.lastRefill = now
}
