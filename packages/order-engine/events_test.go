package engine

import (
	"testing"
	"time"
)

func TestEventTypeConstants(t *testing.T) {
	if EventOrderStatusChanged != "order.status_changed" {
		t.Errorf("EventOrderStatusChanged = %q; want %q", EventOrderStatusChanged, "order.status_changed")
	}
	if EventShipmentStatusChanged != "shipment.status_changed" {
		t.Errorf("EventShipmentStatusChanged = %q; want %q", EventShipmentStatusChanged, "shipment.status_changed")
	}
}

func TestOrderTransitionResultConstruction(t *testing.T) {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	result := OrderTransitionResult{
		From: OrderNew,
		To:   OrderConfirmed,
		Event: Event{
			Type:      EventOrderStatusChanged,
			Timestamp: now,
			Payload:   OrderStatusChanged{From: OrderNew, To: OrderConfirmed},
		},
	}
	if result.From != OrderNew {
		t.Errorf("From = %s; want new", result.From)
	}
	if !result.Event.Timestamp.Equal(now) {
		t.Errorf("Timestamp = %v; want %v", result.Event.Timestamp, now)
	}
	if result.SetShippedAt != nil || result.SetDeliveredAt != nil {
		t.Error("side effect pointers should be nil by default")
	}
}

func TestShipmentTransitionResultConstruction(t *testing.T) {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	result := ShipmentTransitionResult{
		From: ShipmentCreated,
		To:   ShipmentLabelReady,
		Event: Event{
			Type:      EventShipmentStatusChanged,
			Timestamp: now,
			Payload:   ShipmentStatusChanged{From: ShipmentCreated, To: ShipmentLabelReady},
		},
	}
	if result.From != ShipmentCreated {
		t.Errorf("From = %s; want created", result.From)
	}
	if result.Event.Type != EventShipmentStatusChanged {
		t.Errorf("Type = %s; want %s", result.Event.Type, EventShipmentStatusChanged)
	}
}
