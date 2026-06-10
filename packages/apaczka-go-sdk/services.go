package apaczka

import "context"

// GetServiceStructure returns the full list of shipping services available
// through the Apaczka broker.
func (c *Client) GetServiceStructure(ctx context.Context) (*ServiceStructureResponse, error) {
	var result ServiceStructureResponse
	if err := c.do(ctx, "service_structure/", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
