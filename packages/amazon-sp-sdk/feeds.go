package amazon

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

// FeedService handles Amazon SP-API Feeds operations.
type FeedService struct {
	client *Client
}

// CreateDocument creates a feed document and returns a presigned upload URL.
func (s *FeedService) CreateDocument(ctx context.Context, contentType string) (*CreateFeedDocumentResponse, error) {
	req := CreateFeedDocumentRequest{ContentType: contentType}
	var resp CreateFeedDocumentResponse
	if err := s.client.do(ctx, http.MethodPost, "/feeds/2021-06-30/documents", req, &resp); err != nil {
		return nil, fmt.Errorf("amazon: create feed document: %w", err)
	}
	return &resp, nil
}

// Upload uploads content to a presigned feed document URL.
// This is a direct HTTP PUT to S3 — no SP-API auth token is used.
func (s *FeedService) Upload(ctx context.Context, uploadURL string, content []byte, contentType string) error {
	// Validate that the presigned URL points to Amazon S3 (SSRF protection).
	if err := validateS3URL(uploadURL); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, bytes.NewReader(content))
	if err != nil {
		return fmt.Errorf("amazon: create upload request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := s.client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("amazon: upload feed document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return fmt.Errorf("amazon: upload feed document failed (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// SubmitFeed submits a feed for processing.
func (s *FeedService) SubmitFeed(ctx context.Context, feedType string, marketplaceIDs []string, documentID string) (*CreateFeedResponse, error) {
	req := CreateFeedRequest{
		FeedType:            feedType,
		MarketplaceIDs:      marketplaceIDs,
		InputFeedDocumentID: documentID,
	}
	var resp CreateFeedResponse
	if err := s.client.do(ctx, http.MethodPost, "/feeds/2021-06-30/feeds", req, &resp); err != nil {
		return nil, fmt.Errorf("amazon: submit feed: %w", err)
	}
	return &resp, nil
}

// validFeedID matches Amazon feed IDs (alphanumeric + hyphens).
var validFeedID = regexp.MustCompile(`^[a-zA-Z0-9\-]+$`)

// GetFeed retrieves feed processing status.
func (s *FeedService) GetFeed(ctx context.Context, feedID string) (*Feed, error) {
	if !validFeedID.MatchString(feedID) {
		return nil, fmt.Errorf("amazon: invalid feed ID %q", feedID)
	}
	var resp Feed
	if err := s.client.do(ctx, http.MethodGet, "/feeds/2021-06-30/feeds/"+feedID, nil, &resp); err != nil {
		return nil, fmt.Errorf("amazon: get feed: %w", err)
	}
	return &resp, nil
}

// SubmitInventoryFeed creates, uploads, and submits a POST_INVENTORY_AVAILABILITY_DATA feed.
// skuQuantities maps seller SKU to available quantity.
// Returns the feed ID for optional status polling.
func (s *FeedService) SubmitInventoryFeed(ctx context.Context, marketplaceID, merchantID string, skuQuantities map[string]int) (string, error) {
	// Validate SKU lengths to prevent excessively large feeds.
	for sku := range skuQuantities {
		if len(sku) > maxSKULength {
			return "", fmt.Errorf("amazon: SKU %q exceeds max length of %d characters", sku, maxSKULength)
		}
	}

	xmlContent := buildInventoryXML(merchantID, skuQuantities)

	contentType := "text/xml; charset=UTF-8"
	doc, err := s.CreateDocument(ctx, contentType)
	if err != nil {
		return "", err
	}

	if err := s.Upload(ctx, doc.URL, xmlContent, contentType); err != nil {
		return "", err
	}

	resp, err := s.SubmitFeed(ctx, "POST_INVENTORY_AVAILABILITY_DATA", []string{marketplaceID}, doc.FeedDocumentID)
	if err != nil {
		return "", err
	}
	return resp.FeedID, nil
}

// XML structures for the inventory feed envelope.

type amazonEnvelope struct {
	XMLName     xml.Name         `xml:"AmazonEnvelope"`
	XSI         string           `xml:"xmlns:xsi,attr"`
	Schema      string           `xml:"xsi:noNamespaceSchemaLocation,attr"`
	Header      envelopeHeader   `xml:"Header"`
	MessageType string           `xml:"MessageType"`
	Messages    []inventoryMsg   `xml:"Message"`
}

type envelopeHeader struct {
	DocumentVersion    string `xml:"DocumentVersion"`
	MerchantIdentifier string `xml:"MerchantIdentifier"`
}

type inventoryMsg struct {
	MessageID int            `xml:"MessageID"`
	Inventory inventoryEntry `xml:"Inventory"`
}

type inventoryEntry struct {
	SKU      string `xml:"SKU"`
	Quantity int    `xml:"Quantity"`
}

// validateS3URL checks that a presigned URL points to Amazon S3 to prevent SSRF.
// Localhost URLs (HTTP) are allowed for testing.
func validateS3URL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("amazon: invalid presigned URL: %w", err)
	}
	host := strings.ToLower(u.Hostname())
	// Allow localhost for testing (httptest servers)
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return nil
	}
	if u.Scheme != "https" {
		return fmt.Errorf("amazon: presigned URL must use HTTPS, got %q", u.Scheme)
	}
	if !strings.HasSuffix(host, ".amazonaws.com") && !strings.HasSuffix(host, ".amazon.com") {
		return fmt.Errorf("amazon: presigned URL host %q is not an Amazon domain", host)
	}
	return nil
}

const maxSKULength = 40 // Amazon seller SKU limit

func buildInventoryXML(merchantID string, skuQuantities map[string]int) []byte {
	env := amazonEnvelope{
		XSI:         "http://www.w3.org/2001/XMLSchema-instance",
		Schema:      "amzn-envelope.xsd",
		Header: envelopeHeader{
			DocumentVersion:    "1.01",
			MerchantIdentifier: merchantID,
		},
		MessageType: "Inventory",
	}

	// Sort SKUs for deterministic output
	skus := make([]string, 0, len(skuQuantities))
	for sku := range skuQuantities {
		skus = append(skus, sku)
	}
	sort.Strings(skus)

	for i, sku := range skus {
		env.Messages = append(env.Messages, inventoryMsg{
			MessageID: i + 1,
			Inventory: inventoryEntry{
				SKU:      sku,
				Quantity: skuQuantities[sku],
			},
		})
	}

	data, _ := xml.MarshalIndent(env, "", "  ")
	return append([]byte(xml.Header), data...)
}

// XML structures for the pricing feed envelope.

type pricingEnvelope struct {
	XMLName     xml.Name       `xml:"AmazonEnvelope"`
	XSI         string         `xml:"xmlns:xsi,attr"`
	Schema      string         `xml:"xsi:noNamespaceSchemaLocation,attr"`
	Header      envelopeHeader `xml:"Header"`
	MessageType string         `xml:"MessageType"`
	Messages    []pricingMsg   `xml:"Message"`
}

type pricingMsg struct {
	MessageID int          `xml:"MessageID"`
	Price     pricingEntry `xml:"Price"`
}

type pricingEntry struct {
	SKU           string        `xml:"SKU"`
	StandardPrice standardPrice `xml:"StandardPrice"`
}

type standardPrice struct {
	Currency string  `xml:"currency,attr"`
	Value    float64 `xml:",chardata"`
}

// SubmitPricingFeed creates, uploads, and submits a POST_PRODUCT_PRICING_DATA feed.
// skuPrices maps seller SKU to price. currency is e.g. "PLN".
// Returns the feed ID for optional status polling.
func (s *FeedService) SubmitPricingFeed(ctx context.Context, marketplaceID, merchantID string, skuPrices map[string]float64, currency string) (string, error) {
	for sku := range skuPrices {
		if len(sku) > maxSKULength {
			return "", fmt.Errorf("amazon: SKU %q exceeds max length of %d characters", sku, maxSKULength)
		}
	}

	xmlContent := buildPricingXML(merchantID, skuPrices, currency)

	contentType := "text/xml; charset=UTF-8"
	doc, err := s.CreateDocument(ctx, contentType)
	if err != nil {
		return "", err
	}

	if err := s.Upload(ctx, doc.URL, xmlContent, contentType); err != nil {
		return "", err
	}

	resp, err := s.SubmitFeed(ctx, "POST_PRODUCT_PRICING_DATA", []string{marketplaceID}, doc.FeedDocumentID)
	if err != nil {
		return "", err
	}
	return resp.FeedID, nil
}

func buildPricingXML(merchantID string, skuPrices map[string]float64, currency string) []byte {
	env := pricingEnvelope{
		XSI:    "http://www.w3.org/2001/XMLSchema-instance",
		Schema: "amzn-envelope.xsd",
		Header: envelopeHeader{
			DocumentVersion:    "1.01",
			MerchantIdentifier: merchantID,
		},
		MessageType: "Price",
	}

	skus := make([]string, 0, len(skuPrices))
	for sku := range skuPrices {
		skus = append(skus, sku)
	}
	sort.Strings(skus)

	for i, sku := range skus {
		env.Messages = append(env.Messages, pricingMsg{
			MessageID: i + 1,
			Price: pricingEntry{
				SKU: sku,
				StandardPrice: standardPrice{
					Currency: currency,
					Value:    skuPrices[sku],
				},
			},
		})
	}

	data, _ := xml.MarshalIndent(env, "", "  ")
	return append([]byte(xml.Header), data...)
}
