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
// Uses POST /shipments per GLS ShipIT REST API.
func (s *ShipmentService) Create(ctx context.Context, req *CreateParcelRequest) (*CreateParcelResponse, error) {
	var raw rawCreateParcelResponse
	if err := s.client.do(ctx, http.MethodPost, "/shipments", req, &raw); err != nil {
		return nil, fmt.Errorf("gls: create parcel: %w", err)
	}

	resp := &CreateParcelResponse{}
	for _, pd := range raw.CreatedShipment.ParcelData {
		if pd.TrackID != "" {
			resp.TrackIDs = append(resp.TrackIDs, pd.TrackID)
			resp.ParcelIDs = append(resp.ParcelIDs, pd.TrackID)
		}
		if pd.PrintData != "" {
			resp.PrintData = append(resp.PrintData, pd.PrintData)
		}
	}
	return resp, nil
}

// GetLabel retrieves the shipping label for a parcel. Returns raw PDF bytes.
func (s *ShipmentService) GetLabel(ctx context.Context, parcelID string) ([]byte, error) {
	path := fmt.Sprintf("/shipments/%s/labels", url.PathEscape(parcelID))
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
// Uses POST /shipments/parceldetails per GLS ShipIT REST API.
func (s *ShipmentService) GetTracking(ctx context.Context, trackID string) (*TrackingResponse, error) {
	reqBody := ParcelDetailsRequest{
		TrackIDs: []string{trackID},
	}
	var resp TrackingResponse
	if err := s.client.do(ctx, http.MethodPost, "/shipments/parceldetails", &reqBody, &resp); err != nil {
		return nil, fmt.Errorf("gls: get tracking: %w", err)
	}
	return &resp, nil
}

// Cancel cancels a parcel by its tracking ID.
// Uses POST /shipments/cancel/{trackID} per GLS ShipIT REST API.
func (s *ShipmentService) Cancel(ctx context.Context, trackID string) error {
	path := fmt.Sprintf("/shipments/cancel/%s", url.PathEscape(trackID))
	if err := s.client.do(ctx, http.MethodPost, path, nil, nil); err != nil {
		return fmt.Errorf("gls: cancel parcel: %w", err)
	}
	return nil
}
