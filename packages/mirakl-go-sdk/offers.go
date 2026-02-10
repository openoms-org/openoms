package mirakl

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// OfferService handles Mirakl offer-related API operations.
type OfferService struct {
	client *Client
}

// UpdateOffers performs a batch stock/price update for offers.
func (s *OfferService) UpdateOffers(ctx context.Context, updates []OfferUpdate) error {
	req := OfferUpdateRequest{Offers: updates}
	if err := s.client.do(ctx, http.MethodPost, "/offers", req, nil); err != nil {
		return fmt.Errorf("mirakl: update offers: %w", err)
	}
	return nil
}

// GetOffer retrieves an offer by its shop SKU.
func (s *OfferService) GetOffer(ctx context.Context, sku string) (*OfferDetail, error) {
	path := "/offers?shop_skus=" + url.QueryEscape(sku)
	var resp OfferResponse
	if err := s.client.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("mirakl: get offer %s: %w", sku, err)
	}
	if len(resp.Offers) == 0 {
		return nil, fmt.Errorf("mirakl: offer %s not found", sku)
	}
	return &resp.Offers[0], nil
}
