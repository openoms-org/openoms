package engine

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

var (
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrUnknownStatus     = errors.New("unknown status")
)

// allowedOrderTransitions maps each order status to its allowed target statuses.
var allowedOrderTransitions = map[OrderStatus][]OrderStatus{
	OrderNew:            {OrderConfirmed, OrderCancelled, OrderOnHold},
	OrderConfirmed:      {OrderProcessing, OrderCancelled, OrderOnHold},
	OrderProcessing:     {OrderReadyToShip, OrderCancelled, OrderOnHold},
	OrderReadyToShip:    {OrderShipped, OrderCancelled, OrderOnHold},
	OrderShipped:        {OrderInTransit, OrderDelivered, OrderRefunded},
	OrderInTransit:      {OrderOutForDelivery, OrderDelivered, OrderRefunded},
	OrderOutForDelivery: {OrderDelivered, OrderRefunded},
	OrderDelivered:      {OrderCompleted, OrderRefunded},
	OrderCompleted:      {OrderRefunded},
	OrderOnHold:         {OrderConfirmed, OrderProcessing, OrderCancelled},
	OrderCancelled:      {OrderRefunded},
	OrderRefunded:       {},
}

// allowedShipmentTransitions maps each shipment status to its allowed targets.
var allowedShipmentTransitions = map[ShipmentStatus][]ShipmentStatus{
	ShipmentCreated:        {ShipmentLabelReady, ShipmentFailed},
	ShipmentLabelReady:     {ShipmentPickedUp, ShipmentFailed},
	ShipmentPickedUp:       {ShipmentInTransit, ShipmentFailed},
	ShipmentInTransit:      {ShipmentOutForDelivery, ShipmentDelivered, ShipmentReturned, ShipmentFailed},
	ShipmentOutForDelivery: {ShipmentDelivered, ShipmentReturned, ShipmentFailed},
	ShipmentDelivered:      {ShipmentReturned},
	ShipmentReturned:       {},
	ShipmentFailed:         {ShipmentCreated},
}

// CanTransitionOrder reports whether an order can move from one status to another.
func CanTransitionOrder(from, to OrderStatus) bool {
	targets, ok := allowedOrderTransitions[from]
	if !ok {
		return false
	}
	return slices.Contains(targets, to)
}

// CanTransitionShipment reports whether a shipment can move from one status to another.
func CanTransitionShipment(from, to ShipmentStatus) bool {
	targets, ok := allowedShipmentTransitions[from]
	if !ok {
		return false
	}
	return slices.Contains(targets, to)
}

// TransitionOrder validates and performs an order status transition.
// It returns a result with the domain event and any side effects.
func TransitionOrder(from, to OrderStatus, now time.Time) (OrderTransitionResult, error) {
	if !from.Valid() {
		return OrderTransitionResult{}, fmt.Errorf("%w: %q", ErrUnknownStatus, from)
	}
	if !to.Valid() {
		return OrderTransitionResult{}, fmt.Errorf("%w: %q", ErrUnknownStatus, to)
	}
	if !CanTransitionOrder(from, to) {
		return OrderTransitionResult{}, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, from, to)
	}

	result := OrderTransitionResult{
		From: from,
		To:   to,
		Event: Event{
			Type:      EventOrderStatusChanged,
			Timestamp: now,
			Payload:   OrderStatusChanged{From: from, To: to},
		},
	}

	switch to {
	case OrderShipped:
		result.SetShippedAt = &now
	case OrderDelivered:
		result.SetDeliveredAt = &now
	}

	return result, nil
}

// TransitionShipment validates and performs a shipment status transition.
func TransitionShipment(from, to ShipmentStatus, now time.Time) (ShipmentTransitionResult, error) {
	if !from.Valid() {
		return ShipmentTransitionResult{}, fmt.Errorf("%w: %q", ErrUnknownStatus, from)
	}
	if !to.Valid() {
		return ShipmentTransitionResult{}, fmt.Errorf("%w: %q", ErrUnknownStatus, to)
	}
	if !CanTransitionShipment(from, to) {
		return ShipmentTransitionResult{}, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, from, to)
	}

	return ShipmentTransitionResult{
		From: from,
		To:   to,
		Event: Event{
			Type:      EventShipmentStatusChanged,
			Timestamp: now,
			Payload:   ShipmentStatusChanged{From: from, To: to},
		},
	}, nil
}
