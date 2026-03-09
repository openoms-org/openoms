package ebay

import (
	"context"
	"fmt"
	"net/url"
	"strings"
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

// CreateOrReplaceInventoryItem creates or replaces an inventory item by SKU.
// PUT /sell/inventory/v1/inventory_item/{sku}
func (s *InventoryService) CreateOrReplaceInventoryItem(ctx context.Context, sku string, item InventoryItem) error {
	path := fmt.Sprintf("/sell/inventory/v1/inventory_item/%s", url.PathEscape(sku))
	if err := s.client.do(ctx, "PUT", path, item, nil); err != nil {
		return fmt.Errorf("ebay: create inventory item %s: %w", sku, err)
	}
	return nil
}

// GetInventoryItem returns an inventory item by SKU.
// GET /sell/inventory/v1/inventory_item/{sku}
func (s *InventoryService) GetInventoryItem(ctx context.Context, sku string) (*InventoryItem, error) {
	path := fmt.Sprintf("/sell/inventory/v1/inventory_item/%s", url.PathEscape(sku))
	var result InventoryItem
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, fmt.Errorf("ebay: get inventory item %s: %w", sku, err)
	}
	return &result, nil
}

// ListInventoryItems returns a paginated list of inventory items.
// GET /sell/inventory/v1/inventory_item?limit=N&offset=N
func (s *InventoryService) ListInventoryItems(ctx context.Context, limit, offset int) (*InventoryItemsResponse, error) {
	path := fmt.Sprintf("/sell/inventory/v1/inventory_item?limit=%d&offset=%d", limit, offset)
	var result InventoryItemsResponse
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, fmt.Errorf("ebay: list inventory items: %w", err)
	}
	return &result, nil
}

// DeleteInventoryItem deletes an inventory item by SKU.
// DELETE /sell/inventory/v1/inventory_item/{sku}
func (s *InventoryService) DeleteInventoryItem(ctx context.Context, sku string) error {
	path := fmt.Sprintf("/sell/inventory/v1/inventory_item/%s", url.PathEscape(sku))
	if err := s.client.do(ctx, "DELETE", path, nil, nil); err != nil {
		return fmt.Errorf("ebay: delete inventory item %s: %w", sku, err)
	}
	return nil
}

// BulkPriceQuantityRequest is a single item in a bulk update request.
type BulkPriceQuantityRequest struct {
	OfferID           string  `json:"offerId"`
	AvailableQuantity *int    `json:"availableQuantity,omitempty"`
	Price             *Amount `json:"price,omitempty"`
}

// BulkUpdatePriceQuantity updates stock and/or price for multiple offers in one call.
// eBay allows up to 25 offers per request.
// POST /sell/inventory/v1/bulk_update_price_quantity
func (s *InventoryService) BulkUpdatePriceQuantity(ctx context.Context, requests []BulkPriceQuantityRequest) ([]bulkUpdateItemResponse, error) {
	apiRequests := make([]map[string]any, len(requests))
	for i, r := range requests {
		item := map[string]any{
			"offerId": r.OfferID,
		}
		if r.AvailableQuantity != nil {
			item["availableQuantity"] = *r.AvailableQuantity
		}
		if r.Price != nil {
			item["pricingSummary"] = map[string]any{
				"price": map[string]string{
					"value":    r.Price.Value,
					"currency": r.Price.Currency,
				},
			}
		}
		apiRequests[i] = item
	}

	body := map[string]any{"requests": apiRequests}
	var result bulkUpdateResponse
	if err := s.client.do(ctx, "POST", "/sell/inventory/v1/bulk_update_price_quantity", body, &result); err != nil {
		return nil, fmt.Errorf("ebay: bulk update price/quantity: %w", err)
	}

	var errs []string
	for _, r := range result.Responses {
		if r.StatusCode >= 400 {
			msg := fmt.Sprintf("offer %s: HTTP %d", r.OfferID, r.StatusCode)
			if len(r.Errors) > 0 {
				msg = fmt.Sprintf("offer %s: %s", r.OfferID, r.Errors[0].Message)
			}
			errs = append(errs, msg)
		}
	}
	if len(errs) > 0 {
		return result.Responses, fmt.Errorf("ebay: bulk update errors: %s", strings.Join(errs, "; "))
	}

	return result.Responses, nil
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
