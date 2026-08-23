package service

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// productStockQuantityRead matches a read of the legacy products.stock_quantity column
// through a Product value. Assignments to it are writes and are fine — imports and the
// product form are what keep the column populated.
var productStockQuantityRead = regexp.MustCompile(`\bproduct\.StockQuantity\b(\s*=[^=])?`)

// legacyStockQuantityReaders is the reviewed set of files in this package that read
// Product.StockQuantity, with why each one is allowed to.
//
// Canonical available stock is `warehouse_stock.quantity - reserved`
// (ProductRepository.AvailableStockBatch). products.stock_quantity is never decremented
// on shipment, so reading it as availability silently oversells. Adding a file here
// means claiming the read is not an availability read — say why in the map.
var legacyStockQuantityReaders = map[string]string{
	// Passes the legacy value to the marketplace stock sync, which recomputes the
	// quantity it pushes from warehouse_stock; the pair only feeds its zero-crossing
	// heuristics. Also exposes it to automation rules under its own name.
	"product_service.go": "legacy column reported as itself, not as availability",
	// Overwrites the field with AvailableStockBatch output before the one strategy
	// that prices off stock reads it.
	"repricing_service.go": "field is overwritten with canonical availability first",
}

// TestNoNewLegacyStockQuantityReaders keeps the CORR-08 cleanup from regressing. The
// grep is the acceptance criterion: every remaining reader of the legacy column is
// reviewed and annotated, and a new one has to be justified here to compile green.
func TestNoNewLegacyStockQuantityReaders(t *testing.T) {
	entries, err := os.ReadDir(".")
	require.NoError(t, err)

	var found []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(filepath.Clean(name))
		require.NoError(t, err)
		for _, m := range productStockQuantityRead.FindAllStringSubmatch(string(src), -1) {
			if m[1] != "" { // an assignment, i.e. a write
				continue
			}
			found = append(found, name)
			break
		}
	}
	sort.Strings(found)

	var allowed []string
	for name := range legacyStockQuantityReaders {
		allowed = append(allowed, name)
	}
	sort.Strings(allowed)

	assert.Equal(t, allowed, found,
		"a file reads the legacy products.stock_quantity column. Use "+
			"ProductRepository.AvailableStockBatch for available stock, or add the file to "+
			"legacyStockQuantityReaders with the reason the read is safe")
}
