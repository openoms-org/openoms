package dpd

// statusMapping maps DPD shipment statuses to OpenOMS shipment_status strings.
var statusMapping = map[string]string{
	"NEW":              "pending",
	"SENT":             "in_transit",
	"IN_TRANSIT":       "in_transit",
	"OUT_FOR_DELIVERY": "out_for_delivery",
	"DELIVERED":        "delivered",
	"RETURNED":         "returned",
	"PICKUP_AT_POINT":  "ready_for_pickup",
}

// MapStatus translates a DPD shipment status to the corresponding
// OpenOMS shipment_status string.
func MapStatus(dpdStatus string) (openomsStatus string, ok bool) {
	openomsStatus, ok = statusMapping[dpdStatus]
	return
}
