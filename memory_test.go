package ratelimiting

import (
	"context"
	"testing"
	"time"
)

func TestMemoryLimiterAllowsThenBlocks(t *testing.T) {
	m := NewMemoryLimiter()
	fixed := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	m.now = func() time.Time { return fixed }
	p := Policy{Class: "auth_password", Limit: 2, Window: time.Minute}
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		d, err := m.Allow(ctx, "rl:auth_password:1.2.3.4", p)
		if err != nil || !d.Allowed {
			t.Fatalf("allow %d: allowed=%v err=%v", i, d.Allowed, err)
		}
	}
	d, err := m.Allow(ctx, "rl:auth_password:1.2.3.4", p)
	if err != nil {
		t.Fatal(err)
	}
	if d.Allowed {
		t.Fatal("expected deny")
	}
	if d.RetryAfter < time.Second {
		t.Fatalf("retry_after=%v", d.RetryAfter)
	}
}
