package apaczka

import "context"

// SearchPoints returns the list of pickup/drop-off points for the given carrier
// type and country code. pointType must be one of INPOST, UPS, or POCZTA.
func (c *Client) SearchPoints(ctx context.Context, pointType, countryCode string) ([]Point, error) {
	req := pointsRequest{CountryCode: countryCode}
	var result pointsResponse
	if err := c.do(ctx, "points/"+pointType+"/", req, &result); err != nil {
		return nil, err
	}
	return result.Points, nil
}
