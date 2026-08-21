package allegro

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshedCredentialJSON_ReplacesAccessAndRefreshTokens(t *testing.T) {
	existing, err := json.Marshal(Credentials{
		ClientID:     "cid",
		ClientSecret: "csecret",
		AccessToken:  "old-at",
		RefreshToken: "old-rt",
		TokenExpiry:  "2026-08-20T10:00:00Z",
		Sandbox:      true,
	})
	require.NoError(t, err)

	exp := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	gotJSON, err := RefreshedCredentialJSON(existing, "new-at", "new-rt", exp)
	require.NoError(t, err)

	var got Credentials
	require.NoError(t, json.Unmarshal(gotJSON, &got))
	assert.Equal(t, "cid", got.ClientID)
	assert.Equal(t, "csecret", got.ClientSecret)
	assert.Equal(t, "new-at", got.AccessToken)
	assert.Equal(t, "new-rt", got.RefreshToken)
	assert.NotEqual(t, "old-rt", got.RefreshToken, "must not keep the stale refresh token")
	assert.Equal(t, exp.Format(time.RFC3339), got.TokenExpiry)
	assert.True(t, got.Sandbox)
}

func TestRefreshedCredentialJSON_RejectsEmptyRefreshToken(t *testing.T) {
	existing, err := json.Marshal(Credentials{
		ClientID:     "cid",
		ClientSecret: "csecret",
		AccessToken:  "old-at",
		RefreshToken: "old-rt",
	})
	require.NoError(t, err)

	_, err = RefreshedCredentialJSON(existing, "new-at", "", time.Now())
	require.Error(t, err)
}

func TestAllegroReconnectCredentialsJSON_WritesBothTokens(t *testing.T) {
	exp := time.Date(2026, 8, 21, 15, 0, 0, 0, time.UTC)
	gotJSON, err := ReconnectCredentialJSON("cid", "csecret", "oauth-at", "oauth-rt", exp, true)
	require.NoError(t, err)

	var got Credentials
	require.NoError(t, json.Unmarshal(gotJSON, &got))
	assert.Equal(t, "oauth-at", got.AccessToken)
	assert.Equal(t, "oauth-rt", got.RefreshToken)
	assert.Equal(t, exp.Format(time.RFC3339), got.TokenExpiry)
	assert.True(t, got.Sandbox)
}
