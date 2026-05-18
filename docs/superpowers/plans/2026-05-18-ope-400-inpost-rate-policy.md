# OPE-400 InPost Rate Policy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop OpenOMS from presenting InPost hardcoded estimated prices as production carrier quotes.

**Architecture:** Treat InPost rate shopping like every carrier without contract-backed/certified rating support: the provider returns `integration.ErrCarrierRatesNotImplemented`, and `RateService` skips it while preserving the existing `200 {"rates":[]}` API contract. InPost shipment creation, labels, tracking, pickup points, dispatch orders and webhooks stay unchanged.

**Tech Stack:** Go 1.25, chi API server, carrier provider abstraction, existing `integration.ErrCarrierRatesNotImplemented`, Markdown project documentation.

---

## Decision

Use the conservative client-ready policy from OPE-400 triage: hide InPost rate-shopping until OpenOMS has a real contract-backed pricing source. This avoids showing operational users prices that look like guaranteed carrier quotes but are only static domestic estimates.

## File Structure

- Modify `apps/api-server/internal/integration/carriers/inpost_test.go`
  - Replace the old hardcoded pricing expectations with a regression test that InPost returns `ErrCarrierRatesNotImplemented`.
- Modify `apps/api-server/internal/integration/carriers/inpost.go`
  - Replace static domestic PLN tiers in `GetRates` with the sentinel error.
- Modify `docs/system-documentation.md`
  - Document that InPost is also skipped by `/v1/shipping/rates` until certified pricing exists.
- Modify `.claude/context/API_CONTRACTS.md`
  - Record the API behavior change for `/v1/shipping/rates`.
- Modify `.claude/context/SECURITY_POSTURE.md`
  - Close the prior InPost follow-up note.
- Modify `docs/audit/provider-readiness-2026-05-11.md`
  - Update the InPost provider readiness row from `controlled for rates` to hidden/not contract-backed for rates.
- Modify `docs/audit/feature-readiness-matrix-2026-05-11.md`
  - Update the Logistics/InPost matrix row so rates are not exposed as a client-ready capability.

## Implementation Tasks

### Task 1: Add Failing InPost Rate Policy Test

**Files:**
- Modify: `apps/api-server/internal/integration/carriers/inpost_test.go`

- [ ] **Step 1: Replace the old pricing test with a sentinel-error regression**

Replace the whole `TestInPostGetRates` function with:

```go
func TestInPostGetRates_NotImplemented(t *testing.T) {
	provider := newTestProvider(t, "http://unused")

	rates, err := provider.GetRates(context.Background(), integration.RateRequest{
		FromCountry: "PL",
		ToCountry:   "PL",
		Weight:      2,
		Width:       20,
		Height:      5,
		Length:      30,
		COD:         100,
	})
	if !errors.Is(err, integration.ErrCarrierRatesNotImplemented) {
		t.Fatalf("GetRates() error = %v, want ErrCarrierRatesNotImplemented", err)
	}
	if rates != nil {
		t.Fatalf("GetRates() rates = %#v, want nil", rates)
	}
}
```

Add `errors` to the imports and remove `math` if it is no longer used. Remove the `floatClose` helper if it is no longer referenced.

- [ ] **Step 2: Run test and verify RED**

Run:

```bash
cd apps/api-server && go test ./internal/integration/carriers -run TestInPostGetRates_NotImplemented -count=1
```

Expected: fail because current `InPostProvider.GetRates` returns hardcoded rates with `nil` error.

### Task 2: Hide InPost Estimated Rates in Provider

**Files:**
- Modify: `apps/api-server/internal/integration/carriers/inpost.go`

- [ ] **Step 1: Replace hardcoded pricing implementation**

Replace `func (p *InPostProvider) GetRates(...)` with:

```go
// GetRates reports that InPost contract-backed rate shopping is not implemented.
func (p *InPostProvider) GetRates(_ context.Context, _ integration.RateRequest) ([]integration.Rate, error) {
	return nil, fmt.Errorf("inpost: %w", integration.ErrCarrierRatesNotImplemented)
}
```

- [ ] **Step 2: Run focused tests and verify GREEN**

Run:

```bash
cd apps/api-server && go test ./internal/integration/carriers -run 'GetRates|CarrierGetRates' -count=1
```

Expected: pass; InPost and all other unsupported carrier rating providers return `ErrCarrierRatesNotImplemented`.

### Task 3: Update Documentation and Readiness Matrix

**Files:**
- Modify: `docs/system-documentation.md`
- Modify: `.claude/context/API_CONTRACTS.md`
- Modify: `.claude/context/SECURITY_POSTURE.md`
- Modify: `docs/audit/provider-readiness-2026-05-11.md`
- Modify: `docs/audit/feature-readiness-matrix-2026-05-11.md`

- [ ] **Step 1: Update API/system docs wording**

In `docs/system-documentation.md`, update the rate-shopping note to list InPost together with unsupported rating providers:

```markdown
Rate shopping zwraca tylko stawki z providerow z zaakceptowana implementacja wyceny. InPost, DPD, GLS, UPS, Poczta Polska, Orlen Paczka, FedEx i DHL zwracaja `ErrCarrierRatesNotImplemented`, a endpoint pomija je zamiast pokazywac placeholderowe albo szacunkowe ceny.
```

In `.claude/context/API_CONTRACTS.md`, add a new `2026-05-18` recently changed entry:

```markdown
- 2026-05-18: InPost rate-shopping policy:
  - InPost no longer contributes hardcoded domestic estimated PLN rates to `POST /v1/shipping/rates`.
  - `InPostProvider.GetRates` now returns internal `integration.ErrCarrierRatesNotImplemented`; `RateService` skips it and preserves `200 {"rates":[]}` when no carrier can quote.
```

In `.claude/context/SECURITY_POSTURE.md`, replace the OPE-298 follow-up wording with:

```markdown
- OPE-400/OPE-298: hardcoded placeholder or estimated `GetRates` pricing is hidden for InPost, DPD, GLS, UPS, Poczta Polska, Orlen Paczka, FedEx, and DHL. These providers return `integration.ErrCarrierRatesNotImplemented`, and `RateService` skips them so tenants do not choose carriers based on fabricated or non-contract-backed prices.
```

- [ ] **Step 2: Update readiness matrix wording**

In `docs/audit/provider-readiness-2026-05-11.md`, update the InPost carrier row to:

```markdown
| InPost | Create shipment, label, tracking, cancel, dispatch order, punkty odbioru, webhook i Geowidget są podłączone. Rate shopping jest ukryty, dopóki nie ma kontraktowego źródła wyceny. | `ready` dla labels/tracking/points, `blocked` dla rates | Pokazać labels/tracking/points, jeśli credentiale są poprawne; nie pokazywać rate-shopping jako wyceny InPost. |
```

In `docs/audit/feature-readiness-matrix-2026-05-11.md`, update the InPost row to:

```markdown
| InPost | provider | `ready` dla labels/tracking/points, `blocked` dla rates | Pokazać po credentialach; rate shopping ukryty | Manager InPost, token, organization_id, label, tracking, paczkomat; kontraktowe źródło wyceny wymagane przed pokazaniem stawek. |
```

### Task 4: Final Verification

**Files:**
- All modified files

- [ ] **Step 1: Format and focused verification**

Run:

```bash
cd apps/api-server && gofmt -w internal/integration/carriers/inpost.go internal/integration/carriers/inpost_test.go
cd apps/api-server && go test ./internal/integration/carriers ./internal/service -run 'Rate|GetRates|CarrierGetRates' -count=1
```

Expected: pass.

- [ ] **Step 2: Repository checks before PR**

Run:

```bash
git diff --check
./scripts/local-ci.sh --quick
```

Expected: both pass. Before pushing the public repo branch, run full:

```bash
./scripts/local-ci.sh
```

Expected: pass before PR.

## Risk and Rollback

- Risk: tenants with active InPost integration will receive fewer/no `/v1/shipping/rates` options. This is intentional because static estimates must not look like carrier-guaranteed quotes.
- No risk to label creation, tracking, pickup points, dispatch orders or InPost webhooks; those methods are not touched.
- Rollback: revert the OPE-400 PR to restore static estimates, but only if product explicitly accepts that static estimates can be shown as estimates.

## Self-Review

- Spec coverage: OPE-400 decision, API behavior, docs and readiness matrix are covered.
- Placeholder scan: no TBD/TODO placeholders are used.
- Type consistency: the sentinel error remains `integration.ErrCarrierRatesNotImplemented`; service behavior remains unchanged and uses `errors.Is`.
