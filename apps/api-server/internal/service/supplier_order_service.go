package service

import (
	"context"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// SupplierOrderInputLine is one resolved dropship line for the supplier-order request
// builder (OPE-418/Phase-7). The prepare phase resolves each dropship item's identity
// (its product's EAN + the supplier SKU) into this slim, pure-testable shape before the
// request is assembled — keeping BuildSupplierOrderRequest dependency-free.
type SupplierOrderInputLine struct {
	LineID      uuid.UUID // the dropship item id, for missing-identity reporting
	EAN         string
	SupplierSKU string
	Quantity    int
}

// BuildSupplierOrderRequest assembles a SupplierOrderRequest from resolved dropship lines.
// It returns the request, the list of line ids missing a required identity (neither EAN nor
// supplier SKU), and whether any line is ambiguous. "Ambiguous" is reserved for a future
// identity-resolution step and returns false for v1. A missing-identity result must NOT be
// submitted — the caller raises supplier_order_missing_data.
func BuildSupplierOrderRequest(clientOrderNumber string, lines []SupplierOrderInputLine) (integration.SupplierOrderRequest, []string, bool) {
	req := integration.SupplierOrderRequest{ClientOrderNumber: clientOrderNumber}
	var missing []string
	for i := range lines {
		ean := strings.TrimSpace(lines[i].EAN)
		sku := strings.TrimSpace(lines[i].SupplierSKU)
		if ean == "" && sku == "" {
			missing = append(missing, lines[i].LineID.String())
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
// statuses. Every value MUST be a valid dropship_orders status (pending/sent/confirmed/
// shipped/delivered/cancelled — see model.UpdateDropshipStatusRequest.Validate); the
// reconcile poller writes these directly. An unmapped raw status returns "" so the caller
// raises external_status_unmapped. "processing"/"in progress" map to the nearest valid
// state, confirmed (the supplier has accepted and is working the order).
var canonicalSupplierStatuses = map[string]string{
	"ACCEPTED": "confirmed", "CONFIRMED": "confirmed",
	"PROCESSING": "confirmed", "IN_PROGRESS": "confirmed",
	"SHIPPED": "shipped", "SENT": "shipped", "DISPATCHED": "shipped",
	"DELIVERED": "delivered",
	"CANCELLED": "cancelled", "REJECTED": "cancelled",
}

// MapSupplierStatus maps a raw supplier status (case-insensitive) to a canonical status, or
// "" when unmapped. The raw value is preserved separately by the caller.
func MapSupplierStatus(raw string) string {
	return canonicalSupplierStatuses[strings.ToUpper(strings.TrimSpace(raw))]
}

// SupplierOrderService is the gated entry point (OPE-418/Phase-7): it enqueues the
// supplier.order.submit event for API-capable routable dropship units. When disabled it is a
// no-op (Enabled() is false, EnqueueSubmit returns early), so the dropship gate's API branch
// keeps its current behavior. The enqueue is BEST-EFFORT (logs-and-continues on error) so it
// can never break or roll back the dropship routing it is hooked into.
type SupplierOrderService struct {
	enabled       bool
	pool          *pgxpool.Pool
	fulfillment   *FulfillmentService
	orchestration *repository.OrchestrationRepository
}

// NewSupplierOrderService constructs the gated entry point. enabled comes from
// cfg.SupplierOrderEnabled; pool is the RLS-scoped app pool.
func NewSupplierOrderService(enabled bool, pool *pgxpool.Pool, fulfillment *FulfillmentService, orchestration *repository.OrchestrationRepository) *SupplierOrderService {
	return &SupplierOrderService{enabled: enabled, pool: pool, fulfillment: fulfillment, orchestration: orchestration}
}

// Enabled reports whether the supplier-order engine is active. Nil-safe.
func (s *SupplierOrderService) Enabled() bool { return s != nil && s.enabled }

// EnqueueSubmit enqueues a supplier.order.submit event for a routable API dropship unit. It
// anchors the outbox row to the dropship unit's existing fulfillment process (processID,
// already created by the gate's EnsureUnit) and dedups on the (order, supplier) idempotency
// key, so a re-route never double-enqueues. BEST-EFFORT: a no-op when disabled, and on any
// error it logs and returns without affecting the caller's dropship routing.
func (s *SupplierOrderService) EnqueueSubmit(ctx context.Context, tenantID, orderID, supplierID, unitID, processID uuid.UUID) {
	if !s.Enabled() || s.pool == nil || s.orchestration == nil {
		return
	}
	if processID == uuid.Nil {
		slog.Warn("supplier order: enqueue submit skipped, unit has no process (best-effort)",
			"tenant_id", tenantID, "order_id", orderID, "supplier_id", supplierID)
		return
	}
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		_, _, eerr := s.orchestration.EnqueueEvent(ctx, tx, model.OrchestrationOutboxEvent{
			TenantID:       tenantID,
			ProcessID:      processID,
			UnitID:         &unitID,
			EventType:      EventSupplierOrderSubmit,
			IdempotencyKey: EventSupplierOrderSubmit + ":" + orderID.String() + ":" + supplierID.String(),
			Payload: map[string]any{
				"order_id":    orderID.String(),
				"supplier_id": supplierID.String(),
				"unit_id":     unitID.String(),
			},
		})
		return eerr
	})
	if err != nil {
		slog.Warn("supplier order: enqueue submit failed (best-effort, ignored)",
			"tenant_id", tenantID, "order_id", orderID, "supplier_id", supplierID, "error", err)
	}
}
