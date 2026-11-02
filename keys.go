package ratelimiting

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const upperHex = "0123456789ABCDEF"

// KeyForOrg returns the rate-limit bucket key for an organization route:
//
//	org:<orgID>:<route>
//
// The route is a template, not a concrete request path (for example
// "/v1/orgs/:orgID/members", never "/v1/orgs/42/members"). Leading and trailing
// whitespace is removed from both parts and the route is lowercased; path
// parameters are not resolved or collapsed. Colons, percent signs, whitespace
// and control characters are percent-escaped with uppercase hexadecimal, so an
// org id or route cannot merge into another bucket by contributing a separator.
//
// An empty (or whitespace-only) orgID or route returns the empty string.
// Callers must treat that as a configuration error and must not pass it to
// Limiter.Allow, which would put unrelated requests in the same bucket.
func KeyForOrg(orgID, route string) string {
	return keyFor("org", orgID, route)
}

// KeyForIP returns the rate-limit bucket key for an IP-addressed route:
//
//	ip:<ip>:<route>
//
// The escape and normalization rules are the same as KeyForOrg. The IP is
// opaque: it is not parsed or validated, so IPv6 addresses keep their colons in
// escaped form. An empty (or whitespace-only) ip or route returns the empty
// string and must not be passed to Limiter.Allow.
func KeyForIP(ip, route string) string {
	return keyFor("ip", ip, route)
}

func keyFor(prefix, subject, route string) string {
	subject = strings.TrimSpace(subject)
	route = strings.ToLower(strings.TrimSpace(route))
	if subject == "" || route == "" {
		return ""
	}
	return prefix + ":" + escapeKeyPart(subject) + ":" + escapeKeyPart(route)
}

// escapeKeyPart makes a part safe to place between the fixed ":" separators.
// Escaping "%" as well as ":" and whitespace keeps the encoding injective, so
// no two different parts produce the same escaped form.
func escapeKeyPart(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			out = appendEscapedByte(out, s[i])
			i++
			continue
		}
		if r == ':' || r == '%' || unicode.IsSpace(r) || unicode.IsControl(r) {
			for j := 0; j < size; j++ {
				out = appendEscapedByte(out, s[i+j])
			}
		} else {
			out = append(out, s[i:i+size]...)
		}
		i += size
	}
	return string(out)
}

func appendEscapedByte(out []byte, c byte) []byte {
	return append(out, '%', upperHex[c>>4], upperHex[c&0x0f])
}
