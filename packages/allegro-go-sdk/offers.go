package allegro

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

// UpdateStock updates the stock quantity for an offer.
func (s *OfferService) UpdateStock(ctx context.Context, offerID string, quantity int) error {
	body := map[string]any{
		"stock": map[string]any{
			"available": quantity,
		},
	}
	return s.client.do(ctx, "PATCH", fmt.Sprintf("/sale/product-offers/%s", offerID), body, nil)
}

// UpdatePrice updates the selling price for an offer.
func (s *OfferService) UpdatePrice(ctx context.Context, offerID string, amount float64, currency string) error {
	body := map[string]any{
		"sellingMode": map[string]any{
			"price": map[string]any{
				"amount":   fmt.Sprintf("%.2f", amount),
				"currency": currency,
			},
		},
	}
	return s.client.do(ctx, "PATCH", fmt.Sprintf("/sale/product-offers/%s", offerID), body, nil)
}

// Create creates a new product offer.
func (s *OfferService) Create(ctx context.Context, offer any) (*Offer, error) {
	var result Offer
	if err := s.client.do(ctx, "POST", "/sale/product-offers", offer, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// generateCommandID creates a random hex command ID for Allegro command-based APIs.
func generateCommandID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Deactivate ends (deactivates) a single offer using publication commands.
func (s *OfferService) Deactivate(ctx context.Context, offerID string) error {
	commandID := generateCommandID()
	body := map[string]any{
		"offerCriteria": []map[string]any{
			{
				"offers": []map[string]string{{"id": offerID}},
				"type":   "CONTAINS_OFFERS",
			},
		},
		"publication": map[string]string{
			"action": "END",
		},
	}
	return s.client.do(ctx, "PUT", fmt.Sprintf("/sale/offer-publication-commands/%s", commandID), body, nil)
}

// Activate activates a single offer using publication commands.
func (s *OfferService) Activate(ctx context.Context, offerID string) error {
	commandID := generateCommandID()
	body := map[string]any{
		"offerCriteria": []map[string]any{
			{
				"offers": []map[string]string{{"id": offerID}},
				"type":   "CONTAINS_OFFERS",
			},
		},
		"publication": map[string]string{
			"action": "ACTIVATE",
		},
	}
	return s.client.do(ctx, "PUT", fmt.Sprintf("/sale/offer-publication-commands/%s", commandID), body, nil)
}

// BulkUpdateStock updates the stock quantity for multiple offers at once.
func (s *OfferService) BulkUpdateStock(ctx context.Context, updates []StockUpdate) error {
	commandID := generateCommandID()
	modifications := make([]map[string]any, len(updates))
	for i, u := range updates {
		modifications[i] = map[string]any{
			"id": u.OfferID,
			"input": map[string]any{
				"stock": map[string]int{"available": u.Quantity},
			},
		}
	}
	body := map[string]any{
		"modification": modifications,
	}
	return s.client.do(ctx, "PUT", fmt.Sprintf("/sale/offer-quantity-change-commands/%s", commandID), body, nil)
}

// UploadImageURL uploads an image by URL to Allegro's image hosting service.
// The URL must be publicly accessible (not localhost).
// Returns the hosted image URL (allegroimg.com).
func (s *OfferService) UploadImageURL(ctx context.Context, imageURL string) (string, error) {
	body := map[string]string{"url": imageURL}
	var result struct {
		Location string `json:"location"`
	}
	if err := s.client.doUpload(ctx, "/sale/images", body, &result); err != nil {
		return "", err
	}
	return result.Location, nil
}

// UploadImageBinary uploads raw image bytes to Allegro's image hosting service.
// contentType should be "image/jpeg", "image/png", or "image/webp".
// Returns the hosted image URL (allegroimg.com).
func (s *OfferService) UploadImageBinary(ctx context.Context, data []byte, contentType string) (string, error) {
	return s.client.doUploadBinary(ctx, "/sale/images", data, contentType)
}

// ResponsibleProducer represents a GPSR responsible producer.
type ResponsibleProducer struct {
	ID           string                   `json:"id"`
	Name         string                   `json:"name,omitempty"`
	ProducerData *ResponsibleProducerData `json:"producerData,omitempty"`
}

// ResponsibleProducerData contains the producer's contact details.
type ResponsibleProducerData struct {
	TradeName string                     `json:"tradeName"`
	Address   ResponsibleProducerAddress `json:"address"`
	Contact   ResponsibleProducerContact `json:"contact"`
}

// ResponsibleProducerAddress is the address of the producer.
type ResponsibleProducerAddress struct {
	Street      string `json:"street"`
	PostalCode  string `json:"postalCode"`
	City        string `json:"city"`
	CountryCode string `json:"countryCode"`
}

// ResponsibleProducerContact is the contact info of the producer.
type ResponsibleProducerContact struct {
	Email       string `json:"email"`
	PhoneNumber string `json:"phoneNumber,omitempty"`
}

// ListResponsibleProducers lists the seller's responsible producers.
func (s *OfferService) ListResponsibleProducers(ctx context.Context) ([]ResponsibleProducer, error) {
	var result struct {
		Producers []ResponsibleProducer `json:"responsibleProducers"`
	}
	if err := s.client.do(ctx, "GET", "/sale/responsible-producers", nil, &result); err != nil {
		return nil, err
	}
	return result.Producers, nil
}

// CreateResponsibleProducer creates a new GPSR responsible producer.
func (s *OfferService) CreateResponsibleProducer(ctx context.Context, name string, data ResponsibleProducerData) (*ResponsibleProducer, error) {
	body := map[string]any{
		"name":         name,
		"producerData": data,
	}
	var result ResponsibleProducer
	if err := s.client.do(ctx, "POST", "/sale/responsible-producers", body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// BulkUpdatePrice updates the price for multiple offers at once.
func (s *OfferService) BulkUpdatePrice(ctx context.Context, updates []PriceUpdate) error {
	commandID := generateCommandID()
	modifications := make([]map[string]any, len(updates))
	for i, u := range updates {
		modifications[i] = map[string]any{
			"id": u.OfferID,
			"input": map[string]any{
				"buyNowPrice": map[string]string{
					"amount":   fmt.Sprintf("%.2f", u.Amount),
					"currency": u.Currency,
				},
			},
		}
	}
	body := map[string]any{
		"modification": modifications,
	}
	return s.client.do(ctx, "PUT", fmt.Sprintf("/sale/offer-price-change-commands/%s", commandID), body, nil)
}
