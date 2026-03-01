package gls

import (
	"context"
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
// Labels are returned inline in the response (CreatedShipment.PrintData[].Data).
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
	}
	// PrintData is at CreatedShipment level, not inside ParcelData
	for _, pd := range raw.CreatedShipment.PrintData {
		if pd.Data != "" {
			resp.PrintData = append(resp.PrintData, pd.Data)
		}
	}
	return resp, nil
}

// GetLabel is not supported as a separate API call in the GLS ShipIT REST API.
// Labels are returned inline during shipment creation in CreatedShipment.PrintData[].Data.
// Retrieve the label from the stored create response instead.
func (s *ShipmentService) GetLabel(_ context.Context, _ string) ([]byte, error) {
	return nil, fmt.Errorf("gls: label retrieval not supported as separate API call — labels are embedded in the create shipment response (CreatedShipment.PrintData[].Data)")
}

// GetTracking retrieves tracking events for a tracking ID.
// Uses POST /shipments/parceldetails per GLS ShipIT REST API.
// GLS expects a single TrackID string (not an array).
func (s *ShipmentService) GetTracking(ctx context.Context, trackID string) (*TrackingResponse, error) {
	reqBody := ParcelDetailsRequest{
		TrackID: trackID,
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
