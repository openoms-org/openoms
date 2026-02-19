package btp

import (
	"context"
	"fmt"
)

// WaybillService handles waybill (shipping document) operations.
type WaybillService struct {
	client *Client
}

// Get retrieves waybills by their numbers.
func (s *WaybillService) Get(ctx context.Context, req *WaybillsRequest) (*WaybillsResponse, error) {
	var resp WaybillsResponse
	if err := s.client.do(ctx, "POST", "/Gateway/ClientApi/WaybillsGet", req, &resp); err != nil {
		return nil, fmt.Errorf("btp: get waybills: %w", err)
	}
	return &resp, nil
}

// ConfigureNotify configures webhook notifications for new waybills.
func (s *WaybillService) ConfigureNotify(ctx context.Context, cfg *DocumentNotifyConfig) (*APIResult, error) {
	var resp APIResult
	if err := s.client.do(ctx, "POST", "/Gateway/ClientApi/WaybillNotifyConfigure", cfg, &resp); err != nil {
		return nil, fmt.Errorf("btp: configure waybill notify: %w", err)
	}
	return &resp, nil
}
