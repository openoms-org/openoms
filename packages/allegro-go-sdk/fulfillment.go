package allegro

import (
	"context"
	"fmt"
)

// FulfillmentService handles communication with the fulfillment and shipment endpoints.
type FulfillmentService struct {
	client *Client
}

// UpdateStatus updates the fulfillment status of an order.
// Valid statuses: PROCESSING, READY_FOR_SHIPMENT, SENT, PICKED_UP, CANCELLED.
func (s *FulfillmentService) UpdateStatus(ctx context.Context, orderID, status string) error {
	body := FulfillmentUpdate{Status: status}
	return s.client.do(ctx, "PUT", fmt.Sprintf("/order/checkout-forms/%s/fulfillment", orderID), body, nil)
}

// AddShipment adds a shipment (tracking number) to an order.
func (s *FulfillmentService) AddShipment(ctx context.Context, orderID string, shipment ShipmentInput) error {
	return s.client.do(ctx, "POST", fmt.Sprintf("/order/checkout-forms/%s/shipments", orderID), shipment, nil)
}

// ListShipments retrieves shipments associated with an order.
func (s *FulfillmentService) ListShipments(ctx context.Context, orderID string) ([]OrderShipment, error) {
	var result ShipmentList
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/order/checkout-forms/%s/shipments", orderID), nil, &result); err != nil {
		return nil, err
	}
	return result.Shipments, nil
}

// ListCarriers retrieves the list of available shipping carriers.
func (s *FulfillmentService) ListCarriers(ctx context.Context) ([]Carrier, error) {
	var result CarrierList
	if err := s.client.do(ctx, "GET", "/order/carriers", nil, &result); err != nil {
		return nil, err
	}
	return result.Carriers, nil
}
