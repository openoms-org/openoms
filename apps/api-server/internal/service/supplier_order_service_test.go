package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSupplierOrderRequest_OK(t *testing.T) {
	lines := []SupplierOrderInputLine{{EAN: "590", SupplierSKU: "S1", Quantity: 2}}
	req, missing, ambiguous := BuildSupplierOrderRequest("ORD-1", lines)
	require.Empty(t, missing)
	require.False(t, ambiguous)
	assert.Equal(t, "ORD-1", req.ClientOrderNumber)
	require.Len(t, req.Lines, 1)
	assert.Equal(t, "590", req.Lines[0].EAN)
	assert.Equal(t, "S1", req.Lines[0].ItemID)
	assert.Equal(t, float64(2), req.Lines[0].Quantity)
}

func TestBuildSupplierOrderRequest_SKUOnly(t *testing.T) {
	// A line with a SKU but no EAN is still a resolvable identity (ItemID is set).
	lines := []SupplierOrderInputLine{{SupplierSKU: "S2", Quantity: 1}}
	req, missing, ambiguous := BuildSupplierOrderRequest("ORD-2", lines)
	require.Empty(t, missing)
	require.False(t, ambiguous)
	require.Len(t, req.Lines, 1)
	assert.Equal(t, "", req.Lines[0].EAN)
	assert.Equal(t, "S2", req.Lines[0].ItemID)
}

func TestBuildSupplierOrderRequest_MissingIdentity(t *testing.T) {
	id := uuid.New()
	lines := []SupplierOrderInputLine{{LineID: id, Quantity: 1}} // no EAN and no SKU
	_, missing, _ := BuildSupplierOrderRequest("ORD-1", lines)
	assert.NotEmpty(t, missing, "a line with neither EAN nor SKU is missing required identity")
	assert.Equal(t, id.String(), missing[0])
}

func TestMapSupplierStatus_Canonical(t *testing.T) {
	assert.Equal(t, "confirmed", MapSupplierStatus("ACCEPTED"))
	assert.Equal(t, "confirmed", MapSupplierStatus("confirmed")) // case-insensitive
	assert.Equal(t, "shipped", MapSupplierStatus("SHIPPED"))
	assert.Equal(t, "shipped", MapSupplierStatus(" dispatched ")) // trimmed + cased
	assert.Equal(t, "", MapSupplierStatus("weird_unmapped"))      // empty => unmapped -> blocker upstream
	assert.Equal(t, "", MapSupplierStatus(""))                    // empty stays empty
}
