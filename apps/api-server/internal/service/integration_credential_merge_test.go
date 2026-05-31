package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mergeForTest(t *testing.T, existing, update string) map[string]any {
	t.Helper()
	out, err := mergeCredentialUpdate([]byte(existing), []byte(update))
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(out, &m))
	return m
}

func TestMergeCredentialUpdate_NullClearsWebhookSecret(t *testing.T) {
	m := mergeForTest(t, `{"webhook_secret":"old","api_key":"k"}`, `{"webhook_secret":null}`)

	_, has := m["webhook_secret"]
	assert.False(t, has, "explicit JSON null must clear webhook_secret")
	assert.Equal(t, "k", m["api_key"], "unrelated fields are preserved")
}

func TestMergeCredentialUpdate_EmptyWebhookSecretPreserved(t *testing.T) {
	m := mergeForTest(t, `{"webhook_secret":"old"}`, `{"webhook_secret":""}`)

	assert.Equal(t, "old", m["webhook_secret"],
		"empty (not null) means unchanged — the UI does not echo the secret, so it must not be wiped")
}

func TestMergeCredentialUpdate_NewWebhookSecretRotates(t *testing.T) {
	m := mergeForTest(t, `{"webhook_secret":"old"}`, `{"webhook_secret":"new"}`)

	assert.Equal(t, "new", m["webhook_secret"], "a non-empty value rotates the secret")
}

func TestMergeCredentialUpdate_NullClearsGenericField(t *testing.T) {
	m := mergeForTest(t, `{"api_key":"k","client_id":"c"}`, `{"api_key":null}`)

	_, has := m["api_key"]
	assert.False(t, has, "explicit null clears any credential field")
	assert.Equal(t, "c", m["client_id"])
}
