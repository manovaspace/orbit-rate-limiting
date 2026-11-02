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
