package allegro

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// OfferService handles communication with the offer-related endpoints.
type OfferService struct {
	client *Client
}

// ListOffersParams are the optional parameters for listing offers.
type ListOffersParams struct {
	Limit             int
	Offset            int
	Name              string
	PublicationStatus string // ACTIVE, INACTIVE, ENDED
}

// List retrieves a paginated list of offers.
func (s *OfferService) List(ctx context.Context, params *ListOffersParams) (*OfferList, error) {
	path := "/sale/offers"

	if params != nil {
		v := url.Values{}
		if params.Limit > 0 {
			v.Set("limit", strconv.Itoa(params.Limit))
		}
		if params.Offset > 0 {
			v.Set("offset", strconv.Itoa(params.Offset))
		}
		if params.Name != "" {
			v.Set("name", params.Name)
		}
		if params.PublicationStatus != "" {
			v.Set("publication.status", params.PublicationStatus)
		}
		if encoded := v.Encode(); encoded != "" {
			path += "?" + encoded
		}
	}

	var result OfferList
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves a single offer by ID.
func (s *OfferService) Get(ctx context.Context, offerID string) (*Offer, error) {
	var result Offer
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/sale/product-offers/%s", offerID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
