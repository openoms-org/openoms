package prestashop

import (
	"context"
	"fmt"
)

// StockService handles communication with the stock-related PrestaShop endpoints.
type StockService struct {
	client *Client
}

// GetByProduct retrieves the stock available entry for a product.
func (s *StockService) GetByProduct(ctx context.Context, productID int) (*PSStockAvailable, error) {
	path := fmt.Sprintf("/stock_availables?filter[id_product]=[%d]&display=full", productID)

	var wrapper struct {
		StockAvailables []PSStockAvailable `json:"stock_availables"`
	}
	if err := s.client.do(ctx, "GET", path, nil, &wrapper); err != nil {
		return nil, err
	}
	if len(wrapper.StockAvailables) == 0 {
		return nil, &APIError{StatusCode: 404, Message: "stock not found for product"}
	}
	return &wrapper.StockAvailables[0], nil
}

// UpdateQuantity updates the stock quantity for a stock_available entry.
func (s *StockService) UpdateQuantity(ctx context.Context, stockID int, quantity int) error {
	body := map[string]any{
		"stock_available": map[string]any{
			"quantity": quantity,
		},
	}
	return s.client.do(ctx, "PUT", fmt.Sprintf("/stock_availables/%d", stockID), body, nil)
}
