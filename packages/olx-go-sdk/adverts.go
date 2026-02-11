package olx

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// AdvertService handles communication with the OLX adverts endpoints.
type AdvertService struct {
	client *Client
}

// AdvertListParams are the optional parameters for listing adverts.
type AdvertListParams struct {
	Offset int
	Limit  int
	Status string // "active", "limited", "removed", "disabled"
}

// ListAdverts retrieves a list of the authenticated user's adverts.
func (s *AdvertService) ListAdverts(ctx context.Context, params AdvertListParams) (*AdvertListResponse, error) {
	v := url.Values{}
	if params.Offset > 0 {
		v.Set("offset", strconv.Itoa(params.Offset))
	}
	if params.Limit > 0 {
		v.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Status != "" {
		v.Set("status", params.Status)
	}

	path := "/adverts"
	if encoded := v.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var result AdvertListResponse
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAdvert retrieves a single advert by ID.
func (s *AdvertService) GetAdvert(ctx context.Context, id int64) (*Advert, error) {
	var wrapper struct {
		Data Advert `json:"data"`
	}
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/adverts/%d", id), nil, &wrapper); err != nil {
		return nil, err
	}
	return &wrapper.Data, nil
}

// TransactionService handles communication with the OLX transactions endpoints.
type TransactionService struct {
	client *Client
}

// TransactionListParams are the optional parameters for listing transactions.
type TransactionListParams struct {
	Offset     int
	Limit      int
	CreatedAfter string // ISO8601 timestamp
}

// ListTransactions retrieves a list of transactions for the authenticated seller.
func (s *TransactionService) ListTransactions(ctx context.Context, params TransactionListParams) (*TransactionListResponse, error) {
	v := url.Values{}
	if params.Offset > 0 {
		v.Set("offset", strconv.Itoa(params.Offset))
	}
	if params.Limit > 0 {
		v.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.CreatedAfter != "" {
		v.Set("created_after", params.CreatedAfter)
	}

	path := "/transactions"
	if encoded := v.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var result TransactionListResponse
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
