package apaczka

import "context"

// GetTracking returns the tracking history for the given waybill number.
func (c *Client) GetTracking(ctx context.Context, waybillNumber string) (*TrackingResponse, error) {
	var result TrackingResponse
	if err := c.do(ctx, "tracking/"+waybillNumber+"/", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
