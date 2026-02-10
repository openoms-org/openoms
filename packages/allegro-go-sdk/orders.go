package allegro

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// OrderService handles communication with the order-related endpoints.
type OrderService struct {
	client *Client
}

// ListOrdersParams are the optional parameters for listing orders.
type ListOrdersParams struct {
	Limit             int
	Offset            int
	Status            string
	FulfillmentStatus string
	UpdatedAtGte      *time.Time
	UpdatedAtLte      *time.Time
}

// List retrieves a paginated list of orders.
func (s *OrderService) List(ctx context.Context, params *ListOrdersParams) (*OrderList, error) {
	path := "/order/checkout-forms"

	if params != nil {
		v := url.Values{}
		if params.Limit > 0 {
			v.Set("limit", strconv.Itoa(params.Limit))
		}
		if params.Offset > 0 {
			v.Set("offset", strconv.Itoa(params.Offset))
		}
		if params.Status != "" {
			v.Set("status", params.Status)
		}
		if params.FulfillmentStatus != "" {
			v.Set("fulfillment.status", params.FulfillmentStatus)
		}
		if params.UpdatedAtGte != nil {
			v.Set("updatedAt.gte", params.UpdatedAtGte.Format(time.RFC3339))
		}
		if params.UpdatedAtLte != nil {
			v.Set("updatedAt.lte", params.UpdatedAtLte.Format(time.RFC3339))
		}
		if encoded := v.Encode(); encoded != "" {
			path += "?" + encoded
		}
	}

	var result OrderList
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves a single order by ID.
func (s *OrderService) Get(ctx context.Context, id string) (*Order, error) {
	var result Order
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/order/checkout-forms/%s", id), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
