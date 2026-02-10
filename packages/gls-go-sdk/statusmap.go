package gls

// statusMapping maps GLS shipment statuses to OpenOMS shipment_status strings.
var statusMapping = map[string]string{
	"PREADVICE":   "pending",
	"INTRANSIT":   "in_transit",
	"INWAREHOUSE": "in_transit",
	"INDELIVERY":  "out_for_delivery",
	"DELIVERED":   "delivered",
	"RETURNED":    "returned",
}

// MapStatus translates a GLS shipment status to the corresponding
// OpenOMS shipment_status string.
func MapStatus(glsStatus string) (openomsStatus string, ok bool) {
	openomsStatus, ok = statusMapping[glsStatus]
	return
}
