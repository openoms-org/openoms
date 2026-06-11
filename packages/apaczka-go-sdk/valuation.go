package apaczka

import "context"

// GetValuation returns the price table for all available services given a
// sample order payload. Prices are returned in grosze (Polish sub-unit).
func (c *Client) GetValuation(ctx context.Context, req *ValuationRequest) (*ValuationResponse, error) {
	var result ValuationResponse
	if err := c.do(ctx, "order_valuation/", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
