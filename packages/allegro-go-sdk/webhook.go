package allegro

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
)

// WebhookEvent represents a parsed webhook event from Allegro.
type WebhookEvent struct {
	Type       string          `json:"type"`
	ID         string          `json:"id"`
	OccurredAt string          `json:"occurredAt"`
	Payload    json.RawMessage `json:"payload"`
	Order      *WebhookOrder   `json:"order,omitempty"`
}

// WebhookOrder contains the order reference from an Allegro webhook event.
type WebhookOrder struct {
	ID     string         `json:"id"`
	Seller *WebhookSeller `json:"seller,omitempty"`
}

// WebhookSeller contains the seller reference from an Allegro webhook event.
type WebhookSeller struct {
	ID string `json:"id"`
}

// OrderID returns the order ID from the webhook event, or empty string if not present.
func (e *WebhookEvent) OrderID() string {
	if e.Order != nil {
		return e.Order.ID
	}
	return ""
}

// SellerID returns the seller ID from the webhook event, or empty string if not present.
func (e *WebhookEvent) SellerID() string {
	if e.Order != nil && e.Order.Seller != nil {
		return e.Order.Seller.ID
	}
	return ""
}

// VerifyWebhook verifies the HMAC-SHA256 signature of a webhook request body.
func VerifyWebhook(secret string, signature string, body []byte) error {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return errors.New("allegro: invalid webhook signature")
	}
	return nil
}

// ParseWebhookEvent parses a raw webhook body into a WebhookEvent.
func ParseWebhookEvent(body []byte) (*WebhookEvent, error) {
	var evt WebhookEvent
	if err := json.Unmarshal(body, &evt); err != nil {
		return nil, err
	}
	return &evt, nil
}
