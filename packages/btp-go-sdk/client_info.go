package btp

import (
	"context"
	"fmt"
	"net/url"
)

// ClientInfoService handles client information and reference data operations.
type ClientInfoService struct {
	client *Client
}

// GetClient returns basic client information and delivery points.
func (s *ClientInfoService) GetClient(ctx context.Context) (*ClientInfo, error) {
	var info ClientInfo
	if err := s.client.do(ctx, "GET", "/Gateway/ClientApi/ClientGet", nil, &info); err != nil {
		return nil, fmt.Errorf("btp: get client: %w", err)
	}
	return &info, nil
}

// GetDeliveryModes returns available delivery modes.
func (s *ClientInfoService) GetDeliveryModes(ctx context.Context) (*DeliveryModesResponse, error) {
	var resp DeliveryModesResponse
	if err := s.client.do(ctx, "GET", "/Gateway/ClientApi/DeliveryModesGet", nil, &resp); err != nil {
		return nil, fmt.Errorf("btp: get delivery modes: %w", err)
	}
	return &resp, nil
}

// GetCarriers returns available carriers.
func (s *ClientInfoService) GetCarriers(ctx context.Context) (*CarriersResponse, error) {
	var resp CarriersResponse
	if err := s.client.do(ctx, "GET", "/Gateway/ClientApi/CarriersGet", nil, &resp); err != nil {
		return nil, fmt.Errorf("btp: get carriers: %w", err)
	}
	return &resp, nil
}

// GetPaymentMethods returns available payment methods.
func (s *ClientInfoService) GetPaymentMethods(ctx context.Context) (*PaymentMethodsResponse, error) {
	var resp PaymentMethodsResponse
	if err := s.client.do(ctx, "GET", "/Gateway/ClientApi/PaymentMethodsGet", nil, &resp); err != nil {
		return nil, fmt.Errorf("btp: get payment methods: %w", err)
	}
	return &resp, nil
}

// GetCountryGovAreas returns governing/territorial areas for a country.
func (s *ClientInfoService) GetCountryGovAreas(ctx context.Context, countryID string) (*GovAreasResponse, error) {
	q := url.Values{}
	if countryID != "" {
		q.Set("countryId", countryID)
	}
	path := "/Gateway/ClientApi/CountryGovAreasGet"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var resp GovAreasResponse
	if err := s.client.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, fmt.Errorf("btp: get country gov areas: %w", err)
	}
	return &resp, nil
}
