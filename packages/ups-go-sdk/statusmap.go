package ups

// statusMapping maps UPS shipment status codes to OpenOMS shipment_status strings.
var statusMapping = map[string]string{
	"M":  "label_ready",      // Manifest
	"P":  "picked_up",        // Pickup
	"I":  "in_transit",       // In Transit
	"O":  "out_for_delivery", // Out for Delivery
	"D":  "delivered",        // Delivered
	"DO": "delivered",        // Delivered Origin
	"W":  "in_transit",       // Warehousing
	"X":  "failed",           // Exception
	"RS": "returned",         // Returned
}

// MapStatus translates a UPS shipment status code to the corresponding
// OpenOMS shipment_status string.
func MapStatus(upsStatus string) (openomsStatus string, ok bool) {
	openomsStatus, ok = statusMapping[upsStatus]
	return
}
