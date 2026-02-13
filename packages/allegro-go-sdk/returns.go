package allegro

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// ReturnService handles communication with the customer return endpoints.
type ReturnService struct {
	client *Client
}

// ListReturns retrieves a paginated list of customer returns.
func (s *ReturnService) ListReturns(ctx context.Context, params *ListReturnsParams) (*CustomerReturnList, error) {
	path := "/order/customer-returns"

	if params != nil {
		v := url.Values{}
		if params.Limit > 0 {
			v.Set("limit", strconv.Itoa(params.Limit))
		}
		if params.Offset > 0 {
			v.Set("offset", strconv.Itoa(params.Offset))
		}
		if params.Status != "" {
			v.Set("status", params.Status)
		}
		if encoded := v.Encode(); encoded != "" {
			path += "?" + encoded
		}
	}

	var result CustomerReturnList
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetReturn retrieves a single customer return by ID.
func (s *ReturnService) GetReturn(ctx context.Context, returnID string) (*CustomerReturn, error) {
	var result CustomerReturn
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/order/customer-returns/%s", returnID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// RejectReturn rejects a customer return with a given reason.
func (s *ReturnService) RejectReturn(ctx context.Context, returnID string, rejection ReturnRejection) error {
	return s.client.do(ctx, "POST", fmt.Sprintf("/order/customer-returns/%s/rejection", returnID), rejection, nil)
}
