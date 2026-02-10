package gls

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
)

// ShipmentService handles GLS shipment-related API operations.
type ShipmentService struct {
	client *Client
}

// Create creates new parcels in the GLS system.
func (s *ShipmentService) Create(ctx context.Context, req *CreateParcelRequest) (*CreateParcelResponse, error) {
	var resp CreateParcelResponse
	if err := s.client.do(ctx, http.MethodPost, "/parcels", req, &resp); err != nil {
		return nil, fmt.Errorf("gls: create parcel: %w", err)
	}
	return &resp, nil
}

// GetLabel retrieves the shipping label for a parcel. Returns raw PDF bytes.
func (s *ShipmentService) GetLabel(ctx context.Context, parcelID string) ([]byte, error) {
	path := fmt.Sprintf("/parcels/%s/label", url.PathEscape(parcelID))
	var resp LabelResponse
	if err := s.client.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("gls: get label: %w", err)
	}

	data, err := base64.StdEncoding.DecodeString(resp.LabelData)
	if err != nil {
		return nil, fmt.Errorf("gls: decode label data: %w", err)
	}
	return data, nil
}

// GetTracking retrieves tracking events for a tracking ID.
func (s *ShipmentService) GetTracking(ctx context.Context, trackID string) (*TrackingResponse, error) {
	path := fmt.Sprintf("/tracking/%s", url.PathEscape(trackID))
	var resp TrackingResponse
	if err := s.client.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("gls: get tracking: %w", err)
	}
	return &resp, nil
}

// Cancel cancels a parcel by its ID.
func (s *ShipmentService) Cancel(ctx context.Context, parcelID string) error {
	path := fmt.Sprintf("/parcels/%s", url.PathEscape(parcelID))
	if err := s.client.do(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("gls: cancel parcel: %w", err)
	}
	return nil
}
