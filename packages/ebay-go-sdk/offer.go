package ebay

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// OfferService handles eBay Sell Inventory API offer endpoints.
type OfferService struct {
	client *Client
}

// CreateOffer creates a new offer for an inventory item.
// POST /sell/inventory/v1/offer
func (s *OfferService) CreateOffer(ctx context.Context, offer Offer) (*CreateOfferResponse, error) {
	var result CreateOfferResponse
	if err := s.client.do(ctx, "POST", "/sell/inventory/v1/offer", offer, &result); err != nil {
		return nil, fmt.Errorf("ebay: create offer: %w", err)
	}
	return &result, nil
}

// GetOffer returns an offer by ID.
// GET /sell/inventory/v1/offer/{offerId}
func (s *OfferService) GetOffer(ctx context.Context, offerID string) (*Offer, error) {
	var result Offer
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/sell/inventory/v1/offer/%s", offerID), nil, &result); err != nil {
		return nil, fmt.Errorf("ebay: get offer %s: %w", offerID, err)
	}
	return &result, nil
}

// UpdateOffer updates an existing offer.
// PUT /sell/inventory/v1/offer/{offerId}
func (s *OfferService) UpdateOffer(ctx context.Context, offerID string, offer Offer) error {
	if err := s.client.do(ctx, "PUT", fmt.Sprintf("/sell/inventory/v1/offer/%s", offerID), offer, nil); err != nil {
		return fmt.Errorf("ebay: update offer %s: %w", offerID, err)
	}
	return nil
}

// GetOffers returns offers filtered by SKU.
// GET /sell/inventory/v1/offer?sku=X&limit=N&offset=N
func (s *OfferService) GetOffers(ctx context.Context, sku string, limit, offset int) (*OffersResponse, error) {
	v := url.Values{}
	if sku != "" {
		v.Set("sku", sku)
	}
	if limit > 0 {
		v.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		v.Set("offset", strconv.Itoa(offset))
	}
	path := "/sell/inventory/v1/offer"
	if encoded := v.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var result OffersResponse
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, fmt.Errorf("ebay: get offers: %w", err)
	}
	return &result, nil
}

// PublishOffer publishes an offer, making it live on eBay.
// POST /sell/inventory/v1/offer/{offerId}/publish
func (s *OfferService) PublishOffer(ctx context.Context, offerID string) (*PublishResponse, error) {
	var result PublishResponse
	if err := s.client.do(ctx, "POST", fmt.Sprintf("/sell/inventory/v1/offer/%s/publish", offerID), nil, &result); err != nil {
		return nil, fmt.Errorf("ebay: publish offer %s: %w", offerID, err)
	}
	return &result, nil
}

// WithdrawOffer withdraws a published offer from eBay.
// POST /sell/inventory/v1/offer/{offerId}/withdraw
func (s *OfferService) WithdrawOffer(ctx context.Context, offerID string) error {
	if err := s.client.do(ctx, "POST", fmt.Sprintf("/sell/inventory/v1/offer/%s/withdraw", offerID), nil, nil); err != nil {
		return fmt.Errorf("ebay: withdraw offer %s: %w", offerID, err)
	}
	return nil
}

// DeleteOffer deletes an unpublished offer.
// DELETE /sell/inventory/v1/offer/{offerId}
func (s *OfferService) DeleteOffer(ctx context.Context, offerID string) error {
	if err := s.client.do(ctx, "DELETE", fmt.Sprintf("/sell/inventory/v1/offer/%s", offerID), nil, nil); err != nil {
		return fmt.Errorf("ebay: delete offer %s: %w", offerID, err)
	}
	return nil
}
