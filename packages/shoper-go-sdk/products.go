package shoper

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// ProductService handles communication with the product-related Shoper endpoints.
type ProductService struct {
	client *Client
}

// ProductListParams are the optional parameters for listing products.
type ProductListParams struct {
	Page    int
	Limit   int
	Filters map[string]string // e.g. {"category_id": "5"}
}

// List retrieves a paginated list of products.
func (s *ProductService) List(ctx context.Context, params ProductListParams) (*ListResponse[Product], error) {
	path := "/products"

	v := url.Values{}
	if params.Page > 0 {
		v.Set("page", strconv.Itoa(params.Page))
	}
	if params.Limit > 0 {
		v.Set("limit", strconv.Itoa(params.Limit))
	}
	for k, val := range params.Filters {
		v.Set("filters["+k+"]", val)
	}
	if encoded := v.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var result ListResponse[Product]
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves a single product by ID.
func (s *ProductService) Get(ctx context.Context, id int) (*Product, error) {
	var result Product
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/products/%d", id), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Update updates a product with arbitrary fields.
func (s *ProductService) Update(ctx context.Context, id int, data map[string]any) error {
	return s.client.do(ctx, "PUT", fmt.Sprintf("/products/%d", id), data, nil)
}

// UpdateStock updates the stock quantity for a product.
func (s *ProductService) UpdateStock(ctx context.Context, id int, quantity int) error {
	body := map[string]any{
		"stock": map[string]any{
			"stock": quantity,
		},
	}
	return s.client.do(ctx, "PUT", fmt.Sprintf("/products/%d", id), body, nil)
}
