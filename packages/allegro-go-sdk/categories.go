package allegro

import (
	"context"
	"fmt"
	"net/url"
)

// CategoryService handles Allegro category browsing.
type CategoryService struct {
	client *Client
}

// List lists categories. parentID="" returns root categories.
func (s *CategoryService) List(ctx context.Context, parentID string) (*CategoryList, error) {
	path := "/sale/categories"

	if parentID != "" {
		v := url.Values{}
		v.Set("parent.id", parentID)
		path += "?" + v.Encode()
	}

	var result CategoryList
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves a single category by ID.
func (s *CategoryService) Get(ctx context.Context, categoryID string) (*Category, error) {
	var result Category
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/sale/categories/%s", categoryID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetParameters retrieves required parameters for a category (needed for creating offers).
func (s *CategoryService) GetParameters(ctx context.Context, categoryID string) (*CategoryParameterList, error) {
	var result CategoryParameterList
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/sale/categories/%s/parameters", categoryID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SearchMatching returns category suggestions for a given phrase.
// GET /sale/matching-categories?name=...
func (s *CategoryService) SearchMatching(ctx context.Context, name string) (*MatchingCategoriesResponse, error) {
	v := url.Values{}
	v.Set("name", name)
	var result MatchingCategoriesResponse
	if err := s.client.do(ctx, "GET", "/sale/matching-categories?"+v.Encode(), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
