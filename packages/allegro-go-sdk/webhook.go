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
