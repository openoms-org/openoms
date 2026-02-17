package shoper

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// OrderService handles communication with the order-related Shoper endpoints.
type OrderService struct {
	client *Client
}

// OrderListParams are the optional parameters for listing orders.
type OrderListParams struct {
	Page           int
	Limit          int
	ModifiedAfter  string // ISO8601 date filter: date_modify >= value
	StatusID       int
	SortBy         string // e.g. "date_add", "date_modify"
	SortDirection  string // "asc" or "desc"
}

// List retrieves a paginated list of orders.
func (s *OrderService) List(ctx context.Context, params OrderListParams) (*ListResponse[ShoperOrder], error) {
	path := "/orders"

	v := url.Values{}
	if params.Page > 0 {
		v.Set("page", strconv.Itoa(params.Page))
	}
	if params.Limit > 0 {
		v.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.StatusID > 0 {
		v.Set("filters[status_id]", strconv.Itoa(params.StatusID))
	}
	if params.ModifiedAfter != "" {
		v.Set("filters[date_modify][from]", params.ModifiedAfter)
	}
	if params.SortBy != "" {
		v.Set("sort", params.SortBy)
	}
	if params.SortDirection != "" {
		v.Set("order", params.SortDirection)
	}
	if encoded := v.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var result ListResponse[ShoperOrder]
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves a single order by ID.
func (s *OrderService) Get(ctx context.Context, id int) (*ShoperOrder, error) {
	var result ShoperOrder
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/orders/%d", id), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateStatus updates the status of an order.
func (s *OrderService) UpdateStatus(ctx context.Context, id int, statusID int) error {
	body := map[string]any{
		"status_id": statusID,
	}
	return s.client.do(ctx, "PUT", fmt.Sprintf("/orders/%d", id), body, nil)
}
