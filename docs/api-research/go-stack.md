# Go Technology Stack

## Go Version

**Go 1.24** (released February 2025). Provides improved tooling, `log/slog` maturity, and range-over-func stability.

## Monorepo Structure

Use `go.work` workspace to manage multiple modules in a single repository.

```
go.work          # add to .gitignore
go.work.sum      # add to .gitignore
```

### Module Paths

```
github.com/openoms-org/openoms/apps/api-server
github.com/openoms-org/openoms/packages/shared
```

Each directory under `apps/` and `packages/` is a separate Go module with its own `go.mod`.

## Libraries

| Library | Version | Purpose | Why |
|---|---|---|---|
| `go-chi/chi` | v5.2.3 | HTTP router | stdlib `net/http` compatible, middleware ecosystem, no framework lock-in |
| `jackc/pgx` | v5.8.0 | PostgreSQL driver | Native protocol (not database/sql), RLS-aware with `set_config`, connection pooling via pgxpool |
| `hibiken/asynq` | v0.25.1 | Background job queue | Redis-backed, cron support, retries, web UI, simple task/handler model |
| `golang-jwt/jwt` | v5.2.4 | JWT tokens | Ed25519 signing support, well-maintained, JOSE compliant |
| `nhooyr.io/websocket` | latest | WebSocket | Modern API, concurrent-safe read/write, context-aware, passes autobahn tests |
| `golang-migrate/migrate` | v4.19.1 | Database migrations | SQL-based migration files, CLI + library, supports pgx driver |
| `log/slog` | stdlib | Structured logging | Standard library, JSON output, zero dependencies, extensible handlers |
| `caarlos0/env` | v11 | Configuration | Parses env vars into structs with tags, validation, defaults |
| `golang.org/x/crypto/bcrypt` | latest | Password hashing | Standard bcrypt implementation, part of Go extended stdlib |
| `google/uuid` | v1.6 | UUID generation | RFC 4122 compliant, v4 and v7 support |
| `stretchr/testify` | v1.9 | Test assertions | `assert` and `require` packages, readable test failures, widely adopted |

## Build Tools

### Taskfile

Use [Taskfile](https://taskfile.dev) (`Taskfile.yml`) instead of Makefiles. Cross-platform, YAML-based, with dependency resolution.

```yaml
# Taskfile.yml (example structure)
version: "3"

tasks:
  dev:
    desc: Run API server in development mode
    cmds:
      - go run ./cmd/server

  test:
    desc: Run all tests
    cmds:
      - go test ./...

  lint:
    desc: Run linter
    cmds:
      - golangci-lint run

  migrate:
    desc: Run database migrations
    cmds:
      - migrate -path ./migrations -database $DATABASE_URL up
```

### Linter

**golangci-lint v2** — single binary that runs multiple linters in parallel.

## Architecture

```
cmd/server/main.go          # Entrypoint: config, DI, server startup
internal/
  handler/                   # HTTP handlers (accept request, call service, write response)
  service/                   # Business logic (orchestration, validation, rules)
  repository/                # Data access (SQL queries, pgx)
  middleware/                 # HTTP middleware (auth, tenant, logging)
  model/                     # Domain types and DTOs
  config/                    # Configuration structs and loading
```

### Layer Responsibilities

```
Handler -> Service -> Repository
```

- **Handler**: Parse HTTP request, validate input, call service, marshal response. No business logic.
- **Service**: Business rules, orchestration across repositories, transaction management.
- **Repository**: Raw data access. Parameterized SQL only. Returns domain models.

### The `internal/` Convention

All application code lives under `internal/` to prevent external imports. Only `cmd/` and explicitly exported `packages/` are importable.

## Key Conventions

### Parameterized SQL and RLS

All queries must use parameterized SQL. Tenant context is set via `set_config`:

```go
// Set tenant context for RLS before any queries
_, err := conn.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenantID)
```

Never interpolate tenant IDs or user input into SQL strings.

### Context Propagation

Pass `context.Context` as the first argument to all functions that perform I/O. Use context for:

- Request cancellation and timeouts
- Tenant/user propagation through middleware
- Structured logging fields via `slog`

### Connection Acquisition

Use the `AcquireFunc` pattern from pgxpool to ensure connections are returned:

```go
err := pool.AcquireFunc(ctx, func(conn *pgxpool.Conn) error {
    _, err := conn.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenantID)
    if err != nil {
        return err
    }
    return conn.QueryRow(ctx, "SELECT ...").Scan(&result)
})
```

This guarantees the connection is released even if the callback panics.

## golangci-lint Configuration

```yaml
# .golangci.yml
version: "2"

linters:
  enable:
    - errcheck
    - govet
    - staticcheck
    - unused
    - gosimple
    - revive
    - gocritic
    - modernize
```

| Linter | Purpose |
|---|---|
| `errcheck` | Detect unchecked errors |
| `govet` | Report suspicious constructs |
| `staticcheck` | Advanced static analysis |
| `unused` | Find unused code |
| `gosimple` | Suggest code simplifications |
| `revive` | Extensible linter (replaces golint) |
| `gocritic` | Opinionated code quality checks |
| `modernize` | Suggest modern Go idioms (Go 1.21+) |
