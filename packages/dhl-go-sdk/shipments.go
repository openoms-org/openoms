package dhl

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
)

// ShipmentService handles DHL shipment-related API operations.
type ShipmentService struct {
	client *Client
}

// Create creates a new shipment in the DHL system.
func (s *ShipmentService) Create(ctx context.Context, req *CreateShipmentRequest) (*ShipmentResponse, error) {
	var resp ShipmentResponse
	if err := s.client.do(ctx, http.MethodPost, "/v1/shipments", req, &resp); err != nil {
		return nil, fmt.Errorf("dhl: create shipment: %w", err)
	}
	return &resp, nil
}

// GetLabel retrieves the shipping label for a shipment. Returns raw PDF bytes.
func (s *ShipmentService) GetLabel(ctx context.Context, shipmentID string) ([]byte, error) {
	path := fmt.Sprintf("/v1/shipments/%s/label", url.PathEscape(shipmentID))
	var resp LabelResponse
	if err := s.client.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("dhl: get label: %w", err)
	}

	data, err := base64.StdEncoding.DecodeString(resp.LabelData)
	if err != nil {
		return nil, fmt.Errorf("dhl: decode label data: %w", err)
	}
	return data, nil
}

// GetTracking retrieves tracking events for a tracking number.
func (s *ShipmentService) GetTracking(ctx context.Context, trackingNumber string) (*TrackingResponse, error) {
	path := fmt.Sprintf("/v1/tracking?trackingNumber=%s", url.QueryEscape(trackingNumber))
	var resp TrackingResponse
	if err := s.client.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("dhl: get tracking: %w", err)
	}
	return &resp, nil
}

// Cancel cancels a shipment by its ID.
func (s *ShipmentService) Cancel(ctx context.Context, shipmentID string) error {
	path := fmt.Sprintf("/v1/shipments/%s", url.PathEscape(shipmentID))
	if err := s.client.do(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("dhl: cancel shipment: %w", err)
	}
	return nil
}
