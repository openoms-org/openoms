package olx

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// CategoryService handles communication with the OLX categories endpoints.
type CategoryService struct {
	client *Client
}

// ListCategories retrieves categories, optionally filtered by parent_id.
func (s *CategoryService) ListCategories(ctx context.Context, parentID int) (*CategoryListResponse, error) {
	v := url.Values{}
	if parentID > 0 {
		v.Set("parent_id", strconv.Itoa(parentID))
	}
	path := "/categories"
	if encoded := v.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var result CategoryListResponse
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetCategory retrieves a single category by ID.
func (s *CategoryService) GetCategory(ctx context.Context, id int) (*Category, error) {
	var wrapper struct {
		Data Category `json:"data"`
	}
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/categories/%d", id), nil, &wrapper); err != nil {
		return nil, err
	}
	return &wrapper.Data, nil
}

// GetAttributes retrieves the attribute definitions for a category.
func (s *CategoryService) GetAttributes(ctx context.Context, categoryID int) (*CategoryAttributeListResponse, error) {
	var result CategoryAttributeListResponse
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/categories/%d/attributes", categoryID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SuggestCategory suggests categories based on an advert title.
func (s *CategoryService) SuggestCategory(ctx context.Context, query string) (*CategorySuggestionResponse, error) {
	v := url.Values{}
	v.Set("q", query)
	var result CategorySuggestionResponse
	if err := s.client.do(ctx, "GET", "/categories/suggestion?"+v.Encode(), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
