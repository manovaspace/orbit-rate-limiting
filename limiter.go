package ratelimiting

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/manovaspace/orbit-rate-limiting/internal/domain"
	"github.com/manovaspace/orbit-rate-limiting/internal/infrastructure/memory"
	redislimit "github.com/manovaspace/orbit-rate-limiting/internal/infrastructure/redis"
)

// Policy describes a token-bucket class.
type Policy struct {
	Class  string
	Limit  int
	Window time.Duration
}

// Decision is the result of Allow.
type Decision struct {
	Allowed    bool
	Limit      int
	Remaining  int
	ResetAt    time.Time
	RetryAfter time.Duration
}

// Limiter is the rate-limit port.
type Limiter interface {
	Allow(ctx context.Context, key string, policy Policy) (Decision, error)
}

// DefaultAuthPolicies returns ADR-018 v1 defaults.
func DefaultAuthPolicies() map[string]Policy {
	defaults := domain.DefaultAuthPolicies()
	out := make(map[string]Policy, len(defaults))
	for class, policy := range defaults {
		out[class] = fromDomainPolicy(policy)
	}
	return out
}

// OTPRequestIdentifierPolicy is the per-identifier OTP request limit.
func OTPRequestIdentifierPolicy() Policy {
	return fromDomainPolicy(domain.OTPRequestIdentifierPolicy())
}

// MemoryLimiter is a process-local token bucket for tests.
type MemoryLimiter struct {
	inner *memory.Limiter
}

func NewMemoryLimiter() *MemoryLimiter {
	return &MemoryLimiter{inner: memory.New()}
}

func (m *MemoryLimiter) Allow(_ context.Context, key string, policy Policy) (Decision, error) {
	// The memory bucket performs no I/O and the previous release ignored ctx.
	decision, err := m.inner.Allow(context.Background(), key, toDomainPolicy(policy))
	return fromDomainDecision(decision), err
}

// RedisLimiter implements Limiter with a Redis token bucket (Lua).
type RedisLimiter struct {
	inner *redislimit.Limiter
}

func NewRedisLimiter(client *redis.Client) *RedisLimiter {
	return &RedisLimiter{inner: redislimit.New(client)}
}

func (r *RedisLimiter) Allow(ctx context.Context, key string, policy Policy) (Decision, error) {
	decision, err := r.inner.Allow(ctx, key, toDomainPolicy(policy))
	return fromDomainDecision(decision), err
}

func toDomainPolicy(p Policy) domain.Policy {
	return domain.Policy{Class: p.Class, Limit: p.Limit, Window: p.Window}
}

func fromDomainPolicy(p domain.Policy) Policy {
	return Policy{Class: p.Class, Limit: p.Limit, Window: p.Window}
}

func fromDomainDecision(d domain.Decision) Decision {
	return Decision{
		Allowed:    d.Allowed,
		Limit:      d.Limit,
		Remaining:  d.Remaining,
		ResetAt:    d.ResetAt,
		RetryAfter: d.RetryAfter,
	}
}

var (
	_ Limiter = (*MemoryLimiter)(nil)
	_ Limiter = (*RedisLimiter)(nil)
)
