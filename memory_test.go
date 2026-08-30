package ratelimiting

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
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

func TestMemoryLimiter_KeyIsolation(t *testing.T) {
	m := NewMemoryLimiter()
	fixed := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	m.now = func() time.Time { return fixed }
	p := Policy{Class: "auth_password", Limit: 3, Window: time.Minute}
	ctx := context.Background()

	keyA := "rl:auth_password:user_a"
	keyB := "rl:auth_password:user_b"

	// Exhaust Key A
	for i := 0; i < 3; i++ {
		d, err := m.Allow(ctx, keyA, p)
		if err != nil {
			t.Fatalf("keyA allow %d unexpected error: %v", i, err)
		}
		if !d.Allowed {
			t.Fatalf("keyA allow %d expected true, got false", i)
		}
		if d.Remaining != 2-i {
			t.Fatalf("keyA allow %d expected remaining %d, got %d", i, 2-i, d.Remaining)
		}
	}

	// 4th request on Key A should be blocked
	dA, err := m.Allow(ctx, keyA, p)
	if err != nil {
		t.Fatalf("keyA 4th allow unexpected error: %v", err)
	}
	if dA.Allowed {
		t.Fatalf("keyA 4th allow expected false, got true")
	}
	if dA.Remaining != 0 {
		t.Fatalf("keyA 4th allow expected remaining 0, got %d", dA.Remaining)
	}

	// Key B should remain completely unblocked and have full capacity
	for i := 0; i < 3; i++ {
		dB, err := m.Allow(ctx, keyB, p)
		if err != nil {
			t.Fatalf("keyB allow %d unexpected error: %v", i, err)
		}
		if !dB.Allowed {
			t.Fatalf("keyB allow %d expected true, got false (isolation violated)", i)
		}
		if dB.Remaining != 2-i {
			t.Fatalf("keyB allow %d expected remaining %d, got %d", i, 2-i, dB.Remaining)
		}
	}

	// 4th request on Key B should be blocked
	dB, err := m.Allow(ctx, keyB, p)
	if err != nil {
		t.Fatalf("keyB 4th allow unexpected error: %v", err)
	}
	if dB.Allowed {
		t.Fatalf("keyB 4th allow expected false, got true")
	}
}

func TestMemoryLimiter_WindowExpiration(t *testing.T) {
	m := NewMemoryLimiter()
	var (
		mu      sync.Mutex
		simTime = time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	)
	m.now = func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return simTime
	}

	advanceSimTime := func(d time.Duration) {
		mu.Lock()
		simTime = simTime.Add(d)
		mu.Unlock()
	}

	ctx := context.Background()
	p := Policy{Class: "auth_otp_request", Limit: 2, Window: 10 * time.Second}
	key := "rl:auth_otp:192.0.2.1"

	// T0: Consume all 2 tokens
	d1, err := m.Allow(ctx, key, p)
	if err != nil || !d1.Allowed || d1.Remaining != 1 {
		t.Fatalf("T0 request 1 failed: allowed=%v, remaining=%d, err=%v", d1.Allowed, d1.Remaining, err)
	}

	d2, err := m.Allow(ctx, key, p)
	if err != nil || !d2.Allowed || d2.Remaining != 0 {
		t.Fatalf("T0 request 2 failed: allowed=%v, remaining=%d, err=%v", d2.Allowed, d2.Remaining, err)
	}

	// T0: 3rd request must be denied
	d3, err := m.Allow(ctx, key, p)
	if err != nil || d3.Allowed {
		t.Fatalf("T0 request 3 expected deny, got allowed=%v, err=%v", d3.Allowed, err)
	}
	if d3.RetryAfter <= 0 {
		t.Fatalf("T0 request 3 expected positive retry after, got %v", d3.RetryAfter)
	}

	// Advance time by half the window (5s). With Limit=2 over 10s (0.2 tokens/sec), 5s replenishes 1 token.
	advanceSimTime(5 * time.Second)

	d4, err := m.Allow(ctx, key, p)
	if err != nil || !d4.Allowed {
		t.Fatalf("T+5s request 4 expected allowed after partial replenishment, got allowed=%v, err=%v", d4.Allowed, err)
	}

	// Immediately next request at T+5s should be denied (1 token consumed)
	d5, err := m.Allow(ctx, key, p)
	if err != nil || d5.Allowed {
		t.Fatalf("T+5s request 5 expected deny, got allowed=%v, err=%v", d5.Allowed, err)
	}

	// Advance time past the entire window (e.g. 15s more)
	advanceSimTime(15 * time.Second)

	// Now full capacity (2 tokens) should be replenished
	d6, err := m.Allow(ctx, key, p)
	if err != nil || !d6.Allowed || d6.Remaining != 1 {
		t.Fatalf("T+20s request 6 expected allowed, got allowed=%v, remaining=%d, err=%v", d6.Allowed, d6.Remaining, err)
	}

	d7, err := m.Allow(ctx, key, p)
	if err != nil || !d7.Allowed || d7.Remaining != 0 {
		t.Fatalf("T+20s request 7 expected allowed, got allowed=%v, remaining=%d, err=%v", d7.Allowed, d7.Remaining, err)
	}

	d8, err := m.Allow(ctx, key, p)
	if err != nil || d8.Allowed {
		t.Fatalf("T+20s request 8 expected deny, got allowed=%v, err=%v", d8.Allowed, err)
	}

	// Test zero / negative limit and window bypass
	unlimitedP := Policy{Class: "unlimited", Limit: 0, Window: time.Minute}
	dUnl, err := m.Allow(ctx, "unlimited_key", unlimitedP)
	if err != nil || !dUnl.Allowed {
		t.Fatalf("zero limit policy should be allowed, got %v, err=%v", dUnl.Allowed, err)
	}
}

func TestMemoryLimiter_Concurrent(t *testing.T) {
	m := NewMemoryLimiter()
	fixed := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	m.now = func() time.Time { return fixed }

	const (
		limit         = 50
		numGoroutines = 200
	)
	p := Policy{Class: "concurrent_test", Limit: limit, Window: time.Minute}
	ctx := context.Background()

	var (
		wg           sync.WaitGroup
		startBarrier = make(chan struct{})
		allowedCount atomic.Int64
		deniedCount  atomic.Int64
		errCount     atomic.Int64
	)

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			<-startBarrier
			d, err := m.Allow(ctx, "rl:concurrent:single_key", p)
			if err != nil {
				errCount.Add(1)
				return
			}
			if d.Allowed {
				allowedCount.Add(1)
			} else {
				deniedCount.Add(1)
			}
		}()
	}

	close(startBarrier)
	wg.Wait()

	if errCount.Load() > 0 {
		t.Fatalf("encountered %d errors during concurrent requests", errCount.Load())
	}
	if allowedCount.Load() != limit {
		t.Fatalf("expected exactly %d allowed requests, got %d", limit, allowedCount.Load())
	}
	if deniedCount.Load() != (numGoroutines - limit) {
		t.Fatalf("expected exactly %d denied requests, got %d", numGoroutines-limit, deniedCount.Load())
	}

	// Concurrent multi-key test to stress map access across many keys
	const (
		numKeys          = 20
		goroutinesPerKey = 30
		perKeyLimit      = 10
	)
	multiPolicy := Policy{Class: "concurrent_multi", Limit: perKeyLimit, Window: time.Minute}

	var (
		multiWG           sync.WaitGroup
		multiStartBarrier = make(chan struct{})
		keyAllowedCounts  [numKeys]atomic.Int64
		keyDeniedCounts   [numKeys]atomic.Int64
	)

	multiWG.Add(numKeys * goroutinesPerKey)
	for k := 0; k < numKeys; k++ {
		keyIdx := k
		keyName := fmt.Sprintf("rl:concurrent:multi_key_%d", keyIdx)
		for i := 0; i < goroutinesPerKey; i++ {
			go func() {
				defer multiWG.Done()
				<-multiStartBarrier
				d, err := m.Allow(ctx, keyName, multiPolicy)
				if err != nil {
					t.Errorf("multi-key error on key %d: %v", keyIdx, err)
					return
				}
				if d.Allowed {
					keyAllowedCounts[keyIdx].Add(1)
				} else {
					keyDeniedCounts[keyIdx].Add(1)
				}
			}()
		}
	}

	close(multiStartBarrier)
	multiWG.Wait()

	for k := 0; k < numKeys; k++ {
		if got := keyAllowedCounts[k].Load(); got != int64(perKeyLimit) {
			t.Errorf("key %d: expected %d allowed, got %d", k, perKeyLimit, got)
		}
		if got := keyDeniedCounts[k].Load(); got != int64(goroutinesPerKey-perKeyLimit) {
			t.Errorf("key %d: expected %d denied, got %d", k, goroutinesPerKey-perKeyLimit, got)
		}
	}
}
