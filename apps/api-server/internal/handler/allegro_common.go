package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// allegroCredentials represents the parsed credential JSON stored in integrations.
type allegroCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenExpiry  string `json:"token_expiry"`
	Sandbox      bool   `json:"sandbox"`
}

// buildAllegroClient creates an authenticated Allegro SDK client from the request context.
// It includes a token refresh callback that persists refreshed tokens to the database.
func buildAllegroClient(r *http.Request, integrationService *service.IntegrationService, encryptionKey []byte) (*allegrosdk.Client, error) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	credJSON, _, err := integrationService.GetDecryptedCredentialsByProvider(r.Context(), tenantID, "allegro")
	if err != nil {
		return nil, fmt.Errorf("failed to get allegro credentials: %w", err)
	}

	var creds allegroCredentials
	if err := json.Unmarshal(credJSON, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	if creds.AccessToken == "" {
		return nil, fmt.Errorf("allegro integration not authorized (no access token)")
	}

	expiry, _ := time.Parse(time.RFC3339, creds.TokenExpiry)

	opts := []allegrosdk.Option{
		allegrosdk.WithTokens(creds.AccessToken, creds.RefreshToken, expiry),
		allegrosdk.WithOnTokenRefresh(func(accessToken, refreshToken string, exp time.Time) {
			// Persist refreshed tokens to the database
			newCreds := allegroCredentials{
				ClientID:     creds.ClientID,
				ClientSecret: creds.ClientSecret,
				AccessToken:  accessToken,
				RefreshToken: refreshToken,
				TokenExpiry:  exp.Format(time.RFC3339),
				Sandbox:      creds.Sandbox,
			}
			newJSON, err := json.Marshal(newCreds)
			if err != nil {
				slog.Error("allegro: failed to marshal refreshed credentials", "error", err)
				return
			}
			if err := integrationService.UpdateCredentialsByProvider(r.Context(), tenantID, "allegro", newJSON); err != nil {
				slog.Error("allegro: failed to persist refreshed tokens", "error", err, "tenant_id", tenantID)
			} else {
				slog.Info("allegro: refreshed tokens persisted", "tenant_id", tenantID)
			}
		}),
	}
	if creds.Sandbox {
		opts = append(opts, allegrosdk.WithSandbox())
	}

	client := allegrosdk.NewClient(creds.ClientID, creds.ClientSecret, opts...)
	return client, nil
}
