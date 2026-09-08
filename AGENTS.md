# AGENTS.md — ai-runtime

MuxCore sidecar module (`ai-runtime`).

## Module identity

| Field | Value |
|-------|-------|
| Directory | `ai-runtime` |
| Role | `ai` |
| Capability | `ai.runtime` |

## Build

```bash
cd ai-runtime
go test ./...
make lint
```

## Agent rules

- Modules run as gRPC sidecars; capabilities are the security boundary.
- TLS required in production (`MUXCORE_INSECURE_DISABLE_TLS` is dev-only).
- Match existing Go patterns; run `gofmt` and package tests before finishing.
