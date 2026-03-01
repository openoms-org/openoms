package erli

// statusMapping maps Erli order statuses to OpenOMS order status strings.
// Erli has exactly 3 order statuses per the official API docs:
//   - "pending"   — order placed but not yet paid (PayU awaiting payment)
//   - "purchased" — order paid (PayU success or cash-on-delivery)
//   - "cancelled" — order cancelled
var statusMapping = map[string]string{
	"pending":   "new",
	"purchased": "confirmed",
	"cancelled": "cancelled",
}

// MapStatus translates an Erli order status to the corresponding
// OpenOMS order status string.
func MapStatus(erliStatus string) (openomsStatus string, ok bool) {
	openomsStatus, ok = statusMapping[erliStatus]
	return
}
