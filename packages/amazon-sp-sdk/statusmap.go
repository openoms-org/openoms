package amazon

// statusMapping maps Amazon SP-API order statuses to OpenOMS order status strings.
var statusMapping = map[string]string{
	"Pending":              "pending",
	"Unshipped":            "confirmed",
	"PartiallyShipped":     "confirmed",
	"Shipped":              "shipped",
	"Canceled":             "cancelled",
	"Unfulfillable":        "cancelled",
	"InvoiceUnconfirmed":   "pending",
	"PendingAvailability":  "pending",
}

// MapStatus translates an Amazon SP-API order status to the corresponding
// OpenOMS order status string.
func MapStatus(amazonStatus string) (openomsStatus string, ok bool) {
	openomsStatus, ok = statusMapping[amazonStatus]
	return
}
