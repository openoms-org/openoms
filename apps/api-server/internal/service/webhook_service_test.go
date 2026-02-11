package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerifyHMAC_Valid(t *testing.T) {
	secret := "my-webhook-secret"
	body := []byte(`{"event":"order.created"}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	signature := hex.EncodeToString(mac.Sum(nil))

	assert.True(t, verifyHMAC(body, signature, secret))
}

func TestVerifyHMAC_Invalid(t *testing.T) {
	body := []byte(`{"event":"order.created"}`)
	assert.False(t, verifyHMAC(body, "wrong-signature", "secret"))
}

func TestVerifyHMAC_EmptySignature(t *testing.T) {
	body := []byte(`{"event":"order.created"}`)
	assert.False(t, verifyHMAC(body, "", "secret"))
}

func TestExtractEventType_Allegro(t *testing.T) {
	body := []byte(`{"type":"ORDER_CREATED","data":{}}`)
	assert.Equal(t, "ORDER_CREATED", extractEventType("allegro", body))
}

func TestExtractEventType_InPost(t *testing.T) {
	body := []byte(`{"event":"parcel.delivered","data":{}}`)
	assert.Equal(t, "parcel.delivered", extractEventType("inpost", body))
}

func TestExtractEventType_Unknown_Provider(t *testing.T) {
	body := []byte(`{"event":"test"}`)
	assert.Equal(t, "unknown", extractEventType("other", body))
}

func TestExtractEventType_Invalid_JSON(t *testing.T) {
	body := []byte(`not json`)
	assert.Equal(t, "unknown", extractEventType("allegro", body))
}

func TestExtractEventType_Missing_Type_Field(t *testing.T) {
	body := []byte(`{"data":{}}`)
	assert.Equal(t, "unknown", extractEventType("allegro", body))
}

func TestSecretForProvider_Allegro(t *testing.T) {
	svc := NewWebhookService(nil, nil, "allegro-secret", "inpost-secret")
	secret, err := svc.secretForProvider("allegro")
	assert.NoError(t, err)
	assert.Equal(t, "allegro-secret", secret)
}

func TestSecretForProvider_InPost(t *testing.T) {
	svc := NewWebhookService(nil, nil, "allegro-secret", "inpost-secret")
	secret, err := svc.secretForProvider("inpost")
	assert.NoError(t, err)
	assert.Equal(t, "inpost-secret", secret)
}

func TestSecretForProvider_Unknown(t *testing.T) {
	svc := NewWebhookService(nil, nil, "a", "b")
	_, err := svc.secretForProvider("unknown")
	assert.ErrorIs(t, err, ErrUnknownProvider)
}
