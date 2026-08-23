package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// --- stubs for syncOrderStatusForShipment (no DB: the tx is passed straight through) ---

type statusUpdate struct {
	orderID     uuid.UUID
	status      string
	shippedAt   *time.Time
	deliveredAt *time.Time
}

type syncOrderRepoStub struct {
	order   *model.Order
	updates []statusUpdate
}

func (s *syncOrderRepoStub) List(_ context.Context, _ pgx.Tx, _ model.OrderListFilter) ([]model.Order, int, error) {
	return nil, 0, nil
}

func (s *syncOrderRepoStub) FindByID(_ context.Context, _ pgx.Tx, _ uuid.UUID) (*model.Order, error) {
	if s.order == nil {
		return nil, nil
	}
	out := *s.order
	// Mirror the repository: a re-fetch after UpdateStatus reads the new status.
	if n := len(s.updates); n > 0 {
		out.Status = s.updates[n-1].status
	}
	return &out, nil
}

func (s *syncOrderRepoStub) FindByIDs(_ context.Context, _ pgx.Tx, _ []uuid.UUID) (map[uuid.UUID]*model.Order, error) {
	return map[uuid.UUID]*model.Order{}, nil
}

func (s *syncOrderRepoStub) Create(_ context.Context, _ pgx.Tx, _ *model.Order) error { return nil }

func (s *syncOrderRepoStub) CreateIfExternalIDNotExists(_ context.Context, _ pgx.Tx, _ *model.Order) (bool, error) {
	return false, nil
}

func (s *syncOrderRepoStub) Update(_ context.Context, _ pgx.Tx, _ uuid.UUID, _ model.UpdateOrderRequest) error {
	return nil
}

func (s *syncOrderRepoStub) UpdateStatus(_ context.Context, _ pgx.Tx, id uuid.UUID, status string, shippedAt, deliveredAt *time.Time) error {
	s.updates = append(s.updates, statusUpdate{orderID: id, status: status, shippedAt: shippedAt, deliveredAt: deliveredAt})
	return nil
}

func (s *syncOrderRepoStub) FindByExternalID(_ context.Context, _ pgx.Tx, _, _ string) (*model.Order, error) {
	return nil, nil
}

func (s *syncOrderRepoStub) Delete(_ context.Context, _ pgx.Tx, _ uuid.UUID) error { return nil }

func (s *syncOrderRepoStub) CountThisMonth(_ context.Context, _ pgx.Tx) (int, error) { return 0, nil }

type auditRepoStub struct {
	entries []model.AuditEntry
}

func (a *auditRepoStub) Log(_ context.Context, _ pgx.Tx, entry model.AuditEntry) error {
	a.entries = append(a.entries, entry)
	return nil
}

func (a *auditRepoStub) ListByEntity(_ context.Context, _ pgx.Tx, _ string, _ uuid.UUID) ([]model.AuditLogEntry, error) {
	return nil, nil
}

func (a *auditRepoStub) List(_ context.Context, _ pgx.Tx, _ model.AuditListFilter) ([]model.AuditLogEntry, int, error) {
	return nil, 0, nil
}

type syncShipmentRepoStub struct {
	shipments []model.Shipment
}

func (s *syncShipmentRepoStub) List(_ context.Context, _ pgx.Tx, _ model.ShipmentListFilter) ([]model.Shipment, int, error) {
	return s.shipments, len(s.shipments), nil
}

func (s *syncShipmentRepoStub) FindByID(_ context.Context, _ pgx.Tx, _ uuid.UUID) (*model.Shipment, error) {
	return nil, nil
}

func (s *syncShipmentRepoStub) FindByExternalID(_ context.Context, _ pgx.Tx, _ string) (*model.Shipment, error) {
	return nil, nil
}

func (s *syncShipmentRepoStub) CountByOrder(_ context.Context, _ pgx.Tx, _ uuid.UUID) (int, error) {
	return len(s.shipments), nil
}

func (s *syncShipmentRepoStub) Create(_ context.Context, _ pgx.Tx, _ *model.Shipment) error {
	return nil
}

func (s *syncShipmentRepoStub) Update(_ context.Context, _ pgx.Tx, _ uuid.UUID, _ model.UpdateShipmentRequest) error {
	return nil
}

func (s *syncShipmentRepoStub) UpdateStatus(_ context.Context, _ pgx.Tx, _ uuid.UUID, _ string) error {
	return nil
}

func (s *syncShipmentRepoStub) UpdateStatusIfCurrent(_ context.Context, _ pgx.Tx, _ uuid.UUID, _, _ string) (bool, error) {
	return false, nil
}

func (s *syncShipmentRepoStub) Delete(_ context.Context, _ pgx.Tx, _ uuid.UUID) error { return nil }

// newSyncFixture builds a ShipmentService over the stubs plus the shipment being
// transitioned and the order it belongs to.
func newSyncFixture(orderStatus string, shippedAt *time.Time, siblings ...model.Shipment) (*ShipmentService, *syncOrderRepoStub, *auditRepoStub, *model.Shipment) {
	orderID := uuid.New()
	shipment := &model.Shipment{ID: uuid.New(), OrderID: orderID}
	orders := &syncOrderRepoStub{order: &model.Order{ID: orderID, Status: orderStatus, ShippedAt: shippedAt}}
	audit := &auditRepoStub{}
	shipments := &syncShipmentRepoStub{shipments: append([]model.Shipment{{ID: shipment.ID, OrderID: orderID, Status: "delivered"}}, siblings...)}
	svc := &ShipmentService{orderRepo: orders, auditRepo: audit, shipmentRepo: shipments}
	return svc, orders, audit, shipment
}

// TestSyncOrderStatusForShipment_PickedUpShipsOrder is the core CORR-01 case: a
// carrier picking the package up moves the order to shipped and reports the change
// so its side effects (notably the one-time stock decrement) can fan out.
func TestSyncOrderStatusForShipment_PickedUpShipsOrder(t *testing.T) {
	svc, orders, audit, shipment := newSyncFixture(model.OrderStatusNew, nil)

	change, err := svc.syncOrderStatusForShipment(context.Background(), nil, uuid.New(), shipment, "picked_up", uuid.New(), "127.0.0.1")
	require.NoError(t, err)
	require.NotNil(t, change, "a carrier-driven ship must report a status change to fan out")

	assert.Equal(t, model.OrderStatusNew, change.oldStatus)
	assert.Equal(t, model.OrderStatusShipped, change.newStatus)
	assert.True(t, change.firstShip, "order had never shipped -> stock decrement is owed")
	require.NotNil(t, change.order)
	assert.Equal(t, model.OrderStatusShipped, change.order.Status, "the change carries the post-update order")

	require.Len(t, orders.updates, 1)
	assert.Equal(t, model.OrderStatusShipped, orders.updates[0].status)
	assert.NotNil(t, orders.updates[0].shippedAt, "shipped_at is stamped")
	assert.Nil(t, orders.updates[0].deliveredAt)

	require.Len(t, audit.entries, 1, "the synced status change is audited like an operator's")
	assert.Equal(t, "order.status_changed", audit.entries[0].Action)
	changes, ok := audit.entries[0].Changes.(map[string]string)
	require.True(t, ok, "audit changes are a string map")
	assert.Equal(t, shipment.ID.String(), changes["shipment_id"])
	assert.Equal(t, model.OrderStatusShipped, changes["to"])
}

// TestSyncOrderStatusForShipment_ReShipDoesNotDecrementAgain verifies the firstShip
// gate: an order that already carries shipped_at must not decrement stock a second
// time when another package reports movement.
func TestSyncOrderStatusForShipment_ReShipDoesNotDecrementAgain(t *testing.T) {
	shippedAt := time.Now().Add(-time.Hour)
	svc, _, _, shipment := newSyncFixture(model.OrderStatusProcessing, &shippedAt)

	change, err := svc.syncOrderStatusForShipment(context.Background(), nil, uuid.New(), shipment, "in_transit", uuid.New(), "")
	require.NoError(t, err)
	require.NotNil(t, change)
	assert.False(t, change.firstShip, "shipped_at already set -> no second decrement")
}

// TestSyncOrderStatusForShipment_AlreadyShippedIsNoop verifies the pre-existing
// guard: an order already shipped or delivered is left alone, so no duplicate
// emails/webhooks/stock effects fire.
func TestSyncOrderStatusForShipment_AlreadyShippedIsNoop(t *testing.T) {
	for _, status := range []string{model.OrderStatusShipped, model.OrderStatusDelivered} {
		svc, orders, audit, shipment := newSyncFixture(status, nil)

		change, err := svc.syncOrderStatusForShipment(context.Background(), nil, uuid.New(), shipment, "picked_up", uuid.New(), "")
		require.NoError(t, err)
		assert.Nil(t, change, "order %q needs no sync", status)
		assert.Empty(t, orders.updates)
		assert.Empty(t, audit.entries)
	}
}

// TestSyncOrderStatusForShipment_DeliveredWaitsForAllPackages verifies the
// multi-package rule: the order is delivered only once every shipment is.
func TestSyncOrderStatusForShipment_DeliveredWaitsForAllPackages(t *testing.T) {
	orderID := uuid.New()
	pending := model.Shipment{ID: uuid.New(), OrderID: orderID, Status: "in_transit"}
	svc, orders, _, shipment := newSyncFixture(model.OrderStatusShipped, nil, pending)

	change, err := svc.syncOrderStatusForShipment(context.Background(), nil, uuid.New(), shipment, "delivered", uuid.New(), "")
	require.NoError(t, err)
	assert.Nil(t, change, "a sibling package is still in transit")
	assert.Empty(t, orders.updates)
}

// TestSyncOrderStatusForShipment_DeliveredWhenAllPackagesDelivered verifies the
// delivered sync, including the delivered_at stamp.
func TestSyncOrderStatusForShipment_DeliveredWhenAllPackagesDelivered(t *testing.T) {
	svc, orders, audit, shipment := newSyncFixture(model.OrderStatusShipped, nil)
	svc.shipmentRepo.(*syncShipmentRepoStub).shipments = append(
		svc.shipmentRepo.(*syncShipmentRepoStub).shipments,
		model.Shipment{ID: uuid.New(), OrderID: shipment.OrderID, Status: "delivered"},
	)

	change, err := svc.syncOrderStatusForShipment(context.Background(), nil, uuid.New(), shipment, "delivered", uuid.New(), "")
	require.NoError(t, err)
	require.NotNil(t, change)
	assert.Equal(t, model.OrderStatusDelivered, change.newStatus)

	require.Len(t, orders.updates, 1)
	assert.NotNil(t, orders.updates[0].deliveredAt, "delivered_at is stamped")
	assert.Nil(t, orders.updates[0].shippedAt)
	assert.Len(t, audit.entries, 1)
}

// TestSyncOrderStatusForShipment_UnrelatedStatusIsNoop verifies shipment statuses
// that carry no order meaning do not touch the order.
func TestSyncOrderStatusForShipment_UnrelatedStatusIsNoop(t *testing.T) {
	for _, status := range []string{"created", "label_printed", "returned", "cancelled"} {
		svc, orders, _, shipment := newSyncFixture(model.OrderStatusNew, nil)

		change, err := svc.syncOrderStatusForShipment(context.Background(), nil, uuid.New(), shipment, status, uuid.New(), "")
		require.NoError(t, err)
		assert.Nil(t, change, "shipment status %q implies no order change", status)
		assert.Empty(t, orders.updates)
	}
}

// TestSyncOrderStatusForShipment_MissingOrderIsNoop verifies a shipment whose order
// is gone does not fail the shipment transition.
func TestSyncOrderStatusForShipment_MissingOrderIsNoop(t *testing.T) {
	svc, _, _, shipment := newSyncFixture(model.OrderStatusNew, nil)
	svc.orderRepo.(*syncOrderRepoStub).order = nil

	change, err := svc.syncOrderStatusForShipment(context.Background(), nil, uuid.New(), shipment, "picked_up", uuid.New(), "")
	require.NoError(t, err)
	assert.Nil(t, change)
}
