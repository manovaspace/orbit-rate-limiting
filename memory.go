package ratelimiting

import (
	"context"
	"sync"
	"time"
)

type memoryBucket struct {
	tokens    float64
	last      time.Time
	limit     int
	window    time.Duration
}

// MemoryLimiter is a process-local token bucket for tests.
type MemoryLimiter struct {
	mu      sync.Mutex
	buckets map[string]*memoryBucket
	now     func() time.Time
}

func NewMemoryLimiter() *MemoryLimiter {
	return &MemoryLimiter{
		buckets: make(map[string]*memoryBucket),
		now:     time.Now,
	}
}

func (m *MemoryLimiter) Allow(_ context.Context, key string, policy Policy) (Decision, error) {
	if policy.Limit <= 0 || policy.Window <= 0 {
		return Decision{Allowed: true, Limit: policy.Limit}, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	b, ok := m.buckets[key]
	if !ok || b.limit != policy.Limit || b.window != policy.Window {
		b = &memoryBucket{tokens: float64(policy.Limit), last: now, limit: policy.Limit, window: policy.Window}
		m.buckets[key] = b
	}
	elapsed := now.Sub(b.last).Seconds()
	rate := float64(policy.Limit) / policy.Window.Seconds()
	b.tokens = min(float64(policy.Limit), b.tokens+elapsed*rate)
	b.last = now
	resetAt := now.Add(policy.Window)
	if b.tokens < 1 {
		retry := time.Duration((1 - b.tokens) / rate * float64(time.Second))
		if retry < time.Second {
			retry = time.Second
		}
		return Decision{
			Allowed:    false,
			Limit:      policy.Limit,
			Remaining:  0,
			ResetAt:    resetAt,
			RetryAfter: retry,
		}, nil
	}
	b.tokens--
	return Decision{
		Allowed:   true,
		Limit:     policy.Limit,
		Remaining: int(b.tokens),
		ResetAt:   resetAt,
	}, nil
}
