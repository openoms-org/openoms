package erli

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// OfferService handles Erli offer-related API operations.
type OfferService struct {
	client *Client
}

// UpdateStock updates the stock quantity for an offer.
func (s *OfferService) UpdateStock(ctx context.Context, offerID string, quantity int) error {
	path := fmt.Sprintf("/offers/%s", url.PathEscape(offerID))
	body := map[string]any{"stock": quantity}
	if err := s.client.do(ctx, http.MethodPatch, path, body, nil); err != nil {
		return fmt.Errorf("erli: update stock for %s: %w", offerID, err)
	}
	return nil
}

// UpdatePrice updates the price for an offer.
func (s *OfferService) UpdatePrice(ctx context.Context, offerID string, price float64) error {
	path := fmt.Sprintf("/offers/%s", url.PathEscape(offerID))
	body := map[string]any{"price": price}
	if err := s.client.do(ctx, http.MethodPatch, path, body, nil); err != nil {
		return fmt.Errorf("erli: update price for %s: %w", offerID, err)
	}
	return nil
}

// Create creates a new offer on Erli and returns the new offer ID.
func (s *OfferService) Create(ctx context.Context, req CreateOfferRequest) (string, error) {
	var resp CreateOfferResponse
	if err := s.client.do(ctx, http.MethodPost, "/offers", req, &resp); err != nil {
		return "", fmt.Errorf("erli: create offer: %w", err)
	}
	return resp.ID, nil
}
