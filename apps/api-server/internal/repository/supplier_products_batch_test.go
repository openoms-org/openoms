package repository

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func TestBuildSupplierProductsUpsert(t *testing.T) {
	tid, sid := uuid.New(), uuid.New()
	a := &model.SupplierProduct{ID: uuid.New(), TenantID: tid, SupplierID: sid, ExternalID: "A"}
	b := &model.SupplierProduct{ID: uuid.New(), TenantID: tid, SupplierID: sid, ExternalID: "B"}

	q, args := buildSupplierProductsUpsert([]*model.SupplierProduct{a, b})

	assert.Contains(t, q, "INSERT INTO supplier_products")
	assert.Contains(t, q, "ON CONFLICT (tenant_id, supplier_id, external_id)")
	assert.Contains(t, q, "RETURNING id, external_id")
	// 13 columns per row → second row starts at $14.
	assert.Contains(t, q, "($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)")
	assert.Contains(t, q, "($14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26)")
	assert.Len(t, args, 26)
	assert.Equal(t, a.ID, args[0], "first arg of row 1 is its id")
	assert.Equal(t, b.ID, args[13], "first arg of row 2 is its id")
	// product_id ($4) is included in the INSERT but absent from DO UPDATE (existing link survives).
	assert.NotContains(t, q, "product_id = EXCLUDED.product_id")
}
