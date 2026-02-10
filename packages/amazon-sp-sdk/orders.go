package amazon

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// OrderService handles communication with the Amazon SP-API Orders endpoints.
type OrderService struct {
	client *Client
}

// List retrieves orders created after a given time for the specified marketplace IDs.
// createdAfter should be an ISO 8601 date-time string.
// nextToken is used for pagination; pass empty string for the first page.
func (s *OrderService) List(ctx context.Context, createdAfter string, marketplaceIDs []string, nextToken string) (*OrdersResponse, error) {
	v := url.Values{}

	if createdAfter != "" {
		v.Set("CreatedAfter", createdAfter)
	}
	if len(marketplaceIDs) > 0 {
		v.Set("MarketplaceIds", strings.Join(marketplaceIDs, ","))
	}
	if nextToken != "" {
		v.Set("NextToken", nextToken)
	}

	path := "/orders/v0/orders"
	if encoded := v.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var result OrdersResponse
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, fmt.Errorf("amazon: list orders: %w", err)
	}
	return &result, nil
}

// Get retrieves a single order by Amazon order ID.
func (s *OrderService) Get(ctx context.Context, orderID string) (*Order, error) {
	var result GetOrderResponse
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/orders/v0/orders/%s", orderID), nil, &result); err != nil {
		return nil, fmt.Errorf("amazon: get order %s: %w", orderID, err)
	}
	return &result.Payload, nil
}

// GetItems retrieves the items for a given order ID.
// nextToken is used for pagination; pass empty string for the first page.
func (s *OrderService) GetItems(ctx context.Context, orderID string, nextToken string) (*OrderItemsResponse, error) {
	path := fmt.Sprintf("/orders/v0/orders/%s/orderItems", orderID)
	if nextToken != "" {
		path += "?NextToken=" + url.QueryEscape(nextToken)
	}

	var result OrderItemsResponse
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, fmt.Errorf("amazon: get order items %s: %w", orderID, err)
	}
	return &result, nil
}
