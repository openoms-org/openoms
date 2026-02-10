package ups

// statusMapping maps UPS shipment status codes to OpenOMS shipment_status strings.
var statusMapping = map[string]string{
	"M":  "pending",      // Manifest
	"I":  "in_transit",   // In Transit
	"D":  "delivered",    // Delivered
	"X":  "exception",    // Exception
	"P":  "picked_up",    // Pickup
	"RS": "returned",     // Returned
}

// MapStatus translates a UPS shipment status code to the corresponding
// OpenOMS shipment_status string.
func MapStatus(upsStatus string) (openomsStatus string, ok bool) {
	openomsStatus, ok = statusMapping[upsStatus]
	return
}
