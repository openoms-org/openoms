package apaczka

import (
	"context"
	"encoding/base64"
	"fmt"
)

// GetWaybill fetches the shipping label for a given order and returns the raw
// PDF bytes (the API delivers the label as a base64-encoded string).
func (c *Client) GetWaybill(ctx context.Context, orderID string) ([]byte, error) {
	var result waybillResponse
	if err := c.do(ctx, "waybill/"+orderID+"/", nil, &result); err != nil {
		return nil, err
	}

	data, err := base64.StdEncoding.DecodeString(result.Waybill)
	if err != nil {
		return nil, fmt.Errorf("apaczka: decode waybill base64: %w", err)
	}
	return data, nil
}
