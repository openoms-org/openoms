package dpd

// statusMapping maps DPD shipment statuses to OpenOMS shipment_status strings.
var statusMapping = map[string]string{
	"NEW":              "label_ready",
	"SENT":             "picked_up",
	"IN_TRANSIT":       "in_transit",
	"OUT_FOR_DELIVERY": "out_for_delivery",
	"DELIVERED":        "delivered",
	"RETURNED":         "returned",
	"PICKUP_AT_POINT":  "out_for_delivery",
	"CANCELLED":        "failed",
	"FAILED":           "failed",
	"REFUSED":          "failed",
	"LOST":             "failed",
	"DESTROYED":        "failed",
}

// MapStatus translates a DPD shipment status to the corresponding
// OpenOMS shipment_status string.
func MapStatus(dpdStatus string) (openomsStatus string, ok bool) {
	openomsStatus, ok = statusMapping[dpdStatus]
	return
}
