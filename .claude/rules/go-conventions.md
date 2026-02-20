---
paths:
  - "apps/api-server/**/*.go"
---

# Go Conventions — OpenOMS API Server

## Code Style
- `go fmt` enforced (CI auto-formats on push to main)
- `golangci-lint v2.9` with revive linter (unused params, etc.)
- Unused function params: rename to `_` (e.g., `func(_ context.Context, conn *pgx.Conn)`)

## Error Handling
- Always wrap errors with context: `fmt.Errorf("create order: %w", err)`
- Use sentinel errors in service layer: `var ErrOrderNotFound = errors.New("order not found")`
- Handler error switch: specific errors first (ErrNotFound → 404), then validation (→ 400), then generic (→ 500)
- Always `slog.Error()` before returning 500 to client

## Database
- All tenant-scoped queries: `database.WithTenant(ctx, pool, tenantID, func(tx pgx.Tx) error {...})`
- JSONB params: `string(marshaledJSON)` with `$N::jsonb` SQL cast
- No `fmt.Sprintf` for SQL values — always parameterized ($1, $2, ...)
- SECURITY DEFINER functions only for: login, register, public returns
- New tables: enable RLS, create tenant policy, GRANT to openoms role

## Naming
- Handlers: `XxxHandler` with methods matching HTTP verbs (`Create`, `List`, `Get`, `Update`, `Delete`)
- Services: `XxxService` with business logic methods
- Repositories: `XxxRepository` with data access methods
- Models: `model.Xxx` structs with JSON tags

## Logging
- Use `log/slog` (not `log` or `fmt.Println`)
- Structured fields: `slog.Error("message", "key", value, "error", err)`
- Never log: passwords, tokens, encryption keys, credentials, PII
