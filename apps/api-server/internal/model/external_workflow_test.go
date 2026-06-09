package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildRedactedPayload_AllowlistOnly(t *testing.T) {
	order := map[string]any{
		"id":             "o1",
		"status":         "new",
		"customer_email": "secret@example.com",
		"total_amount":   123.45,
	}
	out := BuildRedactedPayload(order, []string{"id", "status", "total_amount"})
	assert.Equal(t, "o1", out["id"])
	assert.Equal(t, "new", out["status"])
	assert.Equal(t, 123.45, out["total_amount"])
	_, leaked := out["customer_email"]
	assert.False(t, leaked, "unlisted PII must never be serialized")
	assert.Len(t, out, 3)
}

func TestExternalWorkflowToken_ScopeAllows(t *testing.T) {
	tok := ExternalWorkflowToken{Scopes: []string{"set_status", "add_tag"}}
	assert.True(t, tok.ScopeAllows("set_status"))
	assert.False(t, tok.ScopeAllows("add_note"))
	assert.False(t, ExternalWorkflowToken{}.ScopeAllows("set_status")) // empty = resolve-only
}

func TestCallbackCommand_IsWhitelisted(t *testing.T) {
	assert.True(t, IsWhitelistedCallbackCommand("set_status"))
	assert.True(t, IsWhitelistedCallbackCommand("add_tag"))
	assert.True(t, IsWhitelistedCallbackCommand("add_note"))
	assert.False(t, IsWhitelistedCallbackCommand("create_invoice")) // not callback-permitted
	assert.False(t, IsWhitelistedCallbackCommand("delete_everything"))
}

func TestHashToken_StableAndHex(t *testing.T) {
	h1 := HashExternalWorkflowToken("abc")
	h2 := HashExternalWorkflowToken("abc")
	assert.Equal(t, h1, h2)
	assert.Len(t, h1, 64) // sha256 hex
	assert.NotEqual(t, h1, HashExternalWorkflowToken("abd"))
}

func TestSignAndVerifyHMAC(t *testing.T) {
	body := []byte(`{"correlation_nonce":"n1","status":"succeeded"}`)
	sig := SignExternalWorkflowBody(body, "secret")
	assert.True(t, VerifyExternalWorkflowSignature(body, sig, "secret"))
	assert.False(t, VerifyExternalWorkflowSignature(body, sig, "wrong"))
	assert.False(t, VerifyExternalWorkflowSignature(body, "sha256=deadbeef", "secret"))
}
