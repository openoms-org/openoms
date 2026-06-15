package service

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
)

func TestDedupeFeedByExternalID(t *testing.T) {
	in := []integration.SupplierProduct{
		{ExternalID: "A", Name: "first"},
		{ExternalID: "B", Name: "b"},
		{ExternalID: "A", Name: "last"}, // later A wins
	}
	out := dedupeFeedByExternalID(in)
	assert.Len(t, out, 2)
	byID := map[string]string{}
	for _, p := range out {
		byID[p.ExternalID] = p.Name
	}
	assert.Equal(t, "last", byID["A"], "last occurrence of a duplicated external_id wins")
	assert.Equal(t, "b", byID["B"])
}

func TestDedupeFeedByExternalID_EmptyExternalIDCoalesces(t *testing.T) {
	// Empty external_id is a normal value of the unique key (tenant_id, supplier_id, external_id),
	// so the old per-row upsert collapsed empty-id rows to one (last-wins). The batch path must do
	// the same — otherwise two '' rows in one multi-row INSERT trip "ON CONFLICT cannot affect row
	// a second time".
	in := []integration.SupplierProduct{
		{ExternalID: "", Name: "first"},
		{ExternalID: "", Name: "last"},
	}
	out := dedupeFeedByExternalID(in)
	assert.Len(t, out, 1, "empty external_id rows coalesce to one")
	assert.Equal(t, "last", out[0].Name, "last occurrence wins")
}
