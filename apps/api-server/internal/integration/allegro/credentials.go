package allegro

import (
	"encoding/json"
	"fmt"
	"time"
)

// RefreshedCredentialJSON replaces both access and refresh tokens on a successful
// Allegro refresh. The previous refresh token is not kept.
func RefreshedCredentialJSON(existing []byte, accessToken, refreshToken string, expiry time.Time) ([]byte, error) {
	if accessToken == "" || refreshToken == "" {
		return nil, fmt.Errorf("allegro: refresh response missing access_token or refresh_token")
	}
	var creds Credentials
	if err := json.Unmarshal(existing, &creds); err != nil {
		return nil, fmt.Errorf("allegro: parse credentials: %w", err)
	}
	creds.AccessToken = accessToken
	creds.RefreshToken = refreshToken
	creds.TokenExpiry = expiry.Format(time.RFC3339)
	return json.Marshal(creds)
}

// ReconnectCredentialJSON is the credential blob written after OAuth reconnect.
func ReconnectCredentialJSON(clientID, clientSecret, accessToken, refreshToken string, expiry time.Time, sandbox bool) ([]byte, error) {
	if accessToken == "" || refreshToken == "" {
		return nil, fmt.Errorf("allegro: reconnect response missing access_token or refresh_token")
	}
	return json.Marshal(Credentials{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenExpiry:  expiry.Format(time.RFC3339),
		Sandbox:      sandbox,
	})
}

// TokenRefreshPersist writes rotated Allegro tokens. A non-nil error fails the refresh.
type TokenRefreshPersist func(accessToken, refreshToken string, expiry time.Time) error

// PersistFn builds a persist callback from existing credential JSON and a writer.
func PersistFn(existing []byte, write func([]byte) error) TokenRefreshPersist {
	return func(accessToken, refreshToken string, expiry time.Time) error {
		newJSON, err := RefreshedCredentialJSON(existing, accessToken, refreshToken, expiry)
		if err != nil {
			return err
		}
		return write(newJSON)
	}
}
