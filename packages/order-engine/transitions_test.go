package engine

import (
	"errors"
	"testing"
	"time"
)

func TestAllowedOrderTransitions(t *testing.T) {
	allowed := []struct {
		from, to OrderStatus
	}{
		{OrderNew, OrderConfirmed},
		{OrderNew, OrderCancelled},
		{OrderNew, OrderOnHold},
		{OrderConfirmed, OrderProcessing},
		{OrderConfirmed, OrderCancelled},
		{OrderConfirmed, OrderOnHold},
		{OrderProcessing, OrderReadyToShip},
		{OrderProcessing, OrderCancelled},
		{OrderProcessing, OrderOnHold},
		{OrderReadyToShip, OrderShipped},
		{OrderReadyToShip, OrderCancelled},
		{OrderReadyToShip, OrderOnHold},
		{OrderShipped, OrderInTransit},
		{OrderShipped, OrderDelivered},
		{OrderShipped, OrderRefunded},
		{OrderInTransit, OrderOutForDelivery},
		{OrderInTransit, OrderDelivered},
		{OrderInTransit, OrderRefunded},
		{OrderOutForDelivery, OrderDelivered},
		{OrderOutForDelivery, OrderRefunded},
		{OrderDelivered, OrderCompleted},
		{OrderDelivered, OrderRefunded},
		{OrderCompleted, OrderRefunded},
		{OrderOnHold, OrderConfirmed},
		{OrderOnHold, OrderProcessing},
		{OrderOnHold, OrderCancelled},
		{OrderCancelled, OrderRefunded},
	}

	for _, tc := range allowed {
		if !CanTransitionOrder(tc.from, tc.to) {
			t.Errorf("CanTransitionOrder(%s, %s) = false; want true", tc.from, tc.to)
		}
	}
}

func TestDisallowedOrderTransitions(t *testing.T) {
	disallowed := []struct {
		from, to OrderStatus
	}{
		{OrderNew, OrderDelivered},
		{OrderNew, OrderShipped},
		{OrderConfirmed, OrderDelivered},
		{OrderRefunded, OrderNew},
		{OrderRefunded, OrderCancelled},
		{OrderCompleted, OrderNew},
		{OrderDelivered, OrderNew},
		{OrderShipped, OrderNew},
		{OrderCancelled, OrderNew},
	}

	for _, tc := range disallowed {
		if CanTransitionOrder(tc.from, tc.to) {
			t.Errorf("CanTransitionOrder(%s, %s) = true; want false", tc.from, tc.to)
		}
	}
}

func TestAllowedShipmentTransitions(t *testing.T) {
	allowed := []struct {
		from, to ShipmentStatus
	}{
		{ShipmentCreated, ShipmentLabelReady},
		{ShipmentCreated, ShipmentFailed},
		{ShipmentLabelReady, ShipmentPickedUp},
		{ShipmentLabelReady, ShipmentFailed},
		{ShipmentPickedUp, ShipmentInTransit},
		{ShipmentPickedUp, ShipmentFailed},
		{ShipmentInTransit, ShipmentOutForDelivery},
		{ShipmentInTransit, ShipmentDelivered},
		{ShipmentInTransit, ShipmentReturned},
		{ShipmentInTransit, ShipmentFailed},
		{ShipmentOutForDelivery, ShipmentDelivered},
		{ShipmentOutForDelivery, ShipmentReturned},
		{ShipmentOutForDelivery, ShipmentFailed},
		{ShipmentDelivered, ShipmentReturned},
		{ShipmentFailed, ShipmentCreated},
	}

	for _, tc := range allowed {
		if !CanTransitionShipment(tc.from, tc.to) {
			t.Errorf("CanTransitionShipment(%s, %s) = false; want true", tc.from, tc.to)
		}
	}
}

func TestDisallowedShipmentTransitions(t *testing.T) {
	disallowed := []struct {
		from, to ShipmentStatus
	}{
		{ShipmentCreated, ShipmentDelivered},
		{ShipmentReturned, ShipmentCreated},
		{ShipmentReturned, ShipmentDelivered},
		{ShipmentDelivered, ShipmentCreated},
		{ShipmentFailed, ShipmentDelivered},
	}

	for _, tc := range disallowed {
		if CanTransitionShipment(tc.from, tc.to) {
			t.Errorf("CanTransitionShipment(%s, %s) = true; want false", tc.from, tc.to)
		}
	}
}

func TestTerminalStates(t *testing.T) {
	// Refunded is terminal for orders
	for to := range knownOrderStatuses {
		if CanTransitionOrder(OrderRefunded, to) {
			t.Errorf("OrderRefunded should be terminal, but can transition to %s", to)
		}
	}

	// Returned is terminal for shipments
	for to := range knownShipmentStatuses {
		if CanTransitionShipment(ShipmentReturned, to) {
			t.Errorf("ShipmentReturned should be terminal, but can transition to %s", to)
		}
	}
}

func TestTransitionOrderSuccess(t *testing.T) {
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	result, err := TransitionOrder(OrderNew, OrderConfirmed, now)
	if err != nil {
		t.Fatalf("TransitionOrder(new, confirmed) error: %v", err)
	}
	if result.From != OrderNew || result.To != OrderConfirmed {
		t.Errorf("result From/To = %s/%s; want new/confirmed", result.From, result.To)
	}
	if result.Event.Type != EventOrderStatusChanged {
		t.Errorf("event type = %s; want %s", result.Event.Type, EventOrderStatusChanged)
	}
	payload, ok := result.Event.Payload.(OrderStatusChanged)
	if !ok {
		t.Fatal("event payload is not OrderStatusChanged")
	}
	if payload.From != OrderNew || payload.To != OrderConfirmed {
		t.Errorf("payload From/To = %s/%s; want new/confirmed", payload.From, payload.To)
	}
	if result.SetShippedAt != nil {
		t.Error("SetShippedAt should be nil for new->confirmed")
	}
	if result.SetDeliveredAt != nil {
		t.Error("SetDeliveredAt should be nil for new->confirmed")
	}
}

func TestTransitionOrderSideEffects(t *testing.T) {
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	// Transition to shipped sets SetShippedAt
	result, err := TransitionOrder(OrderReadyToShip, OrderShipped, now)
	if err != nil {
		t.Fatalf("TransitionOrder(ready_to_ship, shipped) error: %v", err)
	}
	if result.SetShippedAt == nil || !result.SetShippedAt.Equal(now) {
		t.Errorf("SetShippedAt = %v; want %v", result.SetShippedAt, now)
	}
	if result.SetDeliveredAt != nil {
		t.Error("SetDeliveredAt should be nil for shipped transition")
	}

	// Transition to delivered sets SetDeliveredAt
	result, err = TransitionOrder(OrderShipped, OrderDelivered, now)
	if err != nil {
		t.Fatalf("TransitionOrder(shipped, delivered) error: %v", err)
	}
	if result.SetDeliveredAt == nil || !result.SetDeliveredAt.Equal(now) {
		t.Errorf("SetDeliveredAt = %v; want %v", result.SetDeliveredAt, now)
	}
	if result.SetShippedAt != nil {
		t.Error("SetShippedAt should be nil for delivered transition")
	}
}

func TestTransitionOrderInvalid(t *testing.T) {
	now := time.Now()

	_, err := TransitionOrder(OrderNew, OrderDelivered, now)
	if !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("TransitionOrder(new, delivered) error = %v; want ErrInvalidTransition", err)
	}
}

func TestTransitionOrderUnknownStatus(t *testing.T) {
	now := time.Now()

	_, err := TransitionOrder(OrderStatus("bogus"), OrderNew, now)
	if !errors.Is(err, ErrUnknownStatus) {
		t.Errorf("TransitionOrder(bogus, new) error = %v; want ErrUnknownStatus", err)
	}

	_, err = TransitionOrder(OrderNew, OrderStatus("bogus"), now)
	if !errors.Is(err, ErrUnknownStatus) {
		t.Errorf("TransitionOrder(new, bogus) error = %v; want ErrUnknownStatus", err)
	}
}

func TestTransitionShipmentSuccess(t *testing.T) {
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	result, err := TransitionShipment(ShipmentCreated, ShipmentLabelReady, now)
	if err != nil {
		t.Fatalf("TransitionShipment(created, label_ready) error: %v", err)
	}
	if result.From != ShipmentCreated || result.To != ShipmentLabelReady {
		t.Errorf("result From/To = %s/%s; want created/label_ready", result.From, result.To)
	}
	if result.Event.Type != EventShipmentStatusChanged {
		t.Errorf("event type = %s; want %s", result.Event.Type, EventShipmentStatusChanged)
	}
	payload, ok := result.Event.Payload.(ShipmentStatusChanged)
	if !ok {
		t.Fatal("event payload is not ShipmentStatusChanged")
	}
	if payload.From != ShipmentCreated || payload.To != ShipmentLabelReady {
		t.Errorf("payload From/To = %s/%s; want created/label_ready", payload.From, payload.To)
	}
}

func TestTransitionShipmentInvalid(t *testing.T) {
	now := time.Now()

	_, err := TransitionShipment(ShipmentCreated, ShipmentDelivered, now)
	if !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("TransitionShipment(created, delivered) error = %v; want ErrInvalidTransition", err)
	}
}

func TestTransitionShipmentUnknownStatus(t *testing.T) {
	now := time.Now()

	_, err := TransitionShipment(ShipmentStatus("bogus"), ShipmentCreated, now)
	if !errors.Is(err, ErrUnknownStatus) {
		t.Errorf("TransitionShipment(bogus, created) error = %v; want ErrUnknownStatus", err)
	}

	_, err = TransitionShipment(ShipmentCreated, ShipmentStatus("bogus"), now)
	if !errors.Is(err, ErrUnknownStatus) {
		t.Errorf("TransitionShipment(created, bogus) error = %v; want ErrUnknownStatus", err)
	}
}

func TestCanTransitionOrderUnknownFrom(t *testing.T) {
	if CanTransitionOrder(OrderStatus("unknown"), OrderNew) {
		t.Error("CanTransitionOrder with unknown 'from' should return false")
	}
}

func TestCanTransitionShipmentUnknownFrom(t *testing.T) {
	if CanTransitionShipment(ShipmentStatus("unknown"), ShipmentCreated) {
		t.Error("CanTransitionShipment with unknown 'from' should return false")
	}
}
