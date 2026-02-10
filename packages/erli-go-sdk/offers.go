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

// Create creates a new offer on Erli.
func (s *OfferService) Create(ctx context.Context, offer map[string]any) error {
	if err := s.client.do(ctx, http.MethodPost, "/offers", offer, nil); err != nil {
		return fmt.Errorf("erli: create offer: %w", err)
	}
	return nil
}
