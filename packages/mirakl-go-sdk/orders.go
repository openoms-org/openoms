package mirakl

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// OrderService handles Mirakl order-related API operations.
type OrderService struct {
	client *Client
}

// List retrieves orders updated since lastUpdated.
// The lastUpdated parameter should be an ISO 8601 date string.
func (s *OrderService) List(ctx context.Context, lastUpdated string) (*OrdersResponse, error) {
	path := "/orders?order_state_codes=SHIPPING"
	if lastUpdated != "" {
		path += "&start_update_date=" + url.QueryEscape(lastUpdated)
	}

	var resp OrdersResponse
	if err := s.client.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("mirakl: list orders: %w", err)
	}
	return &resp, nil
}

// Get retrieves a single order by its ID.
func (s *OrderService) Get(ctx context.Context, orderID string) (*Order, error) {
	path := fmt.Sprintf("/orders/%s", url.PathEscape(orderID))
	var resp OrdersResponse
	if err := s.client.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("mirakl: get order %s: %w", orderID, err)
	}
	if len(resp.Orders) == 0 {
		return nil, fmt.Errorf("mirakl: order %s not found", orderID)
	}
	return &resp.Orders[0], nil
}

// AcceptOrderLine accepts a specific order line.
func (s *OrderService) AcceptOrderLine(ctx context.Context, orderLineID string) error {
	path := fmt.Sprintf("/orders/%s/accept", url.PathEscape(orderLineID))
	if err := s.client.do(ctx, http.MethodPut, path, nil, nil); err != nil {
		return fmt.Errorf("mirakl: accept order line %s: %w", orderLineID, err)
	}
	return nil
}
