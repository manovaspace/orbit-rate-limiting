// Package application holds the rules shared by every limiter adapter. Every
// dependency is a domain port; it imports no adapter.
package application

import (
	"time"

	"github.com/manovaspace/orbit-rate-limiting/internal/domain"
)

// Unlimited reports the decision both adapters return for a policy whose limit
// or window is not positive. The second result is false when the policy is a
// real bucket and the adapter must consume a token.
func Unlimited(policy domain.Policy) (domain.Decision, bool) {
	if policy.Limit > 0 && policy.Window > 0 {
		return domain.Decision{}, false
	}
	return domain.Decision{Allowed: true, Limit: policy.Limit}, true
}

// MinimumRetryAfter clamps a computed retry delay to the one-second floor both
// adapters expose through Retry-After.
func MinimumRetryAfter(d time.Duration) time.Duration {
	if d < time.Second {
		return time.Second
	}
	return d
}
