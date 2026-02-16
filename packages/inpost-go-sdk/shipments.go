package inpost

import (
	"context"
	"fmt"
	"net/http"
)

// ShipmentService handles shipment-related API operations.
type ShipmentService struct {
	client *Client
}

// Create creates a new shipment in the InPost system.
func (s *ShipmentService) Create(ctx context.Context, req *CreateShipmentRequest) (*Shipment, error) {
	path := fmt.Sprintf("/v1/organizations/%s/shipments", s.client.orgID)
	var shipment Shipment
	if err := s.client.do(ctx, http.MethodPost, path, req, &shipment); err != nil {
		return nil, err
	}
	return &shipment, nil
}

// Buy confirms (buys/pays for) a shipment so that labels can be generated.
// InPost flow: Create → Buy → Get Label.
// In simplified mode (service specified), offerID can be 0 to auto-select.
func (s *ShipmentService) Buy(ctx context.Context, id int64, offerID int64) (*Shipment, error) {
	path := fmt.Sprintf("/v1/shipments/%d/buy", id)
	var body any
	if offerID > 0 {
		body = map[string]int64{"offer_id": offerID}
	}
	var shipment Shipment
	if err := s.client.do(ctx, http.MethodPost, path, body, &shipment); err != nil {
		return nil, err
	}
	return &shipment, nil
}

// Get retrieves a shipment by its ID.
func (s *ShipmentService) Get(ctx context.Context, id int64) (*Shipment, error) {
	path := fmt.Sprintf("/v1/shipments/%d", id)
	var shipment Shipment
	if err := s.client.do(ctx, http.MethodGet, path, nil, &shipment); err != nil {
		return nil, err
	}
	return &shipment, nil
}

// Delete removes a shipment that has not yet been confirmed (bought).
// InPost ShipX only allows deletion of shipments in pre-confirmed statuses
// (e.g. "created", "offers_prepared", "offer_selected").
func (s *ShipmentService) Delete(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/v1/shipments/%d", id)
	if err := s.client.do(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return err
	}
	return nil
}
