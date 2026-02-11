package fedex

// statusMapping maps FedEx event type codes to OpenOMS shipment_status strings.
var statusMapping = map[string]string{
	"OC":  "created",          // Shipment information sent to FedEx
	"PU":  "picked_up",        // Picked up
	"PL":  "label_ready",      // Label created
	"IT":  "in_transit",       // In transit
	"IX":  "in_transit",       // In transit (international)
	"AR":  "in_transit",       // Arrived at FedEx location
	"DP":  "in_transit",       // Departed FedEx location
	"CC":  "in_transit",       // Customs cleared
	"CD":  "in_transit",       // Clearance delay
	"OD":  "out_for_delivery", // On FedEx vehicle for delivery
	"DL":  "delivered",        // Delivered
	"DE":  "failed",           // Delivery exception
	"CA":  "failed",           // Cancelled
	"RS":  "returned",         // Return to shipper
	"SE":  "failed",           // Shipment exception
	"HP":  "out_for_delivery", // Ready for pickup at FedEx location
}

// MapStatus translates a FedEx event type code to the corresponding
// OpenOMS shipment_status string.
func MapStatus(fedexStatus string) (openomsStatus string, ok bool) {
	openomsStatus, ok = statusMapping[fedexStatus]
	return
}
