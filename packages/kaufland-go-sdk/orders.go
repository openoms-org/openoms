package kaufland

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// OrderService handles communication with the Kaufland Seller API order unit endpoints.
type OrderService struct {
	client *Client
}

// OrderUnitListParams are the optional parameters for listing order units.
type OrderUnitListParams struct {
	// CreatedAfter filters order units created after this ISO8601 timestamp.
	CreatedAfter string
	// Status filters by order unit status.
	Status string
	Limit  int
	Offset int
	// Sort controls sorting. Default: "ts_created_iso:desc"
	Sort string
}

// GetOrderUnits retrieves a list of order units.
// https://sellerapi.kaufland.com/?page=order-units#get-order-units
func (s *OrderService) GetOrderUnits(ctx context.Context, params OrderUnitListParams) (*OrderUnitListResponse, error) {
	v := url.Values{}
	if params.CreatedAfter != "" {
		v.Set("ts_created_from_iso", params.CreatedAfter)
	}
	if params.Status != "" {
		v.Set("status", params.Status)
	}
	if params.Limit > 0 {
		v.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Offset > 0 {
		v.Set("offset", strconv.Itoa(params.Offset))
	}
	if params.Sort != "" {
		v.Set("sort", params.Sort)
	}

	path := "/order-units"
	if encoded := v.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var result OrderUnitListResponse
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetOrderUnit retrieves a single order unit by ID.
// https://sellerapi.kaufland.com/?page=order-units#get-order-unit
func (s *OrderService) GetOrderUnit(ctx context.Context, id int64) (*OrderUnit, error) {
	var wrapper struct {
		Data OrderUnit `json:"data"`
	}
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/order-units/%d", id), nil, &wrapper); err != nil {
		return nil, err
	}
	return &wrapper.Data, nil
}
