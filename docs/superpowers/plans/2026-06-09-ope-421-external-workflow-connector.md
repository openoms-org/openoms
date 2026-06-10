# OPE-421/Phase-13 External-Workflow Connector — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A gated, generic external-workflow connector: an automation action hands a signed event to an external engine via the orchestration outbox, and a token+HMAC+single-use-nonce callback resolves that one event and may submit at most one whitelisted typed command — the engine never mutates durable state directly.

**Architecture:** Reuse the merged orchestration outbox/attempts. The outbound action enqueues an `automation.external_workflow` event; the `ExternalWorkflowHandler` dispatches a signed POST and parks the event with `next_attempt_at = now + timeout` (the deadline *is* the timeout); the callback resolves the correlated event and optionally enqueues one in-scope follow-on command. All behind `EXTERNAL_WORKFLOW_ENABLED` (default off).

**Tech Stack:** Go 1.25 (api-server), pgx/v5, PostgreSQL 16 FORCE RLS, golang-migrate, chi, testify. Spec: `docs/superpowers/specs/2026-06-09-ope-421-external-workflow-connector-design.md`.

**Conventions every task follows:**
- Run from `apps/api-server/`. Unit: `go test ./internal/<pkg>/ -run <Name> -count=1`. Integration: `DATABASE_URL=postgres://openoms:openoms-dev-password@127.0.0.1:5433/openoms?sslmode=disable go test -tags integration ./tests/integration/ -run <Name> -count=1`.
- `gofmt -w -s <file>` before each commit. Before any push: CI lint is **golangci-lint v2.9.0** — `/tmp/glci29/golangci-lint run --new-from-rev=main --timeout=5m` must be `0 issues`.
- **Migration-Safety (3 CI rules):** up migration must be additive (no DROP/RENAME/ALTER TYPE/SET NOT NULL/TRUNCATE/ADD-NOT-NULL-without-DEFAULT); never edit a merged migration; every `CREATE INDEX` and its `-- migrate:index-lock-ok` marker on **one line**.
- Commit after every task. No push/PR steps here (the executing workflow handles branch+PR; never push to main).

---

## File Structure

| File | Responsibility | Create/Modify |
|---|---|---|
| `migrations/000042_external_workflow.up.sql` / `.down.sql` | `external_workflow_tokens` (FORCE RLS) + `orchestration_attempts.external_execution_id` column | Create |
| `internal/model/fulfillment.go` | `BlockerExternalWorkflowTimeout` code + category | Modify |
| `internal/model/external_workflow.go` | token struct, scopes, callback DTOs, redaction allowlist + builder | Create |
| `internal/model/external_workflow_test.go` | redaction + scope + HMAC unit tests | Create |
| `internal/config/config.go` | `ExternalWorkflowEnabled` flag | Modify |
| `internal/repository/external_workflow_repository.go` | token issue/hashed-lookup/touch (tenant-scoped) | Create |
| `internal/service/external_workflow_service.go` | enqueue, callback resolution + one follow-on command, scope/HMAC checks, audit | Create |
| `internal/service/external_workflow_handler.go` | `ExternalWorkflowHandler` dispatcher (signed POST, pending-callback, timeout→blocker) | Create |
| `internal/handler/external_workflow_callback_handler.go` | `POST /v1/external-workflows/callback` | Create |
| `internal/automation/actions.go` | `external_workflow` executor case → enqueue | Modify |
| `internal/model/automation_actions.go` | add `external_workflow` to the action registry | Modify |
| `internal/router/router.go` | callback route (gated) | Modify |
| `cmd/server/main.go` | construct + wire service/handler, register dispatcher handler (gated) | Modify |
| `tests/integration/external_workflow_test.go` | token lookup, callback resolve + follow-on, replay, RLS, timeout, flag-off | Create |

---

## Task 1: Migration 000042 — token table + attempts column

**Files:**
- Create: `migrations/000042_external_workflow.up.sql`
- Create: `migrations/000042_external_workflow.down.sql`

- [ ] **Step 1: Write the up migration**

Create `migrations/000042_external_workflow.up.sql` (mirror the 000041 FORCE-RLS + dynamic-role-GRANT template):

```sql
-- OPE-421/Phase-13 external-workflow connector. Additive: a tenant-scoped FORCE-RLS
-- token table for callback auth, plus a column on orchestration_attempts to record the
-- external engine's execution id. No destructive ops.

CREATE TABLE public.external_workflow_tokens (
    id             uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    tenant_id      uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    integration_id uuid NOT NULL REFERENCES public.integrations(id) ON DELETE CASCADE,
    token_hash     text NOT NULL,
    scopes         text[] NOT NULL DEFAULT '{}',
    expires_at     timestamptz,
    last_used_at   timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uq_external_workflow_tokens_hash ON public.external_workflow_tokens (tenant_id, token_hash); -- migrate:index-lock-ok
CREATE INDEX idx_external_workflow_tokens_integration ON public.external_workflow_tokens (tenant_id, integration_id); -- migrate:index-lock-ok

ALTER TABLE public.external_workflow_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.external_workflow_tokens FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.external_workflow_tokens USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Record the external engine's execution id on the dispatch attempt (Phase-13 requirement).
ALTER TABLE public.orchestration_attempts ADD COLUMN IF NOT EXISTS external_execution_id text;

DO $$
DECLARE
    app_role text;
BEGIN
    FOREACH app_role IN ARRAY ARRAY['openoms_app', 'openoms'] LOOP
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = app_role) THEN
            EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON public.external_workflow_tokens TO %I', app_role);
        END IF;
    END LOOP;
END;
$$;
```

(`ADD COLUMN IF NOT EXISTS <col> text` is nullable with no default → passes the ADD-NOT-NULL Migration-Safety rule.)

- [ ] **Step 2: Write the down migration**

Create `migrations/000042_external_workflow.down.sql`:

```sql
-- OPE-421/Phase-13 rollback (additive tables/columns).
ALTER TABLE public.orchestration_attempts DROP COLUMN IF EXISTS external_execution_id;
DROP TABLE IF EXISTS public.external_workflow_tokens;
```

- [ ] **Step 3: Migration-Safety + CONCURRENTLY-per-line check**

Run:
```bash
f=migrations/000042_external_workflow.up.sql
DANGEROUS="DROP[[:space:]]+COLUMN|DROP[[:space:]]+TABLE|DROP[[:space:]]+INDEX|RENAME[[:space:]]+COLUMN|ALTER[[:space:]][^;]*TYPE|ALTER[[:space:]][^;]*SET[[:space:]]+NOT[[:space:]]+NULL|TRUNCATE"
tr '\n' ' ' < "$f" | grep -inEi "$DANGEROUS" || echo "PASS: no destructive ops"
grep -inE 'create[[:space:]]+(unique[[:space:]]+)?index' "$f" | grep -viE 'concurrently' | grep -viE 'migrate:index-lock-ok' && echo "BLOCKED: unmarked index" || echo "PASS: all indexes marked"
```
Expected: both `PASS`.

- [ ] **Step 4: Roundtrip on a scratch DB**

Run:
```bash
docker exec openoms-postgres-1 dropdb -U openoms --if-exists ewf_mig; docker exec openoms-postgres-1 createdb -U openoms ewf_mig
M="postgres://openoms:openoms-dev-password@127.0.0.1:5433/ewf_mig?sslmode=disable"
migrate -path migrations -database "$M" up && migrate -path migrations -database "$M" down 1 && migrate -path migrations -database "$M" up
docker exec openoms-postgres-1 psql -U openoms -d ewf_mig -tAc "SELECT relforcerowsecurity FROM pg_class WHERE relname='external_workflow_tokens';"
docker exec openoms-postgres-1 dropdb -U openoms --if-exists ewf_mig
```
Expected: up/down/up clean; `relforcerowsecurity = t`.

- [ ] **Step 5: Commit**

```bash
git add migrations/000042_external_workflow.up.sql migrations/000042_external_workflow.down.sql
git commit -m "OPE-421: migration 000042 external-workflow token table + attempt execution id"
```

---

## Task 2: Blocker code `external_workflow_timeout`

**Files:**
- Modify: `internal/model/fulfillment.go`
- Test: `internal/model/fulfillment_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/model/fulfillment_test.go`:

```go
func TestBlockerExternalWorkflowTimeout(t *testing.T) {
	assert.True(t, IsValidBlockerCode(BlockerExternalWorkflowTimeout))
	assert.Equal(t, "external_workflow_timeout", BlockerExternalWorkflowTimeout)
	assert.Equal(t, "integration", blockerCategories[BlockerExternalWorkflowTimeout])
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/model/ -run TestBlockerExternalWorkflowTimeout -count=1`
Expected: FAIL — `undefined: BlockerExternalWorkflowTimeout`.

- [ ] **Step 3: Add the constant + category**

In `internal/model/fulfillment.go`, in the blocker-code `const` block add:
```go
	BlockerExternalWorkflowTimeout = "external_workflow_timeout"
```
In the `blockerCategories` map add:
```go
	BlockerExternalWorkflowTimeout: "integration",
```
If a `validBlockerCodes` slice backs `IsValidBlockerCode`, add `BlockerExternalWorkflowTimeout` to it (mirror `BlockerChannelStockStale`).

- [ ] **Step 4: Run it to verify it passes**

Run: `gofmt -w -s internal/model/fulfillment.go && go test ./internal/model/ -run TestBlockerExternalWorkflowTimeout -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/model/fulfillment.go internal/model/fulfillment_test.go
git commit -m "OPE-421: add external_workflow_timeout blocker code"
```

---

## Task 3: Config flag `EXTERNAL_WORKFLOW_ENABLED`

**Files:** Modify `internal/config/config.go`.

- [ ] **Step 1: Add the field**

Next to `FulfillmentProcessEnabled`, add:
```go
	// ExternalWorkflowEnabled turns on the OPE-421/Phase-13 external-workflow connector:
	// the external_workflow action dispatches + the callback route is registered. Default
	// false -> the action is a no-op and the callback route returns 404.
	ExternalWorkflowEnabled bool `env:"EXTERNAL_WORKFLOW_ENABLED" envDefault:"false"`
```

- [ ] **Step 2: Build**

Run: `gofmt -w -s internal/config/config.go && go build ./... && go test ./internal/config/ -count=1`
Expected: build clean; config tests pass / no test files.

- [ ] **Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "OPE-421: EXTERNAL_WORKFLOW_ENABLED config flag (default off)"
```

---

## Task 4: Model — token, scopes, callback DTOs, redaction builder, HMAC helpers

**Files:**
- Create: `internal/model/external_workflow.go`
- Create: `internal/model/external_workflow_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/model/external_workflow_test.go`:

```go
package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildRedactedPayload_AllowlistOnly(t *testing.T) {
	order := map[string]any{
		"id":             "o1",
		"status":         "new",
		"customer_email": "secret@example.com",
		"total_amount":   123.45,
	}
	out := BuildRedactedPayload(order, []string{"id", "status", "total_amount"})
	assert.Equal(t, "o1", out["id"])
	assert.Equal(t, "new", out["status"])
	assert.Equal(t, 123.45, out["total_amount"])
	_, leaked := out["customer_email"]
	assert.False(t, leaked, "unlisted PII must never be serialized")
	assert.Len(t, out, 3)
}

func TestExternalWorkflowToken_ScopeAllows(t *testing.T) {
	tok := ExternalWorkflowToken{Scopes: []string{"set_status", "add_tag"}}
	assert.True(t, tok.ScopeAllows("set_status"))
	assert.False(t, tok.ScopeAllows("add_note"))
	assert.False(t, ExternalWorkflowToken{}.ScopeAllows("set_status")) // empty = resolve-only
}

func TestCallbackCommand_IsWhitelisted(t *testing.T) {
	assert.True(t, IsWhitelistedCallbackCommand("set_status"))
	assert.True(t, IsWhitelistedCallbackCommand("add_tag"))
	assert.True(t, IsWhitelistedCallbackCommand("add_note"))
	assert.False(t, IsWhitelistedCallbackCommand("create_invoice")) // not callback-permitted
	assert.False(t, IsWhitelistedCallbackCommand("delete_everything"))
}

func TestHashToken_StableAndHex(t *testing.T) {
	h1 := HashExternalWorkflowToken("abc")
	h2 := HashExternalWorkflowToken("abc")
	assert.Equal(t, h1, h2)
	assert.Len(t, h1, 64) // sha256 hex
	assert.NotEqual(t, h1, HashExternalWorkflowToken("abd"))
}

func TestSignAndVerifyHMAC(t *testing.T) {
	body := []byte(`{"correlation_nonce":"n1","status":"succeeded"}`)
	sig := SignExternalWorkflowBody(body, "secret")
	assert.True(t, VerifyExternalWorkflowSignature(body, sig, "secret"))
	assert.False(t, VerifyExternalWorkflowSignature(body, sig, "wrong"))
	assert.False(t, VerifyExternalWorkflowSignature(body, "sha256=deadbeef", "secret"))
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/model/ -run 'RedactedPayload|ExternalWorkflowToken|CallbackCommand|HashToken|SignAndVerify' -count=1`
Expected: FAIL — undefined symbols.

- [ ] **Step 3: Write the model**

Create `internal/model/external_workflow.go`:

```go
package model

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"time"

	"github.com/google/uuid"
)

// ExternalWorkflowToken is an RBAC-scoped, hashed bearer token authenticating an external
// workflow engine's callbacks for one integration (OPE-421/Phase-13). Mirrors
// supplier_portal_tokens. The raw token is shown once at creation; only token_hash is stored.
type ExternalWorkflowToken struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	IntegrationID uuid.UUID  `json:"integration_id"`
	TokenHash     string     `json:"-"`
	Scopes        []string   `json:"scopes"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	LastUsedAt    *time.Time `json:"last_used_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// ScopeAllows reports whether the token may submit the given callback command. An empty
// scope set means resolve-only (no follow-on command permitted).
func (t ExternalWorkflowToken) ScopeAllows(command string) bool {
	return slices.Contains(t.Scopes, command)
}

// IsExpired reports whether the token has an expiry in the past.
func (t ExternalWorkflowToken) IsExpired(now time.Time) bool {
	return t.ExpiresAt != nil && now.After(*t.ExpiresAt)
}

// whitelistedCallbackCommands are the only typed commands a callback may submit (each is
// applied through the audited orchestration outbox; the external engine never writes state).
var whitelistedCallbackCommands = []string{"set_status", "add_tag", "add_note"}

// IsWhitelistedCallbackCommand reports whether c is a command a callback may submit.
func IsWhitelistedCallbackCommand(c string) bool { return slices.Contains(whitelistedCallbackCommands, c) }

// ExternalWorkflowCallbackRequest is the callback body posted by the external engine.
type ExternalWorkflowCallbackRequest struct {
	CorrelationNonce string `json:"correlation_nonce"`
	Status           string `json:"status"` // "succeeded" | "failed"
	ExternalExecID   string `json:"external_execution_id,omitempty"`
	// Optional single follow-on command, applied only if in the token's scopes.
	Command       string `json:"command,omitempty"`        // set_status | add_tag | add_note
	CommandValue  string `json:"command_value,omitempty"`  // e.g. the status / tag / note text
}

// BuildRedactedPayload projects an order map onto an allowlist of field paths. Nothing
// outside the allowlist is included — secrets, internal ids and unlisted PII never leave.
func BuildRedactedPayload(order map[string]any, allowlist []string) map[string]any {
	out := make(map[string]any, len(allowlist))
	for _, key := range allowlist {
		if v, ok := order[key]; ok {
			out[key] = v
		}
	}
	return out
}

// HashExternalWorkflowToken returns the hex sha256 of a raw token (the stored token_hash).
func HashExternalWorkflowToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// SignExternalWorkflowBody returns the "sha256=<hex>" HMAC of body under secret (the
// X-Signature-256 header value), matching the outgoing-webhook signing convention.
func SignExternalWorkflowBody(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// VerifyExternalWorkflowSignature constant-time compares a provided "sha256=..." signature
// against the expected HMAC of body under secret.
func VerifyExternalWorkflowSignature(body []byte, provided, secret string) bool {
	expected := SignExternalWorkflowBody(body, secret)
	return hmac.Equal([]byte(provided), []byte(expected))
}
```

- [ ] **Step 4: Run them to verify they pass**

Run: `gofmt -w -s internal/model/external_workflow.go && go test ./internal/model/ -run 'RedactedPayload|ExternalWorkflowToken|CallbackCommand|HashToken|SignAndVerify' -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/model/external_workflow.go internal/model/external_workflow_test.go
git commit -m "OPE-421: external-workflow model (token, scopes, callback DTO, redaction, HMAC)"
```

---

## Task 5: Token repository

**Files:** Create `internal/repository/external_workflow_repository.go`. (Tested via Task 12 integration.)

- [ ] **Step 1: Write the repository**

Create `internal/repository/external_workflow_repository.go`:

```go
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// ErrExternalWorkflowTokenNotFound is returned when a token hash does not resolve.
var ErrExternalWorkflowTokenNotFound = errors.New("external workflow token not found")

// ExternalWorkflowTokenRepository is tenant-scoped (methods take a pgx.Tx from
// database.WithTenant). The callback path resolves a token cross-tenant on the privileged
// pool first (FindByHashCrossTenant) to learn the tenant, then re-scopes via WithTenant.
type ExternalWorkflowTokenRepository struct{}

// NewExternalWorkflowTokenRepository constructs the repository.
func NewExternalWorkflowTokenRepository() *ExternalWorkflowTokenRepository {
	return &ExternalWorkflowTokenRepository{}
}

const externalWorkflowTokenColumns = `id, tenant_id, integration_id, token_hash, scopes, expires_at, last_used_at, created_at`

// Issue stores a new token (the caller hashes the raw token first) and returns it.
func (r *ExternalWorkflowTokenRepository) Issue(ctx context.Context, tx pgx.Tx, t model.ExternalWorkflowToken) (*model.ExternalWorkflowToken, error) {
	out, err := scanExternalWorkflowToken(tx.QueryRow(ctx,
		`INSERT INTO external_workflow_tokens (tenant_id, integration_id, token_hash, scopes, expires_at)
		 VALUES ($1,$2,$3,$4,$5) RETURNING `+externalWorkflowTokenColumns,
		t.TenantID, t.IntegrationID, t.TokenHash, t.Scopes, t.ExpiresAt))
	if err != nil {
		return nil, fmt.Errorf("issue external workflow token: %w", err)
	}
	return out, nil
}

// FindByHashCrossTenant resolves a token hash WITHOUT a tenant context — it runs on the
// privileged pool (bypassing RLS) so the callback (which has no JWT/tenant) can learn which
// tenant a token belongs to. The caller then re-scopes all further work via WithTenant.
func (r *ExternalWorkflowTokenRepository) FindByHashCrossTenant(ctx context.Context, q Querier, tokenHash string) (*model.ExternalWorkflowToken, error) {
	out, err := scanExternalWorkflowToken(q.QueryRow(ctx,
		`SELECT `+externalWorkflowTokenColumns+` FROM external_workflow_tokens WHERE token_hash = $1`, tokenHash))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExternalWorkflowTokenNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find external workflow token: %w", err)
	}
	return out, nil
}

// TouchLastUsed records that a token was just used (tenant-scoped).
func (r *ExternalWorkflowTokenRepository) TouchLastUsed(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	_, err := tx.Exec(ctx, `UPDATE external_workflow_tokens SET last_used_at = now() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("touch external workflow token: %w", err)
	}
	return nil
}

func scanExternalWorkflowToken(row pgx.Row) (*model.ExternalWorkflowToken, error) {
	var t model.ExternalWorkflowToken
	if err := row.Scan(&t.ID, &t.TenantID, &t.IntegrationID, &t.TokenHash, &t.Scopes,
		&t.ExpiresAt, &t.LastUsedAt, &t.CreatedAt); err != nil {
		return nil, err
	}
	return &t, nil
}
```

- [ ] **Step 2: Build**

Run: `gofmt -w -s internal/repository/external_workflow_repository.go && go build ./...`
Expected: builds clean.

- [ ] **Step 3: Commit**

```bash
git add internal/repository/external_workflow_repository.go
git commit -m "OPE-421: external-workflow token repository (issue, cross-tenant hash lookup, touch)"
```

---

## Task 6: Service — enqueue + callback resolution + follow-on command

**Files:** Create `internal/service/external_workflow_service.go`. (Behavior covered by Task 12 integration.)

The service is the gated brain. It enqueues the correlated outbox event (ensure-process for the order, like OPE-421's `EnsureProcessForOrder`), and resolves callbacks (verify nonce in-flight + single-use, mark the event, optionally enqueue one in-scope follow-on command, audit).

- [ ] **Step 1: Write the service**

Create `internal/service/external_workflow_service.go`:

```go
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// EventExternalWorkflow is the orchestration outbox event type for an external-workflow
// dispatch. The ExternalWorkflowHandler (dispatcher) consumes it.
const EventExternalWorkflow = "automation.external_workflow"

// EventExternalWorkflowCommand is the follow-on command enqueued from a callback. It reuses
// the existing set_status orchestration path where applicable; add_tag/add_note are applied
// by the handler registered for this type.
const EventExternalWorkflowCommand = "automation.external_workflow_command"

// ErrCorrelationNotResolvable is returned when a callback's nonce does not match an in-flight,
// unresolved event for the token's integration (unknown, already resolved, or wrong integration).
var ErrCorrelationNotResolvable = errors.New("external workflow correlation not resolvable")

// ExternalWorkflowService is gated (Enabled): when off, Enqueue is a no-op and the callback
// path is never reached (the route is unregistered). It owns the correlation enqueue and the
// callback resolution; the actual outbound POST is the ExternalWorkflowHandler (dispatcher).
type ExternalWorkflowService struct {
	enabled       bool
	pool          *pgxpool.Pool
	workerPool    *pgxpool.Pool // privileged, for the cross-tenant token lookup
	fulfillment   *FulfillmentService
	orchestration *repository.OrchestrationRepository
	tokens        *repository.ExternalWorkflowTokenRepository
	audit         repository.AuditRepo
}

// NewExternalWorkflowService constructs the service.
func NewExternalWorkflowService(enabled bool, pool, workerPool *pgxpool.Pool, fulfillment *FulfillmentService,
	orchestration *repository.OrchestrationRepository, tokens *repository.ExternalWorkflowTokenRepository, audit repository.AuditRepo) *ExternalWorkflowService {
	return &ExternalWorkflowService{enabled: enabled, pool: pool, workerPool: workerPool,
		fulfillment: fulfillment, orchestration: orchestration, tokens: tokens, audit: audit}
}

// Enabled reports whether the connector is active. Nil-safe.
func (s *ExternalWorkflowService) Enabled() bool { return s != nil && s.enabled }

// EnqueueDispatch enqueues a correlated external-workflow event for an order, inside the
// caller's tenant tx (the automation action runs in one). No-op when disabled. Ensures the
// order has a fulfillment process (the outbox process_id FK), then enqueues the event
// carrying the integration, a single-use correlation nonce, and the redacted payload.
func (s *ExternalWorkflowService) EnqueueDispatch(ctx context.Context, tx pgx.Tx, tenantID, orderID, integrationID uuid.UUID, redacted map[string]any) (string, error) {
	if !s.Enabled() {
		return "", nil
	}
	proc, err := s.fulfillment.EnsureProcessForOrderUnconditional(ctx, tx, tenantID, orderID)
	if err != nil {
		return "", fmt.Errorf("ensure process for external workflow: %w", err)
	}
	nonce := uuid.NewString()
	payload := map[string]any{
		"integration_id":    integrationID.String(),
		"correlation_nonce": nonce,
		"order_id":          orderID.String(),
		"redacted_payload":  redacted,
	}
	if _, _, err := s.orchestration.EnqueueEvent(ctx, tx, model.OrchestrationOutboxEvent{
		TenantID:       tenantID,
		ProcessID:      proc.ID,
		EventType:      EventExternalWorkflow,
		IdempotencyKey: EventExternalWorkflow + ":" + orderID.String() + ":" + nonce,
		Payload:        payload,
	}); err != nil {
		return "", fmt.Errorf("enqueue external workflow event: %w", err)
	}
	return nonce, nil
}

// ResolveCallback authenticates + resolves a callback. tokenHash is the hashed bearer token;
// body is the raw request body (already HMAC-verified by the handler against the integration
// secret). It finds the token cross-tenant, then re-scopes via WithTenant to find the in-flight
// correlated event, marks it succeeded/failed, optionally enqueues one in-scope follow-on
// command, records the external exec id, and audits. Returns the resolved tenant for the handler.
func (s *ExternalWorkflowService) ResolveCallback(ctx context.Context, tokenHash string, req model.ExternalWorkflowCallbackRequest, ip string, now time.Time) (uuid.UUID, error) {
	tok, err := s.tokens.FindByHashCrossTenant(ctx, s.workerPool, tokenHash)
	if err != nil {
		return uuid.Nil, err // ErrExternalWorkflowTokenNotFound -> handler maps to 401
	}
	if tok.IsExpired(now) {
		return uuid.Nil, repository.ErrExternalWorkflowTokenNotFound
	}
	tenantID := tok.TenantID
	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		event, e := s.orchestration.FindPendingByEventAndNonce(ctx, tx, EventExternalWorkflow, req.CorrelationNonce, tok.IntegrationID)
		if e != nil {
			return ErrCorrelationNotResolvable
		}
		// Resolve the correlated event (single-use: it is no longer pending after this).
		success := req.Status == "succeeded"
		if success {
			if e := s.orchestration.MarkSucceeded(ctx, tx, event.ID); e != nil {
				return e
			}
		} else {
			if e := s.orchestration.MarkFailedPermanent(ctx, tx, event.ID, "external workflow reported failure"); e != nil {
				return e
			}
		}
		if req.ExternalExecID != "" {
			_ = s.orchestration.SetLatestAttemptExternalExecID(ctx, tx, event.ID, req.ExternalExecID)
		}
		// Optional single follow-on command (only if whitelisted AND in the token scope).
		if success && req.Command != "" {
			if !model.IsWhitelistedCallbackCommand(req.Command) || !tok.ScopeAllows(req.Command) {
				return fmt.Errorf("callback command %q not permitted", req.Command) // -> 403
			}
			cmdPayload := map[string]any{"order_id": payloadOrderID(event), "command": req.Command, "value": req.CommandValue}
			cmdJSON, _ := json.Marshal(cmdPayload)
			if _, _, e := s.orchestration.EnqueueEvent(ctx, tx, model.OrchestrationOutboxEvent{
				TenantID:       tenantID,
				ProcessID:      event.ProcessID,
				EventType:      EventExternalWorkflowCommand,
				IdempotencyKey: EventExternalWorkflowCommand + ":" + req.CorrelationNonce,
				Payload:        json.RawMessage(cmdJSON),
			}); e != nil {
				return e
			}
		}
		if e := s.tokens.TouchLastUsed(ctx, tx, tok.ID); e != nil {
			return e
		}
		if s.audit != nil {
			_ = s.audit.Log(ctx, tx, model.AuditEntry{
				TenantID: tenantID, UserID: uuid.Nil, Action: "external_workflow.callback",
				EntityType: "orchestration_outbox", EntityID: event.ID,
				Changes:   map[string]string{"status": req.Status, "command": req.Command, "external_execution_id": req.ExternalExecID},
				IPAddress: ip,
			})
		}
		return nil
	})
	return tenantID, err
}

// payloadOrderID extracts the order_id from an external-workflow event payload.
func payloadOrderID(event model.OrchestrationOutboxEvent) string {
	if m, ok := event.Payload.(map[string]any); ok {
		if v, ok := m["order_id"].(string); ok {
			return v
		}
	}
	return ""
}
```

This task references three repository methods + one fulfillment method that the next step adds.

- [ ] **Step 2: Add the supporting repository/service methods**

In `internal/repository/orchestration_repository.go` add:

```go
// FindPendingByEventAndNonce loads a still-pending event of the given type whose payload
// carries the correlation nonce AND integration id (tenant-scoped). Used by the callback to
// find the one event it may resolve; not-found / already-resolved -> pgx.ErrNoRows.
func (r *OrchestrationRepository) FindPendingByEventAndNonce(ctx context.Context, q Querier, eventType, nonce string, integrationID uuid.UUID) (*model.OrchestrationOutboxEvent, error) {
	row := q.QueryRow(ctx, `SELECT `+orchestrationOutboxColumns+`
		FROM orchestration_outbox
		WHERE status = 'pending' AND event_type = $1
		  AND payload->>'correlation_nonce' = $2
		  AND payload->>'integration_id' = $3`, eventType, nonce, integrationID.String())
	return scanOutboxEvent(row)
}

// SetLatestAttemptExternalExecID records the external engine's execution id on the most
// recent attempt of an outbox event (best-effort).
func (r *OrchestrationRepository) SetLatestAttemptExternalExecID(ctx context.Context, q Querier, outboxID uuid.UUID, execID string) error {
	_, err := q.Exec(ctx, `UPDATE orchestration_attempts SET external_execution_id = $2
		WHERE id = (SELECT id FROM orchestration_attempts WHERE outbox_id = $1 ORDER BY attempt_number DESC LIMIT 1)`,
		outboxID, execID)
	return err
}
```

(Verify `orchestrationOutboxColumns` + `scanOutboxEvent` + `MarkFailedPermanent` exist in that file from OPE-415; reuse them verbatim. `scanOutboxEvent` must unmarshal `payload` into `any` — confirm by reading the existing scan helper.)

In `internal/service/fulfillment_service.go` add an unconditional ensure-process (the gated `EnsureProcessForOrder` no-ops when `FULFILLMENT_PROCESS_ENABLED` is off, but the external-workflow connector has its own flag, so it needs a process regardless — mirror the OPE-423a backfill which also creates unconditionally):

```go
// EnsureProcessForOrderUnconditional creates/reuses the order's fulfillment process WITHOUT
// the FULFILLMENT_PROCESS_ENABLED gate, for callers gated by their own flag (OPE-421
// external-workflow). Idempotent (GetProcessByOrder first). Does NOT enqueue order.created.
func (s *FulfillmentService) EnsureProcessForOrderUnconditional(ctx context.Context, tx pgx.Tx, tenantID, orderID uuid.UUID) (*model.FulfillmentProcess, error) {
	if existing, err := s.fulfillment.GetProcessByOrder(ctx, tx, orderID); err == nil {
		return existing, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("lookup process: %w", err)
	}
	return s.fulfillment.CreateProcess(ctx, tx, model.FulfillmentProcess{TenantID: tenantID, OrderID: orderID})
}
```

- [ ] **Step 3: Build**

Run: `gofmt -w -s internal/service/external_workflow_service.go internal/repository/orchestration_repository.go internal/service/fulfillment_service.go && go build ./...`
Expected: builds clean (fix any signature mismatch against the actual `MarkFailedPermanent`/`scanOutboxEvent` by reading those functions).

- [ ] **Step 4: Commit**

```bash
git add internal/service/external_workflow_service.go internal/repository/orchestration_repository.go internal/service/fulfillment_service.go
git commit -m "OPE-421: external-workflow service (enqueue, callback resolve + scoped follow-on, audit)"
```

---

## Task 7: Dispatcher handler — signed POST, pending-callback, timeout→blocker

**Files:** Create `internal/service/external_workflow_handler.go`. (Behavior covered by Task 12.)

- [ ] **Step 1: Write the handler**

Create `internal/service/external_workflow_handler.go`:

```go
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// externalWorkflowConfig is the per-integration dispatch config decrypted from the
// integration credentials JSONB (outbound_url, signing_secret, timeout, criticality).
type externalWorkflowConfig struct {
	OutboundURL   string `json:"outbound_url"`
	SigningSecret string `json:"signing_secret"`
	TimeoutSecs   int    `json:"timeout_seconds"`
	Criticality   string `json:"criticality"` // "warning" | "blocker"
}

// ExternalWorkflowHandler implements OrchestrationHandler for EventExternalWorkflow. On first
// dispatch it POSTs the signed payload and parks the event with next_attempt_at = now+timeout
// (the deadline). If the callback resolves it first, the worker never re-dispatches. If the
// deadline passes unresolved, the worker re-dispatches and this handler — finding the event
// past its deadline with no callback — fails it permanently so the worker opens a timeout blocker.
type ExternalWorkflowHandler struct {
	pool       *pgxpool.Pool
	httpClient *http.Client // the SSRF-safe client (noPrivateDialer), shared with webhook dispatch
	orch       *repository.OrchestrationRepository
	loadConfig func(ctx context.Context, tx pgx.Tx, integrationID string) (externalWorkflowConfig, error)
}

// Handle dispatches or times out one external-workflow event. (Constructor + loadConfig wiring
// are in main.go; loadConfig decrypts the integration credentials.)
func (h *ExternalWorkflowHandler) Handle(ctx context.Context, event model.OrchestrationOutboxEvent) error {
	payload, _ := event.Payload.(map[string]any)
	integrationID, _ := payload["integration_id"].(string)
	nonce, _ := payload["correlation_nonce"].(string)

	// If this is a re-dispatch (the event was already dispatched once and its deadline passed
	// with no callback), the attempt count is > 0 -> treat as timeout.
	if event.Attempts > 0 {
		return model.Permanent(fmt.Errorf("external workflow callback timed out (nonce %s)", nonce))
	}

	var cfg externalWorkflowConfig
	err := database.WithTenant(ctx, h.pool, event.TenantID, func(tx pgx.Tx) error {
		c, e := h.loadConfig(ctx, tx, integrationID)
		cfg = c
		return e
	})
	if err != nil {
		return fmt.Errorf("load external workflow config: %w", err) // retryable
	}

	body, _ := json.Marshal(payload["redacted_payload"])
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, cfg.OutboundURL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature-256", model.SignExternalWorkflowBody(body, cfg.SigningSecret))
	req.Header.Set("X-OpenOMS-Event", EventExternalWorkflow)
	req.Header.Set("X-OpenOMS-Correlation", nonce)
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("dispatch external workflow: %w", err) // retryable transport error
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("external workflow returned %d", resp.StatusCode) // retryable
	}

	// Success: parked pending-callback. Return a sentinel that the worker maps to
	// "re-queue with next_attempt_at = now + timeout" rather than succeeded. See note below.
	timeout := time.Duration(cfg.TimeoutSecs) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Minute
	}
	return &deferUntilError{at: time.Now().Add(timeout)}
}

// deferUntilError signals the worker to leave the event pending with next_attempt_at = at
// (the callback deadline) instead of marking it succeeded.
type deferUntilError struct{ at time.Time }

func (e *deferUntilError) Error() string { return "deferred until callback deadline" }
```

The worker must recognize `*deferUntilError` and call `MarkFailedRetry(id, "awaiting callback", e.at)` (which sets status=pending + next_attempt_at). Add that branch to `internal/worker/orchestration_worker.go` where it inspects the dispatch error:

```go
	var deferErr *service.deferUntilError
	if errors.As(err, &deferErr) {
		_ = w.repo.MarkFailedRetry(ctx, w.pool, event.ID, "awaiting external workflow callback", deferErr.at)
		continue
	}
```

(If `deferUntilError` must be referenced from the worker package, export it as `service.DeferUntil` with an `At` field. Adjust the handler accordingly so the worker can type-assert it.)

- [ ] **Step 2: Build**

Run: `gofmt -w -s internal/service/external_workflow_handler.go internal/worker/orchestration_worker.go && go build ./...`
Expected: builds clean (export `DeferUntil` if the worker needs it; reconcile the import direction — worker already imports service).

- [ ] **Step 3: Commit**

```bash
git add internal/service/external_workflow_handler.go internal/worker/orchestration_worker.go
git commit -m "OPE-421: external-workflow dispatcher (signed POST, pending-callback deadline, timeout->blocker)"
```

---

## Task 8: `external_workflow` action type

**Files:**
- Modify: `internal/model/automation_actions.go` (registry)
- Modify: `internal/automation/actions.go` (executor case)
- Test: `internal/model/automation_actions_test.go`

- [ ] **Step 1: Add the registry entry + test**

In `internal/model/automation_actions.go`, add to `automationActionSpecs`:
```go
	"external_workflow": {"integration_id"},
```
In `internal/model/automation_actions_test.go` `TestIsValidActionType`, add `"external_workflow"` to the valid list.

- [ ] **Step 2: Add the executor case**

In `internal/automation/actions.go`, in the `ExecuteAction` switch add:
```go
	case "external_workflow":
		return e.executeExternalWorkflow(ctx, tenantID, action, event)
```
And the method (it builds the redacted payload from the integration allowlist + enqueues via the service — wire the service into `DefaultActionExecutor` with a `SetExternalWorkflow` setter mirroring the OPE-421 `SetOrchestration` pattern; gate on `Enabled()`):
```go
// executeExternalWorkflow hands the event to an external workflow engine via the
// external-workflow connector (OPE-421/Phase-13). Gated/best-effort: a no-op when the
// connector is nil/disabled.
func (e *DefaultActionExecutor) executeExternalWorkflow(_ context.Context, tenantID uuid.UUID, action Action, event Event) error {
	if e.externalWorkflow == nil || !e.externalWorkflow.Enabled() {
		return nil
	}
	if event.EntityType != "order" {
		return fmt.Errorf("external_workflow action only supports order entities, got %q", event.EntityType)
	}
	integrationIDStr, _ := action.Params["integration_id"].(string)
	integrationID, err := uuid.Parse(integrationIDStr)
	if err != nil {
		return fmt.Errorf("external_workflow: invalid integration_id: %w", err)
	}
	return e.externalWorkflow.DispatchForOrder(context.Background(), tenantID, event.EntityID, integrationID)
}
```
Add `DispatchForOrder(ctx, tenantID, orderID, integrationID)` to `ExternalWorkflowService` — it opens its own `WithTenant` tx, loads the integration's `outbound_field_allowlist` + builds the redacted payload from the order, and calls `EnqueueDispatch`. (The allowlist + order load mirror `executeSendMarketplaceMessage`'s template/order load.)

- [ ] **Step 3: Run + build**

Run: `gofmt -w -s internal/model/automation_actions.go internal/model/automation_actions_test.go internal/automation/actions.go && go test ./internal/model/ -run ActionType -count=1 && go build ./...`
Expected: registry test passes; build clean.

- [ ] **Step 4: Commit**

```bash
git add internal/model/automation_actions.go internal/model/automation_actions_test.go internal/automation/actions.go internal/service/external_workflow_service.go
git commit -m "OPE-421: external_workflow automation action (gated enqueue via outbox)"
```

---

## Task 9: Callback HTTP handler + gated route

**Files:**
- Create: `internal/handler/external_workflow_callback_handler.go`
- Modify: `internal/router/router.go`

- [ ] **Step 1: Write the handler**

Create `internal/handler/external_workflow_callback_handler.go`:

```go
package handler

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// ExternalWorkflowCallbackHandler serves the external engine's callback. Auth = bearer token
// (hashed lookup) + HMAC body signature + single-use correlation nonce. No JWT.
type ExternalWorkflowCallbackHandler struct {
	svc            *service.ExternalWorkflowService
	integrationSec func(integrationID string) (string, error) // loads the signing secret for HMAC verify
}

// NewExternalWorkflowCallbackHandler constructs the handler.
func NewExternalWorkflowCallbackHandler(svc *service.ExternalWorkflowService) *ExternalWorkflowCallbackHandler {
	return &ExternalWorkflowCallbackHandler{svc: svc}
}

// Callback handles POST /v1/external-workflows/callback.
func (h *ExternalWorkflowCallbackHandler) Callback(w http.ResponseWriter, r *http.Request) {
	bearer := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if bearer == "" {
		writeError(w, http.StatusUnauthorized, "missing token")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "body too large")
		return
	}
	var req model.ExternalWorkflowCallbackRequest
	if err := jsonUnmarshalStrict(body, &req); err != nil || req.CorrelationNonce == "" {
		writeError(w, http.StatusBadRequest, "invalid callback body")
		return
	}
	// The service verifies the token + finds the integration; the HMAC is verified against the
	// integration secret inside the service after the token resolves the integration. (Pass the
	// raw body + signature header through so the service can do the constant-time compare.)
	tenantID, err := h.svc.ResolveCallbackHTTP(r.Context(), model.HashExternalWorkflowToken(bearer),
		r.Header.Get("X-Signature-256"), body, req, clientIP(r), time.Now())
	switch {
	case errors.Is(err, repository.ErrExternalWorkflowTokenNotFound):
		writeError(w, http.StatusUnauthorized, "invalid token")
	case errors.Is(err, service.ErrExternalWorkflowBadSignature):
		writeError(w, http.StatusUnauthorized, "invalid signature")
	case errors.Is(err, service.ErrCorrelationNotResolvable):
		writeError(w, http.StatusConflict, "correlation not resolvable")
	case err != nil && strings.Contains(err.Error(), "not permitted"):
		writeError(w, http.StatusForbidden, "command not permitted")
	case err != nil:
		writeServerError(w, err)
	default:
		_ = tenantID
		writeJSON(w, http.StatusOK, map[string]string{"result": "resolved"})
	}
}
```

Add to `ExternalWorkflowService` a thin `ResolveCallbackHTTP(ctx, tokenHash, sigHeader, body, req, ip, now)` that: resolves the token cross-tenant, verifies `model.VerifyExternalWorkflowSignature(body, sigHeader, integrationSecret)` (loading the integration secret for `tok.IntegrationID`) → `ErrExternalWorkflowBadSignature` on mismatch, then delegates to the `ResolveCallback` logic from Task 6. Define `var ErrExternalWorkflowBadSignature = errors.New("external workflow bad signature")` in the service. Reuse the existing `writeError`/`writeJSON`/`writeServerError`/`clientIP`/`jsonUnmarshalStrict` handler helpers (confirm their exact names by reading a sibling handler).

- [ ] **Step 2: Register the gated route**

In `internal/router/router.go`, register only when the flag is on (off ⇒ route absent ⇒ chi returns 404):
```go
	if deps.Config.ExternalWorkflowEnabled && deps.ExternalWorkflowCallback != nil {
		r.With(middleware.RateLimitWith(deps.RateLimiter, 60, 1*time.Minute), middleware.MaxBodySize(1<<20)).
			Post("/v1/external-workflows/callback", deps.ExternalWorkflowCallback.Callback)
	}
```
Add `ExternalWorkflowCallback *handler.ExternalWorkflowCallbackHandler` to `RouterDeps`. (Place the route at the same `/v1` group level as other token-authed public routes; it is intentionally outside the JWT group.)

- [ ] **Step 3: Build**

Run: `gofmt -w -s internal/handler/external_workflow_callback_handler.go internal/router/router.go internal/service/external_workflow_service.go && go build ./...`
Expected: builds clean.

- [ ] **Step 4: Commit**

```bash
git add internal/handler/external_workflow_callback_handler.go internal/router/router.go internal/service/external_workflow_service.go
git commit -m "OPE-421: external-workflow callback endpoint (HMAC+token+nonce, gated route)"
```

---

## Task 10: Wire it all in main.go

**Files:** Modify `cmd/server/main.go`.

- [ ] **Step 1: Construct + wire (gated)**

After the orchestration repo/service + `fulfillmentService` are built, add:
```go
	externalWorkflowTokenRepo := repository.NewExternalWorkflowTokenRepository()
	externalWorkflowService := service.NewExternalWorkflowService(
		cfg.ExternalWorkflowEnabled, pool, workerPool, fulfillmentService, orchestrationRepo,
		externalWorkflowTokenRepo, auditRepo)
	automationExecutor.SetExternalWorkflow(externalWorkflowService)
	externalWorkflowCallbackHandler := handler.NewExternalWorkflowCallbackHandler(externalWorkflowService)
```
Add `externalWorkflowCallbackHandler` to `RouterDeps`. Register the dispatcher handler inside the existing `ORCHESTRATION_WORKER_ENABLED` block, additionally gated on `ExternalWorkflowEnabled`:
```go
	if cfg.ExternalWorkflowEnabled {
		orchestrationDispatcher.Register(service.EventExternalWorkflow,
			service.NewExternalWorkflowHandler(pool, webhookSafeHTTPClient, orchestrationRepo, loadExternalWorkflowConfig))
		orchestrationDispatcher.Register(service.EventExternalWorkflowCommand,
			service.NewExternalWorkflowCommandHandler(/* applies set_status/add_tag/add_note via the order service */))
	}
```
`webhookSafeHTTPClient` is the SSRF-safe client used by webhook dispatch (reuse it; expose it from the webhook dispatch service or build one with `noPrivateDialer`). `loadExternalWorkflowConfig` decrypts the integration credentials → `externalWorkflowConfig`. The command handler applies the follow-on command (set_status via the existing `AutomationStatusTransitionHandler` path; add_tag/add_note via the order service).

- [ ] **Step 2: Build + vet + full unit tests**

Run: `gofmt -w -s cmd/server/main.go && go build ./... && go vet ./... && go test ./... 2>&1 | grep -v "^ok\|no test files" | tail`
Expected: build + vet clean; no failing unit packages.

- [ ] **Step 3: Commit**

```bash
git add cmd/server/main.go
git commit -m "OPE-421: wire external-workflow service, handlers, dispatcher registration (gated)"
```

---

## Task 11: DB-bound integration tests

**Files:** Create `tests/integration/external_workflow_test.go` (`//go:build integration`).

Mirror the harness (`harness_test.go`, `fulfillment_read_api_test.go`): `appPool` + `WithTenant`, `superPool`, `seedTenant`, `seedFulfillmentOrder`.

- [ ] **Step 1: Write the integration tests**

Create `tests/integration/external_workflow_test.go` covering:
1. **Token hashed lookup** — issue a token (store `HashExternalWorkflowToken(raw)`), then `FindByHashCrossTenant(superPool, hash)` returns it; an unknown hash → `ErrExternalWorkflowTokenNotFound`.
2. **Callback resolves the correlated event + one in-scope follow-on** — seed an order + enqueue an `EventExternalWorkflow` event with a nonce; call `ResolveCallback` with `status=succeeded, command=add_tag` (token scope includes `add_tag`) → the event is no longer pending (succeeded) AND exactly one `EventExternalWorkflowCommand` event is enqueued; an audit row exists.
3. **Replay rejected** — calling `ResolveCallback` again with the same nonce → `ErrCorrelationNotResolvable` (single-use).
4. **Out-of-scope command rejected** — `command=set_status` with a token scope of only `{add_tag}` → error containing "not permitted"; the resolution still marked the event.
5. **Cross-tenant RLS** — a token + event for tenant A; tenant B's `WithTenant` count of `external_workflow_tokens` excludes A's.
6. **Timeout → blocker** — enqueue an event, run the handler twice (Attempts>0 on the second) → the second yields a permanent error → assert a `external_workflow_timeout` blocker is created via the worker path (or assert `model.Permanent` is returned and the worker's blocker path is exercised by the existing orchestration worker test harness).

Use real strict assertions; build each `EventExternalWorkflow` event payload with `correlation_nonce` + `integration_id` matching the token's integration.

- [ ] **Step 2: Run**

Run:
```bash
DATABASE_URL="postgres://openoms:openoms-dev-password@127.0.0.1:5433/openoms?sslmode=disable" \
go test -tags integration ./tests/integration/ -run 'ExternalWorkflow' -count=1
```
Expected: PASS. (Adjust seed columns to the real schema as needed; keep the assertions.)

- [ ] **Step 3: Commit**

```bash
git add tests/integration/external_workflow_test.go
git commit -m "OPE-421: external-workflow integration tests (token, callback resolve, replay, RLS, timeout)"
```

---

## Task 12: Full validation sweep

**Files:** none.

- [ ] **Step 1: gofmt + build + vet + full unit**

```bash
test -z "$(gofmt -l .)" && echo "gofmt clean" || gofmt -l .
go build ./... && go vet ./... && go vet -tags integration ./tests/integration/
go test ./... 2>&1 | grep -v "^ok\|no test files" | tail
```
Expected: clean; no failing unit packages.

- [ ] **Step 2: CI-pinned lint + migration roundtrip**

```bash
/tmp/glci29/golangci-lint run --new-from-rev=main --timeout=5m
```
Expected: `0 issues`. (Migration roundtrip already verified in Task 1.)

- [ ] **Step 3: Flag-off no-op + 404 sanity**

Confirm by reading the diff: the `external_workflow` executor case returns nil when the service is disabled; the callback route is only registered when `ExternalWorkflowEnabled`; the dispatcher handlers are only registered under both flags. So the default build is byte-for-byte unchanged and the callback path returns chi's 404.

- [ ] **Step 4: Full integration regression**

```bash
DATABASE_URL="postgres://openoms:openoms-dev-password@127.0.0.1:5433/openoms?sslmode=disable" \
go test -tags integration ./tests/integration/ -count=1 2>&1 | tail -3
```
Expected: `ok` (no regression).

---

## Self-Review (completed by plan author)

- **Spec coverage:** signed outbound action (Tasks 7,8) ✓; HMAC callback endpoint (Task 9) ✓; callback → typed commands only, never direct state (Task 6 follow-on via outbox) ✓; RBAC-scoped token model (Tasks 1,4,5) ✓; external execution id in orchestration_attempts (Tasks 1,6) ✓; redacted payload (Task 4 `BuildRedactedPayload`, Task 8 allowlist) ✓; timeout→warning/blocker (Task 7 + `criticality`) ✓; audit (Task 6) ✓; gated `EXTERNAL_WORKFLOW_ENABLED` (Task 3 + gated route/handler registration) ✓; test matrix (Tasks 4,11,12) ✓.
- **Reuse vs spec:** the spec's "pending-callback via next_attempt_at" is realized through a `DeferUntil` sentinel the worker maps to `MarkFailedRetry(id, deadline)` (Task 7) — the deadline IS the timeout (Task 7's `Attempts>0` re-dispatch branch). Migration is **000042** (000041 was supplier-availability).
- **Type consistency:** `EventExternalWorkflow`, `EventExternalWorkflowCommand`, `ExternalWorkflowToken`, `ExternalWorkflowCallbackRequest`, `BuildRedactedPayload`, `HashExternalWorkflowToken`, `Sign/VerifyExternalWorkflowSignature`, `ExternalWorkflowService.{Enabled,EnqueueDispatch,DispatchForOrder,ResolveCallback,ResolveCallbackHTTP}`, `ExternalWorkflowTokenRepository.{Issue,FindByHashCrossTenant,TouchLastUsed}`, `OrchestrationRepository.{FindPendingByEventAndNonce,SetLatestAttemptExternalExecID}`, `EnsureProcessForOrderUnconditional`, `ErrExternalWorkflowTokenNotFound`, `ErrCorrelationNotResolvable`, `ErrExternalWorkflowBadSignature`, `BlockerExternalWorkflowTimeout` are used consistently across tasks.
- **Implementation note for the executor:** before writing each task, READ the real signatures it reuses — `MarkFailedPermanent`, `MarkFailedRetry`, `scanOutboxEvent`, `orchestrationOutboxColumns`, the webhook SSRF client, `AuditRepo.Log`, `model.AuditEntry`, the handler `writeError`/`writeJSON`/`clientIP` helpers, the integration credential decrypt — and reconcile any mismatch (signatures/field names) against the actual code. The plan's code is a faithful guide, not a substitute for the real signatures.

## Deferred (out of scope, per spec)
External engine deployment / network policies / runbook (private infra repo); multi-template payloads (`template_id` + templates table); the registration/token-mint dashboard UI (API/model only here); outbound→alerting fan-out (already exists in private infra).
