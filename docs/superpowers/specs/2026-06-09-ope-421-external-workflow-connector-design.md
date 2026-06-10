# OPE-421 / Phase-13 — External-Workflow ("Custom Action") Connector Design

**Status:** Approved design (no implementation yet)
**Date:** 2026-06-09
**Epic:** OPE-403 tor B (Fulfillment Orchestration) — Phase 13, an OPE-421 followup
**Scope decision:** the **app-side** connector only. Generic "external workflow engine" — NOT hardcoded to any specific tool. The reference consumer's deployment (network policies, runbook, the hub itself) lives in the private infrastructure repo and is an explicit follow-up, out of this spec.

## Goal

Give automation a **safe extension point**: an automation action can hand an event to an external workflow engine to enrich it (call a third-party API, transform data, branch on external state), and the external engine can hand a result back — **without ever owning or directly mutating core fulfillment state**. The external engine can only resolve the specific event it was handed, and at most submit one whitelisted typed command that flows through the already-merged, audited orchestration path.

## Context (already merged)

- Fulfillment orchestration (ADR-424): `orchestration_outbox` (idempotent enqueue on `(tenant_id, idempotency_key)`, statuses pending/claimed/succeeded/failed, retry/backoff, permanent→`fulfillment_blocker`), `orchestration_attempts` (per-attempt status + error), `OrchestrationWorker` (claim `FOR UPDATE SKIP LOCKED` → dispatch → finish), `OrchestrationDispatcher` (event_type → handler registry), gated `ORCHESTRATION_WORKER_ENABLED`.
- Automation `set_status → outbox` routing (OPE-421) + the typed-action registry (`model.automationActionSpecs`, save-time validation).
- Outgoing webhook HMAC signing (`webhook_dispatch_service` / `webhook_service`: `X-Signature-256`, `X-OpenOMS-Event`, SSRF-safe `noPrivateDialer` client) and incoming HMAC verification — reused here.
- RBAC-scoped token pattern: `supplier_portal_tokens` (`token_hash`, `expires_at`, `last_used_at`) — mirrored here.
- `integrations` table: per-tenant, `credentials` JSONB AES-256-GCM encrypted.

## Architecture

```
automation rule (action: external_workflow)
        │  fire
        ▼
enqueue orchestration_outbox event  type=automation.external_workflow
        │   {integration_id, correlation_nonce, redacted_payload, order_id}
        ▼
OrchestrationWorker → ExternalWorkflowHandler (dispatch)
        │   signed POST (X-Signature-256 + X-OpenOMS-Correlation) → integration.outbound_url
        │   record attempt + external_execution_id; leave event PENDING-CALLBACK (claimed, not succeeded)
        ▼
   external workflow engine does its work, then ...
        │
POST /v1/external-workflows/callback   (HMAC body + RBAC token + correlation_nonce)
        │   verify → resolve the correlated attempt/event (succeeded|failed)
        │   optional: ONE whitelisted command (set_status|add_tag|add_note) → follow-on outbox event
        │   audit
        ▼
   (no callback within timeout) → sweep → resolve timed-out → external_workflow_timeout blocker (per criticality)
```

All of it is gated behind `EXTERNAL_WORKFLOW_ENABLED` (default `false`) — off ⇒ the action type is registered but its dispatch/callback paths are inert, so behavior is unchanged.

## Components

### 1. Registration

A per-tenant integration of provider `external_workflow`, reusing the existing `integrations` table. Its AES-encrypted `credentials` JSONB holds:
- `outbound_url` — where the signed event is POSTed.
- `signing_secret` — HMAC key for both the outbound signature and inbound callback verification.
- `timeout_seconds` — how long to wait for a callback before timing out.
- `criticality` — `warning` (a non-halting, visible blocker) or `blocker` (a hard blocker that holds the process).
- `outbound_field_allowlist` — the order fields permitted in the outbound payload (see Redaction).

No new registration table (the `integrations` infra already does encryption, per-tenant scoping, and CRUD).

### 2. RBAC-scoped callback token — new table `external_workflow_tokens`

Mirrors `supplier_portal_tokens`. One or more rotateable tokens per integration. Tenant-scoped, FORCE RLS.

| Column | Type | Notes |
|---|---|---|
| `id` | uuid pk | |
| `tenant_id` | uuid NOT NULL → tenants | RLS key |
| `integration_id` | uuid NOT NULL → integrations | the external_workflow integration |
| `token_hash` | text NOT NULL | SHA-256 of the bearer token (the raw token is shown once at creation) |
| `scopes` | text[] NOT NULL DEFAULT `'{}'` | the whitelisted callback commands this token may submit, e.g. `{set_status, add_tag, add_note}`; empty = resolve-only |
| `expires_at` | timestamptz | optional expiry |
| `last_used_at` | timestamptz | updated on each accepted callback |
| `created_at` | timestamptz NOT NULL DEFAULT now() | |

`UNIQUE (tenant_id, token_hash)`. The token authenticates the *caller* (which integration); the per-call `correlation_nonce` selects *which event* and provides replay protection.

### 3. Outbound action — `external_workflow`

A new automation action type added to the OPE-421 registry (`model.automationActionSpecs`): required param `integration_id`. When fired by the engine, the action:
1. Resolves the `external_workflow` integration; if missing/disabled → no-op (or a `manual_stock_review`-style note — see Errors).
2. Builds a **redacted payload** from the order using `outbound_field_allowlist` (no secrets/PII outside the allowlist) — mirroring the existing `send_marketplace_message` variable-allowlist approach.
3. Generates a `correlation_nonce` (random, single-use).
4. Enqueues an `orchestration_outbox` event `automation.external_workflow` with payload `{integration_id, correlation_nonce, redacted_payload, order_id}` and idempotency key `automation.external_workflow:<rule_id>:<order_id>:<nonce>`.

### 4. Dispatch handler — `ExternalWorkflowHandler`

Registered on the `OrchestrationDispatcher` for `automation.external_workflow` (only when `ORCHESTRATION_WORKER_ENABLED` **and** `EXTERNAL_WORKFLOW_ENABLED`). On dispatch:
1. Loads the integration + decrypts the signing secret.
2. Builds the signed request (reuse `webhook_dispatch`): body = the redacted payload + correlation, headers `X-Signature-256` (HMAC-SHA256 over the body), `X-OpenOMS-Event: automation.external_workflow`, `X-OpenOMS-Correlation: <nonce>`. POSTs via the SSRF-safe client to `outbound_url`.
3. Records the attempt; stores the synchronously-returned external execution id in `orchestration_attempts.external_execution_id` (new column).
4. **Does not mark the outbox event succeeded** — it puts the event back to `pending` with `next_attempt_at = now + timeout_seconds`. This reuses the merged "claim only *due* rows" machinery as the timeout deadline: the worker will not re-pick the event before the deadline, the callback resolves it (→ `succeeded`/`failed`) first in the normal case, and **if the deadline passes with no callback the event becomes due again** and the handler — seeing the correlation still unresolved — applies the timeout policy (§6). A non-2xx / transport error on the POST is a normal retryable attempt failure (existing backoff); on exhaustion, the existing permanent→blocker path. (This means the dispatch handler is idempotent: a re-dispatch of an already-dispatched, still-in-flight correlation re-sends nothing — it checks the correlation state first.)

### 5. Callback endpoint — `POST /v1/external-workflows/callback`

No JWT. Authenticated by: (a) the bearer **token** (looked up by `token_hash`), (b) the **HMAC** signature of the body against the integration's `signing_secret`, and (c) the **correlation_nonce** echoed in the body. The handler:
1. Verifies the HMAC (constant-time compare) and resolves the token → integration + scopes; updates `last_used_at`.
2. Looks up the in-flight `automation.external_workflow` outbox event by `(tenant, correlation_nonce)`; rejects if not found, already resolved, expired, or belongs to a different integration (**single-use** ⇒ replay protection).
3. Resolves the attempt + outbox event as `succeeded` or `failed` per the callback's `status`.
4. If the callback carries an optional **result command** (`set_status` | `add_tag` | `add_note`) **and** that command is in the token's `scopes`: enqueues exactly ONE follow-on outbox event applying it through the existing orchestrator (idempotent, state-machine-respecting, audited). A command outside scope → 403, the resolution still stands.
5. Writes an `audit_log` entry (`action = external_workflow.callback`, the integration, command, external execution id).

Rate-limited (like the other public/token endpoints). Returns 200 with the resolution outcome.

### 6. Timeout policy

The timeout is the `next_attempt_at = now + timeout_seconds` deadline set at dispatch (§4): when it passes with no callback, the event becomes due, the worker re-dispatches, and the handler — finding the correlation still unresolved past its deadline — resolves it as timed-out (`failed`) and creates an `external_workflow_timeout` blocker whose severity follows the integration's `criticality` (`warning` → an action-required-but-non-halting blocker; `blocker` → a hard blocker that holds the process). No separate sweep loop is needed — the existing due-row claim machinery *is* the timeout. (This naturally also bounds the OPE-415-noted "crash-after-claim stuck rows" risk for this event type, since the deadline guarantees the event is revisited.)

### 7. Redaction

The outbound payload is built strictly from `outbound_field_allowlist` (an allowlist of order field paths). Nothing else leaves OpenOMS — credentials, internal ids, and unlisted customer PII are never serialized. Mirrors the established `send_marketplace_message` variable-allowlist enforcement. v1 ships one allowlist per integration; multi-template payloads are deferred.

## Data model changes

- New table `external_workflow_tokens` (FORCE RLS, tenant_isolation, app-role grants — OPE-414 pattern).
- New column `orchestration_attempts.external_execution_id text` (additive, nullable) — the Phase-13 "store execution id in orchestration_attempts" requirement.
- New blocker code `external_workflow_timeout` (category `integration`) in `model/fulfillment.go`.
- Migration is additive only (Migration-Safety: no destructive ops; each new index + its `-- migrate:index-lock-ok` on one line).

## Gating / safety

- `EXTERNAL_WORKFLOW_ENABLED` (env, default `false`). Off ⇒ the dispatch handler is not registered and the callback endpoint returns `404 feature_not_available` (the readiness-gate pattern), so the connector is fully inert.
- Every durable effect flows through the merged outbox (idempotent, audited, retryable, blocker-on-failure). The external engine never writes durable state directly.
- Defense in depth on the callback: token (hashed, scoped, rotateable, expirable) + HMAC body signature + single-use correlation nonce + RLS + rate limit.
- Outbound is SSRF-safe (`noPrivateDialer`) and signed.

## Testing

- **Unit:** redaction allowlist (no unlisted field leaks); the new action's enqueue (correlation nonce + idempotency key); the resolver's status mapping; scope enforcement (command outside scope rejected); HMAC sign/verify round-trip.
- **DB-bound integration:** token issue + hashed lookup; callback resolves the correlated event (succeeded/failed) and enqueues exactly one in-scope follow-on command; replay of a consumed nonce is rejected; cross-tenant isolation on `external_workflow_tokens` and the correlated event lookup; timeout sweep creates the `external_workflow_timeout` blocker per criticality; flag-off ⇒ callback 404 + no dispatch.
- **Callback signature tests:** wrong HMAC rejected; expired/unknown token rejected; nonce for another integration rejected.

## Explicitly deferred (separate work)

- The external engine's **deployment**: in-cluster install, network policies allowing it to reach only the approved OpenOMS endpoints, and the operational runbook — these live in the private infrastructure repo.
- Multi-template outbound payloads (a `template_id` action param + a templates table) — v1 uses one allowlist per integration.
- Outbound→alerting fan-out (engine → ticketing/chat) — already exists in the private infra and is unrelated to this connector boundary.
- A dashboard UI to register integrations / mint tokens (the API + model only here).
