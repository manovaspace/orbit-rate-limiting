# orbit-rate-limiting

[![CI](https://github.com/manovaspace/orbit-rate-limiting/actions/workflows/ci.yml/badge.svg)](https://github.com/manovaspace/orbit-rate-limiting/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)

Go library: Redis (or in-memory) token-bucket rate limiter for API edges.

Part of the [Manova / Orbit](https://github.com/manovaspace) open toolkit.

## Install

```bash
go get github.com/manovaspace/orbit-rate-limiting@latest
```

## Usage

```go
import ratelimiting "github.com/manovaspace/orbit-rate-limiting"

lim := ratelimiting.NewRedisLimiter(redisClient)
d, err := lim.Allow(ctx, "rl:auth_password:1.2.3.4", ratelimiting.DefaultAuthPolicies()["auth_password"])
```

In-memory limiter (tests / single process):

```go
lim := ratelimiting.NewMemoryLimiter()
```

Typical gateway env: `REDIS_ADDR`, `RATE_LIMIT_TRUST_PROXY`.

## Bucket naming

Keys are bucket identities: two requests with the same key share a token
bucket. The package exposes two helpers so gateways and products derive keys
the same way:

- `KeyForOrg(orgID, route)` returns `org:<orgID>:<route>`.
- `KeyForIP(ip, route)` returns `ip:<ip>:<route>`.

`route` is the route template, not the concrete request path. Pass
`/v1/orgs/:orgID/members`, never `/v1/orgs/42/members`. The helpers trim leading
and trailing whitespace from each part and lowercase the route; they do not
resolve or collapse path parameters.

Characters that could merge two keys are percent-escaped with uppercase
hexadecimal: `:`, `%`, whitespace and control characters. This makes the mapping
injective, so `KeyForOrg("a:b", "route")` and
`KeyForOrg("a", "b:route")` produce different keys.

```go
KeyForOrg("org_123", "/V1/Orgs/:OrgID") // org:org_123:/v1/orgs/%3Aorgid
KeyForIP("2001:db8::1", "/v1/login")    // ip:2001%3Adb8%3A%3A1:/v1/login
```

An empty (or whitespace-only) org id, IP or route returns the empty string.
Treat that as a configuration error; never pass `""` to `Allow`, where unrelated
requests would share one bucket.

Which key to use:

- Sign-up, OTP request/verify and invite-accept are unauthenticated: use
  `KeyForIP`.
- Authenticated organization routes use `KeyForOrg`.
- Plan-based limits come later; this version has no plan tier.

## Layout

```
limiter.go                                  public facade (unchanged API)
internal/domain/                            Policy, Decision, ADR-018 defaults, Limiter port
internal/application/                       shared bypass rule and Retry-After floor
internal/infrastructure/memory/             process-local token bucket
internal/infrastructure/redis/              Redis token bucket (Lua)
```

The domain imports no infrastructure package. The public facade only maps the
exported types to the domain types and wires the adapter; it keeps the same
signatures every consumer compiles against.

## Development

```bash
go test ./...
```

The library has no database, HTTP server or event bus, so tests need no service
and no environment variable.

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md). Security reports: [SECURITY.md](./SECURITY.md).

## License

MIT — see [LICENSE](./LICENSE).
