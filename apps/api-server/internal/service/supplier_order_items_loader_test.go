package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// ── fakes for the loader's narrow repo surfaces ──────────────────────────────

type fakeDropshipOrdersReader struct{ orders []model.DropshipOrder }

func (f *fakeDropshipOrdersReader) FindByOrderID(context.Context, pgx.Tx, uuid.UUID) ([]model.DropshipOrder, error) {
	return f.orders, nil
}

type fakeDropshipItemsReader struct {
	items map[uuid.UUID][]model.DropshipOrderItem
}

func (f *fakeDropshipItemsReader) ListByDropshipOrderID(_ context.Context, _ pgx.Tx, dsID uuid.UUID) ([]model.DropshipOrderItem, error) {
	return f.items[dsID], nil
}

type fakeProductReader struct{ products map[uuid.UUID]*model.Product }

func (f *fakeProductReader) FindByID(_ context.Context, _ pgx.Tx, id uuid.UUID) (*model.Product, error) {
	return f.products[id], nil
}

type fakeSupplierProductMappingReader struct {
	mappings map[uuid.UUID]*model.SupplierProduct // keyed by product id
}

func (f *fakeSupplierProductMappingReader) FindBySupplierAndProductID(_ context.Context, _ pgx.Tx, _, productID uuid.UUID) (*model.SupplierProduct, error) {
	return f.mappings[productID], nil
}

// loaderFixture builds a loader over one dropship order with a single line for productID.
func loaderFixture(t *testing.T, supplierID, productID uuid.UUID, product *model.Product, mapping *model.SupplierProduct, itemSKU string) ([]SupplierOrderInputLine, []string, error) {
	t.Helper()
	dsID := uuid.New()
	dsReader := &fakeDropshipOrdersReader{orders: []model.DropshipOrder{{ID: dsID, SupplierID: supplierID, Status: "pending"}}}
	itemReader := &fakeDropshipItemsReader{items: map[uuid.UUID][]model.DropshipOrderItem{
		dsID: {{ID: uuid.New(), ProductID: &productID, SKU: itemSKU, ProductName: "Line", Quantity: 3}},
	}}
	products := map[uuid.UUID]*model.Product{}
	if product != nil {
		products[productID] = product
	}
	mappings := map[uuid.UUID]*model.SupplierProduct{}
	if mapping != nil {
		mappings[productID] = mapping
	}
	loader := NewSupplierOrderItemsLoader(dsReader, itemReader, &fakeProductReader{products: products},
		&fakeSupplierProductMappingReader{mappings: mappings})
	return loader(context.Background(), nil, uuid.New(), supplierID)
}

// Mapping present: ItemID is the SUPPLIER's catalogue external id, EAN comes from the
// product, and the tenant's internal SKU never leaks into the line (OPE-516).
func TestSupplierOrderItemsLoader_MappingPresent(t *testing.T) {
	supplierID, productID := uuid.New(), uuid.New()
	product := &model.Product{ID: productID, SKU: strPtr("TENANT-SKU-1"), EAN: strPtr("5901234567890")}
	mapping := &model.SupplierProduct{SupplierID: supplierID, ProductID: &productID, ExternalID: "BTP-77123", SKU: strPtr("MFG-CODE-1")}

	lines, missing, err := loaderFixture(t, supplierID, productID, product, mapping, "TENANT-SKU-1")
	require.NoError(t, err)
	require.Empty(t, missing)
	require.Len(t, lines, 1)
	assert.Equal(t, "BTP-77123", lines[0].SupplierSKU, "ItemID must be the supplier catalogue external id")
	assert.Equal(t, "5901234567890", lines[0].EAN)
	assert.Equal(t, 3, lines[0].Quantity)
	assert.NotEqual(t, "TENANT-SKU-1", lines[0].SupplierSKU, "tenant SKU must never be sent as the supplier item id")
}

// No mapping: the line falls back to EAN-only identity (ItemID empty) — NEVER the tenant's
// internal SKU.
func TestSupplierOrderItemsLoader_EANOnlyFallback(t *testing.T) {
	supplierID, productID := uuid.New(), uuid.New()
	product := &model.Product{ID: productID, SKU: strPtr("TENANT-SKU-2"), EAN: strPtr("5900000000017")}

	lines, missing, err := loaderFixture(t, supplierID, productID, product, nil, "TENANT-SKU-2")
	require.NoError(t, err)
	require.Empty(t, missing)
	require.Len(t, lines, 1)
	assert.Equal(t, "", lines[0].SupplierSKU, "no mapping -> EAN-only, ItemID stays empty")
	assert.Equal(t, "5900000000017", lines[0].EAN)
}

// Neither mapping nor EAN: the line carries no identity, so the request builder routes it
// into the missing-identity list (supplier_order_missing_data path).
func TestSupplierOrderItemsLoader_BothMissing(t *testing.T) {
	supplierID, productID := uuid.New(), uuid.New()
	product := &model.Product{ID: productID, SKU: strPtr("TENANT-SKU-3")} // no EAN

	lines, missing, err := loaderFixture(t, supplierID, productID, product, nil, "TENANT-SKU-3")
	require.NoError(t, err)
	require.Empty(t, missing, "the loader defers missing-identity classification to the builder")
	require.Len(t, lines, 1)
	assert.Equal(t, "", lines[0].SupplierSKU)
	assert.Equal(t, "", lines[0].EAN)

	_, builderMissing, _ := BuildSupplierOrderRequest("ORD-X", lines)
	assert.Len(t, builderMissing, 1, "identity-less line must land in the missing list")
}

// A line not mapped to any product resolves no identity either (the dropship item's
// free-text SKU is tenant-entered and must not be trusted as a supplier id).
func TestSupplierOrderItemsLoader_UnmappedLineHasNoIdentity(t *testing.T) {
	supplierID := uuid.New()
	dsID := uuid.New()
	dsReader := &fakeDropshipOrdersReader{orders: []model.DropshipOrder{{ID: dsID, SupplierID: supplierID, Status: "pending"}}}
	itemReader := &fakeDropshipItemsReader{items: map[uuid.UUID][]model.DropshipOrderItem{
		dsID: {{ID: uuid.New(), ProductID: nil, SKU: "FREE-TEXT-SKU", ProductName: "Unmapped", Quantity: 1}},
	}}
	loader := NewSupplierOrderItemsLoader(dsReader, itemReader,
		&fakeProductReader{products: map[uuid.UUID]*model.Product{}},
		&fakeSupplierProductMappingReader{mappings: map[uuid.UUID]*model.SupplierProduct{}})

	lines, missing, err := loader(context.Background(), nil, uuid.New(), supplierID)
	require.NoError(t, err)
	require.Empty(t, missing)
	require.Len(t, lines, 1)
	assert.Equal(t, "", lines[0].SupplierSKU, "free-text item SKU must not be sent to the supplier")
	assert.Equal(t, "", lines[0].EAN)
}

// Lines belonging to a different supplier's dropship order are not loaded.
func TestSupplierOrderItemsLoader_FiltersBySupplier(t *testing.T) {
	supplierA, supplierB := uuid.New(), uuid.New()
	dsA, dsB := uuid.New(), uuid.New()
	dsReader := &fakeDropshipOrdersReader{orders: []model.DropshipOrder{
		{ID: dsA, SupplierID: supplierA, Status: "pending"},
		{ID: dsB, SupplierID: supplierB, Status: "pending"},
	}}
	itemReader := &fakeDropshipItemsReader{items: map[uuid.UUID][]model.DropshipOrderItem{
		dsA: {{ID: uuid.New(), SKU: "A-1", ProductName: "A", Quantity: 1}},
		dsB: {{ID: uuid.New(), SKU: "B-1", ProductName: "B", Quantity: 2}},
	}}
	loader := NewSupplierOrderItemsLoader(dsReader, itemReader,
		&fakeProductReader{products: map[uuid.UUID]*model.Product{}},
		&fakeSupplierProductMappingReader{mappings: map[uuid.UUID]*model.SupplierProduct{}})

	lines, _, err := loader(context.Background(), nil, uuid.New(), supplierA)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	assert.Equal(t, 1, lines[0].Quantity)
}
