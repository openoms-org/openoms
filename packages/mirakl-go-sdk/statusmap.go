package mirakl

// statusMapping maps Mirakl order states to OpenOMS order status strings.
var statusMapping = map[string]string{
	"STAGING":              "pending",
	"WAITING_ACCEPTANCE":   "confirmed",
	"SHIPPING":             "confirmed",
	"SHIPPED":              "shipped",
	"RECEIVED":             "delivered",
	"CLOSED":               "delivered",
	"REFUSED":              "cancelled",
	"CANCELED":             "cancelled",
}

// MapStatus translates a Mirakl order state to the corresponding
// OpenOMS order status string.
func MapStatus(miraklStatus string) (openomsStatus string, ok bool) {
	openomsStatus, ok = statusMapping[miraklStatus]
	return
}
