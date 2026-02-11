package woocommerce

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// ProductService handles communication with the product-related WooCommerce endpoints.
type ProductService struct {
	client *Client
}

// ProductListParams are the optional parameters for listing products.
type ProductListParams struct {
	Page    int
	PerPage int
	Search  string
	SKU     string
	Status  string // "draft", "pending", "private", "publish"
	Order   string // "asc" or "desc"
	OrderBy string // "date", "id", "include", "title", "slug"
}

// WooProduct represents a WooCommerce product.
type WooProduct struct {
	ID            int      `json:"id"`
	Name          string   `json:"name"`
	SKU           string   `json:"sku"`
	Price         string   `json:"price"`
	RegularPrice  string   `json:"regular_price"`
	StockQuantity *int     `json:"stock_quantity"`
	StockStatus   string   `json:"stock_status"`
	Status        string   `json:"status"`
	Description   string   `json:"description"`
	ShortDesc     string   `json:"short_description"`
	Categories    []WooCat `json:"categories"`
}

// WooCat represents a WooCommerce product category reference.
type WooCat struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// List retrieves a list of products with optional filtering.
func (s *ProductService) List(ctx context.Context, params ProductListParams) ([]WooProduct, error) {
	path := "/products"

	v := url.Values{}
	if params.Page > 0 {
		v.Set("page", strconv.Itoa(params.Page))
	}
	if params.PerPage > 0 {
		v.Set("per_page", strconv.Itoa(params.PerPage))
	}
	if params.Search != "" {
		v.Set("search", params.Search)
	}
	if params.SKU != "" {
		v.Set("sku", params.SKU)
	}
	if params.Status != "" {
		v.Set("status", params.Status)
	}
	if params.Order != "" {
		v.Set("order", params.Order)
	}
	if params.OrderBy != "" {
		v.Set("orderby", params.OrderBy)
	}
	if encoded := v.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var result []WooProduct
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Get retrieves a single product by ID.
func (s *ProductService) Get(ctx context.Context, id int) (*WooProduct, error) {
	var result WooProduct
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
		"stock_quantity": quantity,
		"manage_stock":   true,
	}
	return s.client.do(ctx, "PUT", fmt.Sprintf("/products/%d", id), body, nil)
}

// UpdatePrice updates the regular price for a product.
func (s *ProductService) UpdatePrice(ctx context.Context, id int, price string) error {
	body := map[string]any{
		"regular_price": price,
	}
	return s.client.do(ctx, "PUT", fmt.Sprintf("/products/%d", id), body, nil)
}

// Create creates a new product.
func (s *ProductService) Create(ctx context.Context, data map[string]any) (*WooProduct, error) {
	var result WooProduct
	if err := s.client.do(ctx, "POST", "/products", data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
