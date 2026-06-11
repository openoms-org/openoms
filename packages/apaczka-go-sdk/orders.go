package apaczka

import "context"

// CreateOrder submits a new shipment order to the Apaczka broker.
func (c *Client) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*Order, error) {
	var result createOrderResponse
	if err := c.do(ctx, "order_send/", req, &result); err != nil {
		return nil, err
	}
	return &result.Order, nil
}

// GetOrderStatus retrieves the current status and details for an existing order.
func (c *Client) GetOrderStatus(ctx context.Context, orderID string) (*Order, error) {
	var result getOrderResponse
	if err := c.do(ctx, "order/"+orderID+"/", nil, &result); err != nil {
		return nil, err
	}
	return &result.Order, nil
}

// CancelOrder cancels an existing order. Returns nil on success.
func (c *Client) CancelOrder(ctx context.Context, orderID string) error {
	return c.do(ctx, "cancel_order/"+orderID+"/", nil, nil)
}
