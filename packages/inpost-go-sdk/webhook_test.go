package inpost

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerifyWebhookValid(t *testing.T) {
	secret := "my-webhook-secret"
	body := []byte(`{"type":"status_changed","payload":{"id":123}}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	signature := hex.EncodeToString(mac.Sum(nil))

	if err := VerifyWebhook(secret, signature, body); err != nil {
		t.Fatalf("expected valid signature, got error: %v", err)
	}
}

func TestVerifyWebhookInvalid(t *testing.T) {
	secret := "my-webhook-secret"
	body := []byte(`{"type":"status_changed","payload":{"id":123}}`)

	err := VerifyWebhook(secret, "deadbeefdeadbeef", body)
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
}

func TestVerifyWebhookWrongSecret(t *testing.T) {
	secret := "correct-secret"
	wrongSecret := "wrong-secret"
	body := []byte(`{"type":"test"}`)

	mac := hmac.New(sha256.New, []byte(wrongSecret))
	mac.Write(body)
	signature := hex.EncodeToString(mac.Sum(nil))

	err := VerifyWebhook(secret, signature, body)
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestParseWebhookEvent(t *testing.T) {
	body := []byte(`{"type":"status_changed","payload":{"shipment_id":42,"status":"delivered"}}`)

	event, err := ParseWebhookEvent(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Type != "status_changed" {
		t.Fatalf("expected type status_changed, got %s", event.Type)
	}
	if event.Payload == nil {
		t.Fatal("expected non-nil payload")
	}
}

func TestParseWebhookEventInvalid(t *testing.T) {
	_, err := ParseWebhookEvent([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
