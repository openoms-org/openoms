package ebay

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// OrderService handles communication with the eBay Fulfillment API order endpoints.
type OrderService struct {
	client *Client
}

// OrderSearchParams are the optional parameters for searching orders.
type OrderSearchParams struct {
	// Filter is an eBay-style filter string, e.g. "creationdate:[2024-01-01T00:00:00Z..]"
	Filter string
	Limit  int
	Offset int
}

// GetOrders retrieves a list of orders using the Fulfillment API.
// https://developer.ebay.com/api-docs/sell/fulfillment/resources/order/methods/getOrders
func (s *OrderService) GetOrders(ctx context.Context, params OrderSearchParams) (*OrderSearchResponse, error) {
	v := url.Values{}
	if params.Filter != "" {
		v.Set("filter", params.Filter)
	}
	if params.Limit > 0 {
		v.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Offset > 0 {
		v.Set("offset", strconv.Itoa(params.Offset))
	}

	path := "/sell/fulfillment/v1/order"
	if encoded := v.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var result OrderSearchResponse
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetOrder retrieves a single order by its eBay order ID.
// https://developer.ebay.com/api-docs/sell/fulfillment/resources/order/methods/getOrder
func (s *OrderService) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	var result Order
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/sell/fulfillment/v1/order/%s", orderID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
