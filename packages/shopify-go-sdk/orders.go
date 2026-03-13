package shopify

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// OrderService handles communication with the order-related Shopify endpoints.
type OrderService struct {
	client *Client
}

// OrderListParams are the optional parameters for listing orders.
type OrderListParams struct {
	Limit             int
	SinceID           int64  // return orders after this ID
	UpdatedAtMin      string // ISO8601 date filter
	Status            string // "open", "closed", "cancelled", "any"
	FinancialStatus   string // "authorized", "pending", "paid", "partially_paid", etc.
	FulfillmentStatus string // "shipped", "partial", "unshipped", "any"
	Fields            string // comma-separated list of fields
}

// List retrieves a list of orders.
func (s *OrderService) List(ctx context.Context, params OrderListParams) ([]Order, error) {
	path := "/orders.json"

	v := url.Values{}
	if params.Limit > 0 {
		v.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.SinceID > 0 {
		v.Set("since_id", strconv.FormatInt(params.SinceID, 10))
	}
	if params.UpdatedAtMin != "" {
		v.Set("updated_at_min", params.UpdatedAtMin)
	}
	if params.Status != "" {
		v.Set("status", params.Status)
	}
	if params.FinancialStatus != "" {
		v.Set("financial_status", params.FinancialStatus)
	}
	if params.FulfillmentStatus != "" {
		v.Set("fulfillment_status", params.FulfillmentStatus)
	}
	if params.Fields != "" {
		v.Set("fields", params.Fields)
	}
	if encoded := v.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var wrapper struct {
		Orders []Order `json:"orders"`
	}
	if err := s.client.do(ctx, "GET", path, nil, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Orders, nil
}

// Get retrieves a single order by ID.
func (s *OrderService) Get(ctx context.Context, id int64) (*Order, error) {
	var wrapper struct {
		Order Order `json:"order"`
	}
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/orders/%d.json", id), nil, &wrapper); err != nil {
		return nil, err
	}
	return &wrapper.Order, nil
}

// Update updates an order with arbitrary fields.
func (s *OrderService) Update(ctx context.Context, id int64, data map[string]any) error {
	body := map[string]any{
		"order": data,
	}
	return s.client.do(ctx, "PUT", fmt.Sprintf("/orders/%d.json", id), body, nil)
}
