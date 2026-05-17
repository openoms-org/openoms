# OPE-298 Hide Stub Carrier Rates Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop `/v1/shipping/rates` from returning fabricated carrier prices for carriers that do not have a real rate-shopping integration.

**Architecture:** Add a shared sentinel error in the carrier integration contract for providers whose rating API is not implemented. Unsupported carriers return that error instead of hardcoded rates, and `RateService` treats it as a non-fatal skip while still warning on real provider failures. InPost rates remain unchanged in this PR because `OPE-298` targets the fake non-InPost provider prices; InPost pricing should be tracked separately if we decide to hide all estimated pricing.

**Tech Stack:** Go 1.25, chi service layer, carrier provider interfaces, `errors.Is`, existing carrier unit tests.

---

## Scope

`OPE-298` lists six carriers with TODO/stubbed pricing:

- DPD
- GLS
- UPS
- Poczta Polska
- Orlen Paczka
- FedEx

During inspection, DHL was also found to return hardcoded estimates from `GetRates`, even though it is not listed in the issue because the method is not marked with a TODO. To keep the production endpoint honest, include DHL in this PR as the same behavior class: no real rating API, no returned price. InPost stays unchanged and should be reviewed as a follow-up because the current product audit treats it as controlled/accepted estimate behavior.

## Files

- Modify: `apps/api-server/internal/integration/carrier.go`
  - Add `ErrCarrierRatesNotImplemented`.
- Modify: `apps/api-server/internal/integration/carriers/dpd.go`
  - Replace fake rate table with sentinel error.
- Modify: `apps/api-server/internal/integration/carriers/gls.go`
  - Replace fake rate table with sentinel error.
- Modify: `apps/api-server/internal/integration/carriers/ups.go`
  - Replace fake rate table with sentinel error.
- Modify: `apps/api-server/internal/integration/carriers/poczta_polska.go`
  - Replace fake rate table with sentinel error.
- Modify: `apps/api-server/internal/integration/carriers/orlen_paczka.go`
  - Replace fake rate table with sentinel error.
- Modify: `apps/api-server/internal/integration/carriers/fedex.go`
  - Replace fake rate table with sentinel error.
- Modify: `apps/api-server/internal/integration/carriers/dhl.go`
  - Replace fake rate table with sentinel error.
- Modify: `apps/api-server/internal/integration/carriers/dpd_test.go`
  - Replace fake-pricing tests with not-implemented assertions.
- Modify: `apps/api-server/internal/integration/carriers/gls_test.go`
  - Replace fake-pricing tests with not-implemented assertions.
- Modify: `apps/api-server/internal/integration/carriers/dhl_test.go`
  - Replace fake-pricing tests with not-implemented assertions.
- Create: `apps/api-server/internal/integration/carriers/rates_not_implemented_test.go`
  - Cover UPS, Poczta Polska, Orlen Paczka, FedEx direct provider behavior.
- Modify: `apps/api-server/internal/service/rate_service.go`
  - Skip `ErrCarrierRatesNotImplemented` without warning.
- Modify: `docs/system-documentation.md`
  - Document that non-certified carrier rating providers return `ErrCarrierRatesNotImplemented`.
- Local context update: `.claude/context/API_CONTRACTS.md` (ignored by git)
  - Update `/v1/shipping/rates` contract notes.
- Local context update: `.claude/context/SECURITY_POSTURE.md` (ignored by git)
  - Mark the hardcoded placeholder carrier rates risk as remediated for the affected providers and leave an InPost follow-up note.
- Local context update: `.claude/context/PROJECT_STATE.md` (ignored by git)
  - Remove stale references to the remediated placeholder-rate backlog item.

---

### Task 1: Write Failing Carrier Tests

**Files:**
- Modify: `apps/api-server/internal/integration/carriers/dpd_test.go`
- Modify: `apps/api-server/internal/integration/carriers/gls_test.go`
- Modify: `apps/api-server/internal/integration/carriers/dhl_test.go`
- Create: `apps/api-server/internal/integration/carriers/rates_not_implemented_test.go`

- [ ] **Step 1: Add `errors` import where needed**

In `dpd_test.go`, `gls_test.go`, and `dhl_test.go`, add:

```go
import (
	"context"
	"encoding/json"
	"errors"
	...
)
```

- [ ] **Step 2: Replace DPD fake pricing tests**

Replace `TestDPD_GetRates_DomesticPricing` and `TestDPD_GetRates_CODSurcharge` with:

```go
func TestDPD_GetRates_NotImplemented(t *testing.T) {
	provider := newTestDPDProvider(t, "http://unused")

	rates, err := provider.GetRates(context.Background(), integration.RateRequest{
		FromCountry: "PL",
		ToCountry:   "PL",
		Weight:      5.0,
		COD:         100.0,
	})
	if !errors.Is(err, integration.ErrCarrierRatesNotImplemented) {
		t.Fatalf("GetRates() error = %v, want ErrCarrierRatesNotImplemented", err)
	}
	if rates != nil {
		t.Fatalf("GetRates() rates = %#v, want nil", rates)
	}
}
```

- [ ] **Step 3: Replace GLS fake pricing tests**

Replace `TestGLS_GetRates_DomesticPricing`, `TestGLS_GetRates_InternationalReturnsEmpty`, and `TestGLS_GetRates_OverweightReturnsEmpty` with:

```go
func TestGLS_GetRates_NotImplemented(t *testing.T) {
	provider := newTestGLSProvider(t, "http://unused")

	rates, err := provider.GetRates(context.Background(), integration.RateRequest{
		FromCountry: "PL",
		ToCountry:   "PL",
		Weight:      5.0,
	})
	if !errors.Is(err, integration.ErrCarrierRatesNotImplemented) {
		t.Fatalf("GetRates() error = %v, want ErrCarrierRatesNotImplemented", err)
	}
	if rates != nil {
		t.Fatalf("GetRates() rates = %#v, want nil", rates)
	}
}
```

- [ ] **Step 4: Replace DHL fake pricing tests**

Replace `TestDHL_GetRates_DomesticPricing_WeightTiers` and `TestDHL_GetRates_InternationalReturnsEmpty` with:

```go
func TestDHL_GetRates_NotImplemented(t *testing.T) {
	provider := newTestDHLProvider(t, "http://unused")

	rates, err := provider.GetRates(context.Background(), integration.RateRequest{
		FromCountry: "PL",
		ToCountry:   "PL",
		Weight:      3.0,
	})
	if !errors.Is(err, integration.ErrCarrierRatesNotImplemented) {
		t.Fatalf("GetRates() error = %v, want ErrCarrierRatesNotImplemented", err)
	}
	if rates != nil {
		t.Fatalf("GetRates() rates = %#v, want nil", rates)
	}
}
```

- [ ] **Step 5: Add coverage for carriers without existing rate tests**

Create `rates_not_implemented_test.go`:

```go
package carriers

import (
	"context"
	"errors"
	"testing"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
)

func TestCarrierGetRates_NotImplementedProviders(t *testing.T) {
	tests := []struct {
		name     string
		provider integration.CarrierProvider
	}{
		{name: "ups", provider: &UPSProvider{}},
		{name: "poczta_polska", provider: &PocztaPolskaProvider{}},
		{name: "orlen_paczka", provider: &OrlenPaczkaProvider{}},
		{name: "fedex", provider: &FedExProvider{}},
	}

	req := integration.RateRequest{
		FromCountry: "PL",
		ToCountry:   "PL",
		Weight:      5,
		COD:         100,
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rates, err := tc.provider.GetRates(context.Background(), req)
			if !errors.Is(err, integration.ErrCarrierRatesNotImplemented) {
				t.Fatalf("GetRates() error = %v, want ErrCarrierRatesNotImplemented", err)
			}
			if rates != nil {
				t.Fatalf("GetRates() rates = %#v, want nil", rates)
			}
		})
	}
}
```

- [ ] **Step 6: Run tests and verify RED**

Run:

```bash
cd apps/api-server
go test ./internal/integration/carriers -run 'GetRates|CarrierGetRates' -count=1
```

Expected: FAIL because `integration.ErrCarrierRatesNotImplemented` does not exist yet and/or providers still return fake rates.

---

### Task 2: Add Sentinel Error To Carrier Contract

**Files:**
- Modify: `apps/api-server/internal/integration/carrier.go`

- [ ] **Step 1: Add `errors` import**

```go
import (
	"context"
	"errors"
	"time"
)
```

- [ ] **Step 2: Add sentinel error near `Rate` types**

```go
// ErrCarrierRatesNotImplemented indicates a carrier provider does not have a
// certified real-time or contract-backed rating implementation.
var ErrCarrierRatesNotImplemented = errors.New("carrier rates not implemented")
```

- [ ] **Step 3: Run targeted tests**

Run:

```bash
cd apps/api-server
go test ./internal/integration/carriers -run 'GetRates|CarrierGetRates' -count=1
```

Expected: FAIL because carrier methods still return fake rates, but the sentinel now compiles.

---

### Task 3: Replace Fake Carrier Rate Tables

**Files:**
- Modify: `apps/api-server/internal/integration/carriers/dhl.go`
- Modify: `apps/api-server/internal/integration/carriers/dpd.go`
- Modify: `apps/api-server/internal/integration/carriers/gls.go`
- Modify: `apps/api-server/internal/integration/carriers/ups.go`
- Modify: `apps/api-server/internal/integration/carriers/poczta_polska.go`
- Modify: `apps/api-server/internal/integration/carriers/orlen_paczka.go`
- Modify: `apps/api-server/internal/integration/carriers/fedex.go`

- [ ] **Step 1: Replace DPD implementation**

```go
func (p *DPDProvider) GetRates(_ context.Context, _ integration.RateRequest) ([]integration.Rate, error) {
	return nil, fmt.Errorf("dpd: %w", integration.ErrCarrierRatesNotImplemented)
}
```

- [ ] **Step 2: Replace GLS implementation**

```go
func (p *GLSProvider) GetRates(_ context.Context, _ integration.RateRequest) ([]integration.Rate, error) {
	return nil, fmt.Errorf("gls: %w", integration.ErrCarrierRatesNotImplemented)
}
```

- [ ] **Step 3: Replace UPS implementation**

```go
func (p *UPSProvider) GetRates(_ context.Context, _ integration.RateRequest) ([]integration.Rate, error) {
	return nil, fmt.Errorf("ups: %w", integration.ErrCarrierRatesNotImplemented)
}
```

- [ ] **Step 4: Replace Poczta Polska implementation**

```go
func (p *PocztaPolskaProvider) GetRates(_ context.Context, _ integration.RateRequest) ([]integration.Rate, error) {
	return nil, fmt.Errorf("poczta_polska: %w", integration.ErrCarrierRatesNotImplemented)
}
```

- [ ] **Step 5: Replace Orlen Paczka implementation**

```go
func (p *OrlenPaczkaProvider) GetRates(_ context.Context, _ integration.RateRequest) ([]integration.Rate, error) {
	return nil, fmt.Errorf("orlen_paczka: %w", integration.ErrCarrierRatesNotImplemented)
}
```

- [ ] **Step 6: Replace FedEx implementation**

```go
func (p *FedExProvider) GetRates(_ context.Context, _ integration.RateRequest) ([]integration.Rate, error) {
	return nil, fmt.Errorf("fedex: %w", integration.ErrCarrierRatesNotImplemented)
}
```

- [ ] **Step 7: Replace DHL implementation**

```go
func (p *DHLProvider) GetRates(_ context.Context, _ integration.RateRequest) ([]integration.Rate, error) {
	return nil, fmt.Errorf("dhl: %w", integration.ErrCarrierRatesNotImplemented)
}
```

- [ ] **Step 8: Run targeted tests and verify GREEN**

Run:

```bash
cd apps/api-server
go test ./internal/integration/carriers -run 'GetRates|CarrierGetRates' -count=1
```

Expected: PASS.

---

### Task 4: Make RateService Skip Unsupported Rate Providers Cleanly

**Files:**
- Modify: `apps/api-server/internal/service/rate_service.go`

- [ ] **Step 1: Add `errors` import**

```go
import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"sync"
)
```

- [ ] **Step 2: Update provider error handling**

Replace the carrier `GetRates` error branch with:

```go
rates, err := carrier.GetRates(ctx, req)
if err != nil {
	if errors.Is(err, integration.ErrCarrierRatesNotImplemented) {
		slog.Debug("rate_service: carrier rates not implemented",
			"provider", provider)
		return
	}

	slog.Warn("rate_service: carrier GetRates failed",
		"provider", provider, "error", err)
	return
}
```

- [ ] **Step 3: Fix stale map comment**

Change the map comment to:

```go
// Carrier provider names that can be instantiated for rate shopping.
// Providers without real rating support must return ErrCarrierRatesNotImplemented.
```

- [ ] **Step 4: Run service/package tests**

Run:

```bash
cd apps/api-server
go test ./internal/service ./internal/integration/carriers -run 'Rate|GetRates|CarrierGetRates' -count=1
```

Expected: PASS.

---

### Task 5: Update Documentation

**Files:**
- Modify: `docs/system-documentation.md`
- Local context update: `.claude/context/API_CONTRACTS.md` (ignored by git)
- Local context update: `.claude/context/SECURITY_POSTURE.md` (ignored by git)
- Local context update: `.claude/context/PROJECT_STATE.md` (ignored by git)

- [ ] **Step 1: Update provider interface docs**

In `docs/system-documentation.md`, change the `GetRates` interface line to:

```text
|    GetRates(ctx, req) -> rates or ErrCarrierRatesNotImplemented |
```

- [ ] **Step 2: Update carrier integration matrix**

In `docs/system-documentation.md`, add a note under the carrier provider table:

```markdown
Rate shopping returns only contract-backed/certified carrier rates. DPD, GLS, UPS, Poczta Polska, Orlen Paczka, FedEx, and DHL currently return `ErrCarrierRatesNotImplemented`, so `/v1/shipping/rates` skips them instead of exposing placeholder prices.
```

- [ ] **Step 3: Update API contract**

In `.claude/context/API_CONTRACTS.md`, add or update the `/v1/shipping/rates` note:

```markdown
- `POST /v1/shipping/rates` aggregates active carrier integrations but skips providers returning `integration.ErrCarrierRatesNotImplemented`; unsupported carrier rating must not be represented as zero-price or estimated production rates.
```

- [ ] **Step 4: Update security posture**

In `.claude/context/SECURITY_POSTURE.md`, add a current note:

```markdown
- OPE-298: hardcoded placeholder `GetRates` pricing was removed for DPD, GLS, UPS, Poczta Polska, Orlen Paczka, FedEx, and DHL. These providers now return `integration.ErrCarrierRatesNotImplemented`, and `RateService` skips them so tenants do not choose carriers based on fabricated prices. Follow-up: decide whether InPost estimated pricing should remain visible as controlled behavior or move behind an explicit product flag.
```

- [ ] **Step 5: Update project state**

In `.claude/context/PROJECT_STATE.md`, update the DPD and GLS production-ready notes so they say the historical placeholder-rate finding was remediated in `OPE-298`.

---

### Task 6: Verification And PR Prep

**Files:**
- No new production files.

- [ ] **Step 1: Format Go files**

Run:

```bash
gofmt -w apps/api-server/internal/integration/carrier.go apps/api-server/internal/integration/carriers/dhl.go apps/api-server/internal/integration/carriers/dpd.go apps/api-server/internal/integration/carriers/gls.go apps/api-server/internal/integration/carriers/ups.go apps/api-server/internal/integration/carriers/poczta_polska.go apps/api-server/internal/integration/carriers/orlen_paczka.go apps/api-server/internal/integration/carriers/fedex.go apps/api-server/internal/integration/carriers/dhl_test.go apps/api-server/internal/integration/carriers/dpd_test.go apps/api-server/internal/integration/carriers/gls_test.go apps/api-server/internal/integration/carriers/rates_not_implemented_test.go apps/api-server/internal/service/rate_service.go
```

- [ ] **Step 2: Run targeted tests**

Run:

```bash
cd apps/api-server
go test ./internal/integration/carriers ./internal/service -run 'Rate|GetRates|CarrierGetRates' -count=1
```

Expected: PASS.

- [ ] **Step 3: Run broader API tests**

Run:

```bash
cd apps/api-server
go test ./internal/integration/carriers ./internal/service ./internal/handler -count=1
```

Expected: PASS.

- [ ] **Step 4: Run repository validation**

Run:

```bash
git diff --check
./scripts/local-ci.sh
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api-server/internal/integration/carrier.go apps/api-server/internal/integration/carriers/dhl.go apps/api-server/internal/integration/carriers/dpd.go apps/api-server/internal/integration/carriers/gls.go apps/api-server/internal/integration/carriers/ups.go apps/api-server/internal/integration/carriers/poczta_polska.go apps/api-server/internal/integration/carriers/orlen_paczka.go apps/api-server/internal/integration/carriers/fedex.go apps/api-server/internal/integration/carriers/dhl_test.go apps/api-server/internal/integration/carriers/dpd_test.go apps/api-server/internal/integration/carriers/gls_test.go apps/api-server/internal/integration/carriers/rates_not_implemented_test.go apps/api-server/internal/service/rate_service.go docs/system-documentation.md docs/superpowers/plans/2026-05-17-ope-298-hide-stub-carrier-rates.md
git commit -m "OPE-298: hide unsupported carrier rate estimates"
```

---

## Risk And Rollback

- **User-facing behavior:** tenants may see fewer or no rate-shopping options when only unsupported carriers are active. This is intentional and safer than showing fabricated prices.
- **Operational risk:** shipment creation, labels, tracking, cancel, pickup points, and carrier setup are not removed.
- **Rollback:** revert the PR to restore the previous fake rates. This is not recommended except as an emergency UI compatibility rollback because it reintroduces inaccurate prices.
- **Follow-up:** create or reuse a Linear task for InPost estimated pricing policy and optional UI messaging/feature flag.

## Self-Review

- Spec coverage: the six OPE-298 providers are covered; DHL is included as same-risk behavior discovered during implementation planning; InPost is explicitly out of scope and tracked as follow-up.
- Placeholder scan: no TBD/TODO implementation steps are left in this plan.
- Type consistency: sentinel name is consistently `integration.ErrCarrierRatesNotImplemented`; service logic uses `errors.Is`.
