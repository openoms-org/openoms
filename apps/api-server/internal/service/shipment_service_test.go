package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// stubProductRepo implements repository.ProductRepo for testing calculateOrderWeight.
type stubProductRepo struct {
	products map[uuid.UUID]*model.Product
}

func (s *stubProductRepo) List(_ context.Context, _ pgx.Tx, _ model.ProductListFilter) ([]model.Product, int, error) {
	return nil, 0, nil
}
func (s *stubProductRepo) FindByID(_ context.Context, _ pgx.Tx, id uuid.UUID) (*model.Product, error) {
	if p, ok := s.products[id]; ok {
		return p, nil
	}
	return nil, nil
}
func (s *stubProductRepo) FindByIDs(_ context.Context, _ pgx.Tx, ids []uuid.UUID) ([]model.Product, error) {
	var result []model.Product
	for _, id := range ids {
		if p, ok := s.products[id]; ok {
			result = append(result, *p)
		}
	}
	return result, nil
}
func (s *stubProductRepo) FindBySKU(_ context.Context, _ pgx.Tx, _ string) (*model.Product, error) {
	return nil, nil
}
func (s *stubProductRepo) AvailableStockBatch(_ context.Context, _ pgx.Tx, _ []uuid.UUID) (map[uuid.UUID]int, error) {
	return nil, nil
}
func (s *stubProductRepo) FindByEAN(_ context.Context, _ pgx.Tx, _ string) (*model.Product, error) {
	return nil, nil
}
func (s *stubProductRepo) FindIDsByEANs(_ context.Context, _ pgx.Tx, _ []string) (map[string]uuid.UUID, error) {
	return map[string]uuid.UUID{}, nil
}
func (s *stubProductRepo) Create(_ context.Context, _ pgx.Tx, _ *model.Product) error { return nil }
func (s *stubProductRepo) Update(_ context.Context, _ pgx.Tx, _ uuid.UUID, _ model.UpdateProductRequest) error {
	return nil
}
func (s *stubProductRepo) Delete(_ context.Context, _ pgx.Tx, _ uuid.UUID) error { return nil }

type fakeShipmentLookupQuerier struct {
	query string
	args  []any
	row   pgx.Row
}

func (f *fakeShipmentLookupQuerier) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	f.query = sql
	f.args = args
	return f.row
}

type fakeShipmentLookupRow struct {
	shipmentID uuid.UUID
	tenantID   uuid.UUID
	status     string
	err        error
}

func (r fakeShipmentLookupRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != 3 {
		return errors.New("unexpected destination count")
	}
	*(dest[0].(*uuid.UUID)) = r.shipmentID
	*(dest[1].(*uuid.UUID)) = r.tenantID
	*(dest[2].(*string)) = r.status
	return nil
}

func TestShipmentService_UpdateStatusByTrackingNumber_RequiresWorkerPool(t *testing.T) {
	svc := NewShipmentService(nil, nil, nil, nil, nil, nil, nil)

	err := svc.UpdateStatusByTrackingNumber(context.Background(), "123", "inpost", "delivered")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "worker database pool is not configured")
}

func TestShipmentService_UpdateStatusByTrackingNumber_UsesWorkerPoolForCrossTenantLookup(t *testing.T) {
	shipmentID := uuid.New()
	tenantID := uuid.New()
	lookup := &fakeShipmentLookupQuerier{row: fakeShipmentLookupRow{
		shipmentID: shipmentID,
		tenantID:   tenantID,
		status:     "delivered",
	}}
	svc := &ShipmentService{workerQuerier: lookup}

	err := svc.UpdateStatusByTrackingNumber(context.Background(), "TRK123", "inpost", "delivered")

	require.NoError(t, err)
	assert.Contains(t, lookup.query, "FROM shipments")
	assert.Equal(t, []any{"TRK123", "inpost"}, lookup.args)
}

func TestShipmentService_Create_ValidationError_MissingOrderID(t *testing.T) {
	svc := NewShipmentService(nil, nil, nil, nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateShipmentRequest{
		Provider: "inpost",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "order_id")
}

func TestShipmentService_Create_ValidationError_MissingProvider(t *testing.T) {
	svc := NewShipmentService(nil, nil, nil, nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateShipmentRequest{
		OrderID: uuid.New(),
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "provider")
}

func TestShipmentService_Update_ValidationError_NoFields(t *testing.T) {
	svc := NewShipmentService(nil, nil, nil, nil, nil, nil, nil)

	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateShipmentRequest{}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestShipmentService_TransitionStatus_ValidationError_EmptyStatus(t *testing.T) {
	svc := NewShipmentService(nil, nil, nil, nil, nil, nil, nil)

	_, err := svc.TransitionStatus(context.Background(), uuid.New(), uuid.New(),
		model.ShipmentStatusTransitionRequest{Status: ""}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestShipmentService_TransitionStatus_ValidationError_WhitespaceStatus(t *testing.T) {
	svc := NewShipmentService(nil, nil, nil, nil, nil, nil, nil)

	_, err := svc.TransitionStatus(context.Background(), uuid.New(), uuid.New(),
		model.ShipmentStatusTransitionRequest{Status: "   "}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "status")
}

func TestShipmentService_Create_ValidationError_BothMissing(t *testing.T) {
	svc := NewShipmentService(nil, nil, nil, nil, nil, nil, nil)

	// Both OrderID (uuid.Nil) and Provider ("") are missing — should fail on order_id first
	_, err := svc.Create(context.Background(), uuid.New(), model.CreateShipmentRequest{
		// OrderID defaults to uuid.Nil, Provider defaults to ""
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "order_id")
}

func TestShipmentService_Update_ValidationError_NegativeWeightAccepted(t *testing.T) {
	// UpdateShipmentRequest does NOT validate weight values (no negative check).
	// Verify that the model validation passes for a negative weight — it only
	// checks that at least one field is provided.
	negWeight := -5.0
	req := model.UpdateShipmentRequest{
		Weight: &negWeight,
	}
	err := req.Validate()
	assert.NoError(t, err, "negative weight should pass validation — no range check exists in UpdateShipmentRequest")
}

func TestShipmentService_Create_ValidationError_ProviderTooLong(t *testing.T) {
	svc := NewShipmentService(nil, nil, nil, nil, nil, nil, nil)

	// CreateShipmentRequest.Validate() checks validateMaxLength("provider", ..., 100)
	var longProvider strings.Builder
	for range 101 {
		longProvider.WriteString("x")
	}

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateShipmentRequest{
		OrderID:  uuid.New(),
		Provider: longProvider.String(),
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "provider")
}

// ---------------------------------------------------------------------------
// calculateOrderWeight tests
// ---------------------------------------------------------------------------

func ptrFloat(v float64) *float64 { return &v }

func TestCalculateOrderWeight_TwoProducts(t *testing.T) {
	pid1 := uuid.New()
	pid2 := uuid.New()
	repo := &stubProductRepo{products: map[uuid.UUID]*model.Product{
		pid1: {ID: pid1, Weight: ptrFloat(1.5)},
		pid2: {ID: pid2, Weight: ptrFloat(0.8)},
	}}
	svc := &ShipmentService{productRepo: repo}

	items, _ := json.Marshal([]map[string]any{
		{"product_id": pid1, "quantity": 2},
		{"product_id": pid2, "quantity": 3},
	})
	order := &model.Order{Items: items}

	got := svc.calculateOrderWeight(context.Background(), nil, order)
	// 1.5*2 + 0.8*3 = 3.0 + 2.4 = 5.4
	assert.InDelta(t, 5.4, got, 0.001)
}

func TestCalculateOrderWeight_NilProductRepo(t *testing.T) {
	svc := &ShipmentService{productRepo: nil}
	items, _ := json.Marshal([]map[string]any{
		{"product_id": uuid.New(), "quantity": 1},
	})
	order := &model.Order{Items: items}

	got := svc.calculateOrderWeight(context.Background(), nil, order)
	assert.Equal(t, 0.0, got)
}

func TestCalculateOrderWeight_EmptyItems(t *testing.T) {
	svc := &ShipmentService{productRepo: &stubProductRepo{}}
	order := &model.Order{Items: json.RawMessage(`[]`)}

	got := svc.calculateOrderWeight(context.Background(), nil, order)
	assert.Equal(t, 0.0, got)
}

func TestCalculateOrderWeight_NilItems(t *testing.T) {
	svc := &ShipmentService{productRepo: &stubProductRepo{}}
	order := &model.Order{}

	got := svc.calculateOrderWeight(context.Background(), nil, order)
	assert.Equal(t, 0.0, got)
}

func TestCalculateOrderWeight_InvalidJSON(t *testing.T) {
	svc := &ShipmentService{productRepo: &stubProductRepo{}}
	order := &model.Order{Items: json.RawMessage(`not json`)}

	got := svc.calculateOrderWeight(context.Background(), nil, order)
	assert.Equal(t, 0.0, got)
}

func TestCalculateOrderWeight_SkipsNilProductID(t *testing.T) {
	pid := uuid.New()
	repo := &stubProductRepo{products: map[uuid.UUID]*model.Product{
		pid: {ID: pid, Weight: ptrFloat(2.0)},
	}}
	svc := &ShipmentService{productRepo: repo}

	items, _ := json.Marshal([]map[string]any{
		{"quantity": 1},
		{"product_id": pid, "quantity": 3},
	})
	order := &model.Order{Items: items}

	got := svc.calculateOrderWeight(context.Background(), nil, order)
	assert.InDelta(t, 6.0, got, 0.001)
}

func TestCalculateOrderWeight_SkipsZeroQuantity(t *testing.T) {
	pid := uuid.New()
	repo := &stubProductRepo{products: map[uuid.UUID]*model.Product{
		pid: {ID: pid, Weight: ptrFloat(1.0)},
	}}
	svc := &ShipmentService{productRepo: repo}

	items, _ := json.Marshal([]map[string]any{
		{"product_id": pid, "quantity": 0},
	})
	order := &model.Order{Items: items}

	got := svc.calculateOrderWeight(context.Background(), nil, order)
	assert.Equal(t, 0.0, got)
}

func TestCalculateOrderWeight_SkipsProductWithoutWeight(t *testing.T) {
	pid1 := uuid.New()
	pid2 := uuid.New()
	repo := &stubProductRepo{products: map[uuid.UUID]*model.Product{
		pid1: {ID: pid1, Weight: ptrFloat(2.0)},
		pid2: {ID: pid2, Weight: nil},
	}}
	svc := &ShipmentService{productRepo: repo}

	items, _ := json.Marshal([]map[string]any{
		{"product_id": pid1, "quantity": 1},
		{"product_id": pid2, "quantity": 5},
	})
	order := &model.Order{Items: items}

	got := svc.calculateOrderWeight(context.Background(), nil, order)
	assert.InDelta(t, 2.0, got, 0.001)
}

func TestCalculateOrderWeight_SkipsProductNotInRepo(t *testing.T) {
	pid1 := uuid.New()
	pidMissing := uuid.New()
	repo := &stubProductRepo{products: map[uuid.UUID]*model.Product{
		pid1: {ID: pid1, Weight: ptrFloat(3.0)},
	}}
	svc := &ShipmentService{productRepo: repo}

	items, _ := json.Marshal([]map[string]any{
		{"product_id": pid1, "quantity": 1},
		{"product_id": pidMissing, "quantity": 2},
	})
	order := &model.Order{Items: items}

	got := svc.calculateOrderWeight(context.Background(), nil, order)
	assert.InDelta(t, 3.0, got, 0.001)
}

func TestCalculateOrderWeight_DuplicateProductIDSumsQuantities(t *testing.T) {
	pid := uuid.New()
	repo := &stubProductRepo{products: map[uuid.UUID]*model.Product{
		pid: {ID: pid, Weight: ptrFloat(1.0)},
	}}
	svc := &ShipmentService{productRepo: repo}

	items, _ := json.Marshal([]map[string]any{
		{"product_id": pid, "quantity": 2},
		{"product_id": pid, "quantity": 3},
	})
	order := &model.Order{Items: items}

	got := svc.calculateOrderWeight(context.Background(), nil, order)
	// 1.0 * (2+3) = 5.0
	assert.InDelta(t, 5.0, got, 0.001)
}
