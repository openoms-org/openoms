package erli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// OfferService handles Erli product/offer-related API operations.
type OfferService struct {
	client *Client
}

// UpdateStock updates the stock quantity for a product identified by the seller's externalID (SKU).
func (s *OfferService) UpdateStock(ctx context.Context, externalID string, quantity int) error {
	path := fmt.Sprintf("/products/%s", url.PathEscape(externalID))
	body := map[string]any{"stock": quantity}
	if err := s.client.do(ctx, http.MethodPatch, path, body, nil); err != nil {
		return fmt.Errorf("erli: update stock for %s: %w", externalID, err)
	}
	return nil
}

// UpdatePrice updates the price for a product identified by the seller's externalID (SKU).
func (s *OfferService) UpdatePrice(ctx context.Context, externalID string, price float64) error {
	path := fmt.Sprintf("/products/%s", url.PathEscape(externalID))
	body := map[string]any{"price": price}
	if err := s.client.do(ctx, http.MethodPatch, path, body, nil); err != nil {
		return fmt.Errorf("erli: update price for %s: %w", externalID, err)
	}
	return nil
}

// Create creates or updates a product on Erli using the seller's externalID (SKU) as the
// path identifier. Returns the product ID and nil on success (HTTP 200/201), or the
// externalID and ErrProductPendingValidation when the API responds with HTTP 202 Accepted
// (product queued for async validation, not yet live).
func (s *OfferService) Create(ctx context.Context, externalID string, req CreateOfferRequest) (string, error) {
	path := fmt.Sprintf("/products/%s", url.PathEscape(externalID))
	raw, statusCode, err := s.client.doRaw(ctx, http.MethodPost, path, req)
	if err != nil {
		return "", fmt.Errorf("erli: create product %s: %w", externalID, err)
	}

	if statusCode == http.StatusAccepted {
		// API accepted the request for async validation. Product is not yet live.
		return externalID, ErrProductPendingValidation
	}

	var resp CreateOfferResponse
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &resp); err != nil {
			return "", fmt.Errorf("erli: decode create product response: %w", err)
		}
	}
	if resp.ID != "" {
		return resp.ID, nil
	}
	// Fallback: return externalID when no ID in response body.
	return externalID, nil
}
