package fedex

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"
)

// ShipmentService handles FedEx shipment-related API operations.
type ShipmentService struct {
	client *Client
}

// Create creates a new shipment in the FedEx system.
func (s *ShipmentService) Create(ctx context.Context, req *ShipmentRequest) (*ShipmentResponse, error) {
	var resp ShipmentResponse
	if err := s.client.do(ctx, http.MethodPost, "/ship/v1/shipments", req, &resp); err != nil {
		return nil, fmt.Errorf("fedex: create shipment: %w", err)
	}
	return &resp, nil
}

// GetLabel retrieves the shipping label for a tracking number. Returns raw PDF bytes.
func (s *ShipmentService) GetLabel(ctx context.Context, trackingNumber string) ([]byte, error) {
	// FedEx provides labels in the shipment creation response.
	// This endpoint re-retrieves the label via the documents API.
	reqBody := map[string]any{
		"accountNumber": map[string]string{
			"value": s.client.accountNumber,
		},
		"trackingNumber": trackingNumber,
		"labelSpecification": map[string]string{
			"labelFormatType": "COMMON2D",
			"imageType":       "PDF",
			"labelStockType":  "PAPER_4X6",
		},
	}

	var resp struct {
		Output struct {
			LabelResults []struct {
				Labels []struct {
					EncodedLabel string `json:"encodedLabel"`
				} `json:"labels"`
			} `json:"labelResults"`
		} `json:"output"`
	}

	if err := s.client.do(ctx, http.MethodPost, "/ship/v1/shipments/label", reqBody, &resp); err != nil {
		return nil, fmt.Errorf("fedex: get label: %w", err)
	}

	if len(resp.Output.LabelResults) > 0 && len(resp.Output.LabelResults[0].Labels) > 0 {
		encoded := resp.Output.LabelResults[0].Labels[0].EncodedLabel
		data, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("fedex: decode label data: %w", err)
		}
		return data, nil
	}

	return nil, fmt.Errorf("fedex: no label data in response")
}

// GetTracking retrieves tracking events for a tracking number.
func (s *ShipmentService) GetTracking(ctx context.Context, trackingNumber string) (*TrackingResponse, error) {
	reqBody := map[string]any{
		"trackingInfo": []map[string]any{
			{
				"trackingNumberInfo": map[string]string{
					"trackingNumber": trackingNumber,
				},
			},
		},
		"includeDetailedScans": true,
	}

	var resp TrackingResponse
	if err := s.client.do(ctx, http.MethodPost, "/track/v1/trackingnumbers", reqBody, &resp); err != nil {
		return nil, fmt.Errorf("fedex: get tracking: %w", err)
	}
	return &resp, nil
}

// Cancel cancels a shipment by its tracking number.
func (s *ShipmentService) Cancel(ctx context.Context, trackingNumber string) error {
	reqBody := &CancelShipmentRequest{
		AccountNumber:  AccountNumber{Value: s.client.accountNumber},
		TrackingNumber: trackingNumber,
	}

	if err := s.client.do(ctx, http.MethodPut, "/ship/v1/shipments/cancel", reqBody, nil); err != nil {
		return fmt.Errorf("fedex: cancel shipment: %w", err)
	}
	return nil
}

// ParseTrackingEvents extracts integration-friendly tracking events from a FedEx tracking response.
func ParseTrackingEvents(resp *TrackingResponse) []TrackingEvent {
	var events []TrackingEvent

	for _, result := range resp.Output.CompleteTrackResults {
		for _, tr := range result.TrackResults {
			for _, scan := range tr.ScanEvents {
				ts, _ := time.Parse("2006-01-02T15:04:05-07:00", scan.Date)
				location := scan.ScanLocation.City
				if scan.ScanLocation.CountryCode != "" {
					location += ", " + scan.ScanLocation.CountryCode
				}
				events = append(events, TrackingEvent{
					EventType:   scan.EventType,
					Description: scan.EventDescription,
					Date:        ts,
					City:        location,
				})
			}
		}
	}

	return events
}
