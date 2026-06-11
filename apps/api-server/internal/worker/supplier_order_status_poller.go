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
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

const supplierOrderPollInterval = 5 * time.Minute

// SupplierOrderFulfillmentRecorder is the narrow surface the poller uses to record the
// reconcile step + unit transition + typed blockers without importing the service package's
// concrete type beyond what is needed. It is satisfied by *service.FulfillmentService. All
// implementations are gated + best-effort: a nil/disabled implementation makes calls no-ops.
type SupplierOrderFulfillmentRecorder interface {
	Enabled() bool
	EnsureUnit(ctx context.Context, tenantID, orderID uuid.UUID, unitType, dedupeKey string, metadata map[string]any) *model.FulfillmentUnit
	RecordStep(ctx context.Context, tenantID, unitID uuid.UUID, stepKey, status string, metadata map[string]any) *model.FulfillmentStep
	RecordUnitTransition(ctx context.Context, tenantID, unitID uuid.UUID, status string)
	CreateSupplierBlocker(ctx context.Context, tenantID, orderID uuid.UUID, unitID *uuid.UUID, code, description string) *model.FulfillmentBlocker
	AggregateMixedOrder(ctx context.Context, tenantID, orderID uuid.UUID) *model.FulfillmentProcess
}

// SupplierProviderFactory builds a SupplierProvider for a supplier inside a tenant tx (loads
// the supplier + decrypts its integration credentials). Mirrors the supplier-order handler's
// factory. Returns a nil provider with a nil error when no API provider is resolvable.
type SupplierProviderFactory func(ctx context.Context, tx pgx.Tx, tenantID, supplierID uuid.UUID) (integration.SupplierProvider, error)

// SupplierOrderStatusPoller reconciles supplier order status/tracking into dropship_orders
// for adapters that implement integration.SupplierStatusReader (OPE-418/Phase-7). Gated: a
// complete no-op when disabled. It mirrors TrackingPoller — a recurring worker that lists
// tenants on the privileged pool and reconciles each tenant under WithTenant so RLS applies.
type SupplierOrderStatusPoller struct {
	pool         *pgxpool.Pool // privileged worker pool
	enabled      bool
	fulfillment  SupplierOrderFulfillmentRecorder
	dropshipRepo repository.DropshipOrderRepo
	newProvider  SupplierProviderFactory
	logger       *slog.Logger
}

// NewSupplierOrderStatusPoller constructs the reconcile poller (OPE-418/Phase-7).
func NewSupplierOrderStatusPoller(pool *pgxpool.Pool, enabled bool, fulfillment SupplierOrderFulfillmentRecorder,
	dropshipRepo repository.DropshipOrderRepo, newProvider SupplierProviderFactory, logger *slog.Logger) *SupplierOrderStatusPoller {
	if logger == nil {
		logger = slog.Default()
	}
	return &SupplierOrderStatusPoller{
		pool:         pool,
		enabled:      enabled,
		fulfillment:  fulfillment,
		dropshipRepo: dropshipRepo,
		newProvider:  newProvider,
		logger:       logger,
	}
}

// Name returns the worker identifier.
func (w *SupplierOrderStatusPoller) Name() string { return "supplier_order_status_poller" }

// Interval returns how frequently the worker should run.
func (w *SupplierOrderStatusPoller) Interval() time.Duration { return supplierOrderPollInterval }

// Run reconciles one batch of non-terminal supplier orders per tenant. No-op when disabled.
func (w *SupplierOrderStatusPoller) Run(ctx context.Context) error {
	if !w.enabled {
		return nil
	}
	// Cross-tenant tenant list on the privileged pool, then per-tenant WithTenant (mirrors
	// the other cross-tenant workers + the orchestration worker).
	tenantIDs, err := listAllTenantIDs(ctx, w.pool, w.logger)
	if err != nil {
		return err
	}

	for _, tenantID := range tenantIDs {
		if cerr := checkWorkerContext(ctx); cerr != nil {
			return cerr
		}
		if rerr := database.WithTenant(ctx, w.pool, tenantID, func(tx pgx.Tx) error {
			return w.reconcileTenant(ctx, tx, tenantID)
		}); rerr != nil {
			w.logger.Error("supplier order poller: tenant reconcile failed",
				"tenant_id", tenantID, "error", rerr)
		}
	}
	return nil
}

// reconcilableOrder is one non-terminal supplier order to poll.
type reconcilableOrder struct {
	id         uuid.UUID
	orderID    uuid.UUID
	supplierID uuid.UUID
	reference  string
	status     string
}

// reconcileTenant polls status for the tenant's non-terminal supplier orders. Each row's
// canonical status is updated; an unmapped raw status raises external_status_unmapped.
func (w *SupplierOrderStatusPoller) reconcileTenant(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) error {
	rows, err := tx.Query(ctx,
		`SELECT id, order_id, supplier_id, supplier_reference, status
		   FROM dropship_orders
		  WHERE supplier_reference IS NOT NULL AND supplier_reference <> ''
		    AND status NOT IN ('delivered', 'cancelled')`)
	if err != nil {
		return err
	}
	var orders []reconcilableOrder
	for rows.Next() {
		var o reconcilableOrder
		var ref *string
		if scanErr := rows.Scan(&o.id, &o.orderID, &o.supplierID, &ref, &o.status); scanErr != nil {
			rows.Close()
			return scanErr
		}
		if ref != nil {
			o.reference = *ref
		}
		orders = append(orders, o)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for i := range orders {
		if cerr := checkWorkerContext(ctx); cerr != nil {
			return cerr
		}
		w.reconcileOne(ctx, tx, tenantID, orders[i])
	}
	return nil
}

// reconcileOne polls one supplier order's status and reconciles it into dropship_orders + the
// fulfillment unit. Best-effort: any per-order error is logged and skipped so one bad order
// never aborts the tenant batch.
func (w *SupplierOrderStatusPoller) reconcileOne(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, o reconcilableOrder) {
	provider, err := w.newProvider(ctx, tx, tenantID, o.supplierID)
	if err != nil || provider == nil {
		if err != nil {
			w.logger.Warn("supplier order poller: build provider failed",
				"tenant_id", tenantID, "supplier_id", o.supplierID, "error", err)
		}
		return
	}
	reader, ok := provider.(integration.SupplierStatusReader)
	if !ok {
		return // adapter has no status read — operator advances status via the portal
	}
	st, err := reader.GetOrderStatus(ctx, o.reference)
	if err != nil {
		w.logger.Warn("supplier order poller: get status failed",
			"tenant_id", tenantID, "dropship_order_id", o.id, "reference", o.reference, "error", err)
		return
	}
	if st == nil {
		return
	}

	unit := w.fulfillment.EnsureUnit(ctx, tenantID, o.orderID, model.UnitTypeDropship, o.supplierID.String(),
		map[string]any{"supplier_id": o.supplierID.String()})

	canon := service.MapSupplierStatus(st.RawStatus)
	if canon == "" {
		// Unmapped external status -> typed blocker, raw preserved in the description.
		var unitID *uuid.UUID
		if unit != nil {
			unitID = &unit.ID
		}
		w.fulfillment.CreateSupplierBlocker(ctx, tenantID, o.orderID, unitID,
			model.BlockerExternalStatusUnmapped,
			"unmapped supplier status: "+st.RawStatus)
		w.fulfillment.AggregateMixedOrder(ctx, tenantID, o.orderID)
		return
	}
	if canon == o.status {
		return // no change
	}

	upd := model.UpdateDropshipStatusRequest{Status: canon}
	if st.TrackingNumber != "" {
		upd.TrackingNumber = &st.TrackingNumber
	}
	if st.Carrier != "" {
		upd.Carrier = &st.Carrier
	}
	if err := w.dropshipRepo.UpdateFields(ctx, tx, o.id, upd); err != nil {
		w.logger.Error("supplier order poller: update dropship fields failed",
			"tenant_id", tenantID, "dropship_order_id", o.id, "error", err)
		return
	}

	// Record the reconcile step + advance the unit toward the canonical status.
	if unit != nil {
		w.fulfillment.RecordStep(ctx, tenantID, unit.ID, model.StepConfirmSupplierOrder, model.FulfillmentStatusSucceeded,
			map[string]any{"supplier_id": o.supplierID.String(), "canonical_status": canon, "raw_status": st.RawStatus})
		switch canon {
		case "shipped", "delivered":
			w.fulfillment.RecordUnitTransition(ctx, tenantID, unit.ID, model.FulfillmentStatusSucceeded)
		case "cancelled":
			w.fulfillment.RecordUnitTransition(ctx, tenantID, unit.ID, model.FulfillmentStatusCancelled)
		default: // confirmed, processing
			w.fulfillment.RecordUnitTransition(ctx, tenantID, unit.ID, model.FulfillmentStatusRunning)
		}
		w.fulfillment.AggregateMixedOrder(ctx, tenantID, o.orderID)
	}

	w.logger.Info("supplier order poller: status reconciled",
		"tenant_id", tenantID, "dropship_order_id", o.id, "from", o.status, "to", canon)
}

// Compile-time assertion that the poller satisfies the worker.Worker interface.
var _ Worker = (*SupplierOrderStatusPoller)(nil)
