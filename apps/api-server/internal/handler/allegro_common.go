package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"

	allegroint "github.com/openoms-org/openoms/apps/api-server/internal/integration/allegro"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
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

const allegroTokenRefreshPersistTimeout = 10 * time.Second

// allegroClientBase carries the shared integration dependencies used by the
// Allegro sub-handlers. Embedding it provides a common newAllegroClient method
// and removes the duplicated integrationService/encryptionKey field pairs.
type allegroClientBase struct {
	integrationService *service.IntegrationService
	encryptionKey      []byte
}

// newAllegroClient creates an authenticated Allegro SDK client from the
// integration credentials in the request context.
func (b allegroClientBase) newAllegroClient(r *http.Request) (*allegrosdk.Client, error) {
	return buildAllegroClient(r, b.integrationService, b.encryptionKey)
}

// buildAllegroClient creates an authenticated Allegro SDK client from the request context.
// It includes a token refresh callback that persists refreshed tokens to the database.
func buildAllegroClient(r *http.Request, integrationService *service.IntegrationService, _ []byte) (*allegrosdk.Client, error) {
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
	tokenRefreshBaseCtx := context.WithoutCancel(r.Context())

	opts := []allegrosdk.Option{
		allegrosdk.WithTokens(creds.AccessToken, creds.RefreshToken, expiry),
		allegrosdk.WithOnTokenRefresh(func(accessToken, refreshToken string, exp time.Time) error {
			// Persist refreshed tokens even if the request context was canceled.
			persistCtx, cancel := allegroTokenRefreshContext(tokenRefreshBaseCtx)
			defer cancel()

			newJSON, err := allegroint.RefreshedCredentialJSON(credJSON, accessToken, refreshToken, exp)
			if err != nil {
				return err
			}
			if err := integrationService.UpdateCredentialsByProvider(persistCtx, tenantID, "allegro", newJSON); err != nil {
				slog.Error("allegro: failed to persist refreshed tokens", "error", err, "tenant_id", tenantID)
				return err
			}
			slog.Info("allegro: refreshed tokens persisted", "tenant_id", tenantID)
			return nil
		}),
	}
	if creds.Sandbox {
		opts = append(opts, allegrosdk.WithSandbox())
	}

	client := allegrosdk.NewClient(creds.ClientID, creds.ClientSecret, opts...)
	return client, nil
}

func allegroTokenRefreshContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), allegroTokenRefreshPersistTimeout)
}

// isAllegroSandbox checks if the Allegro integration uses the sandbox environment.
func isAllegroSandbox(r *http.Request, integrationService *service.IntegrationService, _ []byte) bool {
	tenantID := middleware.TenantIDFromContext(r.Context())
	credJSON, _, err := integrationService.GetDecryptedCredentialsByProvider(r.Context(), tenantID, "allegro")
	if err != nil {
		return false
	}
	var creds allegroCredentials
	if err := json.Unmarshal(credJSON, &creds); err != nil {
		return false
	}
	return creds.Sandbox
}

// allegroOfferURL returns the public offer URL. The seller-panel path
// /moje-allegro/sprzedaz/oferty/{id} 404s; Allegro serves public pages at /oferta/{id}.
func allegroOfferURL(offerID string, sandbox bool) string {
	if sandbox {
		return "https://allegro.pl.allegrosandbox.pl/oferta/" + offerID
	}
	return "https://allegro.pl/oferta/" + offerID
}

// listingStatusFromAllegroPublication maps Allegro publication.status onto
// product_listings.status. Only a live ACTIVE offer is stored as active.
// DRAFT / INACTIVE / IN_PROGRESS / szkic / checking and any other non-live
// state reuse inactive (Nieaktywna). ENDED is stored as ended.
func listingStatusFromAllegroPublication(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "ACTIVE":
		return "active"
	case "ENDED":
		return "ended"
	default:
		return "inactive"
	}
}

// listingStatusFromAllegroOffer persists publication reality after create.
// A missing publication on HTTP 201 is inactive, not active.
func listingStatusFromAllegroOffer(offer *allegrosdk.Offer) string {
	if offer == nil || offer.Publication == nil {
		return "inactive"
	}
	return listingStatusFromAllegroPublication(offer.Publication.Status)
}

// allegroListingNeedsPublicationHeal is true for leftover rows from the
// create-writes-active bug: stored as active with the seller-panel URL that 404s.
// After a heal rewrites the URL to /oferta/{id}, this returns false.
func allegroListingNeedsPublicationHeal(listing *model.ProductListing) bool {
	if listing == nil || listing.ExternalID == nil || *listing.ExternalID == "" {
		return false
	}
	if listing.Status != "active" || listing.URL == nil {
		return false
	}
	return strings.Contains(*listing.URL, "/moje-allegro/")
}
