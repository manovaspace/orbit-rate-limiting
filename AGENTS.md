---
read_with:
  - handbook/docs/orbit/architecture/contracts.md
  - handbook/docs/orbit/decisions/018-orbit-rate-limiting.md
---

> **Moved.** This module is now the `ratelimiting` package of [orbit-kit](https://github.com/manovaspace/orbit-kit) (`github.com/manovaspace/orbit-kit/ratelimiting`). This repository is archived and gets no more releases. To migrate, replace the import path `github.com/manovaspace/orbit-rate-limiting` with `github.com/manovaspace/orbit-kit/ratelimiting` and run `go get github.com/manovaspace/orbit-kit@latest && go mod tidy`.

# orbit-rate-limiting

Redis (or in-memory) token-bucket limiter for API edges. Status: `beta`. MIT — `github.com/manovaspace/orbit-rate-limiting`.

## Commands

```bash
go test ./...
```

## Architecture

- `internal/domain` — `Policy`, `Decision`, ADR-018 defaults and the `Limiter` port. It imports no Redis client, HTTP server or adapter.
- `internal/application` — the rules every adapter shares: the non-positive-policy bypass and the one-second `Retry-After` floor.
- `internal/infrastructure/memory` — process-local token bucket.
- `internal/infrastructure/redis` — Redis token bucket (Lua).
- `limiter.go` — the public facade. `NewMemoryLimiter`, `NewRedisLimiter`, `Policy`, `Decision`, `Limiter`, `DefaultAuthPolicies` and `OTPRequestIdentifierPolicy` keep their previous names and signatures.
- `.golangci.yml` `depguard` rules enforce both boundaries. Do not weaken them to land a change.

The library has no HTTP server, no database and publishes no events; it is a limiter adapter consumed by `orbit-api-gateway`.

## Docs

- Public README in this repo
- Staff handbook (private): ADR-018, architecture/contracts
