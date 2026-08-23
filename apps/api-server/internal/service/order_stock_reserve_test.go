package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// stockRepoStub implements repository.WarehouseStockRepo over an in-memory row set.
type stockRepoStub struct {
	rows    map[uuid.UUID][]model.WarehouseStock
	listErr error
	calls   int
}

func (s *stockRepoStub) ListByWarehouse(_ context.Context, _ pgx.Tx, _ uuid.UUID, _ model.WarehouseStockListFilter) ([]model.WarehouseStock, int, error) {
	return nil, 0, nil
}

func (s *stockRepoStub) ListByProduct(_ context.Context, _ pgx.Tx, productID uuid.UUID) ([]model.WarehouseStock, error) {
	s.calls++
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.rows[productID], nil
}

func (s *stockRepoStub) Upsert(_ context.Context, _ pgx.Tx, _ *model.WarehouseStock) error {
	return nil
}

func (s *stockRepoStub) AdjustQuantity(_ context.Context, _ pgx.Tx, _, _ uuid.UUID, _ *uuid.UUID, _ int) error {
	return nil
}

// TestReserveStockForOrder_NoRepoOrNoLines verifies the guards that make the reserve a
// no-op, so an order with no stock-bearing lines never touches the warehouse.
func TestReserveStockForOrder_NoRepoOrNoLines(t *testing.T) {
	svc := NewOrderService(nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, svc.reserveStockForOrder(context.Background(), nil, map[uuid.UUID]int{uuid.New(): 2}),
		"unwired warehouse repo -> no-op")

	stock := &stockRepoStub{}
	svc.SetWarehouseStockRepo(stock)
	require.NoError(t, svc.reserveStockForOrder(context.Background(), nil, nil), "no lines -> no-op")
	assert.Zero(t, stock.calls, "no warehouse lookup for an order with no stock-bearing lines")
}

// TestReserveStockForOrder_ReadFailureAborts is the CORR-02 contract: a reserve that
// cannot be completed fails the caller's transaction instead of being logged and
// dropped, so an order can never commit without its reservation.
func TestReserveStockForOrder_ReadFailureAborts(t *testing.T) {
	svc := NewOrderService(nil, nil, nil, nil, nil, nil, nil)
	boom := errors.New("connection reset")
	svc.SetWarehouseStockRepo(&stockRepoStub{listErr: boom})

	err := svc.reserveStockForOrder(context.Background(), nil, map[uuid.UUID]int{uuid.New(): 1})
	require.Error(t, err, "a failed reserve must surface, not be swallowed")
	assert.ErrorIs(t, err, boom)
}

// TestAdjustStockPerProduct_StopsAtFirstFailure verifies the shared walk propagates the
// first per-row failure rather than continuing inside a transaction that is already
// poisoned.
func TestAdjustStockPerProduct_StopsAtFirstFailure(t *testing.T) {
	productA := uuid.New()
	warehouse := uuid.New()
	svc := NewOrderService(nil, nil, nil, nil, nil, nil, nil)
	svc.SetWarehouseStockRepo(&stockRepoStub{rows: map[uuid.UUID][]model.WarehouseStock{
		productA: {
			{ID: uuid.New(), WarehouseID: warehouse, ProductID: productA, Quantity: 5},
			{ID: uuid.New(), WarehouseID: warehouse, ProductID: productA, Quantity: 5},
		},
	}})

	boom := errors.New("deadlock detected")
	visits := 0
	err := svc.adjustStockPerProduct(context.Background(), nil, map[uuid.UUID]int{productA: 8},
		func(_ context.Context, _ pgx.Tx, _ uuid.UUID, _ model.WarehouseStock, _ int) (int, error) {
			visits++
			return 0, boom
		})
	require.ErrorIs(t, err, boom)
	assert.Equal(t, 1, visits, "the walk stops at the first failure")
}

// TestAdjustStockPerProduct_ConsumesAcrossWarehouses verifies the walk drains the
// requested quantity across successive rows and stops once satisfied.
func TestAdjustStockPerProduct_ConsumesAcrossWarehouses(t *testing.T) {
	product := uuid.New()
	svc := NewOrderService(nil, nil, nil, nil, nil, nil, nil)
	svc.SetWarehouseStockRepo(&stockRepoStub{rows: map[uuid.UUID][]model.WarehouseStock{
		product: {
			{ID: uuid.New(), WarehouseID: uuid.New(), ProductID: product, Quantity: 3},
			{ID: uuid.New(), WarehouseID: uuid.New(), ProductID: product, Quantity: 10},
			{ID: uuid.New(), WarehouseID: uuid.New(), ProductID: product, Quantity: 10},
		},
	}})

	var taken []int
	require.NoError(t, svc.adjustStockPerProduct(context.Background(), nil, map[uuid.UUID]int{product: 7},
		func(_ context.Context, _ pgx.Tx, _ uuid.UUID, stock model.WarehouseStock, remaining int) (int, error) {
			take := min(remaining, stock.Quantity)
			taken = append(taken, take)
			return take, nil
		}))
	assert.Equal(t, []int{3, 4}, taken, "7 units drained from two rows; the third row is untouched")
}
