package prestashop

import (
	"context"
	"fmt"
	"net/url"
)

// ProductService handles communication with the product-related PrestaShop endpoints.
type ProductService struct {
	client *Client
}

// ProductListParams are the optional parameters for listing products.
type ProductListParams struct {
	Limit        int
	Offset       int
	SortBy       string // e.g. "id_ASC", "date_upd_DESC"
	FilterActive string // "1" or "0"
	DateUpdFrom  string // ISO8601 date filter
}

// List retrieves a list of products.
func (s *ProductService) List(ctx context.Context, params ProductListParams) ([]PSProduct, error) {
	path := "/products"

	v := url.Values{}
	if params.Limit > 0 {
		start := params.Offset
		v.Set("limit", fmt.Sprintf("%d,%d", start, params.Limit))
	}
	if params.SortBy != "" {
		v.Set("sort", "["+params.SortBy+"]")
	}
	if params.FilterActive != "" {
		v.Set("filter[active]", "["+params.FilterActive+"]")
	}
	if params.DateUpdFrom != "" {
		v.Set("filter[date_upd]", ">["+params.DateUpdFrom+"]")
	}
	v.Set("display", "full")
	if encoded := v.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var wrapper struct {
		Products []PSProduct `json:"products"`
	}
	if err := s.client.do(ctx, "GET", path, nil, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Products, nil
}

// Get retrieves a single product by ID.
func (s *ProductService) Get(ctx context.Context, id int) (*PSProduct, error) {
	var wrapper struct {
		Product PSProduct `json:"product"`
	}
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/products/%d", id), nil, &wrapper); err != nil {
		return nil, err
	}
	return &wrapper.Product, nil
}

// Update updates a product with arbitrary fields.
func (s *ProductService) Update(ctx context.Context, id int, data map[string]any) error {
	body := map[string]any{
		"product": data,
	}
	return s.client.do(ctx, "PUT", fmt.Sprintf("/products/%d", id), body, nil)
}

// UpdatePrice updates the price for a product.
func (s *ProductService) UpdatePrice(ctx context.Context, id int, price string) error {
	return s.Update(ctx, id, map[string]any{"price": price})
}

// ListOrderDetails retrieves all order detail (line item) records for an order.
func (s *ProductService) ListOrderDetails(ctx context.Context, orderID int) ([]PSOrderDetail, error) {
	path := fmt.Sprintf("/order_details?filter[id_order]=[%d]&display=full", orderID)

	var wrapper struct {
		OrderDetails []PSOrderDetail `json:"order_details"`
	}
	if err := s.client.do(ctx, "GET", path, nil, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.OrderDetails, nil
}
