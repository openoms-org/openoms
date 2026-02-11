package inpost

import (
	"context"
	"fmt"
	"net/http"
)

// TrackingService handles tracking-related API operations.
type TrackingService struct {
	client *Client
}

// Get retrieves tracking information for a shipment by its tracking number.
func (s *TrackingService) Get(ctx context.Context, trackingNumber string) (*TrackingResponse, error) {
	path := fmt.Sprintf("/v1/tracking/%s", trackingNumber)
	var resp TrackingResponse
	if err := s.client.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
