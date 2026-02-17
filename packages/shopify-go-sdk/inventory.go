package shopify

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// InventoryService handles communication with the inventory-related Shopify endpoints.
type InventoryService struct {
	client *Client
}

// GetLevels retrieves inventory levels for an inventory item.
func (s *InventoryService) GetLevels(ctx context.Context, inventoryItemID int64) ([]InventoryLevel, error) {
	path := "/inventory_levels.json"

	v := url.Values{}
	v.Set("inventory_item_ids", strconv.FormatInt(inventoryItemID, 10))
	path += "?" + v.Encode()

	var wrapper struct {
		InventoryLevels []InventoryLevel `json:"inventory_levels"`
	}
	if err := s.client.do(ctx, "GET", path, nil, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.InventoryLevels, nil
}

// SetLevel sets the inventory level for an item at a location.
func (s *InventoryService) SetLevel(ctx context.Context, inventoryItemID, locationID int64, available int) error {
	body := map[string]any{
		"location_id":      locationID,
		"inventory_item_id": inventoryItemID,
		"available":         available,
	}
	return s.client.do(ctx, "POST", "/inventory_levels/set.json", body, nil)
}

// AdjustLevel adjusts the inventory level by a delta.
func (s *InventoryService) AdjustLevel(ctx context.Context, inventoryItemID, locationID int64, delta int) error {
	body := map[string]any{
		"location_id":       locationID,
		"inventory_item_id": inventoryItemID,
		"available_adjustment": delta,
	}
	return s.client.do(ctx, "POST", "/inventory_levels/adjust.json", body, nil)
}

// ListLocations retrieves all locations for the shop.
func (s *InventoryService) ListLocations(ctx context.Context) ([]Location, error) {
	var wrapper struct {
		Locations []Location `json:"locations"`
	}
	if err := s.client.do(ctx, "GET", "/locations.json", nil, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Locations, nil
}

// Location represents a Shopify fulfillment location.
type Location struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Active   bool   `json:"active"`
	Address1 string `json:"address1"`
	City     string `json:"city"`
	Zip      string `json:"zip"`
	Country  string `json:"country"`
}

// GetInventoryItem retrieves an inventory item by ID.
func (s *InventoryService) GetInventoryItem(ctx context.Context, id int64) (*InventoryItem, error) {
	var wrapper struct {
		InventoryItem InventoryItem `json:"inventory_item"`
	}
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/inventory_items/%d.json", id), nil, &wrapper); err != nil {
		return nil, err
	}
	return &wrapper.InventoryItem, nil
}

// InventoryItem represents a Shopify inventory item.
type InventoryItem struct {
	ID        int64  `json:"id"`
	SKU       string `json:"sku"`
	Tracked   bool   `json:"tracked"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
