package orlenpaczka

// statusMapping maps Orlen Paczka shipment statuses to OpenOMS shipment_status strings.
var statusMapping = map[string]string{
	"CREATED":               "created",
	"LABEL_READY":           "label_ready",
	"PICKED_UP":             "picked_up",
	"IN_TRANSIT":            "in_transit",
	"AT_DESTINATION_POINT":  "out_for_delivery",
	"READY_FOR_PICKUP":      "out_for_delivery",
	"DELIVERED":             "delivered",
	"PICKED_UP_BY_RECEIVER": "delivered",
	"RETURNED":              "returned",
	"RETURN_IN_TRANSIT":     "returned",
	"FAILED":                "failed",
	"CANCELLED":             "failed",
	"EXPIRED":               "failed",
}

// MapStatus translates an Orlen Paczka shipment status to the corresponding
// OpenOMS shipment_status string.
func MapStatus(orlenStatus string) (openomsStatus string, ok bool) {
	openomsStatus, ok = statusMapping[orlenStatus]
	return
}
