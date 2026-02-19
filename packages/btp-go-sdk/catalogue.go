package btp

import (
	"context"
	"fmt"
	"net/url"
)

// CatalogueService handles product catalogue operations.
type CatalogueService struct {
	client *Client
}

// CatalogueOptions contains optional parameters for GetCatalogue.
type CatalogueOptions struct {
	UseItemCategoryFilter *bool
	LangID                string
}

// GetCatalogue returns the product catalogue.
// Recommended refresh rate: 1-2 times per day.
func (s *CatalogueService) GetCatalogue(ctx context.Context, opts *CatalogueOptions) (*ProdCatalogueDocument, error) {
	path := "/Gateway/ClientApi/ProductCatalogueGet"
	if opts != nil {
		q := url.Values{}
		if opts.UseItemCategoryFilter != nil {
			q.Set("useItemCategoryFilter", fmt.Sprintf("%t", *opts.UseItemCategoryFilter))
		}
		if opts.LangID != "" {
			q.Set("langId", opts.LangID)
		}
		if encoded := q.Encode(); encoded != "" {
			path += "?" + encoded
		}
	}

	var doc ProdCatalogueDocument
	if err := s.client.do(ctx, "GET", path, nil, &doc); err != nil {
		return nil, fmt.Errorf("btp: get product catalogue: %w", err)
	}
	return &doc, nil
}
