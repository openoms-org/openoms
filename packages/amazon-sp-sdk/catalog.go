package amazon

import (
	"context"
	"fmt"
	"net/url"
)

// CatalogService handles communication with the Amazon SP-API Catalog Items endpoints.
type CatalogService struct {
	client *Client
}

// GetItem retrieves a catalog item by ASIN.
func (s *CatalogService) GetItem(ctx context.Context, asin, marketplaceID string) (*CatalogItemResponse, error) {
	v := url.Values{}
	v.Set("marketplaceIds", marketplaceID)
	v.Set("includedData", "summaries")

	path := fmt.Sprintf("/catalog/2022-04-01/items/%s?%s", asin, v.Encode())

	var result CatalogItemResponse
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, fmt.Errorf("amazon: get catalog item %s: %w", asin, err)
	}
	return &result, nil
}
