package prestashop

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// OrderService handles communication with the order-related PrestaShop endpoints.
type OrderService struct {
	client *Client
}

// OrderListParams are the optional parameters for listing orders.
type OrderListParams struct {
	Limit       int
	Offset      int
	SortBy      string // e.g. "id_ASC", "date_upd_DESC"
	DateUpdFrom string // ISO8601 date filter
	State       int    // order state ID filter
}

// List retrieves a list of orders.
func (s *OrderService) List(ctx context.Context, params OrderListParams) ([]PSOrder, error) {
	path := "/orders"

	v := url.Values{}
	if params.Limit > 0 {
		start := params.Offset
		v.Set("limit", fmt.Sprintf("%d,%d", start, params.Limit))
	}
	if params.SortBy != "" {
		v.Set("sort", "["+params.SortBy+"]")
	}
	if params.DateUpdFrom != "" {
		v.Set("filter[date_upd]", ">["+params.DateUpdFrom+"]")
	}
	if params.State > 0 {
		v.Set("filter[current_state]", "["+strconv.Itoa(params.State)+"]")
	}
	v.Set("display", "full")
	if encoded := v.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var wrapper struct {
		Orders []PSOrder `json:"orders"`
	}
	if err := s.client.do(ctx, "GET", path, nil, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Orders, nil
}

// Get retrieves a single order by ID.
func (s *OrderService) Get(ctx context.Context, id int) (*PSOrder, error) {
	var wrapper struct {
		Order PSOrder `json:"order"`
	}
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/orders/%d", id), nil, &wrapper); err != nil {
		return nil, err
	}
	return &wrapper.Order, nil
}

// UpdateState updates the current state of an order.
func (s *OrderService) UpdateState(ctx context.Context, orderID int, stateID int) error {
	body := map[string]any{
		"order_history": map[string]any{
			"id_order":       orderID,
			"id_order_state": stateID,
		},
	}
	return s.client.do(ctx, "POST", "/order_histories", body, nil)
}

// GetAddress retrieves an address by ID.
func (s *OrderService) GetAddress(ctx context.Context, id int) (*PSAddress, error) {
	var wrapper struct {
		Address PSAddress `json:"address"`
	}
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/addresses/%d", id), nil, &wrapper); err != nil {
		return nil, err
	}
	return &wrapper.Address, nil
}

// GetCustomer retrieves a customer by ID.
func (s *OrderService) GetCustomer(ctx context.Context, id int) (*PSCustomer, error) {
	var wrapper struct {
		Customer PSCustomer `json:"customer"`
	}
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/customers/%d", id), nil, &wrapper); err != nil {
		return nil, err
	}
	return &wrapper.Customer, nil
}
