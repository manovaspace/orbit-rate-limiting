package ratelimiting_test

import (
	"context"
	"testing"
	"time"

	ratelimiting "github.com/manovaspace/orbit-rate-limiting"
)

func TestDefaultPoliciesMatchADR018(t *testing.T) {
	policies := ratelimiting.DefaultAuthPolicies()
	for _, class := range []string{"auth_otp_request", "auth_otp_verify", "auth_password", "auth_refresh"} {
		policy, ok := policies[class]
		if !ok {
			t.Fatalf("missing default policy %q", class)
		}
		if policy.Class != class || policy.Limit <= 0 || policy.Window != time.Minute {
			t.Fatalf("unexpected policy for %q: %+v", class, policy)
		}
	}
	if got := policies["auth_otp_request"].Limit; got != 5 {
		t.Fatalf("auth_otp_request limit = %d, want 5", got)
	}
}

func TestOTPRequestIdentifierPolicy(t *testing.T) {
	policy := ratelimiting.OTPRequestIdentifierPolicy()
	if policy.Class != "auth_otp_request" || policy.Limit != 3 || policy.Window != time.Minute {
		t.Fatalf("unexpected identifier policy: %+v", policy)
	}
}

func TestMemoryLimiterImplementsLimiter(t *testing.T) {
	var limiter ratelimiting.Limiter = ratelimiting.NewMemoryLimiter()
	decision, err := limiter.Allow(context.Background(), "rl:auth_password:1.2.3.4", ratelimiting.Policy{
		Class:  "auth_password",
		Limit:  1,
		Window: time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Allowed || decision.Remaining != 0 {
		t.Fatalf("decision = %+v, want one allowed token", decision)
	}
}
