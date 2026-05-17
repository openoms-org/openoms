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
	"030103":           "label_ready",
	"040101":           "picked_up",
	"040102":           "picked_up",
	"050101":           "in_transit",
	"050102":           "in_transit",
	"120100":           "in_transit",
	"120101":           "in_transit",
	"160101":           "in_transit",
	"160103":           "in_transit",
	"160501":           "in_transit",
	"160502":           "in_transit",
	"160503":           "in_transit",
	"170101":           "out_for_delivery",
	"170102":           "out_for_delivery",
	"170501":           "out_for_delivery",
	"190101":           "delivered",
	"190102":           "delivered",
	"190103":           "delivered",
	"190104":           "delivered",
	"190202":           "delivered",
	"190203":           "delivered",
	"190204":           "delivered",
	"501300":           "delivered",
	"501304":           "delivered",
	"501340":           "delivered",
	"511901":           "delivered",
	"511902":           "delivered",
	"511903":           "delivered",
	"600103":           "delivered",
	"600104":           "delivered",
	"701901":           "delivered",
	"701902":           "delivered",
	"230403":           "returned",
	"230408":           "returned",
}

// MapStatus translates a DPD shipment status to the corresponding
// OpenOMS shipment_status string.
func MapStatus(dpdStatus string) (openomsStatus string, ok bool) {
	openomsStatus, ok = statusMapping[dpdStatus]
	return
}
