package engine

import (
	"errors"
	"testing"
)

func TestOrderStatusValid(t *testing.T) {
	valid := []OrderStatus{
		OrderNew, OrderConfirmed, OrderProcessing, OrderReadyToShip,
		OrderShipped, OrderInTransit, OrderOutForDelivery, OrderDelivered,
		OrderCompleted, OrderCancelled, OrderRefunded, OrderOnHold,
	}
	for _, s := range valid {
		if !s.Valid() {
			t.Errorf("OrderStatus %q should be valid", s)
		}
	}
	if OrderStatus("bogus").Valid() {
		t.Error("OrderStatus 'bogus' should not be valid")
	}
	if OrderStatus("").Valid() {
		t.Error("empty OrderStatus should not be valid")
	}
}

func TestShipmentStatusValid(t *testing.T) {
	valid := []ShipmentStatus{
		ShipmentCreated, ShipmentLabelReady, ShipmentPickedUp, ShipmentInTransit,
		ShipmentOutForDelivery, ShipmentDelivered, ShipmentReturned, ShipmentFailed,
	}
	for _, s := range valid {
		if !s.Valid() {
			t.Errorf("ShipmentStatus %q should be valid", s)
		}
	}
	if ShipmentStatus("bogus").Valid() {
		t.Error("ShipmentStatus 'bogus' should not be valid")
	}
}

func TestParseOrderStatus(t *testing.T) {
	os, err := ParseOrderStatus("new")
	if err != nil || os != OrderNew {
		t.Errorf("ParseOrderStatus('new') = %q, %v; want %q, nil", os, err, OrderNew)
	}

	_, err = ParseOrderStatus("nonexistent")
	if !errors.Is(err, ErrUnknownStatus) {
		t.Errorf("ParseOrderStatus('nonexistent') error = %v; want ErrUnknownStatus", err)
	}
}

func TestParseShipmentStatus(t *testing.T) {
	ss, err := ParseShipmentStatus("created")
	if err != nil || ss != ShipmentCreated {
		t.Errorf("ParseShipmentStatus('created') = %q, %v; want %q, nil", ss, err, ShipmentCreated)
	}

	_, err = ParseShipmentStatus("nonexistent")
	if !errors.Is(err, ErrUnknownStatus) {
		t.Errorf("ParseShipmentStatus('nonexistent') error = %v; want ErrUnknownStatus", err)
	}
}
