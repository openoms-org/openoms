package inpost

import (
	"context"
	"fmt"
	"net/http"
)

// DispatchOrderService handles dispatch-order-related API operations.
type DispatchOrderService struct {
	client *Client
}

// Create creates a new dispatch order for the given shipments.
func (s *DispatchOrderService) Create(ctx context.Context, req *CreateDispatchOrderRequest) (*DispatchOrder, error) {
	path := fmt.Sprintf("/v1/organizations/%s/dispatch_orders", s.client.orgID)
	var order DispatchOrder
	if err := s.client.do(ctx, http.MethodPost, path, req, &order); err != nil {
		return nil, err
	}
	return &order, nil
}

// Get retrieves a dispatch order by its ID.
func (s *DispatchOrderService) Get(ctx context.Context, id int64) (*DispatchOrder, error) {
	path := fmt.Sprintf("/v1/organizations/%s/dispatch_orders/%d", s.client.orgID, id)
	var order DispatchOrder
	if err := s.client.do(ctx, http.MethodGet, path, nil, &order); err != nil {
		return nil, err
	}
	return &order, nil
}

// Delete cancels a dispatch order by its ID.
func (s *DispatchOrderService) Delete(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/v1/organizations/%s/dispatch_orders/%d", s.client.orgID, id)
	if err := s.client.do(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return err
	}
	return nil
}
