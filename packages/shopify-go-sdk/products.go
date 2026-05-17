package shopify

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// ProductService handles communication with the product-related Shopify endpoints.
type ProductService struct {
	client *Client
}

// ProductListParams are the optional parameters for listing products.
type ProductListParams struct {
	Limit        int
	SinceID      int64  // return products after this ID
	UpdatedAtMin string // ISO8601 date filter
	Status       string // "active", "archived", "draft"
	Fields       string // comma-separated list of fields
}

// List retrieves a list of products.
func (s *ProductService) List(ctx context.Context, params ProductListParams) ([]Product, error) {
	path := "/products.json"

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
	if params.Fields != "" {
		v.Set("fields", params.Fields)
	}
	if encoded := v.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var wrapper struct {
		Products []Product `json:"products"`
	}
	if err := s.client.do(ctx, "GET", path, nil, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Products, nil
}

// Get retrieves a single product by ID.
func (s *ProductService) Get(ctx context.Context, id int64) (*Product, error) {
	var wrapper struct {
		Product Product `json:"product"`
	}
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/products/%d.json", id), nil, &wrapper); err != nil {
		return nil, err
	}
	return &wrapper.Product, nil
}

// Create creates a product with arbitrary Shopify product fields.
func (s *ProductService) Create(ctx context.Context, data map[string]any) (*Product, error) {
	body := map[string]any{
		"product": data,
	}
	var wrapper struct {
		Product Product `json:"product"`
	}
	if err := s.client.do(ctx, "POST", "/products.json", body, &wrapper); err != nil {
		return nil, err
	}
	return &wrapper.Product, nil
}

// Update updates a product with arbitrary fields.
func (s *ProductService) Update(ctx context.Context, id int64, data map[string]any) error {
	body := map[string]any{
		"product": data,
	}
	return s.client.do(ctx, "PUT", fmt.Sprintf("/products/%d.json", id), body, nil)
}

// GetVariant retrieves a single product variant by ID.
func (s *ProductService) GetVariant(ctx context.Context, variantID int64) (*Variant, error) {
	var wrapper struct {
		Variant Variant `json:"variant"`
	}
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/variants/%d.json", variantID), nil, &wrapper); err != nil {
		return nil, err
	}
	return &wrapper.Variant, nil
}

// UpdateVariant updates a variant with arbitrary fields.
func (s *ProductService) UpdateVariant(ctx context.Context, variantID int64, data map[string]any) error {
	body := map[string]any{
		"variant": data,
	}
	return s.client.do(ctx, "PUT", fmt.Sprintf("/variants/%d.json", variantID), body, nil)
}
