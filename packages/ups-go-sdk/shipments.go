package ups

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
)

// ShipmentService handles UPS shipment-related API operations.
type ShipmentService struct {
	client *Client
}

// Create creates a new shipment in the UPS system.
func (s *ShipmentService) Create(ctx context.Context, req *ShipmentRequest) (*ShipmentResponse, error) {
	var resp ShipmentResponse
	if err := s.client.do(ctx, http.MethodPost, "/shipments/v2409/ship", req, &resp); err != nil {
		return nil, fmt.Errorf("ups: create shipment: %w", err)
	}
	return &resp, nil
}

// GetLabel retrieves the shipping label for a shipment. Returns raw PDF bytes.
func (s *ShipmentService) GetLabel(ctx context.Context, shipmentID string) ([]byte, error) {
	path := fmt.Sprintf("/shipments/v2409/%s/label", url.PathEscape(shipmentID))

	var resp struct {
		LabelImage string `json:"labelImage"`
	}
	if err := s.client.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("ups: get label: %w", err)
	}

	data, err := base64.StdEncoding.DecodeString(resp.LabelImage)
	if err != nil {
		return nil, fmt.Errorf("ups: decode label data: %w", err)
	}
	return data, nil
}

// GetTracking retrieves tracking events for a tracking number.
func (s *ShipmentService) GetTracking(ctx context.Context, trackingNumber string) (*TrackingResponse, error) {
	path := fmt.Sprintf("/track/v1/details/%s", url.PathEscape(trackingNumber))
	var resp TrackingResponse
	if err := s.client.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("ups: get tracking: %w", err)
	}
	return &resp, nil
}

// Cancel cancels a shipment by its ID.
func (s *ShipmentService) Cancel(ctx context.Context, shipmentID string) error {
	path := fmt.Sprintf("/shipments/v2409/%s", url.PathEscape(shipmentID))
	if err := s.client.do(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("ups: cancel shipment: %w", err)
	}
	return nil
}
