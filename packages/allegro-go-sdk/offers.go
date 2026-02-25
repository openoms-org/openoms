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

// ListAll fetches all seller offers by automatically paginating through the List endpoint.
// It uses the maximum page size (1000) and collects all OfferSummary items.
// Returns an empty slice (not nil) when the seller has no offers.
// On error, returns any partially collected offers along with the error.
func (s *OfferService) ListAll(ctx context.Context) ([]OfferSummary, error) {
	const maxPageSize = 1000
	var all []OfferSummary

	for offset := 0; ; offset += maxPageSize {
		if err := ctx.Err(); err != nil {
			return all, err
		}

		page, err := s.List(ctx, &ListOffersParams{
			Limit:  maxPageSize,
			Offset: offset,
		})
		if err != nil {
			return all, fmt.Errorf("list offers offset=%d: %w", offset, err)
		}

		if len(page.Offers) == 0 {
			break
		}
		all = append(all, page.Offers...)
		if len(all) >= page.TotalCount {
			break
		}
	}

	if all == nil {
		all = []OfferSummary{}
	}
	return all, nil
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
// Offers are grouped by quantity value; one command is issued per unique value
// using the Allegro command pattern (modification + offerCriteria).
func (s *OfferService) BulkUpdateStock(ctx context.Context, updates []StockUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	// Group offers by target quantity
	byQty := map[int][]string{}
	for _, u := range updates {
		byQty[u.Quantity] = append(byQty[u.Quantity], u.OfferID)
	}

	for qty, offerIDs := range byQty {
		commandID := generateCommandID()
		offers := make([]map[string]string, len(offerIDs))
		for i, id := range offerIDs {
			offers[i] = map[string]string{"id": id}
		}
		body := map[string]any{
			"modification": map[string]any{
				"changeType": "FIXED",
				"value":      qty,
			},
			"offerCriteria": []map[string]any{
				{
					"type":   "CONTAINS_OFFERS",
					"offers": offers,
				},
			},
		}
		if err := s.client.do(ctx, "PUT",
			fmt.Sprintf("/sale/offer-quantity-change-commands/%s", commandID), body, nil); err != nil {
			return fmt.Errorf("bulk stock update (qty=%d, offers=%d): %w", qty, len(offerIDs), err)
		}
	}
	return nil
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
// Offers are grouped by price+currency; one command is issued per unique price
// using the Allegro command pattern (modification + offerCriteria).
func (s *OfferService) BulkUpdatePrice(ctx context.Context, updates []PriceUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	// Group offers by price+currency
	type priceKey struct {
		Amount   string
		Currency string
	}
	byPrice := map[priceKey][]string{}
	for _, u := range updates {
		key := priceKey{Amount: fmt.Sprintf("%.2f", u.Amount), Currency: u.Currency}
		byPrice[key] = append(byPrice[key], u.OfferID)
	}

	for key, offerIDs := range byPrice {
		commandID := generateCommandID()
		offers := make([]map[string]string, len(offerIDs))
		for i, id := range offerIDs {
			offers[i] = map[string]string{"id": id}
		}
		body := map[string]any{
			"modification": map[string]any{
				"type": "FIXED_PRICE",
				"price": map[string]string{
					"amount":   key.Amount,
					"currency": key.Currency,
				},
			},
			"offerCriteria": []map[string]any{
				{
					"type":   "CONTAINS_OFFERS",
					"offers": offers,
				},
			},
		}
		if err := s.client.do(ctx, "PUT",
			fmt.Sprintf("/sale/offer-price-change-commands/%s", commandID), body, nil); err != nil {
			return fmt.Errorf("bulk price update (price=%s %s, offers=%d): %w",
				key.Amount, key.Currency, len(offerIDs), err)
		}
	}
	return nil
}
