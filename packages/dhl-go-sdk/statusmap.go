package dhl

// statusMapping maps DHL shipment statuses to OpenOMS shipment_status strings.
var statusMapping = map[string]string{
	"CREATED":          "created",
	"PICKED_UP":        "picked_up",
	"IN_TRANSIT":       "in_transit",
	"OUT_FOR_DELIVERY": "out_for_delivery",
	"DELIVERED":        "delivered",
	"RETURNED":         "returned",
	"FAILED":           "failed",
	"LABEL_CREATED":    "label_ready",
	"CUSTOMS":          "in_transit",
	"HELD":             "in_transit",
	"UNKNOWN":          "created",
}

// MapStatus translates a DHL shipment status to the corresponding
// OpenOMS shipment_status string.
func MapStatus(dhlStatus string) (openomsStatus string, ok bool) {
	openomsStatus, ok = statusMapping[dhlStatus]
	return
}
