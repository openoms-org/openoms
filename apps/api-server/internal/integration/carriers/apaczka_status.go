package carriers

import "strings"

// mapApaczkaStatus maps a raw Apaczka shipment status string to the internal
// OMS shipment status. Returns ("", false) for unknown statuses.
//
// Apaczka status strings are normalised: trimmed and uppercased before matching.
func mapApaczkaStatus(raw string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "NEW", "CREATED", "WAITING":
		return "pending", true
	case "SENT", "IN_TRANSIT", "PICKED_UP":
		return "in_transit", true
	case "OUT_FOR_DELIVERY":
		return "out_for_delivery", true
	case "DELIVERED":
		return "delivered", true
	case "RETURNED", "RETURN":
		return "returned", true
	case "CANCELLED", "CANCELED":
		return "cancelled", true
	default:
		return "", false
	}
}
