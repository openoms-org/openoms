package shoper

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// CategoryService handles communication with the category-related Shoper endpoints.
type CategoryService struct {
	client *Client
}

// CategoryListParams are the optional parameters for listing categories.
type CategoryListParams struct {
	Page  int
	Limit int
}

// List retrieves a paginated list of categories.
func (s *CategoryService) List(ctx context.Context, params CategoryListParams) (*ListResponse[Category], error) {
	path := "/categories"

	v := url.Values{}
	if params.Page > 0 {
		v.Set("page", strconv.Itoa(params.Page))
	}
	if params.Limit > 0 {
		v.Set("limit", strconv.Itoa(params.Limit))
	}
	if encoded := v.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var result ListResponse[Category]
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves a single category by ID.
func (s *CategoryService) Get(ctx context.Context, id int) (*Category, error) {
	var result Category
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/categories/%d", id), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
