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

// A full scan of the map happens at most once per sweepInterval, or once each
// time the map doubles past sweepSize. Buckets that are still inside their
// window are never evicted, so when every key is active the scans are
// doubling-spaced and the cost per Allow is amortised O(1).
const (
	// sweepInterval bounds how often idle buckets are swept when the map is
	// small. Sweeping is amortised inside Allow; no goroutine or timer runs.
	sweepInterval = time.Minute
	// sweepSize triggers a sweep regardless of sweepInterval once the map
	// holds this many buckets, so a flood of distinct keys cannot grow it
	// unboundedly between interval sweeps.
	sweepSize = 10_000
)

// Limiter is a process-local token bucket for tests and single processes.
// Idle buckets are evicted once they have refilled, so memory is bounded by
// the keys active within roughly one window (plus the sweep cadence).
type Limiter struct {
	mu        sync.Mutex
	buckets   map[string]*bucket
	now       func() time.Time
	lastSweep time.Time
	// sweptLen is the map size right after the last sweep. The size trigger
	// waits until the map has doubled past it (see sweep).
	sweptLen int
	// sweeps counts full scans; tests read it to verify the doubling bound.
	sweeps int
}

// New returns an empty in-memory limiter.
func New() *Limiter {
	return &Limiter{
		buckets: make(map[string]*bucket),
		now:     time.Now,
	}
}

// sweep deletes buckets that have refilled to their limit, which makes them
// indistinguishable from fresh buckets. It runs at most once per call, when
// sweepInterval has elapsed, or when the map holds at least sweepSize buckets
// and has at least doubled since the previous sweep. Caller must hold m.mu.
func (m *Limiter) sweep(now time.Time) {
	intervalDue := now.Sub(m.lastSweep) >= sweepInterval
	sizeDue := len(m.buckets) >= max(sweepSize, 2*m.sweptLen)
	if !intervalDue && !sizeDue {
		return
	}
	m.sweeps++
	for key, b := range m.buckets {
		if now.Sub(b.last) >= b.window {
			delete(m.buckets, key)
		}
	}
	m.sweptLen = len(m.buckets)
	m.lastSweep = now
}

// Allow consumes one token when the bucket has one.
func (m *Limiter) Allow(_ context.Context, key string, policy domain.Policy) (domain.Decision, error) {
	if decision, ok := application.Unlimited(policy); ok {
		return decision, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	m.sweep(now)
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
