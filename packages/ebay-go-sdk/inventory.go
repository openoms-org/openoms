package ebay

import (
	"context"
	"fmt"
)

// InventoryService handles eBay Inventory API v1 operations.
type InventoryService struct {
	client *Client
}

// UpdateStock updates the available quantity for an offer using the bulk endpoint.
func (s *InventoryService) UpdateStock(ctx context.Context, offerID string, quantity int) error {
	body := map[string]any{
		"requests": []map[string]any{
			{
				"offerId":           offerID,
				"availableQuantity": quantity,
			},
		},
	}
	var result bulkUpdateResponse
	if err := s.client.do(ctx, "POST", "/sell/inventory/v1/bulk_update_price_quantity", body, &result); err != nil {
		return fmt.Errorf("ebay: update stock for %s: %w", offerID, err)
	}
	return checkBulkErrors(result, offerID)
}

// UpdatePrice updates the price for an offer using the bulk endpoint.
func (s *InventoryService) UpdatePrice(ctx context.Context, offerID string, price float64, currency string) error {
	body := map[string]any{
		"requests": []map[string]any{
			{
				"offerId": offerID,
				"pricingSummary": map[string]any{
					"price": map[string]string{
						"value":    fmt.Sprintf("%.2f", price),
						"currency": currency,
					},
				},
			},
		},
	}
	var result bulkUpdateResponse
	if err := s.client.do(ctx, "POST", "/sell/inventory/v1/bulk_update_price_quantity", body, &result); err != nil {
		return fmt.Errorf("ebay: update price for %s: %w", offerID, err)
	}
	return checkBulkErrors(result, offerID)
}

// bulkUpdateResponse is the response from the bulk_update_price_quantity endpoint.
type bulkUpdateResponse struct {
	Responses []bulkUpdateItemResponse `json:"responses"`
}

type bulkUpdateItemResponse struct {
	OfferID    string  `json:"offerId"`
	StatusCode int     `json:"statusCode"`
	Errors     []EbErr `json:"errors,omitempty"`
}

// checkBulkErrors inspects the per-item responses for errors.
func checkBulkErrors(resp bulkUpdateResponse, targetOfferID string) error {
	for _, r := range resp.Responses {
		if r.OfferID == targetOfferID && r.StatusCode >= 400 {
			msg := fmt.Sprintf("HTTP %d", r.StatusCode)
			if len(r.Errors) > 0 {
				msg = r.Errors[0].Message
			}
			return fmt.Errorf("ebay: bulk update for %s: %s", targetOfferID, msg)
		}
	}
	return nil
}
