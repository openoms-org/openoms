package inpost

// statusMapping maps InPost shipment statuses to OpenOMS shipment_status strings.
var statusMapping = map[string]string{
	"created":                    "created",
	"offers_prepared":            "created",
	"offer_selected":             "created",
	"confirmed":                  "label_ready",
	"dispatched_by_sender":       "picked_up",
	"collected_from_sender":      "picked_up",
	"taken_by_courier":           "in_transit",
	"adopted_at_source_branch":   "in_transit",
	"sent_from_source_branch":    "in_transit",
	"adopted_at_sorting_center":  "in_transit",
	"sent_from_sorting_center":   "in_transit",
	"adopted_at_target_branch":   "in_transit",
	"out_for_delivery":           "out_for_delivery",
	"ready_to_pickup":            "out_for_delivery",
	"delivered":                  "delivered",
	"picked_up":                  "delivered",
	"returned_to_sender":         "returned",
	"avizo":                      "out_for_delivery",
	"missing":                    "failed",
	"claim_rejected":             "failed",
}

// MapStatus translates an InPost shipment status to the corresponding
// OpenOMS shipment_status string.
func MapStatus(inpostStatus string) (openomsStatus string, ok bool) {
	openomsStatus, ok = statusMapping[inpostStatus]
	return
}
