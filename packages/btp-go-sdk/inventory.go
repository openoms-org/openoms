package btp

import (
	"context"
	"fmt"
	"net/url"
)

// InventoryService handles inventory report operations.
type InventoryService struct {
	client *Client
}

// InventoryReportOptions contains optional parameters for GetReport.
type InventoryReportOptions struct {
	UseItemCategoryFilter *bool
	LangID                string
}

// GetReport returns the inventory report with prices and stock quantities.
// Recommended refresh rate: 4-24 times per day.
func (s *InventoryService) GetReport(ctx context.Context, opts *InventoryReportOptions) (*InvReportDocument, error) {
	path := "/Gateway/ClientApi/InventoryReportGet"
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

	var doc InvReportDocument
	if err := s.client.do(ctx, "GET", path, nil, &doc); err != nil {
		return nil, fmt.Errorf("btp: get inventory report: %w", err)
	}
	return &doc, nil
}
