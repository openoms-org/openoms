# OPE-309/OPE-306 OAuth Reauth Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make marketplace OAuth refresh workers decode stored JSONB credentials correctly and mark Allegro, Amazon, and eBay integrations as requiring reauthorization on terminal OAuth failures.

**Architecture:** Keep the change inside the API worker layer. Add a small shared JSONB string decoder for encrypted integration credentials, use it from existing cross-tenant integration readers and `OAuthRefresher`, and broaden terminal OAuth classification while preserving transient network errors as retryable.

**Tech Stack:** Go 1.25, pgx/v5, existing OpenOMS worker package tests, provider SDK sentinel errors where available.

---

## Scope

- Public repo only: `/Users/rafs/praca/openoms-dev/public`.
- Fixes Linear issues `OPE-309` and `OPE-306`.
- No dashboard changes.
- No database migration.
- No live Kubernetes changes in this implementation step.

## Files And Responsibilities

- Modify `apps/api-server/internal/worker/tenant_iterator.go`
  - Add shared `decodeIntegrationCredentialsJSONB(raw json.RawMessage) (string, error)`.
  - Replace duplicated JSONB string decoding in `ListActiveIntegrations` and `ListAllActiveMarketplaceIntegrations`.

- Modify `apps/api-server/internal/worker/oauth_refresher.go`
  - Scan `credentials` as `json.RawMessage`.
  - Decode with `decodeIntegrationCredentialsJSONB` before calling `crypto.Decrypt`.

- Modify `apps/api-server/internal/worker/integration_error.go`
  - Detect terminal OAuth credential errors for Allegro, OLX, Amazon, and eBay.
  - Keep temporary/non-auth errors retryable.

- Modify `apps/api-server/internal/worker/tenant_iterator_test.go`
  - Add regression tests for JSONB string credential decoding.

- Modify `apps/api-server/internal/worker/marketplace_order_poller_test.go`
  - Extend `TestIsTerminalOAuthCredentialError` for Allegro, Amazon, eBay and transient errors.

- Modify docs if needed:
  - `docs/system-documentation.md` only if worker behavior documentation needs an explicit OAuth reauth note after implementation review.

## Implementation Tasks

### Task 1: Prove JSONB Credential Decoding Regression

- [ ] Add tests in `apps/api-server/internal/worker/tenant_iterator_test.go`:

```go
func TestDecodeIntegrationCredentialsJSONB_StripsJSONBStringQuotes(t *testing.T) {
	raw, err := json.Marshal("encrypted-ciphertext")
	require.NoError(t, err)

	got, err := decodeIntegrationCredentialsJSONB(raw)

	require.NoError(t, err)
	assert.Equal(t, "encrypted-ciphertext", got)
}

func TestDecodeIntegrationCredentialsJSONB_RejectsNonStringJSON(t *testing.T) {
	_, err := decodeIntegrationCredentialsJSONB(json.RawMessage(`{"not":"a string"}`))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode integration credentials")
}
```

- [ ] Run RED:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/worker -run TestDecodeIntegrationCredentialsJSONB -count=1
```

Expected: fail because `decodeIntegrationCredentialsJSONB` does not exist.

- [ ] Implement `decodeIntegrationCredentialsJSONB` in `tenant_iterator.go`.

- [ ] Run GREEN:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/worker -run TestDecodeIntegrationCredentialsJSONB -count=1
```

Expected: pass.

### Task 2: Wire OAuthRefresher To The Shared Decoder

- [ ] In `oauth_refresher.go`, change `integrationRow.credentials` from `string` to `json.RawMessage`.

- [ ] After scanning the row, call:

```go
credentials, err := decodeIntegrationCredentialsJSONB(ir.credentials)
if err != nil {
	w.logger.Error("oauth refresh: decode credentials", "integration_id", ir.id, "error", err)
	continue
}
ir.credentials = credentials
```

Adjusted for the final field names so `crypto.Decrypt` receives the raw encrypted ciphertext string, not a JSON-quoted value.

- [ ] Reuse the helper in `tenant_iterator.go` for existing `ListActiveIntegrations` and `ListAllActiveMarketplaceIntegrations` paths.

- [ ] Run worker tests:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/worker -count=1
```

Expected: pass.

### Task 3: Prove Terminal OAuth Error Classification

- [ ] Extend `TestIsTerminalOAuthCredentialError` in `marketplace_order_poller_test.go`:

```go
assert.True(t, isTerminalOAuthCredentialError("allegro", allegrosdk.ErrUnauthorized))
assert.True(t, isTerminalOAuthCredentialError("allegro", allegrosdk.ErrForbidden))
assert.True(t, isTerminalOAuthCredentialError("amazon", errors.New(`amazon: token request failed (HTTP 400): {"error":"invalid_grant"}`)))
assert.True(t, isTerminalOAuthCredentialError("amazon", &amazonsdk.APIError{StatusCode: 403}))
assert.True(t, isTerminalOAuthCredentialError("ebay", ebaysdk.ErrUnauthorized))
assert.True(t, isTerminalOAuthCredentialError("ebay", errors.New(`ebay: token refresh failed (HTTP 400): {"error":"invalid_grant"}`)))
assert.False(t, isTerminalOAuthCredentialError("allegro", errors.New("temporary network timeout")))
assert.False(t, isTerminalOAuthCredentialError("amazon", errors.New("temporary network timeout")))
assert.False(t, isTerminalOAuthCredentialError("ebay", errors.New("temporary network timeout")))
```

- [ ] Add needed imports in the test:

```go
allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"
amazonsdk "github.com/openoms-org/openoms/packages/amazon-sp-sdk"
ebaysdk "github.com/openoms-org/openoms/packages/ebay-go-sdk"
```

- [ ] Run RED:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/worker -run TestIsTerminalOAuthCredentialError -count=1
```

Expected: fail for Allegro/Amazon/eBay terminal cases.

- [ ] Implement provider-specific terminal detection in `integration_error.go`.

- [ ] Run GREEN:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/worker -run TestIsTerminalOAuthCredentialError -count=1
```

Expected: pass.

### Task 4: Local Validation And Review

- [ ] Format Go files:

```bash
cd /Users/rafs/praca/openoms-dev/public
gofmt -w -s apps/api-server/internal/worker/tenant_iterator.go apps/api-server/internal/worker/oauth_refresher.go apps/api-server/internal/worker/integration_error.go apps/api-server/internal/worker/tenant_iterator_test.go apps/api-server/internal/worker/marketplace_order_poller_test.go
```

- [ ] Run targeted package tests:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/worker -count=1
```

- [ ] Run API tests if worker package is clean:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./... 2>&1 | tail -20
```

- [ ] Run diff review:

```bash
cd /Users/rafs/praca/openoms-dev/public
git diff --check
git diff --stat
git diff
```

- [ ] Commit only after tests pass and the diff matches scope:

```bash
cd /Users/rafs/praca/openoms-dev/public
git add apps/api-server/internal/worker/tenant_iterator.go apps/api-server/internal/worker/oauth_refresher.go apps/api-server/internal/worker/integration_error.go apps/api-server/internal/worker/tenant_iterator_test.go apps/api-server/internal/worker/marketplace_order_poller_test.go docs/superpowers/plans/2026-05-16-ope-309-ope-306-oauth-reauth.md
git commit -m "OPE-309/OPE-306: harden oauth reauth handling"
```

- [ ] Before push/PR, run full public local CI on clean HEAD:

```bash
cd /Users/rafs/praca/openoms-dev/public
./scripts/local-ci.sh
```

## Validation Plan

- Targeted RED/GREEN:
  - `go test ./internal/worker -run TestDecodeIntegrationCredentialsJSONB -count=1`
  - `go test ./internal/worker -run TestIsTerminalOAuthCredentialError -count=1`
- Package validation:
  - `go test ./internal/worker -count=1`
- Broader API validation:
  - `go test ./...`
- Pre-push gate:
  - `./scripts/local-ci.sh`

## Risk And Rollback

- Risk: overly broad terminal OAuth classification could mark a transient provider outage as requiring reauth.
- Mitigation: tests keep generic network errors retryable; detection is provider-scoped and focused on known unauthorized/forbidden/invalid-grant shapes.
- Rollback: revert the PR. Existing behavior resumes: OAuthRefresher may fail decrypting JSONB credentials and providers may keep retrying terminal auth errors.

## Open Questions

- Amazon and eBay SDK token refresh functions currently return generic formatted errors for token endpoint failures instead of sentinel errors. This plan keeps classification in the worker to avoid a wider SDK change, but a future SDK cleanup could introduce provider-specific `ErrInvalidGrant` sentinels.
