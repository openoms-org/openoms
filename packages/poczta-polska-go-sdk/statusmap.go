package poczta

// statusMapping maps Poczta Polska shipment statuses to OpenOMS shipment_status strings.
var statusMapping = map[string]string{
	"UTWORZONA":              "created",
	"NADANA":                "picked_up",
	"W_TRANSPORCIE":         "in_transit",
	"W_DORECZENIU":          "out_for_delivery",
	"DORECZONA":             "delivered",
	"ZWROCONA":              "returned",
	"NIEDORECZONA":          "failed",
	"OCZEKUJE_NA_ODBIOR":    "out_for_delivery",
	"WYDANA_DO_DORECZENIA":  "out_for_delivery",
	"PRZYJETA_W_PLACOWCE":   "in_transit",
	"W_SORTOWNI":            "in_transit",
	"CREATED":               "created",
	"SENT":                  "picked_up",
	"IN_TRANSIT":            "in_transit",
	"OUT_FOR_DELIVERY":      "out_for_delivery",
	"DELIVERED":             "delivered",
	"RETURNED":              "returned",
	"FAILED":                "failed",
	"AWAITING_PICKUP":       "out_for_delivery",
	"LABEL_CREATED":         "label_ready",
}

// MapStatus translates a Poczta Polska shipment status to the corresponding
// OpenOMS shipment_status string.
func MapStatus(ppStatus string) (openomsStatus string, ok bool) {
	openomsStatus, ok = statusMapping[ppStatus]
	return
}
