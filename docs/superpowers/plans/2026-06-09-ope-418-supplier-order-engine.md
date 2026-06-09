# OPE-418/Phase-7 Supplier-Order Engine — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn a routable dropship unit into a supplier order through prepare → preflight → submit → reconcile (API suppliers), routing portal/manual/unsupported to an explicit operator step + typed blocker, all behind `SUPPLIER_ORDER_ENABLED`.

**Architecture:** Extend the OPE-418 dropship gate to enqueue a gated `supplier.order.submit` outbox event for API-capable, routable units; a `SupplierOrderHandler` runs the three synchronous phases idempotently (each a fulfillment step); a recurring `SupplierOrderStatusPoller` (mirroring `TrackingPoller`) reconciles status/tracking into `dropship_orders`. Preflight + status-read are optional adapter sub-interfaces.

**Tech Stack:** Go 1.25 (api-server), pgx/v5, PostgreSQL 16 FORCE RLS, chi, testify. Spec: `docs/superpowers/specs/2026-06-09-ope-418-supplier-order-engine-design.md`.

**Conventions every task follows:**
- Run from `apps/api-server/`. Unit: `go test ./internal/<pkg>/ -run <Name> -count=1`. Integration: `DATABASE_URL=postgres://openoms:openoms-dev-password@127.0.0.1:5433/openoms?sslmode=disable go test -tags integration ./tests/integration/ -run <Name> -count=1`.
- `gofmt -w -s <file>` before each commit. Before any push: CI lint is **golangci-lint v2.9.0** — `/tmp/glci29/golangci-lint run --new-from-rev=main --timeout=5m` = 0 issues.
- No migration is expected (blocker codes are app-validated; `dropship_orders` already has the needed columns). If one proves necessary, it is additive only with single-line `-- migrate:index-lock-ok` markers.
- Commit after every task. No push/PR (the executing workflow handles branch+PR).
- **READ the real signatures before each task:** `FulfillmentService.{EnsureUnit,RecordStep,RecordUnitTransition,CreateSupplierBlocker,EnsureProcessForOrderUnconditional}`, `OrchestrationRepository.{EnqueueEvent,MarkSucceeded,MarkFailedRetry,SetLatestAttemptExternalExecID}`, `OrchestrationHandler`/dispatcher `Register`, `model.Permanent`, `DropshipOrderRepo.{FindByOrderID,Create,UpdateFields}` + `model.DropshipOrder`/`UpdateDropshipStatusRequest`, `integration.{SupplierProvider,SupplierOrderRequest,SupplierOrderLine,SupplierOrderResult,NewSupplierProvider}`, `TrackingPoller` (Name/Interval/Run + how it lists active integrations on the worker pool), `OrderRepo.FindByID`, `SupplierRepo`/`supplierCapability`. Reconcile mismatches against reality.

---

## File Structure

| File | Responsibility | Create/Modify |
|---|---|---|
| `internal/model/fulfillment.go` | 6 new supplier-order blocker codes + categories | Modify |
| `internal/model/fulfillment_test.go` | blocker-code validity tests | Modify |
| `internal/config/config.go` | `SupplierOrderEnabled` flag | Modify |
| `internal/integration/supplier.go` | `SupplierPreflighter` + `SupplierStatusReader` sub-interfaces + `SupplierPreflightResult` + `SupplierOrderStatus` | Modify |
| `internal/service/supplier_order_service.go` | request builder, capability resolution, the submit-enqueue + the manual-branch blocker | Create |
| `internal/service/supplier_order_handler.go` | `SupplierOrderHandler` (prepare→preflight→submit, idempotent) | Create |
| `internal/service/dropship_service.go` | gate hook: enqueue submit (api) / manual blocker (portal/manual) when enabled | Modify |
| `internal/worker/supplier_order_status_poller.go` | reconcile poller (status/tracking → dropship_orders + unit step) | Create |
| `cmd/server/main.go` | wire service/handler/poller (gated) | Modify |
| `tests/integration/supplier_order_test.go` | submit, missing-data blocker, manual branch, idempotency, reconcile, RLS, flag-off | Create |

---

## Task 1: Six new supplier-order blocker codes

**Files:** Modify `internal/model/fulfillment.go`; Test `internal/model/fulfillment_test.go`.

- [ ] **Step 1: Write the failing test**

Add to `internal/model/fulfillment_test.go`:

```go
func TestSupplierOrderBlockerCodes(t *testing.T) {
	for _, c := range []string{
		BlockerSupplierOrderMissingData, BlockerSupplierOrderAmbiguousSKU, BlockerSupplierOrderRejected,
		BlockerSupplierPaymentAwaiting, BlockerSupplierPartialFulfillment, BlockerSupplierManualSubmissionRequired,
	} {
		assert.Truef(t, IsValidBlockerCode(c), "%q should be valid", c)
		assert.Equalf(t, "supplier", blockerCategories[c], "%q should be category supplier", c)
	}
}
```

- [ ] **Step 2: Run it (FAIL — undefined)**

Run: `go test ./internal/model/ -run TestSupplierOrderBlockerCodes -count=1`

- [ ] **Step 3: Add the constants + categories**

In `internal/model/fulfillment.go` blocker-code `const` block add:
```go
	BlockerSupplierOrderMissingData         = "supplier_order_missing_data"
	BlockerSupplierOrderAmbiguousSKU        = "supplier_order_ambiguous_sku"
	BlockerSupplierOrderRejected            = "supplier_order_rejected"
	BlockerSupplierPaymentAwaiting          = "supplier_payment_awaiting"
	BlockerSupplierPartialFulfillment       = "supplier_partial_fulfillment"
	BlockerSupplierManualSubmissionRequired = "supplier_manual_submission_required"
```
In `blockerCategories` add each → `"supplier"`. If a `validBlockerCodes` slice backs `IsValidBlockerCode`, add all six (mirror `BlockerSupplierAvailabilityStale`).

- [ ] **Step 4: Run it (PASS)**

Run: `gofmt -w -s internal/model/fulfillment.go && go test ./internal/model/ -run TestSupplierOrderBlockerCodes -count=1`

- [ ] **Step 5: Commit**

```bash
git add internal/model/fulfillment.go internal/model/fulfillment_test.go
git commit -m "OPE-418: supplier-order blocker codes (missing data, ambiguous sku, rejected, payment, partial, manual)"
```

---

## Task 2: Config flag `SUPPLIER_ORDER_ENABLED`

**Files:** Modify `internal/config/config.go`.

- [ ] **Step 1: Add the field**

Next to `SupplierAvailabilityEnabled`:
```go
	// SupplierOrderEnabled turns on the OPE-418/Phase-7 supplier-order engine: the dropship
	// gate enqueues supplier.order.submit for API-capable routable units, and the handler +
	// status poller run. Default false -> the gate's API branch keeps its current behavior
	// (mark the create_dropship_order step ready, no auto-submit).
	SupplierOrderEnabled bool `env:"SUPPLIER_ORDER_ENABLED" envDefault:"false"`
```

- [ ] **Step 2: Build** — `gofmt -w -s internal/config/config.go && go build ./... && go test ./internal/config/ -count=1`

- [ ] **Step 3: Commit** — `git add internal/config/config.go && git commit -m "OPE-418: SUPPLIER_ORDER_ENABLED config flag (default off)"`

---

## Task 3: Optional adapter capability sub-interfaces

**Files:** Modify `internal/integration/supplier.go`; Test `internal/integration/supplier_test.go`.

- [ ] **Step 1: Write the failing test**

Add `internal/integration/supplier_test.go` (or extend an existing one):

```go
func TestSupplierCapabilitySubInterfaces(t *testing.T) {
	// A type implementing both sub-interfaces satisfies them; the bare SupplierProvider does not.
	var p any = &fakeFullSupplier{}
	_, isPreflighter := p.(SupplierPreflighter)
	_, isStatusReader := p.(SupplierStatusReader)
	assert.True(t, isPreflighter)
	assert.True(t, isStatusReader)
}

type fakeFullSupplier struct{}

func (f *fakeFullSupplier) ProviderName() string                                          { return "fake" }
func (f *fakeFullSupplier) FetchProducts(ctx context.Context) ([]SupplierProduct, error)  { return nil, nil }
func (f *fakeFullSupplier) FetchInventory(ctx context.Context) ([]SupplierProduct, error) { return nil, nil }
func (f *fakeFullSupplier) CreateOrder(ctx context.Context, req SupplierOrderRequest) (*SupplierOrderResult, error) {
	return &SupplierOrderResult{ExternalOrderID: "x"}, nil
}
func (f *fakeFullSupplier) Preflight(ctx context.Context, req SupplierOrderRequest) (*SupplierPreflightResult, error) {
	return &SupplierPreflightResult{Accepted: true}, nil
}
func (f *fakeFullSupplier) GetOrderStatus(ctx context.Context, externalID string) (*SupplierOrderStatus, error) {
	return &SupplierOrderStatus{RawStatus: "confirmed"}, nil
}
```

- [ ] **Step 2: Run it (FAIL — undefined types)**

Run: `go test ./internal/integration/ -run TestSupplierCapabilitySubInterfaces -count=1`

- [ ] **Step 3: Add the sub-interfaces + result types**

In `internal/integration/supplier.go` (after the existing `SupplierOrderResult`):

```go
// SupplierPreflightResult is returned by an adapter that supports pre-submission validation.
type SupplierPreflightResult struct {
	Accepted       bool                `json:"accepted"`
	AcceptedTotal  float64             `json:"accepted_total,omitempty"`
	MissingFields  []string            `json:"missing_fields,omitempty"`  // -> supplier_order_missing_data
	BusinessErrors []string            `json:"business_errors,omitempty"` // -> supplier_order_rejected
	SplitLines     []SupplierOrderLine `json:"split_lines,omitempty"`     // partial/split -> supplier_partial_fulfillment
	PaymentDue     bool                `json:"payment_due,omitempty"`     // -> supplier_payment_awaiting
}

// SupplierOrderStatus is returned by an adapter that supports order status reads. RawStatus
// is the supplier's own status string (mapped to canonical downstream; raw is preserved).
type SupplierOrderStatus struct {
	RawStatus      string `json:"raw_status"`
	TrackingNumber string `json:"tracking_number,omitempty"`
	Carrier        string `json:"carrier,omitempty"`
}

// SupplierPreflighter is OPTIONALLY implemented by adapters that support pre-submission
// validation. The supplier-order engine type-asserts it; absent => preflight is skipped
// (unless the availability policy requires it, which routes to a manual step).
type SupplierPreflighter interface {
	Preflight(ctx context.Context, req SupplierOrderRequest) (*SupplierPreflightResult, error)
}

// SupplierStatusReader is OPTIONALLY implemented by adapters that support order status reads.
// Absent => no automatic reconcile poll (an operator advances status via the portal).
type SupplierStatusReader interface {
	GetOrderStatus(ctx context.Context, externalID string) (*SupplierOrderStatus, error)
}
```

- [ ] **Step 4: Run it (PASS)**

Run: `gofmt -w -s internal/integration/supplier.go internal/integration/supplier_test.go && go test ./internal/integration/ -run TestSupplierCapabilitySubInterfaces -count=1`

- [ ] **Step 5: Commit**

```bash
git add internal/integration/supplier.go internal/integration/supplier_test.go
git commit -m "OPE-418: optional supplier adapter sub-interfaces (preflight, status read)"
```

---

## Task 4: Supplier-order service — request builder + canonical status mapping

**Files:** Create `internal/service/supplier_order_service.go`; Test `internal/service/supplier_order_service_test.go`.

This holds the pure, unit-testable pieces: building the `SupplierOrderRequest` from an order + dropship lines (with the missing/ambiguous validation), and mapping a raw supplier status to a canonical one.

- [ ] **Step 1: Write the failing tests**

Create `internal/service/supplier_order_service_test.go`:

```go
package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func TestBuildSupplierOrderRequest_OK(t *testing.T) {
	lines := []model.DropshipOrderItem{{EAN: "590", SupplierSKU: "S1", Quantity: 2}}
	req, missing, ambiguous := BuildSupplierOrderRequest("ORD-1", lines)
	require.Empty(t, missing)
	require.False(t, ambiguous)
	assert.Equal(t, "ORD-1", req.ClientOrderNumber)
	require.Len(t, req.Lines, 1)
	assert.Equal(t, "590", req.Lines[0].EAN)
	assert.Equal(t, float64(2), req.Lines[0].Quantity)
}

func TestBuildSupplierOrderRequest_MissingIdentity(t *testing.T) {
	lines := []model.DropshipOrderItem{{Quantity: 1}} // no EAN and no SKU
	_, missing, _ := BuildSupplierOrderRequest("ORD-1", lines)
	assert.NotEmpty(t, missing, "a line with neither EAN nor SKU is missing required identity")
}

func TestMapSupplierStatus_Canonical(t *testing.T) {
	assert.Equal(t, "confirmed", MapSupplierStatus("ACCEPTED"))
	assert.Equal(t, "shipped", MapSupplierStatus("SHIPPED"))
	assert.Equal(t, "", MapSupplierStatus("weird_unmapped")) // empty => unmapped -> blocker upstream
}
```

(Adjust `model.DropshipOrderItem` field names to the real struct — read `internal/model/dropship.go`.)

- [ ] **Step 2: Run it (FAIL — undefined)**

Run: `go test ./internal/service/ -run 'BuildSupplierOrderRequest|MapSupplierStatus' -count=1`

- [ ] **Step 3: Write the pure builder + mapper**

Create `internal/service/supplier_order_service.go` (the pure helpers; the stateful service struct comes in Task 5):

```go
package service

import (
	"strings"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// BuildSupplierOrderRequest assembles a SupplierOrderRequest from dropship lines. It returns
// the request, the list of lines missing a required identity (EAN or supplier SKU), and
// whether any line is ambiguous (both an EAN and a SKU that disagree is left to the adapter;
// here "ambiguous" is reserved for a future identity-resolution step and returns false for v1).
// A missing-identity result must NOT be submitted — the caller raises supplier_order_missing_data.
func BuildSupplierOrderRequest(clientOrderNumber string, lines []model.DropshipOrderItem) (integration.SupplierOrderRequest, []string, bool) {
	req := integration.SupplierOrderRequest{ClientOrderNumber: clientOrderNumber}
	var missing []string
	for i := range lines {
		ean := strings.TrimSpace(lines[i].EAN)
		sku := strings.TrimSpace(lines[i].SupplierSKU)
		if ean == "" && sku == "" {
			missing = append(missing, lines[i].ID.String())
			continue
		}
		req.Lines = append(req.Lines, integration.SupplierOrderLine{
			EAN:      ean,
			ItemID:   sku,
			Quantity: float64(lines[i].Quantity),
		})
	}
	return req, missing, false
}

// canonicalSupplierStatuses maps common raw supplier statuses to canonical OpenOMS dropship
// statuses. An unmapped raw status returns "" so the caller raises external_status_unmapped.
var canonicalSupplierStatuses = map[string]string{
	"ACCEPTED": "confirmed", "CONFIRMED": "confirmed",
	"PROCESSING": "processing",
	"SHIPPED": "shipped", "SENT": "shipped", "DISPATCHED": "shipped",
	"DELIVERED": "delivered",
	"CANCELLED": "cancelled", "REJECTED": "cancelled",
}

// MapSupplierStatus maps a raw supplier status (case-insensitive) to a canonical status, or
// "" when unmapped. The raw value is preserved separately by the caller.
func MapSupplierStatus(raw string) string {
	return canonicalSupplierStatuses[strings.ToUpper(strings.TrimSpace(raw))]
}
```

- [ ] **Step 4: Run it (PASS)**

Run: `gofmt -w -s internal/service/supplier_order_service.go internal/service/supplier_order_service_test.go && go test ./internal/service/ -run 'BuildSupplierOrderRequest|MapSupplierStatus' -count=1`

- [ ] **Step 5: Commit**

```bash
git add internal/service/supplier_order_service.go internal/service/supplier_order_service_test.go
git commit -m "OPE-418: supplier-order request builder + canonical status mapper (pure)"
```

---

## Task 5: SupplierOrderHandler — prepare → preflight → submit (idempotent)

**Files:** Create `internal/service/supplier_order_handler.go`. (Behavior covered by Task 8.)

Implements `OrchestrationHandler` for the `supplier.order.submit` event. The event payload carries `{order_id, supplier_id, unit_id}`. The handler loads the order + dropship lines, builds the request, runs the three phases recording a step at each, and is idempotent (an existing `dropship_orders.supplier_reference` for the order+supplier → skip submit).

- [ ] **Step 1: Write the handler**

Create `internal/service/supplier_order_handler.go`:

```go
package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// EventSupplierOrderSubmit is the orchestration outbox event type that drives a dropship
// unit through prepare -> preflight -> submit.
const EventSupplierOrderSubmit = "supplier.order.submit"

// supplierOrderDeps is the narrow set of collaborators the handler needs (kept as an
// interface-free struct of the real services; wired in main.go).
type SupplierOrderHandler struct {
	pool         *pgxpool.Pool
	fulfillment  *FulfillmentService
	dropshipRepo dropshipOrderWriter        // FindByOrderID + Create/UpdateFields
	orderRepo    orderReader                // FindByID
	itemsFor     func(ctx context.Context, tx pgx.Tx, orderID, supplierID uuid.UUID) ([]model.DropshipOrderItem, error)
	newProvider  func(ctx context.Context, tx pgx.Tx, supplierID uuid.UUID) (integration.SupplierProvider, error)
}

// Handle runs the three synchronous phases. Errors that cannot be retried create a typed
// blocker via the fulfillment service and return nil (the unit waits); transport errors
// return a plain error (retryable by the worker). The payload is {order_id, supplier_id, unit_id}.
func (h *SupplierOrderHandler) Handle(ctx context.Context, event model.OrchestrationOutboxEvent) error {
	p, _ := event.Payload.(map[string]any)
	orderID := uuid.MustParse(fmt.Sprint(p["order_id"]))
	supplierID := uuid.MustParse(fmt.Sprint(p["supplier_id"]))
	unitID := uuid.MustParse(fmt.Sprint(p["unit_id"]))
	tenantID := event.TenantID

	return database.WithTenant(ctx, h.pool, tenantID, func(tx pgx.Tx) error {
		// Idempotency: already submitted? (a dropship_orders row with a supplier_reference)
		existing, _ := h.dropshipRepo.FindByOrderID(ctx, tx, orderID)
		for i := range existing {
			if existing[i].SupplierID == supplierID && strings.TrimSpace(existing[i].SupplierReference) != "" {
				return nil // already placed — idempotent no-op
			}
		}

		// PREPARE
		lines, err := h.itemsFor(ctx, tx, orderID, supplierID)
		if err != nil {
			return fmt.Errorf("load dropship lines: %w", err) // retryable
		}
		ord, err := h.orderRepo.FindByID(ctx, tx, orderID)
		if err != nil {
			return fmt.Errorf("load order: %w", err) // retryable
		}
		req, missing, ambiguous := BuildSupplierOrderRequest(ord.OrderNumber, lines)
		if ambiguous {
			h.blocker(ctx, tx, tenantID, orderID, unitID, model.BlockerSupplierOrderAmbiguousSKU, "ambiguous SKU/EAN identity")
			return nil
		}
		if len(missing) > 0 {
			h.blocker(ctx, tx, tenantID, orderID, unitID, model.BlockerSupplierOrderMissingData,
				fmt.Sprintf("%d line(s) missing EAN/SKU", len(missing)))
			return nil
		}

		provider, err := h.newProvider(ctx, tx, supplierID)
		if err != nil {
			return fmt.Errorf("build supplier provider: %w", err) // retryable (transient)
		}

		// PREFLIGHT (optional capability)
		h.fulfillment.RecordStep(ctx, tenantID, unitID, model.StepPreflightSupplierOrder, model.FulfillmentStatusRunning, nil)
		if pf, ok := provider.(integration.SupplierPreflighter); ok {
			res, perr := pf.Preflight(ctx, req)
			if perr != nil {
				return fmt.Errorf("supplier preflight: %w", perr) // retryable transport
			}
			if !res.Accepted {
				code := model.BlockerSupplierOrderRejected
				if len(res.MissingFields) > 0 {
					code = model.BlockerSupplierOrderMissingData
				}
				h.blocker(ctx, tx, tenantID, orderID, unitID, code, "supplier preflight rejected")
				return nil
			}
			if res.PaymentDue {
				h.blocker(ctx, tx, tenantID, orderID, unitID, model.BlockerSupplierPaymentAwaiting, "supplier order awaiting payment")
				return nil
			}
			if len(res.SplitLines) > 0 {
				h.blocker(ctx, tx, tenantID, orderID, unitID, model.BlockerSupplierPartialFulfillment, "supplier split requires operator review")
				return nil
			}
		}
		h.fulfillment.RecordStep(ctx, tenantID, unitID, model.StepPreflightSupplierOrder, model.FulfillmentStatusSucceeded, nil)

		// SUBMIT
		h.fulfillment.RecordStep(ctx, tenantID, unitID, model.StepCreateDropshipOrder, model.FulfillmentStatusRunning, nil)
		result, serr := provider.CreateOrder(ctx, req)
		if serr != nil {
			// A business rejection is permanent; transport is retryable. The adapter should
			// wrap business errors as model.Permanent; otherwise treat as retryable.
			if model.IsPermanent(serr) {
				h.blocker(ctx, tx, tenantID, orderID, unitID, model.BlockerSupplierOrderRejected, serr.Error())
				return nil
			}
			return fmt.Errorf("supplier create order: %w", serr) // retryable
		}
		// Persist the supplier order id on a dropship_orders row + record the attempt exec id.
		if e := h.recordSubmitted(ctx, tx, tenantID, orderID, supplierID, result); e != nil {
			return e
		}
		_ = h.fulfillment // SetLatestAttemptExternalExecID is recorded by the worker via the attempt
		h.fulfillment.RecordStep(ctx, tenantID, unitID, model.StepCreateDropshipOrder, model.FulfillmentStatusSucceeded,
			map[string]any{"supplier_reference": result.ExternalOrderID})
		h.fulfillment.RecordUnitTransition(ctx, tenantID, unitID, model.FulfillmentStatusWaitingExternal)
		return nil
	})
}

// blocker records the blocked step + unit transition + a typed supplier blocker (best-effort).
func (h *SupplierOrderHandler) blocker(ctx context.Context, tx pgx.Tx, tenantID, orderID, unitID uuid.UUID, code, desc string) {
	h.fulfillment.RecordStep(ctx, tenantID, unitID, model.StepCreateDropshipOrder, model.FulfillmentStatusBlocked, map[string]any{"reason": code})
	h.fulfillment.RecordUnitTransition(ctx, tenantID, unitID, model.FulfillmentStatusBlocked)
	h.fulfillment.CreateSupplierBlocker(ctx, tenantID, orderID, &unitID, code, desc)
}
```

The handler references `dropshipOrderWriter`, `orderReader`, and `recordSubmitted` — defined next.

- [ ] **Step 2: Add the narrow collaborator interfaces + recordSubmitted**

Add to `supplier_order_handler.go`:

```go
type dropshipOrderWriter interface {
	FindByOrderID(ctx context.Context, tx pgx.Tx, orderID uuid.UUID) ([]model.DropshipOrder, error)
	Create(ctx context.Context, tx pgx.Tx, d *model.DropshipOrder) error
}
type orderReader interface {
	FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Order, error)
}

// recordSubmitted creates (or completes) the dropship_orders row carrying the supplier order id.
func (h *SupplierOrderHandler) recordSubmitted(ctx context.Context, tx pgx.Tx, tenantID, orderID, supplierID uuid.UUID, result *integration.SupplierOrderResult) error {
	d := &model.DropshipOrder{
		TenantID:          tenantID,
		OrderID:           orderID,
		SupplierID:        supplierID,
		Status:            "sent",
		SupplierReference: result.ExternalOrderID,
	}
	if err := h.dropshipRepo.Create(ctx, tx, d); err != nil {
		return fmt.Errorf("record dropship order: %w", err)
	}
	return nil
}
```

(Reconcile every field/method name against the REAL `model.DropshipOrder`, `model.Order` (`OrderNumber`?), `model.DropshipOrderItem`, and `model.IsPermanent`/`model.Permanent`. If `DropshipOrder.Create` needs `supplier_name`/`currency`, set sensible values. If the order's human number field is named differently, use the real one.)

- [ ] **Step 3: Build** — `gofmt -w -s internal/service/supplier_order_handler.go && go build ./...`

- [ ] **Step 4: Commit**

```bash
git add internal/service/supplier_order_handler.go
git commit -m "OPE-418: SupplierOrderHandler (prepare/preflight/submit, idempotent, typed blockers)"
```

---

## Task 6: Gate hook — enqueue submit (api) / manual blocker (portal/manual)

**Files:** Modify `internal/service/dropship_service.go`.

Extend the existing `gateDropshipAvailability` caller / the `switch capability` in the dropship-unit recorder. Inject an optional enqueuer + the flag (a `SetSupplierOrder(svc)` setter mirroring `SetAvailabilityService`).

- [ ] **Step 1: Modify the `SupplierCapAPI` + `SupplierCapPortal/Manual` branches**

In the `case SupplierCapAPI:` branch, after the availability gate passes (the `RecordStep(... StepCreateDropshipOrder, Ready ...)` line), add:
```go
		// OPE-418/Phase-7: when the supplier-order engine is enabled, enqueue the submit
		// event so the SupplierOrderHandler runs prepare->preflight->submit. Off by default
		// -> the step stays "ready" with no auto-submit (current behavior).
		if s.supplierOrder != nil && s.supplierOrder.Enabled() {
			s.supplierOrder.EnqueueSubmit(ctx, tenantID, orderID, supplierID, unit.ID)
		}
```
In the `case SupplierCapPortal, SupplierCapManual:` branch, after the waiting step, add a typed blocker so the operator sees an actionable item:
```go
		if s.supplierOrder != nil && s.supplierOrder.Enabled() {
			s.fulfillment.CreateSupplierBlocker(ctx, tenantID, orderID, &unit.ID,
				model.BlockerSupplierManualSubmissionRequired,
				fmt.Sprintf("supplier %q requires manual order submission (capability %s)", supplierName, capability))
		}
```

Add the field + setter to `DropshipService` (mirror `availability`):
```go
	supplierOrder *SupplierOrderService
```
```go
// SetSupplierOrderService wires the gated supplier-order engine (OPE-418/Phase-7).
func (s *DropshipService) SetSupplierOrderService(svc *SupplierOrderService) { s.supplierOrder = svc }
```

- [ ] **Step 2: Add `Enabled` + `EnqueueSubmit` to `SupplierOrderService`** (in `supplier_order_service.go`)

```go
// SupplierOrderService is the gated entry point: it enqueues the submit event. Stateful
// collaborators (pool + orchestration repo + fulfillment) are injected in main.go.
type SupplierOrderService struct {
	enabled       bool
	pool          *pgxpool.Pool
	fulfillment   *FulfillmentService
	orchestration *repository.OrchestrationRepository
}

func NewSupplierOrderService(enabled bool, pool *pgxpool.Pool, fulfillment *FulfillmentService, orchestration *repository.OrchestrationRepository) *SupplierOrderService {
	return &SupplierOrderService{enabled: enabled, pool: pool, fulfillment: fulfillment, orchestration: orchestration}
}

// Enabled reports whether the engine is active. Nil-safe.
func (s *SupplierOrderService) Enabled() bool { return s != nil && s.enabled }

// EnqueueSubmit enqueues a supplier.order.submit event for a routable API dropship unit,
// inside the caller's tenant tx. Idempotent on the (order, supplier) idempotency key. The
// dropship unit's process already exists (the gate ran EnsureUnit), so a process is present.
func (s *SupplierOrderService) EnqueueSubmit(ctx context.Context, tenantID, orderID, supplierID, unitID uuid.UUID) {
	if !s.Enabled() {
		return
	}
	proc, err := s.fulfillment.EnsureProcessForOrderUnconditional(ctx, currentTx(ctx), tenantID, orderID)
	// NOTE: the gate calls this inside its own WithTenant tx; thread that tx in rather than
	// opening a new one. Implement EnqueueSubmit to take a pgx.Tx param (the gate has one).
	_ = proc
	_ = err
}
```

**Implementation note:** the gate already runs inside a tenant tx where it calls `EnsureUnit`. Change `EnqueueSubmit` to take the gate's `pgx.Tx` (signature `EnqueueSubmit(ctx, tx, tenantID, orderID, supplierID, unitID)`), look up the unit's process (the unit row has `process_id`), and call `orchestration.EnqueueEvent` with `EventType: EventSupplierOrderSubmit`, `IdempotencyKey: EventSupplierOrderSubmit + ":" + orderID + ":" + supplierID`, `Payload: {order_id, supplier_id, unit_id}`, `ProcessID: <unit.process_id>`. (The unit's ProcessID is available from the `unit` the gate already holds — pass `unit.ProcessID` in.) Adjust the call site in Task 6 Step 1 to pass `tx` + `unit.ProcessID`.

- [ ] **Step 3: Build** — `gofmt -w -s internal/service/dropship_service.go internal/service/supplier_order_service.go && go build ./...`

- [ ] **Step 4: Commit**

```bash
git add internal/service/dropship_service.go internal/service/supplier_order_service.go
git commit -m "OPE-418: dropship gate enqueues supplier.order.submit (api) + manual blocker (portal/manual), gated"
```

---

## Task 7: SupplierOrderStatusPoller — reconcile status/tracking

**Files:** Create `internal/worker/supplier_order_status_poller.go`.

Mirror `TrackingPoller`: a `Worker` (Name/Interval/Run) that, gated, iterates tenants, finds `dropship_orders` in non-terminal states with a `supplier_reference`, builds the supplier provider, and — if it implements `SupplierStatusReader` — polls status, maps it, updates the dropship row + records the reconcile step.

- [ ] **Step 1: Write the poller**

Create `internal/worker/supplier_order_status_poller.go`:

```go
package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

const supplierOrderPollInterval = 5 * time.Minute

// SupplierOrderStatusPoller reconciles supplier order status/tracking into dropship_orders
// for adapters that support status reads (OPE-418/Phase-7). Gated: a no-op when disabled.
type SupplierOrderStatusPoller struct {
	pool        *pgxpool.Pool // privileged worker pool
	enabled     bool
	fulfillment *service.FulfillmentService
	newProvider func(ctx context.Context, tx pgx.Tx, supplierID uuid.UUID) (integration.SupplierProvider, error)
	logger      *slog.Logger
}

func NewSupplierOrderStatusPoller(pool *pgxpool.Pool, enabled bool, fulfillment *service.FulfillmentService,
	newProvider func(ctx context.Context, tx pgx.Tx, supplierID uuid.UUID) (integration.SupplierProvider, error), logger *slog.Logger) *SupplierOrderStatusPoller {
	if logger == nil {
		logger = slog.Default()
	}
	return &SupplierOrderStatusPoller{pool: pool, enabled: enabled, fulfillment: fulfillment, newProvider: newProvider, logger: logger}
}

func (w *SupplierOrderStatusPoller) Name() string            { return "supplier_order_status_poller" }
func (w *SupplierOrderStatusPoller) Interval() time.Duration { return supplierOrderPollInterval }

// Run reconciles one batch of non-terminal supplier orders per tenant. No-op when disabled.
func (w *SupplierOrderStatusPoller) Run(ctx context.Context) error {
	if !w.enabled {
		return nil
	}
	// Cross-tenant tenant list on the privileged pool, then per-tenant WithTenant (mirrors
	// the other cross-tenant workers).
	rows, err := w.pool.Query(ctx, "SELECT id FROM tenants")
	if err != nil {
		return err
	}
	var tenantIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if rows.Scan(&id) == nil {
			tenantIDs = append(tenantIDs, id)
		}
	}
	rows.Close()

	for _, tenantID := range tenantIDs {
		if err := checkWorkerContext(ctx); err != nil {
			return err
		}
		_ = database.WithTenant(ctx, w.pool, tenantID, func(tx pgx.Tx) error {
			return w.reconcileTenant(ctx, tx, tenantID)
		})
	}
	return nil
}

// reconcileTenant polls status for the tenant's non-terminal supplier orders. Each row's
// canonical status is updated; an unmapped raw status raises external_status_unmapped.
func (w *SupplierOrderStatusPoller) reconcileTenant(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) error {
	// SELECT id, order_id, supplier_id, supplier_reference FROM dropship_orders
	//   WHERE supplier_reference <> '' AND status NOT IN ('delivered','cancelled')
	// For each: provider := newProvider(supplierID); if reader, ok := provider.(SupplierStatusReader) {
	//   st := reader.GetOrderStatus(ref); canon := service.MapSupplierStatus(st.RawStatus)
	//   if canon == "" -> CreateSupplierBlocker(external_status_unmapped) ; continue
	//   UPDATE dropship_orders SET status=canon, tracking_number=..., carrier=... WHERE id
	//   fulfillment.RecordStep(StepConfirmSupplierOrder, succeeded); transition the unit toward the canon }
	// (Write the concrete SQL + loop here, mirroring TrackingPoller's structure + dropshipRepo.)
	return nil
}
```

(Flesh out `reconcileTenant` with the concrete query + provider loop per the comment, reusing `service.MapSupplierStatus`, the dropship repo's update method (`UpdateFields`), and `fulfillment.RecordStep`/`RecordUnitTransition`/`CreateSupplierBlocker`. Mirror `TrackingPoller.Run`'s real structure for listing + provider construction.)

- [ ] **Step 2: Build** — `gofmt -w -s internal/worker/supplier_order_status_poller.go && go build ./...`

- [ ] **Step 3: Commit**

```bash
git add internal/worker/supplier_order_status_poller.go
git commit -m "OPE-418: SupplierOrderStatusPoller (reconcile supplier status/tracking, gated)"
```

---

## Task 8: Wire it in main.go

**Files:** Modify `cmd/server/main.go`.

- [ ] **Step 1: Construct + wire (gated)**

After `fulfillmentService` + `orchestrationRepo` are built:
```go
	supplierOrderService := service.NewSupplierOrderService(
		cfg.SupplierOrderEnabled, pool, fulfillmentService, orchestrationRepo)
	dropshipService.SetSupplierOrderService(supplierOrderService)
```
Define a `newSupplierProvider(ctx, tx, supplierID)` closure (load the supplier + its integration credentials + `integration.NewSupplierProvider(provider, creds, settings)` — mirror how `supplierCapability` / the sync worker build a provider). Register the dispatcher handler + the poller inside the `ORCHESTRATION_WORKER_ENABLED` block, additionally gated on `SupplierOrderEnabled`:
```go
	if cfg.SupplierOrderEnabled {
		orchestrationDispatcher.Register(service.EventSupplierOrderSubmit,
			service.NewSupplierOrderHandler(pool, fulfillmentService, dropshipOrderRepo, orderRepo, dropshipItemsLoader, newSupplierProvider))
		workerMgr.Register(worker.NewSupplierOrderStatusPoller(workerPool, cfg.SupplierOrderEnabled, fulfillmentService, newSupplierProvider, slog.Default()))
	}
```
Add a `NewSupplierOrderHandler(...)` constructor in the service that fills the struct fields. `dropshipItemsLoader` is the `itemsFor` closure (load dropship items for an order+supplier from the dropship item repo).

- [ ] **Step 2: Build + vet + full unit tests**

Run: `gofmt -w -s cmd/server/main.go && go build ./... && go vet ./... && go test ./... 2>&1 | grep -v "^ok\|no test files" | tail`
Expected: clean; no failing packages.

- [ ] **Step 3: Commit**

```bash
git add cmd/server/main.go internal/service/supplier_order_handler.go internal/service/supplier_order_service.go
git commit -m "OPE-418: wire supplier-order handler + status poller (gated)"
```

---

## Task 9: DB-bound integration tests

**Files:** Create `tests/integration/supplier_order_test.go` (`//go:build integration`).

Mirror the harness; seed a tenant + order + supplier (+ a fake supplier provider registered via `integration.RegisterSupplierProvider` for the test, implementing CreateOrder and optionally the sub-interfaces). Cover:
1. **API submit** — capability api + routable → handler creates a `dropship_orders` row with `supplier_reference` + the create_dropship_order step succeeded + the unit waiting_external.
2. **Missing data** — a dropship line with no EAN/SKU → `supplier_order_missing_data` blocker, no dropship row created.
3. **Manual branch** — a portal/manual supplier → the gate records the waiting step + `supplier_manual_submission_required` blocker, no submit.
4. **Idempotent re-submit** — running the handler twice for the same order+supplier → exactly one dropship_orders row (the second is a no-op).
5. **Reconcile** — seed a dropship row with a supplier_reference + a fake status reader returning "SHIPPED" → the poller updates status to "shipped" + records the confirm step; an unmapped raw status → `external_status_unmapped` blocker.
6. **Cross-tenant RLS** — dropship/units of tenant A invisible to tenant B.
7. **Flag-off** — with the service disabled, the gate's api branch does not enqueue and the handler/poller are no-ops.

- [ ] **Step 1: Write the tests** (concrete code mirroring the harness + a registered fake provider).
- [ ] **Step 2: Run**

```bash
DATABASE_URL="postgres://openoms:openoms-dev-password@127.0.0.1:5433/openoms?sslmode=disable" \
go test -tags integration ./tests/integration/ -run 'SupplierOrder' -count=1
```
Expected: PASS (adjust seed columns to the real schema; keep assertions).

- [ ] **Step 3: Commit**

```bash
git add tests/integration/supplier_order_test.go
git commit -m "OPE-418: supplier-order integration tests (submit, missing-data, manual, idempotency, reconcile, RLS, flag-off)"
```

---

## Task 10: Full validation sweep

- [ ] **Step 1:** `test -z "$(gofmt -l .)" && echo clean; go build ./... && go vet ./... && go vet -tags integration ./tests/integration/ && go test ./... 2>&1 | grep -v "^ok\|no test files" | tail`
- [ ] **Step 2:** `/tmp/glci29/golangci-lint run --new-from-rev=main --timeout=5m` → `0 issues`.
- [ ] **Step 3:** `DATABASE_URL=... go test -tags integration ./tests/integration/ -count=1 | tail -3` → `ok` (no regression).
- [ ] **Step 4:** Flag-off sanity — confirm by reading the diff that `SupplierOrderService.Enabled()` gates the gate enqueue + the manual blocker, and the handler + poller are only registered under `cfg.SupplierOrderEnabled`. Default build unchanged.

---

## Self-Review (completed by plan author)

- **Spec coverage:** prepare/preflight/submit (Task 5) ✓; reconcile poller (Task 7) ✓; capability branch api/portal/manual/unsupported (Task 6, reusing the existing gate's unsupported branch) ✓; optional adapter sub-interfaces (Task 3) ✓; 6 new blockers + reuse (Task 1, Task 5) ✓; idempotency — supplier_reference guard + outbox key (Tasks 5,6) ✓; storage in dropship_orders (Task 5) ✓; gated SUPPLIER_ORDER_ENABLED (Task 2 + gated registration) ✓; test matrix (Tasks 4,9,10) ✓; no migration (blocker codes app-validated, dropship_orders sufficient) ✓.
- **Type consistency:** `EventSupplierOrderSubmit`, `SupplierOrderService.{Enabled,EnqueueSubmit}`, `SupplierOrderHandler`, `BuildSupplierOrderRequest`, `MapSupplierStatus`, `SupplierPreflighter`/`SupplierStatusReader`/`SupplierPreflightResult`/`SupplierOrderStatus`, the 6 `BlockerSupplier*` codes, `SupplierOrderStatusPoller` used consistently.
- **Executor note (important):** this plan touches several real structs whose exact fields/methods MUST be read and reconciled before coding — `model.DropshipOrder` (does it have `SupplierReference`? the SQL column is `supplier_reference`), `model.DropshipOrderItem` (EAN/SupplierSKU/Quantity field names), `model.Order` (the human order-number field), `model.IsPermanent`/`model.Permanent`, `EnsureProcessForOrderUnconditional` (added in the ext-workflow connector — confirm it exists on main), `DropshipOrderRepo.UpdateFields`, how `supplierCapability` builds a provider (for `newSupplierProvider`). Where a name differs, use the real one — the plan's code is a guide, not gospel.
- **EnqueueSubmit tx threading:** the gate calls EnqueueSubmit INSIDE its own WithTenant tx — implement EnqueueSubmit to TAKE that `pgx.Tx` + the unit's `ProcessID` (do not open a second tx). Task 6 Step 2 flags this.

## Deferred (per spec)
Per-vendor adapters beyond btp; multi-document split execution (v1 → `supplier_partial_fulfillment` blocker); purchase-order/backorder submit; operator submission UI beyond the existing portal; a richer per-provider status-mapping table (v1 uses the in-code `MapSupplierStatus`).
