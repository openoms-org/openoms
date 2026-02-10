package inpost

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
)

// WebhookEvent represents an incoming InPost webhook event.
type WebhookEvent struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// VerifyWebhook verifies the HMAC-SHA256 signature of a webhook payload.
// The signature is expected to be a hex-encoded string.
func VerifyWebhook(secret string, signature string, body []byte) error {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return errors.New("inpost: invalid webhook signature")
	}
	return nil
}

// ParseWebhookEvent parses the raw JSON body of a webhook into a WebhookEvent.
func ParseWebhookEvent(body []byte) (*WebhookEvent, error) {
	var event WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("inpost: failed to parse webhook event: %w", err)
	}
	return &event, nil
}
