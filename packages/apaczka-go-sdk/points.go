package apaczka

import "context"

// SearchPoints returns the list of pickup/drop-off points for the given carrier
// type and country code. pointType must be one of INPOST, UPS, or POCZTA.
// subtype is optional; pass "" to omit it from the request.
func (c *Client) SearchPoints(ctx context.Context, pointType, countryCode, subtype string) ([]Point, error) {
	req := pointsRequest{CountryCode: countryCode, Subtype: subtype}
	var result pointsResponse
	if err := c.do(ctx, "points/"+pointType+"/", req, &result); err != nil {
		return nil, err
	}
	if result.Points == nil {
		return []Point{}, nil
	}
	return result.Points, nil
}
