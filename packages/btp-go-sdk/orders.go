package btp

import (
	"context"
	"fmt"
)

// OrderService handles order operations.
type OrderService struct {
	client *Client
}

// Create places a new order in the BTP system.
func (s *OrderService) Create(ctx context.Context, req *OrderRequest) (*OrderResponse, error) {
	var resp OrderResponse
	if err := s.client.do(ctx, "POST", "/Gateway/ClientApi/OrderCreate", req, &resp); err != nil {
		return nil, fmt.Errorf("btp: create order: %w", err)
	}
	return &resp, nil
}

// SetAttachments adds file attachments to an existing order.
func (s *OrderService) SetAttachments(ctx context.Context, req *OrderAttachmentsRequest) (*OrderAttachmentsResponse, error) {
	var resp OrderAttachmentsResponse
	if err := s.client.do(ctx, "POST", "/Gateway/ClientApi/OrderAttachmentsSet", req, &resp); err != nil {
		return nil, fmt.Errorf("btp: set order attachments: %w", err)
	}
	return &resp, nil
}
