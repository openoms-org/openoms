package kaufland

import (
	"context"
	"fmt"
)

// UnitService handles Kaufland Seller API unit endpoints.
type UnitService struct {
	client *Client
}

// UpdateStock updates the stock quantity for a unit.
func (s *UnitService) UpdateStock(ctx context.Context, unitID int64, quantity int) error {
	body := map[string]any{"quantity": quantity}
	return s.client.do(ctx, "PATCH", fmt.Sprintf("/units/%d/", unitID), body, nil)
}

// UpdatePrice updates the listing price for a unit.
func (s *UnitService) UpdatePrice(ctx context.Context, unitID int64, price float64) error {
	body := map[string]any{"listing_price": price}
	return s.client.do(ctx, "PATCH", fmt.Sprintf("/units/%d/", unitID), body, nil)
}
