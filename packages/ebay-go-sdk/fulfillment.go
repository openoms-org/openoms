package ebay

import (
	"context"
	"fmt"
)

// FulfillmentService handles eBay Fulfillment API shipping fulfillment endpoints.
type FulfillmentService struct {
	client *Client
}

// CreateShippingFulfillment pushes tracking/carrier information to eBay for an order.
// POST /sell/fulfillment/v1/order/{orderId}/shipping_fulfillment
func (s *FulfillmentService) CreateShippingFulfillment(ctx context.Context, orderID string, req ShippingFulfillmentRequest) (*ShippingFulfillment, error) {
	path := fmt.Sprintf("/sell/fulfillment/v1/order/%s/shipping_fulfillment", orderID)
	var result ShippingFulfillment
	if err := s.client.do(ctx, "POST", path, req, &result); err != nil {
		return nil, fmt.Errorf("ebay: create shipping fulfillment for order %s: %w", orderID, err)
	}
	return &result, nil
}

// GetShippingFulfillments returns all shipping fulfillments for an order.
// GET /sell/fulfillment/v1/order/{orderId}/shipping_fulfillment
func (s *FulfillmentService) GetShippingFulfillments(ctx context.Context, orderID string) (*ShippingFulfillmentPagedCollection, error) {
	path := fmt.Sprintf("/sell/fulfillment/v1/order/%s/shipping_fulfillment", orderID)
	var result ShippingFulfillmentPagedCollection
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, fmt.Errorf("ebay: get shipping fulfillments for order %s: %w", orderID, err)
	}
	return &result, nil
}

// IssueRefund issues a refund for an order.
// POST /sell/fulfillment/v1/order/{orderId}/issue_refund
func (s *FulfillmentService) IssueRefund(ctx context.Context, orderID string, req IssueRefundRequest) (*IssueRefundResponse, error) {
	path := fmt.Sprintf("/sell/fulfillment/v1/order/%s/issue_refund", orderID)
	var result IssueRefundResponse
	if err := s.client.do(ctx, "POST", path, req, &result); err != nil {
		return nil, fmt.Errorf("ebay: issue refund for order %s: %w", orderID, err)
	}
	return &result, nil
}
