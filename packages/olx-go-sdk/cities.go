package olx

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// CityService handles OLX Partner API city endpoints.
type CityService struct {
	client *Client
}

// ListCities returns cities, optionally filtered by query string.
// OLX API: GET /partner/cities?offset=0&limit=50&query=...
func (s *CityService) ListCities(ctx context.Context, query string, offset, limit int) (*CityListResponse, error) {
	v := url.Values{}
	v.Set("offset", strconv.Itoa(offset))
	v.Set("limit", strconv.Itoa(limit))
	if query != "" {
		v.Set("query", query)
	}
	var result CityListResponse
	if err := s.client.do(ctx, "GET", "/cities?"+v.Encode(), nil, &result); err != nil {
		return nil, fmt.Errorf("olx: list cities: %w", err)
	}
	return &result, nil
}

// GetCity returns a single city by ID.
func (s *CityService) GetCity(ctx context.Context, cityID int) (*City, error) {
	var wrapper struct {
		Data City `json:"data"`
	}
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/cities/%d", cityID), nil, &wrapper); err != nil {
		return nil, fmt.Errorf("olx: get city %d: %w", cityID, err)
	}
	return &wrapper.Data, nil
}
