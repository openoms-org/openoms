package erli

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// OrderService handles Erli order-related API operations.
type OrderService struct {
	client *Client
}

// List retrieves orders with optional cursor-based pagination.
// Only returns paid orders by default.
func (s *OrderService) List(ctx context.Context, cursor string) (*OrdersResponse, error) {
	u, _ := url.Parse("/orders")
	q := url.Values{"status": {"paid"}}
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	u.RawQuery = q.Encode()
	path := u.String()

	var resp OrdersResponse
	if err := s.client.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("erli: list orders: %w", err)
	}
	return &resp, nil
}

// Get retrieves a single order by its ID.
func (s *OrderService) Get(ctx context.Context, orderID string) (*Order, error) {
	path := fmt.Sprintf("/orders/%s", url.PathEscape(orderID))
	var resp Order
	if err := s.client.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("erli: get order %s: %w", orderID, err)
	}
	return &resp, nil
}
