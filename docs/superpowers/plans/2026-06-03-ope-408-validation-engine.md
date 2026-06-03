# OPE-408 Validation Engine & Evidence Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development / executing-plans.

**Goal:** Probe definitions + immutable validation runs/results + evidence (redacted observations, payload hashes) + safety controls (destructive-probe confirmation) + auto-creation of integration gaps from failures. Closes the Studio backend (tor A) before the UI.

**Architecture:** Three platform tables (`provider_validation_probes`, `provider_validation_runs`, `provider_validation_results`). A `ProviderValidationService` owns the run state machine (start → record results → complete), computes the verdict, auto-creates gaps (via the OPE-407 capability repo), and enforces immutability + the destructive-probe gate. A separate `ProviderValidationHandler` wires endpoints under the existing `/v1/platform/providers/{id}/versions/{version_id}` group. The actual probe EXECUTION against live providers is deferred (needs adapters/credentials, OPE-413+); the engine records externally-supplied, already-redacted results.

**Tech Stack:** Go 1.25, chi/v5, pgx/v5, golang-migrate, testify. Builds on OPE-405/407.

---

## Scope

In scope: probe definitions (replace-set, frozen on publish), run lifecycle (`pending → passed|failed|error`), per-probe results (immutable once the run is finalized), evidence stored as redacted observation + payload_hash (never raw secrets), destructive-probe confirmation gate, auto-gap-creation on failure, endpoints, tests.

Out of scope: live probe execution against real providers (pluggable, OPE-413+ certification); evidence retention/cleanup jobs (OPE-425 lifecycle / OPE-422 ops); Studio UI (OPE-411).

## Enums

- probe_type: `auth_check|endpoint_reachability|feed_fetch|feed_parse|sample_catalog_read|sample_stock_read|sample_price_read|order_preflight|sandbox_order_create|order_status_read|shipment_tracking_read|invoice_read|webhook_signature_verification|malformed_payload_test|rate_limit_behavior`
- environment: `sandbox|production`
- run verdict: `pending|passed|failed|error`
- result status: `passed|failed|skipped|error`

## Data model (migration 000035) — no RLS, dual-role grants

```sql
CREATE TABLE public.provider_validation_probes (
    id uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    provider_version_id uuid NOT NULL REFERENCES public.provider_versions(id) ON DELETE CASCADE,
    probe_type text NOT NULL CHECK (probe_type IN (... 15 types ...)),
    label text NOT NULL,
    destructive boolean NOT NULL DEFAULT false,
    required boolean NOT NULL DEFAULT false,
    config jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider_version_id, label)
);
CREATE INDEX idx_provider_vprobes_version ON public.provider_validation_probes (provider_version_id); -- migrate:index-lock-ok

CREATE TABLE public.provider_validation_runs (
    id uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    provider_version_id uuid NOT NULL REFERENCES public.provider_versions(id) ON DELETE CASCADE,
    environment text NOT NULL CHECK (environment IN ('sandbox','production')),
    verdict text NOT NULL DEFAULT 'pending' CHECK (verdict IN ('pending','passed','failed','error')),
    started_by uuid REFERENCES public.users(id) ON DELETE SET NULL,
    started_at timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    notes text NOT NULL DEFAULT ''
);
CREATE INDEX idx_provider_vruns_version ON public.provider_validation_runs (provider_version_id, started_at DESC); -- migrate:index-lock-ok

CREATE TABLE public.provider_validation_results (
    id uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    run_id uuid NOT NULL REFERENCES public.provider_validation_runs(id) ON DELETE CASCADE,
    probe_type text NOT NULL,
    label text NOT NULL,
    status text NOT NULL CHECK (status IN ('passed','failed','skipped','error')),
    observation text NOT NULL DEFAULT '',   -- redacted safe summary (no secrets/PII)
    payload_hash text NOT NULL DEFAULT '',   -- hash for correlation, never the raw payload
    findings text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (run_id, label)
);
```
Dual-role grants DO-block. `.down.sql`: drop results, runs, probes.

## Implementation Tasks (TDD)

### Task 1: Migration + model
- [ ] `migrations/000035_provider_validation.up.sql`/`.down.sql`.
- [ ] `internal/model/provider_validation.go`: `ProviderValidationProbe`, `ProviderValidationRun`, `ProviderValidationResult` structs; enum constants + `IsValidProbeType/Environment/RunVerdict/ResultStatus`; `ValidateProbes([]ProviderValidationProbe) error` (valid type, non-empty unique label); `VerdictFromResults([]ProviderValidationResult) string` (any error→error, any failed→failed, none→error, else passed); `GapForFailedProbe(probeType, status) (gapType, severity string)` (auth_check→auth_failure; feed_parse/malformed_payload_test→parser_failure; order_preflight→missing_order_preflight; shipment_tracking_read→missing_tracking; else provider_business_error; severity error→system_error else action_required); `HashPayload([]byte) string` (sha256 hex).
- [ ] `model/provider_validation_test.go`: enum/validate/verdict/gap-mapping/hash matrices.

### Task 2: Repository
- [ ] `repository/provider_validation_repository.go`: `ReplaceProbes(ctx, q Querier, versionID, probes)` + `ListProbes(ctx, versionID)`; `CreateRun(ctx, q, versionID, env, startedBy) (*Run, error)`; `GetRun(ctx, runID)` (pgx.ErrNoRows); `ListRuns(ctx, versionID)`; `UpsertResult(ctx, q, runID, result)` (ON CONFLICT(run_id,label) DO UPDATE); `ListResults(ctx, runID)`; `FinalizeRun(ctx, q, runID, verdict, finishedAt)`.

### Task 3: Service `provider_validation_service.go`
- [ ] `NewProviderValidationService(pool, vers, caps, val)`. Errors: `ErrInvalidProbe`, `ErrInvalidValidationEnv`, `ErrInvalidResultStatus`, `ErrDestructiveProbeNotConfirmed`, `ErrValidationRunNotFound`, `ErrValidationRunFinalized`. (Reuse `ErrProviderVersionNotFound`/`Frozen` from the registry service package — same package.)
- [ ] `SetProbes`/`GetProbes` (frozen-check, validate, replace in tx).
- [ ] `StartRun(versionID, env, allowDestructive, actorID)`: version exists (404); valid env; load probes; if any destructive && !allowDestructive → `ErrDestructiveProbeNotConfirmed`; create run (pending). (No live execution — results recorded separately.)
- [ ] `RecordResult(runID, probeType, label, status, observation, payloadHash, findings)`: run exists + verdict==pending (else `ErrValidationRunFinalized`); valid status; upsert result.
- [ ] `CompleteRun(runID)`: run pending (else finalized); compute `VerdictFromResults`; finalize (set verdict + finished_at) and auto-create a gap per failed/error result — all in one tx.
- [ ] `ListRuns`/`GetRunWithResults`.

### Task 4: Handler + router
- [ ] `handler/provider_validation_handler.go` with a `ProviderValidation` service interface + methods: GetProbes/UpdateProbes, StartRun (POST /validate), ListRuns, GetRun (GET /validation-runs[/{run_id}]), RecordResult (POST /validation-runs/{run_id}/results), CompleteRun (POST /validation-runs/{run_id}/complete). Map errors → 404/422; audit mutations.
- [ ] RouterDeps `ProviderValidation *handler.ProviderValidationHandler`; routes under the version group: `GET/PATCH /probes` (read/write), `POST /validate` (validate), `GET /validation-runs` (read), `GET /validation-runs/{run_id}` (read), `POST /validation-runs/{run_id}/results` (validate), `POST /validation-runs/{run_id}/complete` (validate). Bootstrap wiring.
- [ ] Handler tests (fake service): destructive→422, finalized→422, run not-found→404, success audits.

### Task 5: Integration test
- [ ] `tests/integration/provider_validation_test.go`: set probes (incl. a destructive one) → StartRun without confirm → `ErrDestructiveProbeNotConfirmed` → StartRun(allowDestructive) → RecordResult passed + RecordResult failed(auth_check) → CompleteRun → verdict=failed, finished_at set, an `auth_failure` gap auto-created (assert via the capability repo ListGaps) → RecordResult after complete → `ErrValidationRunFinalized` → probes frozen after publish. appPool.

### Task 6: Validate + docs
- [ ] go test ./..., vet, gofmt, golangci-lint; integration green; migrate down/up.
- [ ] `docs/system-documentation.md` — validation engine + endpoints under the Studio block.

## Risks
- Run immutability — enforce verdict==pending gate on RecordResult/CompleteRun; CompleteRun finalize+gaps in one tx.
- Evidence safety — model stores only observation (redacted) + payload_hash; no column accepts raw payloads. `HashPayload` provided for callers.
- Destructive gate — StartRun rejects unless `allow_destructive` when any probe is destructive.
- simple-protocol nullable casts (e.g. finished_at) — cast `$n::timestamptz` (learned in OPE-407).

## Self-Review
Covers OPE-420 Task 7: probe definitions; immutable runs (verdict/env/actor/started-finished); probe-level results with observations + safe payload hashes + findings; destructive-probe confirmation; gaps created from failures. Live execution + retention are later tasks.
