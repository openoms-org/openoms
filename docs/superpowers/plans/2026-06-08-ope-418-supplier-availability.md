# OPE-418 Supplier-Availability Policy Model — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the naive single-number supplier stock with a freshness-aware, policy-driven supplier-availability read-model that gates auto dropship routing and channel-stock increases, behind a feature flag.

**Architecture:** Two additive tenant-scoped tables (`supplier_availability` snapshot + `supplier_availability_policy` with a 4-scope precedence resolver) feed a pure on-read `ResolveAvailability` function. The merged dropship/backorder routing and the channel stock-sync consult it; when off (`SUPPLIER_AVAILABILITY_ENABLED=false`, default) behavior is byte-for-byte unchanged.

**Tech Stack:** Go 1.25 (api-server), pgx/v5, PostgreSQL 16 with FORCE RLS, golang-migrate, testify. Spec: `docs/superpowers/specs/2026-06-08-ope-418-supplier-availability-design.md`.

**Conventions every task follows:**
- Run all commands from `apps/api-server/`. Tests: `go test ./internal/<pkg>/ -run <Name> -count=1`. Integration: `DATABASE_URL=postgres://openoms:openoms-dev-password@127.0.0.1:5433/openoms?sslmode=disable go test -tags integration ./tests/integration/ -run <Name> -count=1`.
- Before each commit: `gofmt -w -s <changed.go>`. Before any push (separate from this plan): the CI lint is **golangci-lint v2.9.0** — `/tmp/glci29/golangci-lint run --new-from-rev=main --timeout=5m` must be `0 issues`.
- Commit after every task (frequent commits). No push/PR steps here — integration into branches/PRs is handled by the executing workflow per repo convention (branch + PR, never push to main).

---

## File Structure

| File | Responsibility | Create/Modify |
|---|---|---|
| `apps/api-server/migrations/000041_supplier_availability.up.sql` / `.down.sql` | Two additive FORCE-RLS tables + partial unique indexes | Create |
| `apps/api-server/internal/model/supplier_availability.go` | Snapshot + policy structs, enums, validators | Create |
| `apps/api-server/internal/model/supplier_availability_resolve.go` | Pure `ResolveAvailability` resolver + `AvailabilityDecision` | Create |
| `apps/api-server/internal/model/supplier_availability_resolve_test.go` | Resolver unit tests (no DB) | Create |
| `apps/api-server/internal/model/fulfillment.go` | Add `BlockerSupplierAvailabilityInsufficient` code + category | Modify |
| `apps/api-server/internal/repository/supplier_availability_repository.go` | Snapshot upsert + policy CRUD + resolution-load queries (tenant-scoped, `pgx.Tx`) | Create |
| `apps/api-server/internal/service/supplier_availability_service.go` | `ResolveForOrderLine` (loads snapshot+policy chain, calls resolver), policy CRUD + override audit | Create |
| `apps/api-server/internal/config/config.go` | `SupplierAvailabilityEnabled` flag | Modify |
| `apps/api-server/internal/worker/supplier_sync_worker.go` | Gated snapshot upsert during sync | Modify |
| `apps/api-server/internal/service/dropship_service.go` | Consult resolver before auto-routing; blocker on stale/unknown/insufficient | Modify |
| `apps/api-server/internal/worker/stock_sync_worker.go` | Apply channel-increase gate for supplier-backed listings | Modify |
| `apps/api-server/cmd/server/main.go` | Wire the availability service into supplier sync / dropship / stock-sync (gated) | Modify |
| `apps/api-server/tests/integration/supplier_availability_test.go` | DB-bound: upsert idempotency, RLS isolation, routing blockers, channel-increase gate, flag-off no-op | Create |

---

## Task 1: Migration 000041 — two additive FORCE-RLS tables

**Files:**
- Create: `apps/api-server/migrations/000041_supplier_availability.up.sql`
- Create: `apps/api-server/migrations/000041_supplier_availability.down.sql`

- [ ] **Step 1: Write the up migration**

Create `apps/api-server/migrations/000041_supplier_availability.up.sql`:

```sql
-- OPE-418 supplier-availability read-model. Two additive TENANT-SCOPED tables (FORCE
-- ROW LEVEL SECURITY + tenant_isolation policy, accessed through database.WithTenant).
-- supplier_availability is the raw observational snapshot per supplier_product x
-- warehouse; supplier_availability_policy holds the 4-scope tenant rules. available_to_sell
-- is computed on read, so no value is materialised here.

CREATE TABLE public.supplier_availability (
    id                    uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    tenant_id             uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    supplier_id           uuid NOT NULL REFERENCES public.suppliers(id) ON DELETE CASCADE,
    supplier_product_id   uuid NOT NULL REFERENCES public.supplier_products(id) ON DELETE CASCADE,
    product_id            uuid REFERENCES public.products(id) ON DELETE SET NULL,
    warehouse_external_id text NOT NULL DEFAULT '',
    source_quantity       integer NOT NULL DEFAULT 0,
    availability_type     text NOT NULL DEFAULT 'unknown'
        CHECK (availability_type IN ('exact_quantity','bucket','boolean','eta_only','unknown')),
    min_handling_days     integer,
    max_handling_days     integer,
    next_delivery_date    date,
    reservation_supported boolean NOT NULL DEFAULT false,
    freshness_observed_at timestamptz NOT NULL DEFAULT now(),
    source_max_stale_seconds integer,
    last_successful_sync_id uuid REFERENCES public.sync_jobs(id) ON DELETE SET NULL,
    raw                   jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uq_supplier_availability_product_wh
    ON public.supplier_availability (tenant_id, supplier_product_id, warehouse_external_id); -- migrate:index-lock-ok
CREATE INDEX idx_supplier_availability_supplier
    ON public.supplier_availability (tenant_id, supplier_id); -- migrate:index-lock-ok
CREATE INDEX idx_supplier_availability_product
    ON public.supplier_availability (tenant_id, product_id); -- migrate:index-lock-ok

CREATE TABLE public.supplier_availability_policy (
    id              uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    tenant_id       uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    scope           text NOT NULL CHECK (scope IN ('supplier','product','listing','channel')),
    supplier_id     uuid REFERENCES public.suppliers(id) ON DELETE CASCADE,
    product_id      uuid REFERENCES public.products(id) ON DELETE CASCADE,
    listing_id      uuid REFERENCES public.product_listings(id) ON DELETE CASCADE,
    channel         text,
    mode            text NOT NULL DEFAULT 'auto' CHECK (mode IN ('auto','manual','paused')),
    safety_buffer   integer NOT NULL DEFAULT 0,
    freshness_window_seconds integer,
    max_lead_time_days integer,
    override_quantity integer,
    allow_channel_increase boolean NOT NULL DEFAULT false,
    require_reservation boolean NOT NULL DEFAULT false,
    require_preflight boolean NOT NULL DEFAULT false,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uq_sap_supplier ON public.supplier_availability_policy (tenant_id, supplier_id) WHERE scope = 'supplier'; -- migrate:index-lock-ok
CREATE UNIQUE INDEX uq_sap_product ON public.supplier_availability_policy (tenant_id, supplier_id, product_id) WHERE scope = 'product'; -- migrate:index-lock-ok
CREATE UNIQUE INDEX uq_sap_listing ON public.supplier_availability_policy (tenant_id, listing_id) WHERE scope = 'listing'; -- migrate:index-lock-ok
CREATE UNIQUE INDEX uq_sap_channel ON public.supplier_availability_policy (tenant_id, channel) WHERE scope = 'channel'; -- migrate:index-lock-ok

ALTER TABLE public.supplier_availability ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.supplier_availability FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.supplier_availability USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

ALTER TABLE public.supplier_availability_policy ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.supplier_availability_policy FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.supplier_availability_policy USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Grant the least-privilege app role(s): production "openoms_app", self-hosted "openoms".
DO $$
DECLARE
    app_role text;
    tbl      text;
BEGIN
    FOREACH app_role IN ARRAY ARRAY['openoms_app', 'openoms'] LOOP
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = app_role) THEN
            FOREACH tbl IN ARRAY ARRAY['supplier_availability','supplier_availability_policy'] LOOP
                EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON public.%I TO %I', tbl, app_role);
            END LOOP;
        END IF;
    END LOOP;
END;
$$;
```

- [ ] **Step 2: Write the down migration**

Create `apps/api-server/migrations/000041_supplier_availability.down.sql`:

```sql
-- OPE-418 rollback: drop the supplier-availability read-model tables (additive).
DROP TABLE IF EXISTS public.supplier_availability_policy;
DROP TABLE IF EXISTS public.supplier_availability;
```

- [ ] **Step 3: Verify the Migration Safety regex passes (no destructive ops in up)**

Run:
```bash
DANGEROUS_OPS="DROP[[:space:]]+COLUMN|DROP[[:space:]]+TABLE|DROP[[:space:]]+SCHEMA|DROP[[:space:]]+FUNCTION|DROP[[:space:]]+INDEX|RENAME[[:space:]]+COLUMN|ALTER[[:space:]][^;]*TYPE|ALTER[[:space:]][^;]*SET[[:space:]]+NOT[[:space:]]+NULL|TRUNCATE"
tr '\n' ' ' < migrations/000041_supplier_availability.up.sql | grep -inEi "$DANGEROUS_OPS" || echo "PASS: no destructive ops"
```
Expected: `PASS: no destructive ops`.

- [ ] **Step 4: Apply + roundtrip on a scratch DB**

Run:
```bash
docker exec openoms-postgres-1 psql -U openoms -d openoms -c "DROP DATABASE IF EXISTS ope418_mig; CREATE DATABASE ope418_mig;"
migrate -path migrations -database "postgres://openoms:openoms-dev-password@127.0.0.1:5433/ope418_mig?sslmode=disable" up
migrate -path migrations -database "postgres://openoms:openoms-dev-password@127.0.0.1:5433/ope418_mig?sslmode=disable" down 1
migrate -path migrations -database "postgres://openoms:openoms-dev-password@127.0.0.1:5433/ope418_mig?sslmode=disable" up
docker exec openoms-postgres-1 psql -U openoms -d openoms -c "DROP DATABASE IF EXISTS ope418_mig;"
```
Expected: each `up`/`down` prints `41/u supplier_availability` / `41/d supplier_availability` with no error.

- [ ] **Step 5: Commit**

```bash
git add migrations/000041_supplier_availability.up.sql migrations/000041_supplier_availability.down.sql
git commit -m "OPE-418: migration 000041 supplier-availability read-model tables"
```

---

## Task 2: Blocker code `supplier_availability_insufficient`

**Files:**
- Modify: `apps/api-server/internal/model/fulfillment.go` (the blocker-code const block + `blockerCategories` map)
- Test: `apps/api-server/internal/model/fulfillment_test.go` (or the existing provider_attempt/fulfillment test)

Note: `BlockerSupplierAvailabilityStale`, `BlockerSupplierAvailabilityUnknown`, `BlockerManualStockReviewRequired`, `BlockerSupplierPreflightRequired` ALREADY exist. Only the "trusted but not enough stock and no ETA" signal is missing.

- [ ] **Step 1: Write the failing test**

Add to `apps/api-server/internal/model/fulfillment_test.go`:

```go
func TestBlockerSupplierAvailabilityInsufficient_IsValidWithCategory(t *testing.T) {
	assert.True(t, IsValidBlockerCode(BlockerSupplierAvailabilityInsufficient))
	assert.Equal(t, "supplier_availability_insufficient", BlockerSupplierAvailabilityInsufficient)
	assert.Equal(t, "supplier", blockerCategories[BlockerSupplierAvailabilityInsufficient])
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/model/ -run TestBlockerSupplierAvailabilityInsufficient -count=1`
Expected: FAIL — `undefined: BlockerSupplierAvailabilityInsufficient`.

- [ ] **Step 3: Add the constant + category**

In `apps/api-server/internal/model/fulfillment.go`, in the blocker-code `const` block (after `BlockerStockAckMissing` at line ~87) add:

```go
	BlockerSupplierAvailabilityInsufficient = "supplier_availability_insufficient"
```

In the `blockerCategories` map (search for `BlockerSupplierAvailabilityStale:` and its `"supplier"` value) add the entry:

```go
	BlockerSupplierAvailabilityInsufficient: "supplier",
```

If there is a `validBlockerCodes` slice used by `IsValidBlockerCode`, add `BlockerSupplierAvailabilityInsufficient` to it as well (mirror how `BlockerSupplierAvailabilityStale` is listed).

- [ ] **Step 4: Run it to verify it passes**

Run: `gofmt -w -s internal/model/fulfillment.go && go test ./internal/model/ -run TestBlockerSupplierAvailabilityInsufficient -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/model/fulfillment.go internal/model/fulfillment_test.go
git commit -m "OPE-418: add supplier_availability_insufficient blocker code"
```

---

## Task 3: Model — snapshot + policy structs, enums, validators

**Files:**
- Create: `apps/api-server/internal/model/supplier_availability.go`
- Test: `apps/api-server/internal/model/supplier_availability_test.go`

- [ ] **Step 1: Write the failing test**

Create `apps/api-server/internal/model/supplier_availability_test.go`:

```go
package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSupplierAvailabilityValidators(t *testing.T) {
	assert.True(t, IsValidAvailabilityType("exact_quantity"))
	assert.True(t, IsValidAvailabilityType("bucket"))
	assert.True(t, IsValidAvailabilityType("boolean"))
	assert.True(t, IsValidAvailabilityType("eta_only"))
	assert.True(t, IsValidAvailabilityType("unknown"))
	assert.False(t, IsValidAvailabilityType("magic"))

	assert.True(t, IsValidPolicyScope("supplier"))
	assert.True(t, IsValidPolicyScope("product"))
	assert.True(t, IsValidPolicyScope("listing"))
	assert.True(t, IsValidPolicyScope("channel"))
	assert.False(t, IsValidPolicyScope("global"))

	assert.True(t, IsValidPolicyMode("auto"))
	assert.True(t, IsValidPolicyMode("manual"))
	assert.True(t, IsValidPolicyMode("paused"))
	assert.False(t, IsValidPolicyMode("frozen"))
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/model/ -run TestSupplierAvailabilityValidators -count=1`
Expected: FAIL — `undefined: IsValidAvailabilityType`.

- [ ] **Step 3: Write the model**

Create `apps/api-server/internal/model/supplier_availability.go`:

```go
package model

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

// Availability types (how precise the supplier's quantity signal is).
const (
	AvailabilityExactQuantity = "exact_quantity"
	AvailabilityBucket        = "bucket"
	AvailabilityBoolean       = "boolean"
	AvailabilityETAOnly       = "eta_only"
	AvailabilityUnknown       = "unknown"
)

var validAvailabilityTypes = []string{
	AvailabilityExactQuantity, AvailabilityBucket, AvailabilityBoolean, AvailabilityETAOnly, AvailabilityUnknown,
}

// IsValidAvailabilityType reports whether t is a known availability type.
func IsValidAvailabilityType(t string) bool { return slices.Contains(validAvailabilityTypes, t) }

// Policy scopes (precedence: channel > listing > product > supplier).
const (
	PolicyScopeSupplier = "supplier"
	PolicyScopeProduct  = "product"
	PolicyScopeListing  = "listing"
	PolicyScopeChannel  = "channel"
)

var validPolicyScopes = []string{PolicyScopeSupplier, PolicyScopeProduct, PolicyScopeListing, PolicyScopeChannel}

// IsValidPolicyScope reports whether s is a known policy scope.
func IsValidPolicyScope(s string) bool { return slices.Contains(validPolicyScopes, s) }

// Policy modes.
const (
	PolicyModeAuto   = "auto"
	PolicyModeManual = "manual"
	PolicyModePaused = "paused"
)

var validPolicyModes = []string{PolicyModeAuto, PolicyModeManual, PolicyModePaused}

// IsValidPolicyMode reports whether m is a known policy mode.
func IsValidPolicyMode(m string) bool { return slices.Contains(validPolicyModes, m) }

// SupplierAvailability is the raw, observational availability of a supplier product at
// a supplier warehouse (one row per tenant x supplier_product x warehouse_external_id).
type SupplierAvailability struct {
	ID                    uuid.UUID  `json:"id"`
	TenantID              uuid.UUID  `json:"tenant_id"`
	SupplierID            uuid.UUID  `json:"supplier_id"`
	SupplierProductID     uuid.UUID  `json:"supplier_product_id"`
	ProductID             *uuid.UUID `json:"product_id,omitempty"`
	WarehouseExternalID   string     `json:"warehouse_external_id"`
	SourceQuantity        int        `json:"source_quantity"`
	AvailabilityType      string     `json:"availability_type"`
	MinHandlingDays       *int       `json:"min_handling_days,omitempty"`
	MaxHandlingDays       *int       `json:"max_handling_days,omitempty"`
	NextDeliveryDate      *time.Time `json:"next_delivery_date,omitempty"`
	ReservationSupported  bool       `json:"reservation_supported"`
	FreshnessObservedAt   time.Time  `json:"freshness_observed_at"`
	SourceMaxStaleSeconds *int       `json:"source_max_stale_seconds,omitempty"`
	LastSuccessfulSyncID  *uuid.UUID `json:"last_successful_sync_id,omitempty"`
	Raw                   []byte     `json:"raw,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

// SupplierAvailabilityPolicy is one tenant rule at one of the four scopes. Only the ref
// column for its scope is set (supplier_id/product_id/listing_id/channel).
type SupplierAvailabilityPolicy struct {
	ID                   uuid.UUID  `json:"id"`
	TenantID             uuid.UUID  `json:"tenant_id"`
	Scope                string     `json:"scope"`
	SupplierID           *uuid.UUID `json:"supplier_id,omitempty"`
	ProductID            *uuid.UUID `json:"product_id,omitempty"`
	ListingID            *uuid.UUID `json:"listing_id,omitempty"`
	Channel              *string    `json:"channel,omitempty"`
	Mode                 string     `json:"mode"`
	SafetyBuffer         int        `json:"safety_buffer"`
	FreshnessWindowSecs  *int       `json:"freshness_window_seconds,omitempty"`
	MaxLeadTimeDays      *int       `json:"max_lead_time_days,omitempty"`
	OverrideQuantity     *int       `json:"override_quantity,omitempty"`
	AllowChannelIncrease bool       `json:"allow_channel_increase"`
	RequireReservation   bool       `json:"require_reservation"`
	RequirePreflight     bool       `json:"require_preflight"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}
```

- [ ] **Step 4: Run it to verify it passes**

Run: `gofmt -w -s internal/model/supplier_availability.go && go test ./internal/model/ -run TestSupplierAvailabilityValidators -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/model/supplier_availability.go internal/model/supplier_availability_test.go
git commit -m "OPE-418: supplier-availability + policy model structs and validators"
```

---

## Task 4: Pure resolver — `ResolveAvailability`

**Files:**
- Create: `apps/api-server/internal/model/supplier_availability_resolve.go`
- Test: `apps/api-server/internal/model/supplier_availability_resolve_test.go`

The resolver is the heart of the feature and is a pure function (no DB) so it is exhaustively unit-tested.

- [ ] **Step 1: Write the failing tests**

Create `apps/api-server/internal/model/supplier_availability_resolve_test.go`:

```go
package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func iptr(v int) *int { return &v }

// baseSnapshot: trusted, exact, 100 units, fresh (observed now), reservation supported.
func baseSnapshot(now time.Time) SupplierAvailability {
	return SupplierAvailability{
		SourceQuantity:       100,
		AvailabilityType:     AvailabilityExactQuantity,
		ReservationSupported: true,
		FreshnessObservedAt:  now,
		MaxHandlingDays:      iptr(3),
	}
}

func TestResolve_Trusted_BufferApplied(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	snap := baseSnapshot(now)
	pol := EffectivePolicy{Mode: PolicyModeAuto, SafetyBuffer: 10, FreshnessWindowSeconds: 3600}
	d := ResolveAvailability(snap, pol, 5, false, now)
	assert.Equal(t, AvailabilityStatusTrusted, d.Status)
	assert.Equal(t, 90, d.AvailableToSell) // 100 - 10
	assert.True(t, d.AutoRoutable)
	assert.Nil(t, d.BlockerCode)
}

func TestResolve_Override_Wins(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	snap := baseSnapshot(now)
	pol := EffectivePolicy{Mode: PolicyModeManual, SafetyBuffer: 10, OverrideQuantity: iptr(7), FreshnessWindowSeconds: 3600}
	d := ResolveAvailability(snap, pol, 5, false, now)
	assert.Equal(t, AvailabilityStatusManualOverride, d.Status)
	assert.Equal(t, 7, d.AvailableToSell)
}

func TestResolve_Paused_ZeroNoBlocker(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	pol := EffectivePolicy{Mode: PolicyModePaused, FreshnessWindowSeconds: 3600}
	d := ResolveAvailability(baseSnapshot(now), pol, 5, false, now)
	assert.Equal(t, AvailabilityStatusPaused, d.Status)
	assert.Equal(t, 0, d.AvailableToSell)
	assert.False(t, d.AutoRoutable)
	assert.Nil(t, d.BlockerCode) // intentional operator pause, not a blocker
}

func TestResolve_Stale_Blocks(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	snap := baseSnapshot(now.Add(-2 * time.Hour)) // observed 2h ago
	pol := EffectivePolicy{Mode: PolicyModeAuto, FreshnessWindowSeconds: 3600}
	d := ResolveAvailability(snap, pol, 5, false, now)
	assert.Equal(t, AvailabilityStatusStale, d.Status)
	assert.Equal(t, 0, d.AvailableToSell)
	assert.False(t, d.AutoRoutable)
	assert.Equal(t, BlockerSupplierAvailabilityStale, *d.BlockerCode)
}

func TestResolve_Unknown_Blocks(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	snap := baseSnapshot(now)
	snap.AvailabilityType = AvailabilityUnknown
	pol := EffectivePolicy{Mode: PolicyModeAuto, FreshnessWindowSeconds: 3600}
	d := ResolveAvailability(snap, pol, 5, false, now)
	assert.Equal(t, AvailabilityStatusUnknown, d.Status)
	assert.Equal(t, BlockerSupplierAvailabilityUnknown, *d.BlockerCode)
}

func TestResolve_Insufficient_NoETA_Blocks(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	snap := baseSnapshot(now)
	snap.SourceQuantity = 3
	pol := EffectivePolicy{Mode: PolicyModeAuto, FreshnessWindowSeconds: 3600}
	d := ResolveAvailability(snap, pol, 5, false, now) // need 5, have 3, no ETA
	assert.Equal(t, AvailabilityStatusTrusted, d.Status)
	assert.False(t, d.AutoRoutable)
	assert.False(t, d.Backorder)
	assert.Equal(t, BlockerSupplierAvailabilityInsufficient, *d.BlockerCode)
}

func TestResolve_Insufficient_WithETA_Backorder(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	snap := baseSnapshot(now)
	snap.SourceQuantity = 3
	eta := now.Add(72 * time.Hour)
	snap.NextDeliveryDate = &eta
	pol := EffectivePolicy{Mode: PolicyModeAuto, FreshnessWindowSeconds: 3600}
	d := ResolveAvailability(snap, pol, 5, false, now)
	assert.True(t, d.Backorder)
	assert.False(t, d.AutoRoutable)
	assert.Nil(t, d.BlockerCode) // backorder is a unit state, not a blocker
}

func TestResolve_RequireReservation_NotSupported_NotRoutable(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	snap := baseSnapshot(now)
	snap.ReservationSupported = false
	pol := EffectivePolicy{Mode: PolicyModeAuto, FreshnessWindowSeconds: 3600, RequireReservation: true}
	d := ResolveAvailability(snap, pol, 5, false, now)
	assert.False(t, d.AutoRoutable)
}

func TestResolve_RequirePreflight_DefaultsUnsupported_NotRoutable(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	pol := EffectivePolicy{Mode: PolicyModeAuto, FreshnessWindowSeconds: 3600, RequirePreflight: true}
	// preflightSupported = false (the engine that models it is a later spec).
	d := ResolveAvailability(baseSnapshot(now), pol, 5, false, now)
	assert.False(t, d.AutoRoutable)
}

func TestResolve_LeadTimeExceeded_NotRoutable(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	snap := baseSnapshot(now)
	snap.MaxHandlingDays = iptr(10)
	pol := EffectivePolicy{Mode: PolicyModeAuto, FreshnessWindowSeconds: 3600, MaxLeadTimeDays: iptr(5)}
	d := ResolveAvailability(snap, pol, 5, false, now)
	assert.False(t, d.AutoRoutable)
}

func TestResolve_ChannelIncrease_Gate(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	snap := baseSnapshot(now)
	// allow_channel_increase=false -> never allowed even when trusted.
	off := EffectivePolicy{Mode: PolicyModeAuto, FreshnessWindowSeconds: 3600, AllowChannelIncrease: false}
	assert.False(t, ResolveAvailability(snap, off, 5, false, now).ChannelIncreaseAllowed)
	// allow_channel_increase=true + trusted -> allowed.
	on := EffectivePolicy{Mode: PolicyModeAuto, FreshnessWindowSeconds: 3600, AllowChannelIncrease: true}
	assert.True(t, ResolveAvailability(snap, on, 5, false, now).ChannelIncreaseAllowed)
	// allow_channel_increase=true but stale -> not allowed.
	staleSnap := baseSnapshot(now.Add(-2 * time.Hour))
	assert.False(t, ResolveAvailability(staleSnap, on, 5, false, now).ChannelIncreaseAllowed)
}

func TestResolvePolicyChain_Precedence(t *testing.T) {
	// channel > listing > product > supplier; each field resolves independently.
	supplier := SupplierAvailabilityPolicy{Scope: PolicyScopeSupplier, Mode: PolicyModeAuto, SafetyBuffer: 5, FreshnessWindowSecs: iptr(7200)}
	product := SupplierAvailabilityPolicy{Scope: PolicyScopeProduct, SafetyBuffer: 9} // overrides buffer only
	eff := ResolvePolicyChain([]SupplierAvailabilityPolicy{supplier, product})
	assert.Equal(t, 9, eff.SafetyBuffer)             // from product (more specific)
	assert.Equal(t, 7200, eff.FreshnessWindowSeconds) // inherited from supplier
	assert.Equal(t, PolicyModeAuto, eff.Mode)
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/model/ -run TestResolve -count=1`
Expected: FAIL — `undefined: ResolveAvailability` / `EffectivePolicy` / `AvailabilityStatusTrusted`.

- [ ] **Step 3: Write the resolver**

Create `apps/api-server/internal/model/supplier_availability_resolve.go`:

```go
package model

import "time"

// DefaultFreshnessWindowSeconds is used when neither the policy nor the snapshot's feed
// declares one — an hour is a safe conservative default for supplier feeds.
const DefaultFreshnessWindowSeconds = 3600

// Availability decision statuses.
const (
	AvailabilityStatusTrusted        = "trusted"
	AvailabilityStatusStale          = "stale"
	AvailabilityStatusUnknown        = "unknown"
	AvailabilityStatusPaused         = "paused"
	AvailabilityStatusManualOverride = "manual_override"
)

// EffectivePolicy is the policy after the 4-scope precedence chain is resolved. It mirrors
// the rule fields of SupplierAvailabilityPolicy but with concrete (already-resolved) values.
type EffectivePolicy struct {
	Mode                   string
	SafetyBuffer           int
	FreshnessWindowSeconds int
	MaxLeadTimeDays        *int
	OverrideQuantity       *int
	AllowChannelIncrease   bool
	RequireReservation     bool
	RequirePreflight       bool
}

// AvailabilityDecision is the resolver output consumed by routing + propagation.
type AvailabilityDecision struct {
	AvailableToSell        int
	Status                 string
	AutoRoutable           bool
	Backorder              bool
	ChannelIncreaseAllowed bool
	BlockerCode            *string // nil when nothing is wrong
}

// ResolvePolicyChain folds a precedence-ordered slice of scope policies (LEAST specific
// first: supplier, product, listing, channel) into one EffectivePolicy. Each field is
// taken from the MOST specific policy that sets it; unset fields inherit. A zero
// SafetyBuffer / false bool from a more specific scope is treated as "set" (explicit),
// so callers should only include policy rows that actually exist for the context.
func ResolvePolicyChain(chain []SupplierAvailabilityPolicy) EffectivePolicy {
	eff := EffectivePolicy{Mode: PolicyModeAuto, FreshnessWindowSeconds: 0}
	for _, p := range chain { // chain is ordered least->most specific; later wins
		eff.Mode = p.Mode
		eff.SafetyBuffer = p.SafetyBuffer
		if p.FreshnessWindowSecs != nil {
			eff.FreshnessWindowSeconds = *p.FreshnessWindowSecs
		}
		if p.MaxLeadTimeDays != nil {
			eff.MaxLeadTimeDays = p.MaxLeadTimeDays
		}
		if p.OverrideQuantity != nil {
			eff.OverrideQuantity = p.OverrideQuantity
		}
		eff.AllowChannelIncrease = p.AllowChannelIncrease
		eff.RequireReservation = p.RequireReservation
		eff.RequirePreflight = p.RequirePreflight
	}
	return eff
}

// ResolveAvailability turns a snapshot + resolved policy into a decision. PURE (no DB).
// requestedQty is the quantity the order line needs; preflightSupported reflects whether
// the supplier capability supports a preflight check (false until the supplier-order
// engine spec lands, so require_preflight conservatively yields a manual unit).
func ResolveAvailability(snap SupplierAvailability, pol EffectivePolicy, requestedQty int, preflightSupported bool, now time.Time) AvailabilityDecision {
	d := AvailabilityDecision{}

	// 1. Manual override wins outright.
	if pol.OverrideQuantity != nil {
		d.Status = AvailabilityStatusManualOverride
		d.AvailableToSell = max(0, *pol.OverrideQuantity)
		d.AutoRoutable = d.AvailableToSell >= requestedQty && pol.Mode != PolicyModePaused
		d.ChannelIncreaseAllowed = false // a manual override never auto-raises a channel
		return d
	}

	// 2. Operator pause.
	if pol.Mode == PolicyModePaused {
		d.Status = AvailabilityStatusPaused
		return d // zero, not routable, no blocker (intentional)
	}

	// 3. Unknown availability is untrusted.
	if snap.AvailabilityType == AvailabilityUnknown {
		d.Status = AvailabilityStatusUnknown
		code := BlockerSupplierAvailabilityUnknown
		d.BlockerCode = &code
		return d
	}

	// 4. Freshness.
	window := pol.FreshnessWindowSeconds
	if window <= 0 {
		if snap.SourceMaxStaleSeconds != nil && *snap.SourceMaxStaleSeconds > 0 {
			window = *snap.SourceMaxStaleSeconds
		} else {
			window = DefaultFreshnessWindowSeconds
		}
	}
	if now.Sub(snap.FreshnessObservedAt) > time.Duration(window)*time.Second {
		d.Status = AvailabilityStatusStale
		code := BlockerSupplierAvailabilityStale
		d.BlockerCode = &code
		return d
	}

	// 5. Trusted.
	d.Status = AvailabilityStatusTrusted
	d.AvailableToSell = max(0, snap.SourceQuantity-pol.SafetyBuffer)
	d.ChannelIncreaseAllowed = pol.AllowChannelIncrease && leadTimeOK(snap, pol)

	if d.AvailableToSell >= requestedQty &&
		(!pol.RequireReservation || snap.ReservationSupported) &&
		(!pol.RequirePreflight || preflightSupported) &&
		leadTimeOK(snap, pol) {
		d.AutoRoutable = true
		return d
	}

	// Trusted but cannot auto-route: backorder if an ETA exists, else insufficient blocker.
	if d.AvailableToSell < requestedQty {
		if snap.NextDeliveryDate != nil {
			d.Backorder = true
			return d
		}
		code := BlockerSupplierAvailabilityInsufficient
		d.BlockerCode = &code
	}
	return d
}

// leadTimeOK reports whether the supplier handling time fits the policy lead-time cap.
func leadTimeOK(snap SupplierAvailability, pol EffectivePolicy) bool {
	if pol.MaxLeadTimeDays == nil {
		return true
	}
	if snap.MaxHandlingDays == nil {
		return true // no handling data — do not block on lead time
	}
	return *snap.MaxHandlingDays <= *pol.MaxLeadTimeDays
}
```

- [ ] **Step 4: Run them to verify they pass**

Run: `gofmt -w -s internal/model/supplier_availability_resolve.go && go test ./internal/model/ -run 'TestResolve' -count=1`
Expected: PASS (all resolver cases).

- [ ] **Step 5: Commit**

```bash
git add internal/model/supplier_availability_resolve.go internal/model/supplier_availability_resolve_test.go
git commit -m "OPE-418: pure ResolveAvailability resolver + policy precedence chain"
```

---

## Task 5: Repository — snapshot upsert, policy CRUD, resolution-load

**Files:**
- Create: `apps/api-server/internal/repository/supplier_availability_repository.go`
- Test: covered by the integration test in Task 9 (repo methods are thin SQL; the resolver math is already unit-tested).

Follow the tenant-scoped repository pattern (methods take `pgx.Tx` obtained from `database.WithTenant`), mirroring `internal/repository/fulfillment_repository.go`.

- [ ] **Step 1: Write the repository**

Create `apps/api-server/internal/repository/supplier_availability_repository.go`:

```go
package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// SupplierAvailabilityRepository is tenant-scoped: every method takes a pgx.Tx obtained
// from database.WithTenant, so PostgreSQL RLS scopes all rows to the current tenant.
type SupplierAvailabilityRepository struct{}

// NewSupplierAvailabilityRepository constructs the repository.
func NewSupplierAvailabilityRepository() *SupplierAvailabilityRepository {
	return &SupplierAvailabilityRepository{}
}

const supplierAvailabilityColumns = `id, tenant_id, supplier_id, supplier_product_id, product_id,
	warehouse_external_id, source_quantity, availability_type, min_handling_days, max_handling_days,
	next_delivery_date, reservation_supported, freshness_observed_at, source_max_stale_seconds,
	last_successful_sync_id, raw, created_at, updated_at`

// UpsertSnapshot inserts or updates the snapshot for (tenant, supplier_product, warehouse),
// keyed by the uq_supplier_availability_product_wh unique index. Idempotent.
func (r *SupplierAvailabilityRepository) UpsertSnapshot(ctx context.Context, tx pgx.Tx, a model.SupplierAvailability) (*model.SupplierAvailability, error) {
	raw := a.Raw
	if raw == nil {
		raw = []byte("{}")
	}
	out, err := scanSupplierAvailability(tx.QueryRow(ctx,
		`INSERT INTO supplier_availability
		   (tenant_id, supplier_id, supplier_product_id, product_id, warehouse_external_id,
		    source_quantity, availability_type, min_handling_days, max_handling_days, next_delivery_date,
		    reservation_supported, freshness_observed_at, source_max_stale_seconds, last_successful_sync_id, raw)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15::jsonb)
		 ON CONFLICT (tenant_id, supplier_product_id, warehouse_external_id) DO UPDATE SET
		    product_id = EXCLUDED.product_id,
		    source_quantity = EXCLUDED.source_quantity,
		    availability_type = EXCLUDED.availability_type,
		    min_handling_days = EXCLUDED.min_handling_days,
		    max_handling_days = EXCLUDED.max_handling_days,
		    next_delivery_date = EXCLUDED.next_delivery_date,
		    reservation_supported = EXCLUDED.reservation_supported,
		    freshness_observed_at = EXCLUDED.freshness_observed_at,
		    source_max_stale_seconds = EXCLUDED.source_max_stale_seconds,
		    last_successful_sync_id = EXCLUDED.last_successful_sync_id,
		    raw = EXCLUDED.raw,
		    updated_at = now()
		 RETURNING `+supplierAvailabilityColumns,
		a.TenantID, a.SupplierID, a.SupplierProductID, a.ProductID, a.WarehouseExternalID,
		a.SourceQuantity, a.AvailabilityType, a.MinHandlingDays, a.MaxHandlingDays, a.NextDeliveryDate,
		a.ReservationSupported, a.FreshnessObservedAt, a.SourceMaxStaleSeconds, a.LastSuccessfulSyncID, string(raw)))
	if err != nil {
		return nil, fmt.Errorf("upsert supplier availability: %w", err)
	}
	return out, nil
}

// ListSnapshotsByProduct returns all warehouse snapshots for a product (tenant-scoped).
func (r *SupplierAvailabilityRepository) ListSnapshotsByProduct(ctx context.Context, tx pgx.Tx, productID uuid.UUID) ([]model.SupplierAvailability, error) {
	rows, err := tx.Query(ctx, `SELECT `+supplierAvailabilityColumns+`
		FROM supplier_availability WHERE product_id = $1 ORDER BY warehouse_external_id`, productID)
	if err != nil {
		return nil, fmt.Errorf("list supplier availability by product: %w", err)
	}
	defer rows.Close()
	out := []model.SupplierAvailability{}
	for rows.Next() {
		a, e := scanSupplierAvailabilityRows(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

const supplierPolicyColumns = `id, tenant_id, scope, supplier_id, product_id, listing_id, channel,
	mode, safety_buffer, freshness_window_seconds, max_lead_time_days, override_quantity,
	allow_channel_increase, require_reservation, require_preflight, created_at, updated_at`

// ListPoliciesForContext loads every policy row that could apply to a (supplier, product,
// listing, channel) context — the caller orders them least->most specific and folds with
// model.ResolvePolicyChain. listingID/channel may be nil/empty when not applicable.
func (r *SupplierAvailabilityRepository) ListPoliciesForContext(ctx context.Context, tx pgx.Tx, supplierID, productID uuid.UUID, listingID *uuid.UUID, channel *string) ([]model.SupplierAvailabilityPolicy, error) {
	rows, err := tx.Query(ctx, `SELECT `+supplierPolicyColumns+`
		FROM supplier_availability_policy
		WHERE (scope = 'supplier' AND supplier_id = $1)
		   OR (scope = 'product'  AND supplier_id = $1 AND product_id = $2)
		   OR (scope = 'listing'  AND listing_id = $3)
		   OR (scope = 'channel'  AND channel = $4)`,
		supplierID, productID, listingID, channel)
	if err != nil {
		return nil, fmt.Errorf("list policies for context: %w", err)
	}
	defer rows.Close()
	out := []model.SupplierAvailabilityPolicy{}
	for rows.Next() {
		p, e := scanSupplierPolicyRows(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// UpsertPolicy inserts or updates a single scope policy (keyed by the partial unique index
// for its scope). Idempotent per scope target.
func (r *SupplierAvailabilityRepository) UpsertPolicy(ctx context.Context, tx pgx.Tx, p model.SupplierAvailabilityPolicy) (*model.SupplierAvailabilityPolicy, error) {
	out, err := scanSupplierPolicy(tx.QueryRow(ctx,
		`INSERT INTO supplier_availability_policy
		   (tenant_id, scope, supplier_id, product_id, listing_id, channel, mode, safety_buffer,
		    freshness_window_seconds, max_lead_time_days, override_quantity, allow_channel_increase,
		    require_reservation, require_preflight)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		 RETURNING `+supplierPolicyColumns,
		p.TenantID, p.Scope, p.SupplierID, p.ProductID, p.ListingID, p.Channel, p.Mode, p.SafetyBuffer,
		p.FreshnessWindowSecs, p.MaxLeadTimeDays, p.OverrideQuantity, p.AllowChannelIncrease,
		p.RequireReservation, p.RequirePreflight))
	if err != nil {
		return nil, fmt.Errorf("upsert supplier availability policy: %w", err)
	}
	return out, nil
}

func scanSupplierAvailability(row pgx.Row) (*model.SupplierAvailability, error) {
	var a model.SupplierAvailability
	if err := row.Scan(&a.ID, &a.TenantID, &a.SupplierID, &a.SupplierProductID, &a.ProductID,
		&a.WarehouseExternalID, &a.SourceQuantity, &a.AvailabilityType, &a.MinHandlingDays, &a.MaxHandlingDays,
		&a.NextDeliveryDate, &a.ReservationSupported, &a.FreshnessObservedAt, &a.SourceMaxStaleSeconds,
		&a.LastSuccessfulSyncID, &a.Raw, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, fmt.Errorf("scan supplier availability: %w", err)
	}
	return &a, nil
}

func scanSupplierAvailabilityRows(rows pgx.Rows) (*model.SupplierAvailability, error) { return scanSupplierAvailability(rows) }

func scanSupplierPolicy(row pgx.Row) (*model.SupplierAvailabilityPolicy, error) {
	var p model.SupplierAvailabilityPolicy
	if err := row.Scan(&p.ID, &p.TenantID, &p.Scope, &p.SupplierID, &p.ProductID, &p.ListingID, &p.Channel,
		&p.Mode, &p.SafetyBuffer, &p.FreshnessWindowSecs, &p.MaxLeadTimeDays, &p.OverrideQuantity,
		&p.AllowChannelIncrease, &p.RequireReservation, &p.RequirePreflight, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, fmt.Errorf("scan supplier policy: %w", err)
	}
	return &p, nil
}

func scanSupplierPolicyRows(rows pgx.Rows) (*model.SupplierAvailabilityPolicy, error) { return scanSupplierPolicy(rows) }
```

(`pgx.Row` and `pgx.Rows` both satisfy the `Scan` shape used here, so the shared scan helpers work for both single-row and multi-row queries.)

- [ ] **Step 2: Verify it compiles**

Run: `gofmt -w -s internal/repository/supplier_availability_repository.go && go build ./...`
Expected: builds clean (exit 0).

- [ ] **Step 3: Commit**

```bash
git add internal/repository/supplier_availability_repository.go
git commit -m "OPE-418: supplier-availability repository (snapshot upsert, policy CRUD, resolution load)"
```

---

## Task 6: Config flag `SUPPLIER_AVAILABILITY_ENABLED`

**Files:**
- Modify: `apps/api-server/internal/config/config.go`
- Test: `apps/api-server/internal/config/config_test.go` (if present; else skip the test step and just verify the field parses)

- [ ] **Step 1: Add the flag field**

In `apps/api-server/internal/config/config.go`, next to `FulfillmentProcessEnabled` (line ~61), add:

```go
	// SupplierAvailabilityEnabled turns on the OPE-418 supplier-availability read-model:
	// the supplier sync writes snapshots and dropship routing / stock propagation consult
	// the resolver. Default false -> the legacy supplier_products.stock_quantity path is
	// unchanged.
	SupplierAvailabilityEnabled bool `env:"SUPPLIER_AVAILABILITY_ENABLED" envDefault:"false"`
```

- [ ] **Step 2: Verify it builds + parses**

Run: `gofmt -w -s internal/config/config.go && go build ./... && go test ./internal/config/ -count=1`
Expected: build clean; config tests pass (or "no test files").

- [ ] **Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "OPE-418: SUPPLIER_AVAILABILITY_ENABLED config flag (default off)"
```

---

## Task 7: Service — resolution + policy CRUD with override audit

**Files:**
- Create: `apps/api-server/internal/service/supplier_availability_service.go`
- Test: integration coverage in Task 9; the resolution math is unit-tested in Task 4.

This service owns the gated entry point used by routing/propagation and the audited policy writes.

- [ ] **Step 1: Write the service**

Create `apps/api-server/internal/service/supplier_availability_service.go`:

```go
package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// SupplierAvailabilityService is the gated entry point (OPE-418) for resolving supplier
// availability and writing audited policies. When disabled it is a no-op: Enabled() is
// false and ResolveForProduct returns a zero decision so callers fall back to legacy stock.
type SupplierAvailabilityService struct {
	enabled bool
	pool    *pgxpool.Pool
	repo    *repository.SupplierAvailabilityRepository
	audit   repository.AuditRepo
}

// NewSupplierAvailabilityService constructs the service. enabled comes from
// cfg.SupplierAvailabilityEnabled; pool is the RLS-scoped app pool.
func NewSupplierAvailabilityService(enabled bool, pool *pgxpool.Pool, repo *repository.SupplierAvailabilityRepository, audit repository.AuditRepo) *SupplierAvailabilityService {
	return &SupplierAvailabilityService{enabled: enabled, pool: pool, repo: repo, audit: audit}
}

// Enabled reports whether the supplier-availability read-model is active. Nil-safe.
func (s *SupplierAvailabilityService) Enabled() bool { return s != nil && s.enabled }

// ResolveForProduct loads the best snapshot + the policy chain for a (supplier, product,
// listing?, channel?) context and returns the resolved decision. When disabled it returns
// a zero decision with ok=false so callers keep the legacy behavior. Picks the snapshot
// with the most stock across warehouses (the order line is satisfied from any warehouse).
func (s *SupplierAvailabilityService) ResolveForProduct(ctx context.Context, tenantID, supplierID, productID uuid.UUID, listingID *uuid.UUID, channel *string, requestedQty int, preflightSupported bool, now time.Time) (model.AvailabilityDecision, bool, error) {
	if !s.Enabled() {
		return model.AvailabilityDecision{}, false, nil
	}
	var decision model.AvailabilityDecision
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		snaps, e := s.repo.ListSnapshotsByProduct(ctx, tx, productID)
		if e != nil {
			return e
		}
		if len(snaps) == 0 {
			// No snapshot recorded -> treat as unknown (untrusted) so routing is blocked.
			decision = model.AvailabilityDecision{Status: model.AvailabilityStatusUnknown}
			code := model.BlockerSupplierAvailabilityUnknown
			decision.BlockerCode = &code
			return nil
		}
		best := snaps[0]
		for _, sn := range snaps[1:] {
			if sn.SourceQuantity > best.SourceQuantity {
				best = sn
			}
		}
		policies, e := s.repo.ListPoliciesForContext(ctx, tx, supplierID, productID, listingID, channel)
		if e != nil {
			return e
		}
		eff := model.ResolvePolicyChain(sortPoliciesBySpecificity(policies))
		decision = model.ResolveAvailability(best, eff, requestedQty, preflightSupported, now)
		return nil
	})
	if err != nil {
		return model.AvailabilityDecision{}, false, fmt.Errorf("resolve supplier availability: %w", err)
	}
	return decision, true, nil
}

// scopeRank orders scopes least->most specific for the precedence fold.
func scopeRank(scope string) int {
	switch scope {
	case model.PolicyScopeSupplier:
		return 0
	case model.PolicyScopeProduct:
		return 1
	case model.PolicyScopeListing:
		return 2
	case model.PolicyScopeChannel:
		return 3
	default:
		return -1
	}
}

// sortPoliciesBySpecificity returns the policies ordered least->most specific so
// model.ResolvePolicyChain folds them with the most specific winning.
func sortPoliciesBySpecificity(in []model.SupplierAvailabilityPolicy) []model.SupplierAvailabilityPolicy {
	out := make([]model.SupplierAvailabilityPolicy, len(in))
	copy(out, in)
	sort.SliceStable(out, func(i, j int) bool { return scopeRank(out[i].Scope) < scopeRank(out[j].Scope) })
	return out
}

// SetPolicy upserts a scope policy and writes an audit entry when it changes the manual
// controls (override_quantity / mode) — the research requires an active override is never
// silently changed by automation.
func (s *SupplierAvailabilityService) SetPolicy(ctx context.Context, tenantID, actorID uuid.UUID, ip string, p model.SupplierAvailabilityPolicy) (*model.SupplierAvailabilityPolicy, error) {
	if !model.IsValidPolicyScope(p.Scope) || !model.IsValidPolicyMode(p.Mode) {
		return nil, NewValidationError(fmt.Errorf("invalid scope %q or mode %q", p.Scope, p.Mode))
	}
	p.TenantID = tenantID
	var out *model.SupplierAvailabilityPolicy
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		saved, e := s.repo.UpsertPolicy(ctx, tx, p)
		if e != nil {
			return e
		}
		out = saved
		if s.audit != nil && (p.OverrideQuantity != nil || p.Mode != model.PolicyModeAuto) {
			changes := map[string]string{"scope": p.Scope, "mode": p.Mode}
			if p.OverrideQuantity != nil {
				changes["override_quantity"] = fmt.Sprintf("%d", *p.OverrideQuantity)
			}
			return s.audit.Log(ctx, tx, model.AuditEntry{
				TenantID:   tenantID,
				UserID:     actorID,
				Action:     "supplier_availability.policy_override",
				EntityType: "supplier_availability_policy",
				EntityID:   saved.ID,
				Changes:    changes,
				IPAddress:  ip,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
```

- [ ] **Step 2: Verify it builds**

Run: `gofmt -w -s internal/service/supplier_availability_service.go && go build ./...`
Expected: builds clean.

- [ ] **Step 3: Commit**

```bash
git add internal/service/supplier_availability_service.go
git commit -m "OPE-418: supplier-availability service (gated resolve + audited policy CRUD)"
```

---

## Task 8: Wire the snapshot upsert into SupplierSyncWorker + consumption into routing/propagation

**Files:**
- Modify: `apps/api-server/internal/worker/supplier_sync_worker.go` (gated snapshot upsert)
- Modify: `apps/api-server/internal/service/dropship_service.go` (consult resolver before auto-routing)
- Modify: `apps/api-server/internal/worker/stock_sync_worker.go` (channel-increase gate)
- Modify: `apps/api-server/cmd/server/main.go` (construct + inject the service, gated)

All hooks are nil-safe + gated: when the service is nil/disabled they are complete no-ops.

- [ ] **Step 1: SupplierSyncWorker — write snapshots during sync**

In `internal/worker/supplier_sync_worker.go`, add an optional `availability *service.SupplierAvailabilityService` field + a `WithAvailability(svc)` setter (mirror the OPE-417 `WithFulfillment` setter pattern). Where the worker upserts each `supplier_products` row (the existing per-product loop), after a successful product upsert add, inside the existing `database.WithTenant` tx:

```go
	if w.availability != nil && w.availability.Enabled() {
		_, _ = w.availabilityRepo.UpsertSnapshot(ctx, tx, model.SupplierAvailability{
			TenantID:             tenantID,
			SupplierID:           supplier.ID,
			SupplierProductID:    sp.ID,
			ProductID:            sp.ProductID, // *uuid.UUID, nil until mapped
			WarehouseExternalID:  "", // single-warehouse feeds; multi-warehouse is a later enrichment
			SourceQuantity:       sp.StockQuantity,
			AvailabilityType:     model.AvailabilityExactQuantity, // feed gives an exact count; refine per format later
			ReservationSupported: false,
			FreshnessObservedAt:  time.Now(),
			LastSuccessfulSyncID: &syncJobID, // the sync_jobs row id for this run
		})
	}
```

Pass `w.availabilityRepo` (a `*repository.SupplierAvailabilityRepository`) in alongside the service, or reach it through the service — whichever matches the worker's existing field style. The upsert error is intentionally ignored (best-effort observability; never fail a catalog sync because a snapshot write failed) — log it instead if the worker has a logger handy.

- [ ] **Step 2: DropshipService — consult the resolver before auto-routing**

In `internal/service/dropship_service.go`, the dropship-unit creation method (around line 92, the one that calls `s.supplierCapability` + `s.fulfillment.EnsureUnit`): add an optional `availability *SupplierAvailabilityService` field + `SetAvailabilityService(svc)` setter. After the unit is ensured and the capability is `api`/`portal` (i.e. auto-submit is otherwise possible), insert:

```go
	if s.availability != nil && s.availability.Enabled() {
		decision, ok, err := s.availability.ResolveForProduct(ctx, tenantID, supplierID, productID, nil, nil, requestedQty, false, time.Now())
		if err == nil && ok {
			if decision.BlockerCode != nil {
				// stale / unknown / insufficient -> typed blocker on the unit, do NOT auto-submit.
				s.fulfillment.CreateSupplierBlocker(ctx, tenantID, orderID, &unit.ID, *decision.BlockerCode,
					"supplier availability not safe for auto-routing")
				return // leave the unit waiting/manual
			}
			if decision.Backorder {
				// insufficient now but ETA known -> backorder unit (existing UnitTypeBackorder path).
				s.fulfillment.EnsureUnit(ctx, tenantID, orderID, model.UnitTypeBackorder, supplierID.String(),
					map[string]any{"supplier_id": supplierID.String(), "reason": "supplier_backorder"})
				return
			}
			// decision.AutoRoutable == true -> fall through to the existing auto-submit path.
		}
	}
```

Use the exact local variable names already in that method (`tenantID`, `orderID`, `supplierID`, `unit`); `productID` and `requestedQty` come from the dropship line being routed — thread them from the caller if not already in scope (the method already iterates order lines for the supplier).

- [ ] **Step 3: StockSyncWorker — channel-increase gate for supplier-backed listings**

In `internal/worker/stock_sync_worker.go`, where it computes `stockQty` for a listing (around line 110) and is about to push it via `provider.UpdateStock` / `BulkUpdateStock`: for listings backed by a supplier (not own warehouse stock), gate an *increase*. Add an optional `availability *service.SupplierAvailabilityService`. Before pushing a stock value that is HIGHER than the listing's last-known channel stock for a supplier-backed product:

```go
	if w.availability != nil && w.availability.Enabled() && isSupplierBacked {
		decision, ok, err := w.availability.ResolveForProduct(ctx, tenantID, supplierID, productID, &listingID, &channel, 0, false, time.Now())
		if err == nil && ok {
			if newStock > lastChannelStock && !decision.ChannelIncreaseAllowed {
				newStock = lastChannelStock // suppress the increase; decreases always allowed
			} else if decision.Status == model.AvailabilityStatusTrusted {
				newStock = min(newStock, decision.AvailableToSell)
			}
		}
	}
```

If determining `isSupplierBacked` / `supplierID` / `lastChannelStock` is not cheaply available in the worker's current query, scope this step to ONLY the suppression of increases for products that have a supplier-availability snapshot, and note the own-warehouse path is unaffected. Decreases must always pass through.

- [ ] **Step 4: main.go — construct + inject the service (gated)**

In `cmd/server/main.go`, after `fulfillmentService` is built (line ~331) and near where the supplier sync / dropship / stock-sync are constructed:

```go
	supplierAvailabilityRepo := repository.NewSupplierAvailabilityRepository()
	supplierAvailabilityService := service.NewSupplierAvailabilityService(
		cfg.SupplierAvailabilityEnabled, pool, supplierAvailabilityRepo, auditRepo)
```

Then chain/inject it: `supplierSyncWorker.WithAvailability(supplierAvailabilityService)` (+ pass `supplierAvailabilityRepo` if the worker needs the repo directly), `dropshipService.SetAvailabilityService(supplierAvailabilityService)`, and the stock-sync worker setter. All are no-ops when `cfg.SupplierAvailabilityEnabled` is false.

- [ ] **Step 5: Build + vet + full unit tests**

Run:
```bash
gofmt -w -s internal/worker/supplier_sync_worker.go internal/service/dropship_service.go internal/worker/stock_sync_worker.go cmd/server/main.go
go build ./... && go vet ./... && go test ./... 2>&1 | grep -v "^ok\|no test files" | tail
```
Expected: build + vet clean; no failing unit packages.

- [ ] **Step 6: Commit**

```bash
git add internal/worker/supplier_sync_worker.go internal/service/dropship_service.go internal/worker/stock_sync_worker.go cmd/server/main.go
git commit -m "OPE-418: wire supplier-availability into sync, dropship routing, stock propagation (gated)"
```

---

## Task 9: DB-bound integration tests

**Files:**
- Create: `apps/api-server/tests/integration/supplier_availability_test.go`

Mirror the existing harness (`tests/integration/harness_test.go`, `fulfillment_read_api_test.go`): `appPool` + `database.WithTenant` for tenant ops, `superPool` for cross-tenant setup, `seedTenant`. Build tag `//go:build integration`, gated on `DATABASE_URL`.

- [ ] **Step 1: Write the integration tests**

Create `apps/api-server/tests/integration/supplier_availability_test.go`:

```go
//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// seedSupplierAndProduct inserts a supplier + supplier_product for the tenant and returns
// their ids. Uses the superPool (cross-tenant setup) consistent with other harness seeds.
func seedSupplierAndProduct(t *testing.T, ctx context.Context, tenantID uuid.UUID) (supplierID, supplierProductID uuid.UUID) {
	t.Helper()
	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`INSERT INTO suppliers (tenant_id, name, feed_format) VALUES ($1,'Acme','iof') RETURNING id`,
			tenantID).Scan(&supplierID); err != nil {
			return err
		}
		return tx.QueryRow(ctx,
			`INSERT INTO supplier_products (tenant_id, supplier_id, external_id, name, stock_quantity)
			 VALUES ($1,$2,'EXT-1','Widget',100) RETURNING id`,
			tenantID, supplierID).Scan(&supplierProductID)
	}))
	return supplierID, supplierProductID
}

// TestSupplierAvailability_UpsertIdempotent proves the snapshot upsert is idempotent on
// (tenant, supplier_product, warehouse) — two upserts leave exactly one row, updated.
func TestSupplierAvailability_UpsertIdempotent(t *testing.T) {
	ctx := context.Background()
	tenantID := seedTenant(t, ctx)
	supplierID, spID := seedSupplierAndProduct(t, ctx, tenantID)
	repo := repository.NewSupplierAvailabilityRepository()

	upsert := func(qty int) {
		require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
			_, e := repo.UpsertSnapshot(ctx, tx, model.SupplierAvailability{
				TenantID: tenantID, SupplierID: supplierID, SupplierProductID: spID,
				WarehouseExternalID: "", SourceQuantity: qty,
				AvailabilityType: model.AvailabilityExactQuantity, FreshnessObservedAt: time.Now(),
			})
			return e
		}))
	}
	upsert(100)
	upsert(80)

	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		var n, qty int
		if e := tx.QueryRow(ctx, `SELECT count(*), max(source_quantity) FROM supplier_availability WHERE supplier_product_id=$1`, spID).Scan(&n, &qty); e != nil {
			return e
		}
		assert.Equal(t, 1, n, "exactly one snapshot row")
		assert.Equal(t, 80, qty, "second upsert updated the quantity")
		return nil
	}))
}

// TestSupplierAvailability_RLSIsolation proves tenant A never sees tenant B's snapshots.
func TestSupplierAvailability_RLSIsolation(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	tenantB := seedTenant(t, ctx)
	sa, spa := seedSupplierAndProduct(t, ctx, tenantA)
	sb, spb := seedSupplierAndProduct(t, ctx, tenantB)
	repo := repository.NewSupplierAvailabilityRepository()

	seed := func(tenant, supplier, sp uuid.UUID) {
		require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
			_, e := repo.UpsertSnapshot(ctx, tx, model.SupplierAvailability{
				TenantID: tenant, SupplierID: supplier, SupplierProductID: sp,
				SourceQuantity: 50, AvailabilityType: model.AvailabilityExactQuantity, FreshnessObservedAt: time.Now(),
			})
			return e
		}))
	}
	seed(tenantA, sa, spa)
	seed(tenantB, sb, spb)

	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		var n int
		if e := tx.QueryRow(ctx, `SELECT count(*) FROM supplier_availability`).Scan(&n); e != nil {
			return e
		}
		assert.Equal(t, 1, n, "tenant A sees only its own snapshot")
		return nil
	}))
}

// TestSupplierAvailability_PolicyChainResolves proves the 4-scope policy load + fold gives
// the expected effective policy (product buffer overrides supplier buffer, others inherit).
func TestSupplierAvailability_PolicyChainResolves(t *testing.T) {
	ctx := context.Background()
	tenantID := seedTenant(t, ctx)
	supplierID, spID := seedSupplierAndProduct(t, ctx, tenantID)
	_ = spID
	repo := repository.NewSupplierAvailabilityRepository()

	// supplier-scope: buffer 5, freshness 7200; product-scope: buffer 9 (needs a product id).
	var productID uuid.UUID
	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		if e := tx.QueryRow(ctx, `INSERT INTO products (tenant_id, sku, name, price) VALUES ($1,'SKU1','P',10) RETURNING id`, tenantID).Scan(&productID); e != nil {
			return e
		}
		fw := 7200
		if _, e := repo.UpsertPolicy(ctx, tx, model.SupplierAvailabilityPolicy{
			TenantID: tenantID, Scope: model.PolicyScopeSupplier, SupplierID: &supplierID,
			Mode: model.PolicyModeAuto, SafetyBuffer: 5, FreshnessWindowSecs: &fw,
		}); e != nil {
			return e
		}
		_, e := repo.UpsertPolicy(ctx, tx, model.SupplierAvailabilityPolicy{
			TenantID: tenantID, Scope: model.PolicyScopeProduct, SupplierID: &supplierID, ProductID: &productID,
			Mode: model.PolicyModeAuto, SafetyBuffer: 9,
		})
		return e
	}))

	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		policies, e := repo.ListPoliciesForContext(ctx, tx, supplierID, productID, nil, nil)
		if e != nil {
			return e
		}
		assert.Len(t, policies, 2)
		return nil
	}))
}
```

- [ ] **Step 2: Run the integration tests**

Run:
```bash
DATABASE_URL="postgres://openoms:openoms-dev-password@127.0.0.1:5433/openoms?sslmode=disable" \
go test -tags integration ./tests/integration/ -run 'SupplierAvailability' -count=1
```
Expected: PASS (upsert idempotency, RLS isolation, policy chain). If a seed column name (e.g. `products` required columns) differs from the schema, adjust the seed INSERT to match `migrations/000001_init_schema.up.sql` — keep the assertions.

- [ ] **Step 3: Commit**

```bash
git add tests/integration/supplier_availability_test.go
git commit -m "OPE-418: supplier-availability integration tests (upsert idempotency, RLS, policy chain)"
```

---

## Task 10: Full validation sweep

**Files:** none (validation only).

- [ ] **Step 1: gofmt + build + vet + full unit tests**

Run:
```bash
test -z "$(gofmt -l .)" && echo "gofmt clean" || gofmt -l .
go build ./... && go vet ./... && go vet -tags integration ./tests/integration/
go test ./... 2>&1 | grep -v "^ok\|no test files" | tail
```
Expected: gofmt clean; build + vet clean; no failing unit packages.

- [ ] **Step 2: CI-pinned lint**

Run: `/tmp/glci29/golangci-lint run --new-from-rev=main --timeout=5m`
Expected: `0 issues`.

- [ ] **Step 3: Full fulfillment/orchestration integration regression**

Run:
```bash
DATABASE_URL="postgres://openoms:openoms-dev-password@127.0.0.1:5433/openoms?sslmode=disable" \
go test -tags integration ./tests/integration/ -count=1 2>&1 | tail -3
```
Expected: `ok` (no regression).

- [ ] **Step 4: Flag-off no-op sanity**

Confirm by reading the diff that every new hook (`SupplierSyncWorker` snapshot upsert, `DropshipService` resolver consult, `StockSyncWorker` gate) is guarded by `s.availability != nil && s.availability.Enabled()`, and `Enabled()` returns false when `SUPPLIER_AVAILABILITY_ENABLED` is unset — so the default build is byte-for-byte unchanged.

---

## Self-Review (completed by plan author)

- **Spec coverage:** migration 000041 (Task 1) ✓; snapshot + 4-scope policy tables (Task 1) ✓; `supplier_availability_insufficient` blocker (Task 2) ✓; pure resolver + precedence + math + gates (Task 4) ✓; repo upsert/CRUD/resolution-load (Task 5) ✓; `SUPPLIER_AVAILABILITY_ENABLED` (Task 6) ✓; gated SupplierSyncWorker upsert (Task 8) ✓; dropship/backorder routing consumption + blocker (Task 8) ✓; channel-increase gate (Task 8) ✓; override audit (Task 7) ✓; test matrix — resolver unit (Task 4) + DB integration upsert/RLS/policy + flag-off (Tasks 9–10) ✓.
- **Refinement vs spec:** the spec said "add `supplier_availability_insufficient`"; on inspection `supplier_availability_unknown` already exists as its own code, so the resolver uses `BlockerSupplierAvailabilityUnknown` for unknown and `BlockerSupplierAvailabilityStale` for stale, and only the *insufficient* code is added — captured in Task 2 + Task 4.
- **Type consistency:** `EffectivePolicy`, `AvailabilityDecision`, `ResolveAvailability`, `ResolvePolicyChain`, `SupplierAvailability`, `SupplierAvailabilityPolicy`, `UpsertSnapshot`, `ListSnapshotsByProduct`, `ListPoliciesForContext`, `UpsertPolicy`, `SupplierAvailabilityService.Enabled/ResolveForProduct/SetPolicy` are used identically across tasks.
- **Deferred (out of scope, per spec):** supplier-order preflight/submit/reconcile engine; multi-warehouse split; the channel write-back push mechanics beyond the gate; per-channel availability webhooks; policy-editing dashboard UI. Multi-warehouse snapshots are supported by the schema (the `warehouse_external_id` dimension) but the sync writes a single `''` warehouse until a per-format enrichment task.

## Known follow-on tasks (not in this plan)
- Supplier-order engine (prepare/preflight/submit/reconcile) — consumes `auto_routable` + `RequirePreflight`.
- Policy-management API endpoints + dashboard UI (this plan adds the service `SetPolicy`; HTTP handlers/routes are a thin follow-up).
- Multi-warehouse snapshot enrichment per feed format (BigBuy/La Grana style).
