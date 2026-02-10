package allegro

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerifyWebhookValid(t *testing.T) {
	secret := "my-webhook-secret"
	body := []byte(`{"type":"ORDER_STATUS_CHANGED","id":"evt-1"}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))

	if err := VerifyWebhook(secret, sig, body); err != nil {
		t.Errorf("VerifyWebhook with valid signature returned error: %v", err)
	}
}

func TestVerifyWebhookInvalid(t *testing.T) {
	secret := "my-webhook-secret"
	body := []byte(`{"type":"ORDER_STATUS_CHANGED","id":"evt-1"}`)

	if err := VerifyWebhook(secret, "bad-signature", body); err == nil {
		t.Error("VerifyWebhook with invalid signature returned nil error")
	}
}

func TestVerifyWebhookTamperedBody(t *testing.T) {
	secret := "my-webhook-secret"
	original := []byte(`{"type":"ORDER_STATUS_CHANGED","id":"evt-1"}`)
	tampered := []byte(`{"type":"ORDER_STATUS_CHANGED","id":"evt-2"}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(original)
	sig := hex.EncodeToString(mac.Sum(nil))

	if err := VerifyWebhook(secret, sig, tampered); err == nil {
		t.Error("VerifyWebhook with tampered body returned nil error")
	}
}

func TestParseWebhookEvent(t *testing.T) {
	body := []byte(`{
		"type": "ORDER_STATUS_CHANGED",
		"id": "evt-123",
		"occurredAt": "2024-01-15T10:30:00Z",
		"payload": {"orderId": "order-1"}
	}`)

	evt, err := ParseWebhookEvent(body)
	if err != nil {
		t.Fatalf("ParseWebhookEvent error: %v", err)
	}
	if evt.Type != "ORDER_STATUS_CHANGED" {
		t.Errorf("Type = %q, want %q", evt.Type, "ORDER_STATUS_CHANGED")
	}
	if evt.ID != "evt-123" {
		t.Errorf("ID = %q, want %q", evt.ID, "evt-123")
	}
	if evt.OccurredAt != "2024-01-15T10:30:00Z" {
		t.Errorf("OccurredAt = %q, want %q", evt.OccurredAt, "2024-01-15T10:30:00Z")
	}
	if string(evt.Payload) != `{"orderId": "order-1"}` {
		t.Errorf("Payload = %s, want {\"orderId\": \"order-1\"}", string(evt.Payload))
	}
}

func TestParseWebhookEventInvalid(t *testing.T) {
	_, err := ParseWebhookEvent([]byte(`not json`))
	if err == nil {
		t.Error("ParseWebhookEvent with invalid JSON returned nil error")
	}
}
