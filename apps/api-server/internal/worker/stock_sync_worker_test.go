package worker

import (
	"strings"
	"testing"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/stretchr/testify/assert"
)

// normalizeSQL collapses all whitespace runs to single spaces so query-shape
// assertions are independent of source indentation.
func normalizeSQL(q string) string { return strings.Join(strings.Fields(q), " ") }

// TestStockSyncListingsQuery_FlagOffParity guards the OPE-532 gated-flag parity:
// when the supplier-availability resolver is disabled (default), the gather query
// must be the pre-OPE-418 legacy query — no products join, no supplier columns.
// Only the gated variant may carry the supplier join.
func TestStockSyncListingsQuery_FlagOffParity(t *testing.T) {
	legacy := normalizeSQL(stockSyncListingsQueryLegacy)
	gated := normalizeSQL(stockSyncListingsQueryWithSupplier)

	// Flag-off: exact legacy shape (the pre-OPE-418 query, whitespace-normalized).
	assert.Equal(t,
		"SELECT pl.id, pl.external_id, pl.stock_override, "+
			"GREATEST(COALESCE(SUM(ws.quantity), 0) - COALESCE(SUM(ws.reserved), 0), 0) AS available_qty "+
			"FROM product_listings pl "+
			"LEFT JOIN warehouse_stock ws ON ws.product_id = pl.product_id AND ws.variant_id IS NULL "+
			"WHERE pl.integration_id = $1 AND pl.status = 'active' "+
			"AND pl.external_id IS NOT NULL AND pl.stock_sync_mode = 'auto' "+
			"GROUP BY pl.id, pl.external_id, pl.stock_override "+
			"LIMIT 5000",
		legacy)
	assert.NotContains(t, legacy, "JOIN products")
	assert.NotContains(t, legacy, "dropship_supplier_id")

	// Flag-on: the gated variant adds the products join + supplier columns.
	assert.Contains(t, gated, "JOIN products p ON p.id = pl.product_id")
	assert.Contains(t, gated, "p.dropship_supplier_id")

	// Both variants target the same listing set.
	for _, q := range []string{legacy, gated} {
		assert.Contains(t, q, "FROM product_listings pl")
		assert.Contains(t, q, "LEFT JOIN warehouse_stock ws ON ws.product_id = pl.product_id AND ws.variant_id IS NULL")
		assert.Contains(t, q, "WHERE pl.integration_id = $1 AND pl.status = 'active'")
		assert.Contains(t, q, "pl.stock_sync_mode = 'auto'")
		assert.Contains(t, q, "LIMIT 5000")
	}
}

func TestChunkSlice(t *testing.T) {
	items := make([]integration.StockUpdate, 250)
	for i := range items {
		items[i] = integration.StockUpdate{ExternalOfferID: "offer", Quantity: i}
	}

	chunks := chunkStockUpdates(items, 100)

	assert.Len(t, chunks, 3)
	assert.Len(t, chunks[0], 100)
	assert.Len(t, chunks[1], 100)
	assert.Len(t, chunks[2], 50)
}

func TestChunkSlice_Empty(t *testing.T) {
	chunks := chunkStockUpdates(nil, 100)
	assert.Nil(t, chunks)
}

func TestTruncateErrorMessage(t *testing.T) {
	short := "short error"
	assert.Equal(t, short, truncateErrorMessage(short))

	long := make([]byte, 600)
	for i := range long {
		long[i] = 'x'
	}
	result := truncateErrorMessage(string(long))
	assert.Len(t, result, 500)
}

func TestCloseProvider_WithCloser(t *testing.T) {
	called := false
	p := &mockCloser{onClose: func() { called = true }}
	closeProvider(p)
	assert.True(t, called)
}

func TestCloseProvider_WithoutCloser(_ *testing.T) {
	// Should not panic on a type without Close()
	closeProvider("not a closer")
}

type mockCloser struct {
	onClose func()
}

func (m *mockCloser) Close() {
	if m.onClose != nil {
		m.onClose()
	}
}
