package engine

import "time"

// EventType identifies the kind of domain event.
type EventType string

const (
	EventOrderStatusChanged    EventType = "order.status_changed"
	EventShipmentStatusChanged EventType = "shipment.status_changed"
)

// Event represents a domain event emitted by a state transition.
type Event struct {
	Type      EventType
	Timestamp time.Time
	Payload   any
}

// OrderStatusChanged is the payload for order transition events.
type OrderStatusChanged struct {
	From OrderStatus
	To   OrderStatus
}

// ShipmentStatusChanged is the payload for shipment transition events.
type ShipmentStatusChanged struct {
	From ShipmentStatus
	To   ShipmentStatus
}

// OrderTransitionResult is returned by TransitionOrder on success.
type OrderTransitionResult struct {
	From           OrderStatus
	To             OrderStatus
	Event          Event
	SetShippedAt   *time.Time // non-nil when transitioning to shipped
	SetDeliveredAt *time.Time // non-nil when transitioning to delivered
}

// ShipmentTransitionResult is returned by TransitionShipment on success.
type ShipmentTransitionResult struct {
	From  ShipmentStatus
	To    ShipmentStatus
	Event Event
}
