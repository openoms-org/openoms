package service

import (
	"strings"

	"github.com/google/uuid"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
)

// SupplierOrderInputLine is one resolved dropship line for the supplier-order request
// builder (OPE-418/Phase-7). The prepare phase resolves each dropship item's identity
// (its product's EAN + the supplier SKU) into this slim, pure-testable shape before the
// request is assembled — keeping BuildSupplierOrderRequest dependency-free.
type SupplierOrderInputLine struct {
	LineID      uuid.UUID // the dropship item id, for missing-identity reporting
	EAN         string
	SupplierSKU string
	Quantity    int
}

// BuildSupplierOrderRequest assembles a SupplierOrderRequest from resolved dropship lines.
// It returns the request, the list of line ids missing a required identity (neither EAN nor
// supplier SKU), and whether any line is ambiguous. "Ambiguous" is reserved for a future
// identity-resolution step and returns false for v1. A missing-identity result must NOT be
// submitted — the caller raises supplier_order_missing_data.
func BuildSupplierOrderRequest(clientOrderNumber string, lines []SupplierOrderInputLine) (integration.SupplierOrderRequest, []string, bool) {
	req := integration.SupplierOrderRequest{ClientOrderNumber: clientOrderNumber}
	var missing []string
	for i := range lines {
		ean := strings.TrimSpace(lines[i].EAN)
		sku := strings.TrimSpace(lines[i].SupplierSKU)
		if ean == "" && sku == "" {
			missing = append(missing, lines[i].LineID.String())
			continue
		}
		req.Lines = append(req.Lines, integration.SupplierOrderLine{
			EAN:      ean,
			ItemID:   sku,
			Quantity: float64(lines[i].Quantity),
		})
	}
	return req, missing, false
}

// canonicalSupplierStatuses maps common raw supplier statuses to canonical OpenOMS dropship
// statuses. An unmapped raw status returns "" so the caller raises external_status_unmapped.
var canonicalSupplierStatuses = map[string]string{
	"ACCEPTED": "confirmed", "CONFIRMED": "confirmed",
	"PROCESSING": "processing",
	"SHIPPED":    "shipped", "SENT": "shipped", "DISPATCHED": "shipped",
	"DELIVERED": "delivered",
	"CANCELLED": "cancelled", "REJECTED": "cancelled",
}

// MapSupplierStatus maps a raw supplier status (case-insensitive) to a canonical status, or
// "" when unmapped. The raw value is preserved separately by the caller.
func MapSupplierStatus(raw string) string {
	return canonicalSupplierStatuses[strings.ToUpper(strings.TrimSpace(raw))]
}
