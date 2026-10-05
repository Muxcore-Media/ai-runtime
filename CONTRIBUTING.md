# Contributing to Ai Runtime

## Development Setup

### Prerequisites

- Go 1.26.x
- golangci-lint (optional but recommended)
- Private module fetch (needs GitHub access to `github.com/Muxcore-Media`):

```bash
export GOPRIVATE='github.com/Muxcore-Media/*'
export GONOSUMDB='github.com/Muxcore-Media/*'
git config --global url."ssh://git@github.com/Muxcore-Media/".insteadOf "https://github.com/Muxcore-Media/"
```

### Clone and build

```bash
git clone https://github.com/Muxcore-Media/ai-runtime.git
cd ai-runtime
make build
make test
```

### Run against a local muxcored

```bash
# Terminal 1
MUXCORE_INSECURE_DISABLE_TLS=true muxcored

# Terminal 2
make build
MUXCORE_GRPC_ADDR=localhost:9090 MUXCORE_INSECURE_DISABLE_TLS=true ./ai-runtime
```

## Running Tests

```bash
make test
```

Tests must not depend on a running muxcored instance.

## Linting

```bash
make lint
```

## Pull Request Process

1. Branch from `main`.
2. Make your changes with tests.
3. Run `make ci` locally — it must pass.
4. Open a PR on GitHub (`github.com/Muxcore-Media/ai-runtime`).

## Security Vulnerabilities

Do **not** open a public issue. See [SECURITY.md](SECURITY.md).

## License

By contributing, you agree that your contributions will be licensed under the GPL-3.0 license.
