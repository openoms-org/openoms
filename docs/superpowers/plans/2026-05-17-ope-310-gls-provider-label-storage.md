# OPE-310 GLS Provider Label Storage Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent the GLS carrier provider created through the production factory from panicking when labels are generated without injected object storage.

**Architecture:** Keep the change inside the GLS carrier adapter. GLS returns label bytes inline during `CreateShipment`, and `LabelService.GenerateLabel` calls `CreateShipment` and `GetLabel` on the same provider instance, so the provider should retain those inline label bytes in memory for the immediate `GetLabel` call and return a clear error when no label is available. This avoids widening the global carrier factory API or coupling all carrier providers to object storage for one GLS-specific behavior.

**Tech Stack:** Go 1.25, GLS provider adapter, existing Go unit tests under `apps/api-server/internal/integration/carriers`.

---

## Scope

- Fix only the public repository GLS carrier adapter.
- Do not change production Helm values, enterprise infra, or global carrier factory signatures.
- Do not add fake GLS label retrieval through the carrier API; GLS labels are inline in `CreateShipment`.

## Files

- Modify: `apps/api-server/internal/integration/carriers/gls.go`
- Modify: `apps/api-server/internal/integration/carriers/gls_test.go`
- Add: this plan file under `docs/superpowers/plans/`

## Tasks

### Task 1: Branch and Linear State

- [ ] Move `OPE-310` to `In Progress`.
- [ ] Create branch `fix/OPE-310-gls-label-storage` in `public/`.

### Task 2: RED Test

- [ ] Add a regression test to `apps/api-server/internal/integration/carriers/gls_test.go` that constructs GLS with `NewGLSProvider(..., nil)` and uses a mocked GLS API response with `CreatedShipment.PrintData`.
- [ ] The test must call `CreateShipment` and then `GetLabel` on the same provider instance.
- [ ] Expected RED behavior before implementation: test fails because the current provider dereferences nil storage during `CreateShipment`.

Target command:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/integration/carriers -run TestGLS_CreateShipmentWithoutStorageCachesInlineLabel -count=1
```

### Task 3: GREEN Implementation

- [ ] Add an in-memory label cache field to `GLSProvider`.
- [ ] Initialize the cache in `NewGLSProvider`.
- [ ] In `CreateShipment`, after decoding inline GLS label bytes, store them in the cache by external ID.
- [ ] If object storage is configured, keep the existing upload attempt as an optional side effect.
- [ ] In `GetLabel`, return cached bytes when present.
- [ ] If the cache misses and object storage is configured, keep the existing storage fallback.
- [ ] If neither cache nor storage has the label, return a clear `gls: label not available for external id ...` error instead of panicking.

### Task 4: Regression and Package Validation

- [ ] Run the targeted RED/GREEN test.
- [ ] Run all carrier integration tests:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/integration/carriers -count=1
```

- [ ] Run relevant service tests if provider contract changes are broader:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/service ./internal/integration/carriers -run 'Label|GLS|Carrier' -count=1
```

### Task 5: Self-Review, Full Validation, PR

- [ ] Run `gofmt -w -s` on touched Go files.
- [ ] Run `git diff --check`, `git diff --stat`, and review the full diff.
- [ ] Run full public validation:

```bash
cd /Users/rafs/praca/openoms-dev/public
./scripts/local-ci.sh
```

- [ ] Commit with message `OPE-310: prevent GLS label storage panic`.
- [ ] Push branch and open PR titled `OPE-310: prevent GLS label storage panic`.
- [ ] Check CI, CodeQL, Docker, Security Tests, and CodeRabbit comments before merge.

## Risk and Rollback

- Risk is low and isolated to GLS provider label handling.
- Behavior improves fail-closed semantics: missing GLS labels return an error instead of panic.
- Rollback is reverting the PR; no migration or data change is involved.
- If CodeRabbit or tests reveal that label bytes must be persisted outside the provider instance, create a follow-up to inject `storage.ObjectStorage` into `LabelService`/carrier factory deliberately rather than doing it as a hidden broad refactor.

