package btp

import (
	"context"
	"fmt"
	"net/url"
)

// InvoiceService handles invoice operations.
type InvoiceService struct {
	client *Client
}

// Get retrieves invoices by document IDs and types.
func (s *InvoiceService) Get(ctx context.Context, req *InvoicesRequest) (*InvoicesResponse, error) {
	var resp InvoicesResponse
	if err := s.client.do(ctx, "POST", "/Gateway/ClientApi/InvoicesGet", req, &resp); err != nil {
		return nil, fmt.Errorf("btp: get invoices: %w", err)
	}
	return &resp, nil
}

// GetImage retrieves an invoice image in base64-encoded PDF format.
func (s *InvoiceService) GetImage(ctx context.Context, req *InvoiceImageRequest) (*InvoiceImageResponse, error) {
	q := url.Values{}
	q.Set("documentId", req.DocumentID)
	q.Set("documentType", req.DocumentType)
	q.Set("imageFormat", req.ImageFormat)
	if req.SellerID != "" {
		q.Set("sellerId", req.SellerID)
	}
	path := "/Gateway/ClientApi/InvoiceImageGet?" + q.Encode()

	var resp InvoiceImageResponse
	if err := s.client.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, fmt.Errorf("btp: get invoice image: %w", err)
	}
	return &resp, nil
}

// ConfigureNotify configures webhook notifications for new invoices.
func (s *InvoiceService) ConfigureNotify(ctx context.Context, cfg *DocumentNotifyConfig) (*APIResult, error) {
	var resp APIResult
	if err := s.client.do(ctx, "POST", "/Gateway/ClientApi/InvoiceNotifyConfigure", cfg, &resp); err != nil {
		return nil, fmt.Errorf("btp: configure invoice notify: %w", err)
	}
	return &resp, nil
}
