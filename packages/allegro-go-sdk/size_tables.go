package allegro

import (
	"context"
	"fmt"
)

// SizeTableService handles Allegro size tables (tabele rozmiarow).
type SizeTableService struct {
	client *Client
}

// List lists the seller's size tables.
// GET /sale/size-tables
func (s *SizeTableService) List(ctx context.Context) (*SizeTableList, error) {
	var result SizeTableList
	if err := s.client.do(ctx, "GET", "/sale/size-tables", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Get gets a single size table.
// GET /sale/size-tables/{id}
func (s *SizeTableService) Get(ctx context.Context, tableID string) (*SizeTable, error) {
	var result SizeTable
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/sale/size-tables/%s", tableID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Create creates a new size table.
// POST /sale/size-tables
func (s *SizeTableService) Create(ctx context.Context, table CreateSizeTableRequest) (*SizeTable, error) {
	var result SizeTable
	if err := s.client.do(ctx, "POST", "/sale/size-tables", table, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Update updates a size table.
// PUT /sale/size-tables/{id}
func (s *SizeTableService) Update(ctx context.Context, tableID string, table CreateSizeTableRequest) (*SizeTable, error) {
	var result SizeTable
	if err := s.client.do(ctx, "PUT", fmt.Sprintf("/sale/size-tables/%s", tableID), table, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete deletes a size table.
// DELETE /sale/size-tables/{id}
func (s *SizeTableService) Delete(ctx context.Context, tableID string) error {
	return s.client.do(ctx, "DELETE", fmt.Sprintf("/sale/size-tables/%s", tableID), nil, nil)
}
