package orlenpaczka

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
)

// ShipmentService handles Orlen Paczka shipment-related API operations.
type ShipmentService struct {
	client *Client
}

// Create creates a new shipment in the Orlen Paczka system.
func (s *ShipmentService) Create(ctx context.Context, req *CreateShipmentRequest) (*ShipmentResponse, error) {
	var resp ShipmentResponse
	if err := s.client.do(ctx, http.MethodPost, "/v1/shipments", req, &resp); err != nil {
		return nil, fmt.Errorf("orlenpaczka: create shipment: %w", err)
	}
	return &resp, nil
}

// GetLabel retrieves the shipping label for a shipment. Returns raw PDF bytes.
func (s *ShipmentService) GetLabel(ctx context.Context, shipmentID string) ([]byte, error) {
	path := fmt.Sprintf("/v1/shipments/%s/label", url.PathEscape(shipmentID))
	var resp LabelResponse
	if err := s.client.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("orlenpaczka: get label: %w", err)
	}

	data, err := base64.StdEncoding.DecodeString(resp.LabelData)
	if err != nil {
		return nil, fmt.Errorf("orlenpaczka: decode label data: %w", err)
	}
	return data, nil
}

// GetTracking retrieves tracking events for a tracking number.
func (s *ShipmentService) GetTracking(ctx context.Context, trackingNumber string) (*TrackingResponse, error) {
	path := fmt.Sprintf("/v1/tracking?trackingNumber=%s", url.QueryEscape(trackingNumber))
	var resp TrackingResponse
	if err := s.client.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("orlenpaczka: get tracking: %w", err)
	}
	return &resp, nil
}

// Cancel cancels a shipment by its ID.
func (s *ShipmentService) Cancel(ctx context.Context, shipmentID string) error {
	path := fmt.Sprintf("/v1/shipments/%s", url.PathEscape(shipmentID))
	if err := s.client.do(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("orlenpaczka: cancel shipment: %w", err)
	}
	return nil
}

// PointService handles Orlen Paczka point/pickup point search operations.
type PointService struct {
	client *Client
}

// SearchPoints searches for Orlen Paczka pickup points by city or query string.
func (s *PointService) SearchPoints(ctx context.Context, query string, limit int) (*PointSearchResponse, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	path := fmt.Sprintf("/v1/points?query=%s&limit=%d",
		url.QueryEscape(query),
		limit,
	)

	var resp PointSearchResponse
	if err := s.client.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("orlenpaczka: search points: %w", err)
	}
	return &resp, nil
}
