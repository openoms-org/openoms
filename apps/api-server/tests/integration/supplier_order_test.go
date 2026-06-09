//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
	"github.com/openoms-org/openoms/apps/api-server/internal/worker"
)

// ── Test doubles for the supplier adapter ────────────────────────────────────

// fakeAPISupplier implements the bare SupplierProvider + the optional status reader.
type fakeAPISupplier struct {
	createErr  error
	externalID string
	status     *integration.SupplierOrderStatus
}

func (f *fakeAPISupplier) ProviderName() string { return "fake-api" }
func (f *fakeAPISupplier) FetchProducts(context.Context) ([]integration.SupplierProduct, error) {
	return nil, nil
}
func (f *fakeAPISupplier) FetchInventory(context.Context) ([]integration.SupplierProduct, error) {
	return nil, nil
}
func (f *fakeAPISupplier) CreateOrder(_ context.Context, _ integration.SupplierOrderRequest) (*integration.SupplierOrderResult, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	id := f.externalID
	if id == "" {
		id = "SUP-EXT-1"
	}
	return &integration.SupplierOrderResult{ExternalOrderID: id}, nil
}
func (f *fakeAPISupplier) GetOrderStatus(context.Context, string) (*integration.SupplierOrderStatus, error) {
	return f.status, nil
}

// ── Seed helpers ──────────────────────────────────────────────────────────────

// seedSupplier inserts a supplier (via the superuser pool, bypassing RLS) and returns its id.
// dropship_orders.supplier_id has a FK to suppliers, so the supplier row must exist.
func seedSupplier(t *testing.T, ctx context.Context, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := superPool.Exec(ctx,
		`INSERT INTO suppliers (id, tenant_id, name, feed_format) VALUES ($1,$2,'Fake Supplier','iof')`,
		id, tenantID)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM suppliers WHERE id = $1", id) })
	return id
}

// seedDropshipUnit ensures a dropship fulfillment unit on the order's process and returns it.
func seedDropshipUnit(t *testing.T, ctx context.Context, svc *service.FulfillmentService, tenantID, orderID, supplierID uuid.UUID) *model.FulfillmentUnit {
	t.Helper()
	unit := svc.EnsureUnit(ctx, tenantID, orderID, model.UnitTypeDropship, supplierID.String(),
		map[string]any{"supplier_id": supplierID.String(), "capability": "api"})
	require.NotNil(t, unit)
	return unit
}

// seedPendingDropshipOrder seeds a pending dropship_orders row + items for an (order, supplier)
// via the app pool (so RLS applies), returning the dropship order id.
func seedPendingDropshipOrder(t *testing.T, ctx context.Context, tenantID, orderID, supplierID uuid.UUID, items []model.DropshipOrderItem) uuid.UUID {
	t.Helper()
	dsRepo := repository.NewDropshipOrderRepository()
	itemRepo := repository.NewDropshipOrderItemRepository()
	dsID := uuid.New()
	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		d := &model.DropshipOrder{
			ID:           dsID,
			TenantID:     tenantID,
			OrderID:      orderID,
			SupplierID:   supplierID,
			SupplierName: "Fake Supplier",
			Status:       "pending",
			Currency:     "PLN",
		}
		if e := dsRepo.Create(ctx, tx, d); e != nil {
			return e
		}
		for i := range items {
			it := items[i]
			it.ID = uuid.New()
			it.TenantID = tenantID
			it.DropshipOrderID = dsID
			if e := itemRepo.CreateItem(ctx, tx, &it); e != nil {
				return e
			}
		}
		return nil
	}))
	return dsID
}

// itemsLoaderFor returns a supplierOrderItemsLoader-shaped closure that resolves the dropship
// lines for an (order, supplier) directly from the seeded dropship items (no product lookup).
func itemsLoaderFor() func(ctx context.Context, tx pgx.Tx, orderID, supplierID uuid.UUID) ([]service.SupplierOrderInputLine, []string, error) {
	dsRepo := repository.NewDropshipOrderRepository()
	itemRepo := repository.NewDropshipOrderItemRepository()
	return func(ctx context.Context, tx pgx.Tx, orderID, supplierID uuid.UUID) ([]service.SupplierOrderInputLine, []string, error) {
		dsOrders, err := dsRepo.FindByOrderID(ctx, tx, orderID)
		if err != nil {
			return nil, nil, err
		}
		var lines []service.SupplierOrderInputLine
		for di := range dsOrders {
			if dsOrders[di].SupplierID != supplierID {
				continue
			}
			its, ierr := itemRepo.ListByDropshipOrderID(ctx, tx, dsOrders[di].ID)
			if ierr != nil {
				return nil, nil, ierr
			}
			for ii := range its {
				lines = append(lines, service.SupplierOrderInputLine{
					LineID:      its[ii].ID,
					SupplierSKU: its[ii].SKU,
					Quantity:    its[ii].Quantity,
				})
			}
		}
		return lines, nil, nil
	}
}

func providerFactoryFor(p integration.SupplierProvider) func(ctx context.Context, tx pgx.Tx, tenantID, supplierID uuid.UUID) (integration.SupplierProvider, error) {
	return func(context.Context, pgx.Tx, uuid.UUID, uuid.UUID) (integration.SupplierProvider, error) {
		return p, nil
	}
}

// scopedProviderFactory returns the fake provider only for onlyTenant, and (nil, nil) for any
// other tenant — so the poller's full cross-tenant sweep (it lists all tenants on the
// privileged pool, mirroring production's worker pool) leaves other tests' rows untouched.
func scopedProviderFactory(onlyTenant uuid.UUID, p integration.SupplierProvider) func(ctx context.Context, tx pgx.Tx, tenantID, supplierID uuid.UUID) (integration.SupplierProvider, error) {
	return func(_ context.Context, _ pgx.Tx, tenantID, _ uuid.UUID) (integration.SupplierProvider, error) {
		if tenantID != onlyTenant {
			return nil, nil
		}
		return p, nil
	}
}

// ── Tests ─────────────────────────────────────────────────────────────────────

// TestSupplierOrder_APISubmit: an api supplier + routable unit → the handler updates the
// pending dropship_orders row with a supplier_reference, marks create_dropship_order
// succeeded, and transitions the unit to waiting_external.
func TestSupplierOrder_APISubmit(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	orderID, processID := seedFulfillmentOrder(t, ctx, tenantA, "API Submit Customer")
	supplierID := seedSupplier(t, ctx, tenantA)

	fSvc := newUnitService()
	unit := seedDropshipUnit(t, ctx, fSvc, tenantA, orderID, supplierID)
	dsID := seedPendingDropshipOrder(t, ctx, tenantA, orderID, supplierID,
		[]model.DropshipOrderItem{{SKU: "SKU-1", ProductName: "Widget", Quantity: 2}})

	handler := service.NewSupplierOrderHandler(appPool, fSvc,
		repository.NewDropshipOrderRepository(), repository.NewOrderRepository(),
		itemsLoaderFor(), providerFactoryFor(&fakeAPISupplier{externalID: "SUP-OK-1"}))

	err := handler.Handle(ctx, model.OrchestrationOutboxEvent{
		TenantID:  tenantA,
		ProcessID: processID,
		EventType: service.EventSupplierOrderSubmit,
		Payload: map[string]any{
			"order_id":    orderID.String(),
			"supplier_id": supplierID.String(),
			"unit_id":     unit.ID.String(),
		},
	})
	require.NoError(t, err)

	dsRepo := repository.NewDropshipOrderRepository()
	fRepo := repository.NewFulfillmentRepository()
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		// Exactly one dropship row, now carrying the supplier reference + status sent.
		orders, e := dsRepo.FindByOrderID(ctx, tx, orderID)
		if e != nil {
			return e
		}
		require.Len(t, orders, 1, "no duplicate dropship order created")
		assert.Equal(t, dsID, orders[0].ID)
		require.NotNil(t, orders[0].SupplierReference)
		assert.Equal(t, "SUP-OK-1", *orders[0].SupplierReference)
		assert.Equal(t, "sent", orders[0].Status)

		steps, e := fRepo.ListSteps(ctx, tx, unit.ID)
		if e != nil {
			return e
		}
		var create *model.FulfillmentStep
		for i := range steps {
			if steps[i].StepKey == model.StepCreateDropshipOrder {
				create = &steps[i]
			}
		}
		require.NotNil(t, create, "create_dropship_order step recorded")
		assert.Equal(t, model.FulfillmentStatusSucceeded, create.Status)

		units, e := fRepo.ListUnits(ctx, tx, processID)
		if e != nil {
			return e
		}
		require.Len(t, units, 1)
		assert.Equal(t, model.FulfillmentStatusWaitingExternal, units[0].Status)
		return nil
	}))
}

// TestSupplierOrder_MissingData: a dropship line with no EAN/SKU → supplier_order_missing_data
// blocker, and the dropship row is NOT given a supplier_reference.
func TestSupplierOrder_MissingData(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	orderID, processID := seedFulfillmentOrder(t, ctx, tenantA, "Missing Data Customer")
	supplierID := seedSupplier(t, ctx, tenantA)

	fSvc := newUnitService()
	unit := seedDropshipUnit(t, ctx, fSvc, tenantA, orderID, supplierID)
	seedPendingDropshipOrder(t, ctx, tenantA, orderID, supplierID,
		[]model.DropshipOrderItem{{SKU: "", ProductName: "No Identity", Quantity: 1}}) // no SKU, no product/EAN

	handler := service.NewSupplierOrderHandler(appPool, fSvc,
		repository.NewDropshipOrderRepository(), repository.NewOrderRepository(),
		itemsLoaderFor(), providerFactoryFor(&fakeAPISupplier{}))

	err := handler.Handle(ctx, model.OrchestrationOutboxEvent{
		TenantID:  tenantA,
		ProcessID: processID,
		EventType: service.EventSupplierOrderSubmit,
		Payload: map[string]any{
			"order_id":    orderID.String(),
			"supplier_id": supplierID.String(),
			"unit_id":     unit.ID.String(),
		},
	})
	require.NoError(t, err, "missing-data is a blocker, not a retryable error")

	dsRepo := repository.NewDropshipOrderRepository()
	fRepo := repository.NewFulfillmentRepository()
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		orders, e := dsRepo.FindByOrderID(ctx, tx, orderID)
		if e != nil {
			return e
		}
		require.Len(t, orders, 1)
		assert.Nil(t, orders[0].SupplierReference, "missing-data line must not be submitted")
		assert.Equal(t, "pending", orders[0].Status)

		blockers, e := fRepo.ListBlockers(ctx, tx, processID)
		if e != nil {
			return e
		}
		require.Len(t, blockers, 1)
		assert.Equal(t, model.BlockerSupplierOrderMissingData, blockers[0].Code)
		assert.Equal(t, "supplier", blockers[0].Category)
		return nil
	}))
}

// TestSupplierOrder_IdempotentResubmit: running the handler twice for the same (order,
// supplier) → exactly one dropship row carrying the reference, the second is a no-op.
func TestSupplierOrder_IdempotentResubmit(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	orderID, processID := seedFulfillmentOrder(t, ctx, tenantA, "Idempotent Customer")
	supplierID := seedSupplier(t, ctx, tenantA)

	fSvc := newUnitService()
	unit := seedDropshipUnit(t, ctx, fSvc, tenantA, orderID, supplierID)
	seedPendingDropshipOrder(t, ctx, tenantA, orderID, supplierID,
		[]model.DropshipOrderItem{{SKU: "SKU-IDEM", ProductName: "Repeat", Quantity: 1}})

	handler := service.NewSupplierOrderHandler(appPool, fSvc,
		repository.NewDropshipOrderRepository(), repository.NewOrderRepository(),
		itemsLoaderFor(), providerFactoryFor(&fakeAPISupplier{externalID: "SUP-IDEM-1"}))

	evt := model.OrchestrationOutboxEvent{
		TenantID:  tenantA,
		ProcessID: processID,
		EventType: service.EventSupplierOrderSubmit,
		Payload: map[string]any{
			"order_id":    orderID.String(),
			"supplier_id": supplierID.String(),
			"unit_id":     unit.ID.String(),
		},
	}
	require.NoError(t, handler.Handle(ctx, evt))
	require.NoError(t, handler.Handle(ctx, evt)) // second dispatch — must be a no-op

	dsRepo := repository.NewDropshipOrderRepository()
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		orders, e := dsRepo.FindByOrderID(ctx, tx, orderID)
		if e != nil {
			return e
		}
		require.Len(t, orders, 1, "idempotent re-submit must not create a second dropship order")
		require.NotNil(t, orders[0].SupplierReference)
		assert.Equal(t, "SUP-IDEM-1", *orders[0].SupplierReference)
		return nil
	}))
}

// TestSupplierOrder_Reconcile: the poller maps a "SHIPPED" raw status to "shipped" and
// records the confirm step; an unmapped raw status raises external_status_unmapped.
func TestSupplierOrder_Reconcile(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)

	// --- mapped: SHIPPED -> shipped ---
	orderID, processID := seedFulfillmentOrder(t, ctx, tenantA, "Reconcile Customer")
	supplierID := seedSupplier(t, ctx, tenantA)
	fSvc := newUnitService()
	unit := seedDropshipUnit(t, ctx, fSvc, tenantA, orderID, supplierID)
	ref := "SUP-SHIP-1"
	dsID := seedPendingDropshipOrder(t, ctx, tenantA, orderID, supplierID,
		[]model.DropshipOrderItem{{SKU: "SKU-R", ProductName: "Tracked", Quantity: 1}})
	// Mark it sent with a reference so the poller picks it up.
	dsRepo := repository.NewDropshipOrderRepository()
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		return dsRepo.UpdateFields(ctx, tx, dsID, model.UpdateDropshipStatusRequest{Status: "sent", SupplierReference: &ref})
	}))

	tracking := "TRK-123"
	// The poller lists tenants on its (privileged, in prod the worker) pool, so it gets superPool
	// here; the scoped factory keeps the cross-tenant sweep from touching other tests' rows.
	poller := worker.NewSupplierOrderStatusPoller(superPool, true, fSvc, dsRepo,
		scopedProviderFactory(tenantA, &fakeAPISupplier{status: &integration.SupplierOrderStatus{RawStatus: "SHIPPED", TrackingNumber: tracking, Carrier: "inpost"}}),
		nil)
	require.NoError(t, poller.Run(ctx))

	fRepo := repository.NewFulfillmentRepository()
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		d, e := dsRepo.FindByID(ctx, tx, dsID)
		if e != nil {
			return e
		}
		require.NotNil(t, d)
		assert.Equal(t, "shipped", d.Status)
		require.NotNil(t, d.TrackingNumber)
		assert.Equal(t, tracking, *d.TrackingNumber)
		steps, e := fRepo.ListSteps(ctx, tx, unit.ID)
		if e != nil {
			return e
		}
		var confirm *model.FulfillmentStep
		for i := range steps {
			if steps[i].StepKey == model.StepConfirmSupplierOrder {
				confirm = &steps[i]
			}
		}
		require.NotNil(t, confirm, "confirm_supplier_order step recorded")
		assert.Equal(t, model.FulfillmentStatusSucceeded, confirm.Status)
		return nil
	}))
	_ = processID

	// --- unmapped raw status -> external_status_unmapped blocker (fresh tenant for isolation) ---
	tenantB := seedTenant(t, ctx)
	orderID2, processID2 := seedFulfillmentOrder(t, ctx, tenantB, "Unmapped Customer")
	supplierID2 := seedSupplier(t, ctx, tenantB)
	unit2 := seedDropshipUnit(t, ctx, fSvc, tenantB, orderID2, supplierID2)
	ref2 := "SUP-WEIRD-1"
	dsID2 := seedPendingDropshipOrder(t, ctx, tenantB, orderID2, supplierID2,
		[]model.DropshipOrderItem{{SKU: "SKU-U", ProductName: "Weird", Quantity: 1}})
	require.NoError(t, database.WithTenant(ctx, appPool, tenantB, func(tx pgx.Tx) error {
		return dsRepo.UpdateFields(ctx, tx, dsID2, model.UpdateDropshipStatusRequest{Status: "sent", SupplierReference: &ref2})
	}))
	_ = unit2

	pollerUnmapped := worker.NewSupplierOrderStatusPoller(superPool, true, fSvc, dsRepo,
		scopedProviderFactory(tenantB, &fakeAPISupplier{status: &integration.SupplierOrderStatus{RawStatus: "teleported"}}),
		nil)
	require.NoError(t, pollerUnmapped.Run(ctx))

	require.NoError(t, database.WithTenant(ctx, appPool, tenantB, func(tx pgx.Tx) error {
		blockers, e := fRepo.ListBlockers(ctx, tx, processID2)
		if e != nil {
			return e
		}
		var found bool
		for i := range blockers {
			if blockers[i].Code == model.BlockerExternalStatusUnmapped {
				found = true
			}
		}
		assert.True(t, found, "unmapped supplier status raises external_status_unmapped")
		// Status unchanged (still sent) on an unmapped read.
		d, e := dsRepo.FindByID(ctx, tx, dsID2)
		if e != nil {
			return e
		}
		assert.Equal(t, "sent", d.Status)
		return nil
	}))
}

// TestSupplierOrder_RejectedPermanent: a permanent (business) CreateOrder error → a
// supplier_order_rejected blocker, no reference, and the handler returns nil (no retry).
func TestSupplierOrder_RejectedPermanent(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	orderID, processID := seedFulfillmentOrder(t, ctx, tenantA, "Rejected Customer")
	supplierID := seedSupplier(t, ctx, tenantA)

	fSvc := newUnitService()
	unit := seedDropshipUnit(t, ctx, fSvc, tenantA, orderID, supplierID)
	seedPendingDropshipOrder(t, ctx, tenantA, orderID, supplierID,
		[]model.DropshipOrderItem{{SKU: "SKU-REJ", ProductName: "Rejected", Quantity: 1}})

	handler := service.NewSupplierOrderHandler(appPool, fSvc,
		repository.NewDropshipOrderRepository(), repository.NewOrderRepository(),
		itemsLoaderFor(),
		providerFactoryFor(&fakeAPISupplier{createErr: model.Permanent(errors.New("supplier rejected: out of stock"))}))

	err := handler.Handle(ctx, model.OrchestrationOutboxEvent{
		TenantID:  tenantA,
		ProcessID: processID,
		EventType: service.EventSupplierOrderSubmit,
		Payload: map[string]any{
			"order_id":    orderID.String(),
			"supplier_id": supplierID.String(),
			"unit_id":     unit.ID.String(),
		},
	})
	require.NoError(t, err, "a permanent business rejection becomes a blocker, not a retryable error")

	dsRepo := repository.NewDropshipOrderRepository()
	fRepo := repository.NewFulfillmentRepository()
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		orders, e := dsRepo.FindByOrderID(ctx, tx, orderID)
		if e != nil {
			return e
		}
		require.Len(t, orders, 1)
		assert.Nil(t, orders[0].SupplierReference)
		blockers, e := fRepo.ListBlockers(ctx, tx, processID)
		if e != nil {
			return e
		}
		require.Len(t, blockers, 1)
		assert.Equal(t, model.BlockerSupplierOrderRejected, blockers[0].Code)
		return nil
	}))
}

// TestSupplierOrder_TransportErrorRetryable: a non-permanent CreateOrder error → the handler
// returns an error (retryable by the worker), and NO blocker is opened, NO reference set.
func TestSupplierOrder_TransportErrorRetryable(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	orderID, processID := seedFulfillmentOrder(t, ctx, tenantA, "Transport Customer")
	supplierID := seedSupplier(t, ctx, tenantA)

	fSvc := newUnitService()
	unit := seedDropshipUnit(t, ctx, fSvc, tenantA, orderID, supplierID)
	seedPendingDropshipOrder(t, ctx, tenantA, orderID, supplierID,
		[]model.DropshipOrderItem{{SKU: "SKU-NET", ProductName: "Flaky", Quantity: 1}})

	handler := service.NewSupplierOrderHandler(appPool, fSvc,
		repository.NewDropshipOrderRepository(), repository.NewOrderRepository(),
		itemsLoaderFor(),
		providerFactoryFor(&fakeAPISupplier{createErr: errors.New("connection reset by peer")}))

	err := handler.Handle(ctx, model.OrchestrationOutboxEvent{
		TenantID:  tenantA,
		ProcessID: processID,
		EventType: service.EventSupplierOrderSubmit,
		Payload: map[string]any{
			"order_id":    orderID.String(),
			"supplier_id": supplierID.String(),
			"unit_id":     unit.ID.String(),
		},
	})
	require.Error(t, err, "a transport error is retryable (propagated to the worker's backoff)")
	assert.False(t, model.IsPermanent(err), "transport error is not permanent")

	dsRepo := repository.NewDropshipOrderRepository()
	fRepo := repository.NewFulfillmentRepository()
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		orders, e := dsRepo.FindByOrderID(ctx, tx, orderID)
		if e != nil {
			return e
		}
		require.Len(t, orders, 1)
		assert.Nil(t, orders[0].SupplierReference)
		blockers, e := fRepo.ListBlockers(ctx, tx, processID)
		if e != nil {
			return e
		}
		assert.Empty(t, blockers, "a retryable transport error does not open a blocker")
		return nil
	}))
}

// TestSupplierOrder_EnqueueAndManualBranch exercises the gate-level entry points:
//   - api capability + enabled → EnqueueSubmit enqueues exactly one outbox event, idempotently.
//   - flag-off → no enqueue.
func TestSupplierOrder_EnqueueAndManualBranch(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	orderID, processID := seedFulfillmentOrder(t, ctx, tenantA, "Enqueue Customer")
	supplierID := seedSupplier(t, ctx, tenantA)
	fSvc := newUnitService()
	unit := seedDropshipUnit(t, ctx, fSvc, tenantA, orderID, supplierID)
	orchRepo := repository.NewOrchestrationRepository()

	// Enabled: enqueue twice — idempotency key dedups to one outbox row.
	svcOn := service.NewSupplierOrderService(true, appPool, fSvc, orchRepo)
	require.True(t, svcOn.Enabled())
	svcOn.EnqueueSubmit(ctx, tenantA, orderID, supplierID, unit.ID, unit.ProcessID)
	svcOn.EnqueueSubmit(ctx, tenantA, orderID, supplierID, unit.ID, unit.ProcessID)

	assert.Equal(t, 1, countOutboxByType(t, ctx, tenantA, processID, service.EventSupplierOrderSubmit),
		"EnqueueSubmit is idempotent on the (order, supplier) key")

	// Flag-off: no enqueue for a different order.
	orderID2, processID2 := seedFulfillmentOrder(t, ctx, tenantA, "FlagOff Customer")
	supplierID2 := seedSupplier(t, ctx, tenantA)
	unit2 := seedDropshipUnit(t, ctx, fSvc, tenantA, orderID2, supplierID2)
	svcOff := service.NewSupplierOrderService(false, appPool, fSvc, orchRepo)
	assert.False(t, svcOff.Enabled())
	svcOff.EnqueueSubmit(ctx, tenantA, orderID2, supplierID2, unit2.ID, unit2.ProcessID)
	assert.Equal(t, 0, countOutboxByType(t, ctx, tenantA, processID2, service.EventSupplierOrderSubmit),
		"disabled service enqueues nothing")
}

// countOutboxByType counts orchestration_outbox rows for a process + event type (app pool, RLS).
func countOutboxByType(t *testing.T, ctx context.Context, tenantID, processID uuid.UUID, eventType string) int {
	t.Helper()
	var n int
	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM orchestration_outbox WHERE process_id = $1 AND event_type = $2`,
			processID, eventType).Scan(&n)
	}))
	return n
}

// TestSupplierOrder_RLSIsolation: tenant B cannot see tenant A's supplier dropship orders /
// blockers produced by the engine.
func TestSupplierOrder_RLSIsolation(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	tenantB := seedTenant(t, ctx)
	orderID, processID := seedFulfillmentOrder(t, ctx, tenantA, "RLS Supplier Customer")
	supplierID := seedSupplier(t, ctx, tenantA)

	fSvc := newUnitService()
	unit := seedDropshipUnit(t, ctx, fSvc, tenantA, orderID, supplierID)
	seedPendingDropshipOrder(t, ctx, tenantA, orderID, supplierID,
		[]model.DropshipOrderItem{{SKU: "SKU-RLS", ProductName: "Isolated", Quantity: 1}})

	handler := service.NewSupplierOrderHandler(appPool, fSvc,
		repository.NewDropshipOrderRepository(), repository.NewOrderRepository(),
		itemsLoaderFor(), providerFactoryFor(&fakeAPISupplier{externalID: "SUP-RLS-1"}))
	require.NoError(t, handler.Handle(ctx, model.OrchestrationOutboxEvent{
		TenantID:  tenantA,
		ProcessID: processID,
		EventType: service.EventSupplierOrderSubmit,
		Payload: map[string]any{
			"order_id":    orderID.String(),
			"supplier_id": supplierID.String(),
			"unit_id":     unit.ID.String(),
		},
	}))

	dsRepo := repository.NewDropshipOrderRepository()
	// Tenant B sees no dropship orders for tenant A's order.
	require.NoError(t, database.WithTenant(ctx, appPool, tenantB, func(tx pgx.Tx) error {
		orders, e := dsRepo.FindByOrderID(ctx, tx, orderID)
		if e != nil {
			return e
		}
		assert.Empty(t, orders, "tenant B must not see tenant A's supplier dropship orders")
		return nil
	}))
	// Tenant A sees its row.
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		orders, e := dsRepo.FindByOrderID(ctx, tx, orderID)
		if e != nil {
			return e
		}
		require.Len(t, orders, 1)
		require.NotNil(t, orders[0].SupplierReference)
		return nil
	}))
}

// TestSupplierOrder_FlagOffNoOp: the status poller is a complete no-op when disabled.
func TestSupplierOrder_FlagOffNoOp(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	orderID, processID := seedFulfillmentOrder(t, ctx, tenantA, "Poller FlagOff Customer")
	supplierID := seedSupplier(t, ctx, tenantA)
	fSvc := newUnitService()
	_ = seedDropshipUnit(t, ctx, fSvc, tenantA, orderID, supplierID)
	ref := "SUP-OFF-1"
	dsID := seedPendingDropshipOrder(t, ctx, tenantA, orderID, supplierID,
		[]model.DropshipOrderItem{{SKU: "SKU-OFF", ProductName: "Untouched", Quantity: 1}})
	dsRepo := repository.NewDropshipOrderRepository()
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		return dsRepo.UpdateFields(ctx, tx, dsID, model.UpdateDropshipStatusRequest{Status: "sent", SupplierReference: &ref})
	}))

	// Disabled poller: would otherwise map "SHIPPED" -> shipped, but it must do nothing.
	poller := worker.NewSupplierOrderStatusPoller(appPool, false, fSvc, dsRepo,
		providerFactoryFor(&fakeAPISupplier{status: &integration.SupplierOrderStatus{RawStatus: "SHIPPED"}}), nil)
	require.NoError(t, poller.Run(ctx))

	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		d, e := dsRepo.FindByID(ctx, tx, dsID)
		if e != nil {
			return e
		}
		assert.Equal(t, "sent", d.Status, "disabled poller must not reconcile status")
		return nil
	}))
	_ = processID
}
