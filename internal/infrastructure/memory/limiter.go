// Package memory is the process-local token-bucket adapter.
package memory

import (
	"context"
	"sync"
	"time"

	"github.com/manovaspace/orbit-rate-limiting/internal/application"
	"github.com/manovaspace/orbit-rate-limiting/internal/domain"
)

type bucket struct {
	tokens float64
	last   time.Time
	limit  int
	window time.Duration
}

// Limiter is a process-local token bucket for tests.
type Limiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	now     func() time.Time
}

// New returns an empty in-memory limiter.
func New() *Limiter {
	return &Limiter{
		buckets: make(map[string]*bucket),
		now:     time.Now,
	}
}

// Allow consumes one token when the bucket has one.
func (m *Limiter) Allow(_ context.Context, key string, policy domain.Policy) (domain.Decision, error) {
	if decision, ok := application.Unlimited(policy); ok {
		return decision, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	b, ok := m.buckets[key]
	if !ok || b.limit != policy.Limit || b.window != policy.Window {
		b = &bucket{tokens: float64(policy.Limit), last: now, limit: policy.Limit, window: policy.Window}
		m.buckets[key] = b
	}
	elapsed := now.Sub(b.last).Seconds()
	rate := float64(policy.Limit) / policy.Window.Seconds()
	b.tokens = min(float64(policy.Limit), b.tokens+elapsed*rate)
	b.last = now
	resetAt := now.Add(policy.Window)
	if b.tokens < 1 {
		retry := time.Duration((1 - b.tokens) / rate * float64(time.Second))
		retry = application.MinimumRetryAfter(retry)
		return domain.Decision{
			Allowed:    false,
			Limit:      policy.Limit,
			Remaining:  0,
			ResetAt:    resetAt,
			RetryAfter: retry,
		}, nil
	}
	b.tokens--
	return domain.Decision{
		Allowed:   true,
		Limit:     policy.Limit,
		Remaining: int(b.tokens),
		ResetAt:   resetAt,
	}, nil
}

var _ domain.Limiter = (*Limiter)(nil)
