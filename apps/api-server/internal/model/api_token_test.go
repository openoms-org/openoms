package model

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashAPIToken_StableHexAndNotPlaintext(t *testing.T) {
	raw := "oms_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	h1 := HashAPIToken(raw)
	h2 := HashAPIToken(raw)
	assert.Equal(t, h1, h2)
	assert.Len(t, h1, 64)
	assert.NotEqual(t, raw, h1)
	assert.NotEqual(t, h1, HashAPIToken(raw+"x"))
}

func TestCreateAPITokenRequest_Validate(t *testing.T) {
	assert.EqualError(t, (&CreateAPITokenRequest{}).Validate(), "name is required")
	assert.EqualError(t, (&CreateAPITokenRequest{Name: "   "}).Validate(), "name is required")
	assert.NoError(t, (&CreateAPITokenRequest{Name: "allegro-sync"}).Validate())
}

func TestAPIToken_ListJSONOmitsSecretAndHash(t *testing.T) {
	tok := APIToken{
		Name:      "allegro-sync",
		TokenHash: "should-never-appear",
	}
	body, err := json.Marshal(tok)
	require.NoError(t, err)
	assert.NotContains(t, string(body), "should-never-appear")
	assert.NotContains(t, strings.ToLower(string(body)), "token_hash")
	assert.NotContains(t, string(body), `"token"`)
}

func TestCreatedAPIToken_JSONIncludesTokenOnce(t *testing.T) {
	created := CreatedAPIToken{
		APIToken: APIToken{Name: "allegro-sync", TokenHash: "stored-hash"},
		Token:    "oms_raw-secret",
	}
	body, err := json.Marshal(created)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"token":"oms_raw-secret"`)
	assert.NotContains(t, string(body), "stored-hash")
}
