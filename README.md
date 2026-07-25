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

## Development

```bash
go test ./...
```

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md). Security reports: [SECURITY.md](./SECURITY.md).

## License

MIT — see [LICENSE](./LICENSE).
