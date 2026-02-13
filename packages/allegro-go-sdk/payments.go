package allegro

import (
	"context"
	"net/url"
	"strconv"
)

// PaymentService handles communication with the payment and refund endpoints.
type PaymentService struct {
	client *Client
}

// ListPaymentOps retrieves a paginated list of payment operations.
func (s *PaymentService) ListPaymentOps(ctx context.Context, params *PaymentOpsParams) (*PaymentOpsList, error) {
	path := "/payments/payment-operations"

	if params != nil {
		v := url.Values{}
		if params.Limit > 0 {
			v.Set("limit", strconv.Itoa(params.Limit))
		}
		if params.Offset > 0 {
			v.Set("offset", strconv.Itoa(params.Offset))
		}
		if params.Group != "" {
			v.Set("group", params.Group)
		}
		if encoded := v.Encode(); encoded != "" {
			path += "?" + encoded
		}
	}

	var result PaymentOpsList
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateRefund creates a new refund.
func (s *PaymentService) CreateRefund(ctx context.Context, refund CreateRefundRequest) (*Refund, error) {
	var result Refund
	if err := s.client.do(ctx, "POST", "/payments/refunds", refund, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListRefunds retrieves a paginated list of refunds.
func (s *PaymentService) ListRefunds(ctx context.Context, params *ListRefundsParams) (*RefundList, error) {
	path := "/payments/refunds"

	if params != nil {
		v := url.Values{}
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

	var result RefundList
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
