# AGENTS.md

This file provides guidance for AI coding agents working in this repository.

## Project Overview

Starling Sweeper is a Go application that runs as an Azure Function (custom handler). It listens for Starling Bank webhook events and automatically "sweeps" the pre-existing account balance to a savings goal when a qualifying inbound payment is received.

## Architecture

- **Entry point:** `cmd/main.go` — HTTP server registering handlers for `/_ping` and `/feed-item`
- **Core logic:** `internal/feeditem/` — webhook handler that validates, deduplicates, and processes incoming transactions
- **Caching:** `internal/cache/` — Redis-backed cache used for deduplicating webhook deliveries
- **Health check:** `internal/ping/` — simple ping/version endpoint
- **Azure Functions config:** `host.json`, `feed-item/`, `ping/` directories contain Azure Function bindings

## Language & Toolchain

- **Language:** Go (version specified in `go.mod`, currently 1.24+)
- **Build:** `make build` (or `make build_azure` for the Linux/amd64 production binary)
- **Lint:** `make lint` (uses `golangci-lint` v2 with config in `.golangci.toml`)
- **Test:** `make test` (runs `go test -p 8 ./...` with `ENV=test`)
- **Coverage:** `make coverage`
- **Run locally:** `make start` (requires Azure Functions Core Tools and a `.env` file)

## Code Conventions

- All linters enabled by default in golangci-lint v2 except those explicitly disabled in `.golangci.toml`.
- Use `goimports` for formatting.
- Test files live alongside their source files (e.g., `handler_test.go` next to `handler.go`).
- No `//nolint` directives without an explanation and specific linter name.
- Environment variables are used for configuration (see README for the full list).
- The `internal/` directory enforces package boundaries — all application packages are internal.

## Testing

- Tests use the standard `testing` package.
- Redis is mocked using `github.com/alicebob/miniredis/v2`.
- HTTP responses are mocked using `github.com/karupanerura/go-mock-http-response`.
- Run tests with `make test`. The `ENV=test` variable is required.

## CI/CD

- **Test workflow** (`.github/workflows/test.yml`): Runs unit tests, coverage, and linting on PRs and pushes to `main`.
- **Deploy workflow** (`.github/workflows/deploy.yml`): Deploys to Azure Functions on push to `main`.
- **CodeQL** (`.github/workflows/codeql.yml`): Security scanning.

## Dependencies

- Managed with Go modules (`go.mod` / `go.sum`).
- Renovate is configured for automated dependency updates (`.github/renovate.json`).
- Key dependencies:
  - `github.com/lildude/starling` — Starling Bank API client
  - `github.com/go-redis/redis/v8` — Redis client
  - `golang.org/x/oauth2` — OAuth2 token handling
  - `github.com/joho/godotenv` — `.env` file loading

## Important Notes

- Never commit `.env` or `local.settings.json` files — they contain secrets.
- The webhook signature is validated using a public key (`PUBLIC_KEY` env var) via the Starling client library.
- The application handles three inbound transaction types: `FASTER_PAYMENTS_IN`, `NOSTRO_DEPOSIT`, and `DIRECT_CREDIT`.
