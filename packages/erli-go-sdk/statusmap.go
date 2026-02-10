package erli

// statusMapping maps Erli order statuses to OpenOMS order status strings.
var statusMapping = map[string]string{
	"new":       "pending",
	"paid":      "confirmed",
	"shipped":   "shipped",
	"delivered": "delivered",
	"cancelled": "cancelled",
	"returned":  "returned",
}

// MapStatus translates an Erli order status to the corresponding
// OpenOMS order status string.
func MapStatus(erliStatus string) (openomsStatus string, ok bool) {
	openomsStatus, ok = statusMapping[erliStatus]
	return
}
