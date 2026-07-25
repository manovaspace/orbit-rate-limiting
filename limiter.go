package ratelimiting

import (
	"context"
	"time"
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
	return map[string]Policy{
		"auth_otp_request": {Class: "auth_otp_request", Limit: 5, Window: time.Minute},
		"auth_otp_verify":  {Class: "auth_otp_verify", Limit: 20, Window: time.Minute},
		"auth_password":    {Class: "auth_password", Limit: 10, Window: time.Minute},
		"auth_refresh":     {Class: "auth_refresh", Limit: 30, Window: time.Minute},
	}
}

// OTPRequestIdentifierPolicy is the per-identifier OTP request limit.
func OTPRequestIdentifierPolicy() Policy {
	return Policy{Class: "auth_otp_request", Limit: 3, Window: time.Minute}
}
