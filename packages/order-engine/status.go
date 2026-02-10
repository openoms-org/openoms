package engine

import "fmt"

// OrderStatus represents the lifecycle state of an order.
type OrderStatus string

const (
	OrderNew            OrderStatus = "new"
	OrderConfirmed      OrderStatus = "confirmed"
	OrderProcessing     OrderStatus = "processing"
	OrderReadyToShip    OrderStatus = "ready_to_ship"
	OrderShipped        OrderStatus = "shipped"
	OrderInTransit      OrderStatus = "in_transit"
	OrderOutForDelivery OrderStatus = "out_for_delivery"
	OrderDelivered      OrderStatus = "delivered"
	OrderCompleted      OrderStatus = "completed"
	OrderCancelled      OrderStatus = "cancelled"
	OrderRefunded       OrderStatus = "refunded"
	OrderOnHold         OrderStatus = "on_hold"
)

var knownOrderStatuses = map[OrderStatus]bool{
	OrderNew:            true,
	OrderConfirmed:      true,
	OrderProcessing:     true,
	OrderReadyToShip:    true,
	OrderShipped:        true,
	OrderInTransit:      true,
	OrderOutForDelivery: true,
	OrderDelivered:      true,
	OrderCompleted:      true,
	OrderCancelled:      true,
	OrderRefunded:       true,
	OrderOnHold:         true,
}

// Valid returns true if the status is a recognized order status.
func (s OrderStatus) Valid() bool {
	return knownOrderStatuses[s]
}

// ParseOrderStatus converts a string to OrderStatus, returning ErrUnknownStatus
// if the value is not recognized.
func ParseOrderStatus(s string) (OrderStatus, error) {
	os := OrderStatus(s)
	if !os.Valid() {
		return "", fmt.Errorf("%w: %q", ErrUnknownStatus, s)
	}
	return os, nil
}

// ShipmentStatus represents the lifecycle state of a shipment.
type ShipmentStatus string

const (
	ShipmentCreated        ShipmentStatus = "created"
	ShipmentLabelReady     ShipmentStatus = "label_ready"
	ShipmentPickedUp       ShipmentStatus = "picked_up"
	ShipmentInTransit      ShipmentStatus = "in_transit"
	ShipmentOutForDelivery ShipmentStatus = "out_for_delivery"
	ShipmentDelivered      ShipmentStatus = "delivered"
	ShipmentReturned       ShipmentStatus = "returned"
	ShipmentFailed         ShipmentStatus = "failed"
)

var knownShipmentStatuses = map[ShipmentStatus]bool{
	ShipmentCreated:        true,
	ShipmentLabelReady:     true,
	ShipmentPickedUp:       true,
	ShipmentInTransit:      true,
	ShipmentOutForDelivery: true,
	ShipmentDelivered:      true,
	ShipmentReturned:       true,
	ShipmentFailed:         true,
}

// Valid returns true if the status is a recognized shipment status.
func (s ShipmentStatus) Valid() bool {
	return knownShipmentStatuses[s]
}

// ParseShipmentStatus converts a string to ShipmentStatus, returning
// ErrUnknownStatus if the value is not recognized.
func ParseShipmentStatus(s string) (ShipmentStatus, error) {
	ss := ShipmentStatus(s)
	if !ss.Valid() {
		return "", fmt.Errorf("%w: %q", ErrUnknownStatus, s)
	}
	return ss, nil
}
