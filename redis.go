package ratelimiting

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisLimiter implements Limiter with a Redis token bucket (Lua).
type RedisLimiter struct {
	client *redis.Client
	now    func() time.Time
}

func NewRedisLimiter(client *redis.Client) *RedisLimiter {
	return &RedisLimiter{client: client, now: time.Now}
}

// tokenBucketScript: KEYS[1]=key ARGV[1]=limit ARGV[2]=window_ms ARGV[3]=now_ms
// Returns: allowed (0/1), remaining, retry_after_ms
var tokenBucketScript = redis.NewScript(`
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window_ms = tonumber(ARGV[2])
local now_ms = tonumber(ARGV[3])
local rate = limit / window_ms

local data = redis.call('HMGET', key, 'tokens', 'ts')
local tokens = tonumber(data[1])
local ts = tonumber(data[2])
if tokens == nil then
  tokens = limit
  ts = now_ms
end
local elapsed = math.max(0, now_ms - ts)
tokens = math.min(limit, tokens + elapsed * rate)
local allowed = 0
local retry_ms = 0
if tokens >= 1 then
  tokens = tokens - 1
  allowed = 1
else
  retry_ms = math.ceil((1 - tokens) / rate)
  if retry_ms < 1000 then retry_ms = 1000 end
end
redis.call('HMSET', key, 'tokens', tokens, 'ts', now_ms)
redis.call('PEXPIRE', key, window_ms)
local remaining = math.floor(tokens)
return {allowed, remaining, retry_ms}
`)

func (r *RedisLimiter) Allow(ctx context.Context, key string, policy Policy) (Decision, error) {
	if policy.Limit <= 0 || policy.Window <= 0 {
		return Decision{Allowed: true, Limit: policy.Limit}, nil
	}
	now := r.now()
	windowMS := policy.Window.Milliseconds()
	if windowMS < 1 {
		windowMS = 1
	}
	res, err := tokenBucketScript.Run(ctx, r.client, []string{key},
		policy.Limit, windowMS, now.UnixMilli()).Slice()
	if err != nil {
		return Decision{}, fmt.Errorf("redis rate limit: %w", err)
	}
	allowed := toInt64(res[0]) == 1
	remaining := int(toInt64(res[1]))
	retryMS := toInt64(res[2])
	d := Decision{
		Allowed:   allowed,
		Limit:     policy.Limit,
		Remaining: remaining,
		ResetAt:   now.Add(policy.Window),
	}
	if !allowed {
		d.RetryAfter = time.Duration(retryMS) * time.Millisecond
		d.Remaining = 0
	}
	return d, nil
}

func toInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case string:
		var x int64
		_, _ = fmt.Sscan(n, &x)
		return x
	default:
		return 0
	}
}
