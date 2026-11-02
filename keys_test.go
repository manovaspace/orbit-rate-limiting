package ratelimiting_test

import (
	"net/url"
	"strings"
	"testing"

	ratelimiting "github.com/manovaspace/orbit-rate-limiting"
)

func TestKeyForOrg(t *testing.T) {
	tests := []struct {
		name  string
		org   string
		route string
		want  string
	}{
		{"basic", "org-123", "/v1/orgs", "org:org-123:/v1/orgs"},
		{"route trimmed and lowercased", "org-123", "  /V1/Orgs  ", "org:org-123:/v1/orgs"},
		{"route colon escaped", "org-123", "/v1/orgs/:id", "org:org-123:/v1/orgs/%3Aid"},
		{"org colon escaped", "org:123", "/v1/orgs", "org:org%3A123:/v1/orgs"},
		{"org whitespace escaped", "org 123", "/v1/orgs", "org:org%20123:/v1/orgs"},
		{"percent escaped", "org%3A", "/v1/orgs", "org:org%253A:/v1/orgs"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ratelimiting.KeyForOrg(tc.org, tc.route)
			if got != tc.want {
				t.Fatalf("KeyForOrg(%q, %q) = %q, want %q", tc.org, tc.route, got, tc.want)
			}
			if again := ratelimiting.KeyForOrg(tc.org, tc.route); again != got {
				t.Fatalf("KeyForOrg is not stable: %q then %q", got, again)
			}
		})
	}
}

func TestKeyForIP(t *testing.T) {
	tests := []struct {
		name  string
		ip    string
		route string
		want  string
	}{
		{"ipv4", "203.0.113.7", "/V1/login", "ip:203.0.113.7:/v1/login"},
		{"ipv6 colons escaped", "2001:db8::1", "/v1/login", "ip:2001%3Adb8%3A%3A1:/v1/login"},
		{"route trimmed and lowercased", "203.0.113.7", "  /V1/OTP  ", "ip:203.0.113.7:/v1/otp"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ratelimiting.KeyForIP(tc.ip, tc.route)
			if got != tc.want {
				t.Fatalf("KeyForIP(%q, %q) = %q, want %q", tc.ip, tc.route, got, tc.want)
			}
			if again := ratelimiting.KeyForIP(tc.ip, tc.route); again != got {
				t.Fatalf("KeyForIP is not stable: %q then %q", got, again)
			}
		})
	}
}

func TestKeyBuildersDoNotCollide(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
	}{
		{
			"colon in org versus colon in route",
			ratelimiting.KeyForOrg("a:b", "route"),
			ratelimiting.KeyForOrg("a", "b:route"),
		},
		{
			"colon in route versus colon in org",
			ratelimiting.KeyForOrg("a", "b:c"),
			ratelimiting.KeyForOrg("a:b", "c"),
		},
		{
			"whitespace in org versus whitespace in route",
			ratelimiting.KeyForOrg("a b", "route"),
			ratelimiting.KeyForOrg("a", "b route"),
		},
		{
			"literal escape in org versus escaped separator",
			ratelimiting.KeyForOrg("a%3A", "route"),
			ratelimiting.KeyForOrg("a:", "route"),
		},
		{
			"ip colon in address versus colon in route",
			ratelimiting.KeyForIP("a:b", "route"),
			ratelimiting.KeyForIP("a", "b:route"),
		},
		{
			"ip literal escape versus escaped separator",
			ratelimiting.KeyForIP("a%3A", "route"),
			ratelimiting.KeyForIP("a:", "route"),
		},
		{
			"org prefix versus ip prefix",
			ratelimiting.KeyForOrg("a", "route"),
			ratelimiting.KeyForIP("a", "route"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.a == "" || tc.b == "" {
				t.Fatalf("collision inputs must not be empty: %q, %q", tc.a, tc.b)
			}
			if tc.a == tc.b {
				t.Fatalf("keys collide: %q", tc.a)
			}
		})
	}
}

func TestKeyBuildersRejectEmptyParts(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"empty org", ratelimiting.KeyForOrg("", "/v1/orgs")},
		{"whitespace-only org", ratelimiting.KeyForOrg(" \t ", "/v1/orgs")},
		{"empty org route", ratelimiting.KeyForOrg("org-123", "")},
		{"whitespace-only org route", ratelimiting.KeyForOrg("org-123", " \n ")},
		{"empty ip", ratelimiting.KeyForIP("", "/v1/login")},
		{"whitespace-only ip", ratelimiting.KeyForIP(" \t ", "/v1/login")},
		{"empty ip route", ratelimiting.KeyForIP("203.0.113.7", "")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.key != "" {
				t.Fatalf("key = %q, want empty string", tc.key)
			}
		})
	}
}

func TestKeyPartsRoundTrip(t *testing.T) {
	org := "Org: one/ü\n"
	route := "/V1/Orgs/:OrgID"
	key := ratelimiting.KeyForOrg(org, route)

	parts := strings.Split(key, ":")
	if len(parts) != 3 {
		t.Fatalf("key %q has %d colon-separated parts, want 3", key, len(parts))
	}
	if parts[0] != "org" {
		t.Fatalf("prefix = %q, want %q", parts[0], "org")
	}
	gotOrg, err := url.PathUnescape(parts[1])
	if err != nil {
		t.Fatalf("org part %q does not unescape: %v", parts[1], err)
	}
	if want := strings.TrimSpace(org); gotOrg != want {
		t.Fatalf("org round trip = %q, want %q", gotOrg, want)
	}
	gotRoute, err := url.PathUnescape(parts[2])
	if err != nil {
		t.Fatalf("route part %q does not unescape: %v", parts[2], err)
	}
	if want := strings.ToLower(strings.TrimSpace(route)); gotRoute != want {
		t.Fatalf("route round trip = %q, want %q", gotRoute, want)
	}
}
