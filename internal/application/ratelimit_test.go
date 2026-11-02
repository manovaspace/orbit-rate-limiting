package application_test

import (
	"testing"
	"time"

	"github.com/manovaspace/orbit-rate-limiting/internal/application"
	"github.com/manovaspace/orbit-rate-limiting/internal/domain"
)

func TestUnlimitedAcceptsNonPositivePolicies(t *testing.T) {
	for _, policy := range []domain.Policy{
		{Class: "zero-limit", Limit: 0, Window: time.Minute},
		{Class: "negative-limit", Limit: -1, Window: time.Minute},
		{Class: "zero-window", Limit: 10, Window: 0},
		{Class: "negative-window", Limit: 10, Window: -time.Second},
	} {
		decision, ok := application.Unlimited(policy)
		if !ok {
			t.Fatalf("%s: policy was not treated as unlimited", policy.Class)
		}
		if !decision.Allowed || decision.Limit != policy.Limit {
			t.Fatalf("%s: decision = %+v, want allowed with limit %d", policy.Class, decision, policy.Limit)
		}
	}
}

func TestUnlimitedLeavesRealBucketsToTheAdapter(t *testing.T) {
	if _, ok := application.Unlimited(domain.Policy{Limit: 1, Window: time.Minute}); ok {
		t.Fatal("a positive limit and window must consume a token")
	}
}

func TestMinimumRetryAfterFloorsAtOneSecond(t *testing.T) {
	if got := application.MinimumRetryAfter(10 * time.Millisecond); got != time.Second {
		t.Fatalf("MinimumRetryAfter(10ms) = %v, want 1s", got)
	}
	if got := application.MinimumRetryAfter(3 * time.Second); got != 3*time.Second {
		t.Fatalf("MinimumRetryAfter(3s) = %v, want 3s", got)
	}
}
