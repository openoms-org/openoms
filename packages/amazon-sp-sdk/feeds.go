package amazon

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"
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
func (s *FeedService) Upload(ctx context.Context, url string, content []byte, contentType string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(content))
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

// GetFeed retrieves feed processing status.
func (s *FeedService) GetFeed(ctx context.Context, feedID string) (*Feed, error) {
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
