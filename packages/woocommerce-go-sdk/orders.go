package woocommerce

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// OrderService handles communication with the order-related WooCommerce endpoints.
type OrderService struct {
	client *Client
}

// OrderListParams are the optional parameters for listing orders.
type OrderListParams struct {
	After   string // ISO8601 date, limit to orders after this date
	Before  string // ISO8601 date, limit to orders before this date
	Status  string // Order status (e.g. "processing", "completed")
	Page    int
	PerPage int
	Order   string // "asc" or "desc"
	OrderBy string // "date", "id", "include", "title", "slug"

	// ModifiedAfter filters orders modified after this ISO8601 date.
	ModifiedAfter string
}

// WooOrder represents a WooCommerce order.
type WooOrder struct {
	ID            int           `json:"id"`
	Status        string        `json:"status"`
	Currency      string        `json:"currency"`
	Total         string        `json:"total"`
	TotalTax      string        `json:"total_tax"`
	Billing       WooAddress    `json:"billing"`
	Shipping      WooAddress    `json:"shipping"`
	PaymentMethod string        `json:"payment_method"`
	PaymentTitle  string        `json:"payment_method_title"`
	LineItems     []WooLineItem `json:"line_items"`
	DateCreated   string        `json:"date_created"`
	DateModified  string        `json:"date_modified"`
	CustomerNote  string        `json:"customer_note"`
}

// WooAddress represents a billing or shipping address.
type WooAddress struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Address1  string `json:"address_1"`
	Address2  string `json:"address_2"`
	City      string `json:"city"`
	PostCode  string `json:"postcode"`
	Country   string `json:"country"`
}

// WooLineItem represents a single line item in a WooCommerce order.
type WooLineItem struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	SKU       string  `json:"sku"`
	Quantity  int     `json:"quantity"`
	Total     string  `json:"total"`
	TotalTax  string  `json:"total_tax"`
	Price     float64 `json:"price"`
	ProductID int     `json:"product_id"`
}

// List retrieves a list of orders with optional filtering.
func (s *OrderService) List(ctx context.Context, params OrderListParams) ([]WooOrder, error) {
	path := "/orders"

	v := url.Values{}
	if params.After != "" {
		v.Set("after", params.After)
	}
	if params.Before != "" {
		v.Set("before", params.Before)
	}
	if params.Status != "" {
		v.Set("status", params.Status)
	}
	if params.Page > 0 {
		v.Set("page", strconv.Itoa(params.Page))
	}
	if params.PerPage > 0 {
		v.Set("per_page", strconv.Itoa(params.PerPage))
	}
	if params.Order != "" {
		v.Set("order", params.Order)
	}
	if params.OrderBy != "" {
		v.Set("orderby", params.OrderBy)
	}
	if params.ModifiedAfter != "" {
		v.Set("modified_after", params.ModifiedAfter)
	}
	if encoded := v.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var result []WooOrder
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Get retrieves a single order by ID.
func (s *OrderService) Get(ctx context.Context, id int) (*WooOrder, error) {
	var result WooOrder
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/orders/%d", id), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateStatus updates the status of an order.
func (s *OrderService) UpdateStatus(ctx context.Context, id int, status string) error {
	body := map[string]any{
		"status": status,
	}
	return s.client.do(ctx, "PUT", fmt.Sprintf("/orders/%d", id), body, nil)
}
