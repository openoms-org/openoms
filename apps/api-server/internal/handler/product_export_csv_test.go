package handler

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// The stock column of the product CSV export must carry the canonical available stock
// (warehouse_stock based), not the legacy products.stock_quantity column, so that the
// operator report matches the product list in the dashboard.
func TestProductExportRow_StockColumnUsesAvailableStock(t *testing.T) {
	stockIdx := indexOf(t, productExportHeader, "stock_quantity")

	p := model.Product{
		ID:             uuid.New(),
		Name:           "Widget",
		Price:          19.99,
		StockQuantity:  42, // legacy, stale
		AvailableStock: 7,  // canonical
	}

	row := productExportRow(p)

	require.Len(t, row, len(productExportHeader))
	assert.Equal(t, "7", row[stockIdx])
	assert.NotContains(t, row, "42")
}

// The export -> import round trip is a customer-facing contract: the CSV import parser
// keys off the "stock_quantity" header, so the column names must stay untouched even
// though the value now comes from a different source.
func TestProductExportHeader_IsUnchanged(t *testing.T) {
	assert.Equal(t, []string{
		"id", "name", "sku", "ean", "price", "stock_quantity",
		"category", "tags", "weight", "width", "height", "length",
		"short_description", "status",
	}, productExportHeader)
}

func TestProductExportRow_FormatsOptionalFields(t *testing.T) {
	sku := "SKU-1"
	ean := "5901234123457"
	category := "Tools"
	weight := 1.5
	width := 10.0
	height := 20.0
	depth := 30.0

	p := model.Product{
		ID:               uuid.New(),
		Name:             "Widget",
		SKU:              &sku,
		EAN:              &ean,
		Price:            19.9,
		AvailableStock:   3,
		Category:         &category,
		Tags:             []string{"a", "b"},
		Weight:           &weight,
		Width:            &width,
		Height:           &height,
		Depth:            &depth,
		DescriptionShort: "short",
	}

	row := productExportRow(p)

	require.Len(t, row, len(productExportHeader))
	assert.Equal(t, p.ID.String(), row[indexOf(t, productExportHeader, "id")])
	assert.Equal(t, "Widget", row[indexOf(t, productExportHeader, "name")])
	assert.Equal(t, sku, row[indexOf(t, productExportHeader, "sku")])
	assert.Equal(t, ean, row[indexOf(t, productExportHeader, "ean")])
	assert.Equal(t, "19.90", row[indexOf(t, productExportHeader, "price")])
	assert.Equal(t, "3", row[indexOf(t, productExportHeader, "stock_quantity")])
	assert.Equal(t, category, row[indexOf(t, productExportHeader, "category")])
	assert.Equal(t, "a,b", row[indexOf(t, productExportHeader, "tags")])
	assert.Equal(t, "1.50", row[indexOf(t, productExportHeader, "weight")])
	assert.Equal(t, "10.00", row[indexOf(t, productExportHeader, "width")])
	assert.Equal(t, "20.00", row[indexOf(t, productExportHeader, "height")])
	assert.Equal(t, "30.00", row[indexOf(t, productExportHeader, "length")])
	assert.Equal(t, "short", row[indexOf(t, productExportHeader, "short_description")])
	assert.Equal(t, "active", row[indexOf(t, productExportHeader, "status")])
}

func TestProductExportRow_EmptyOptionalFields(t *testing.T) {
	row := productExportRow(model.Product{ID: uuid.New(), Name: "Bare"})

	require.Len(t, row, len(productExportHeader))
	for _, col := range []string{"sku", "ean", "category", "tags", "weight", "width", "height", "length"} {
		assert.Empty(t, row[indexOf(t, productExportHeader, col)], "column %s", col)
	}
	assert.Equal(t, "0", row[indexOf(t, productExportHeader, "stock_quantity")])
}

func indexOf(t *testing.T, cols []string, name string) int {
	t.Helper()
	for i, c := range cols {
		if c == name {
			return i
		}
	}
	t.Fatalf("column %q not found in header %v", name, cols)
	return -1
}
