package allegro

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// ProductCatalogService handles Allegro product catalog (matching system).
type ProductCatalogService struct {
	client *Client
}

// Search searches the Allegro product catalog.
func (s *ProductCatalogService) Search(ctx context.Context, params *SearchProductsParams) (*ProductCatalogList, error) {
	path := "/sale/products"

	if params != nil {
		v := url.Values{}
		if params.Phrase != "" {
			v.Set("phrase", params.Phrase)
		}
		if params.CategoryID != "" {
			v.Set("category.id", params.CategoryID)
		}
		if params.Limit > 0 {
			v.Set("limit", strconv.Itoa(params.Limit))
		}
		if params.Offset > 0 {
			v.Set("offset", strconv.Itoa(params.Offset))
		}
		if encoded := v.Encode(); encoded != "" {
			path += "?" + encoded
		}
	}

	var result ProductCatalogList
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves a single product from the Allegro catalog.
func (s *ProductCatalogService) Get(ctx context.Context, productID string) (*CatalogProduct, error) {
	var result CatalogProduct
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/sale/products/%s", productID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
