package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"
	amazonsdk "github.com/openoms-org/openoms/packages/amazon-sp-sdk"
	olxsdk "github.com/openoms-org/openoms/packages/olx-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/crypto"
	allegroIntegration "github.com/openoms-org/openoms/apps/api-server/internal/integration/allegro"
	amazonIntegration "github.com/openoms-org/openoms/apps/api-server/internal/integration/amazon"
	olxIntegration "github.com/openoms-org/openoms/apps/api-server/internal/integration/olx"
)

// OAuthRefresher periodically checks for expiring OAuth tokens and refreshes them.
type OAuthRefresher struct {
	pool          *pgxpool.Pool
	encryptionKey []byte
	logger        *slog.Logger
}

func NewOAuthRefresher(pool *pgxpool.Pool, encryptionKey []byte, logger *slog.Logger) *OAuthRefresher {
	return &OAuthRefresher{
		pool:          pool,
		encryptionKey: encryptionKey,
		logger:        logger,
	}
}

func (w *OAuthRefresher) Name() string {
	return "oauth_refresher"
}

func (w *OAuthRefresher) Interval() time.Duration {
	return 30 * time.Minute
}

// Run checks for expiring OAuth tokens and refreshes them.
func (w *OAuthRefresher) Run(ctx context.Context) error {
	w.logger.Info("checking for expiring tokens")

	rows, err := w.pool.Query(ctx,
		`SELECT id, tenant_id, credentials, provider
		 FROM integrations
		 WHERE status = 'active' AND provider IN ('allegro', 'olx', 'amazon')`,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	type integrationRow struct {
		id          string
		tenantID    string
		credentials string
		provider    string
	}

	var integrations []integrationRow
	for rows.Next() {
		var ir integrationRow
		if err := rows.Scan(&ir.id, &ir.tenantID, &ir.credentials, &ir.provider); err != nil {
			return err
		}
		integrations = append(integrations, ir)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	refreshed := 0
	for _, ir := range integrations {
		credJSON, err := crypto.Decrypt(ir.credentials, w.encryptionKey)
		if err != nil {
			w.logger.Error("oauth refresh: decrypt failed", "integration_id", ir.id, "error", err)
			continue
		}

		var tokenExpiry string
		switch ir.provider {
		case "allegro":
			var creds allegroIntegration.AllegroCredentials
			if err := json.Unmarshal(credJSON, &creds); err != nil {
				w.logger.Error("oauth refresh: parse credentials", "integration_id", ir.id, "error", err)
				continue
			}
			tokenExpiry = creds.TokenExpiry
		case "olx":
			var creds olxIntegration.OLXCredentials
			if err := json.Unmarshal(credJSON, &creds); err != nil {
				w.logger.Error("oauth refresh: parse credentials", "integration_id", ir.id, "error", err)
				continue
			}
			tokenExpiry = creds.TokenExpiry
			if creds.RefreshToken == "" {
				continue // no refresh token, skip (client_credentials flow)
			}
		case "amazon":
			var creds amazonIntegration.AmazonCredentials
			if err := json.Unmarshal(credJSON, &creds); err != nil {
				w.logger.Error("oauth refresh: parse credentials", "integration_id", ir.id, "error", err)
				continue
			}
			tokenExpiry = creds.TokenExpiry
			if creds.RefreshToken == "" {
				continue
			}
		default:
			continue
		}

		// Check if token expires within 2 hours
		expiry, err := time.Parse(time.RFC3339, tokenExpiry)
		if err != nil {
			w.logger.Error("oauth refresh: parse token expiry", "integration_id", ir.id, "error", err)
			continue
		}

		if time.Until(expiry) > 2*time.Hour {
			continue // token still valid, skip
		}

		w.logger.Info("worker: refreshing token",
			"operation", "integration.oauth_refresh",
			"tenant_id", ir.tenantID,
			"entity_id", ir.id,
			"provider", ir.provider, "expires_at", tokenExpiry)

		var newCredJSON []byte

		switch ir.provider {
		case "allegro":
			newCredJSON, err = w.refreshAllegro(ctx, credJSON, expiry)
		case "olx":
			newCredJSON, err = w.refreshOLX(ctx, credJSON, expiry)
		case "amazon":
			newCredJSON, err = w.refreshAmazon(ctx, credJSON, expiry)
		}
		if err != nil {
			w.logger.Error("oauth refresh: refresh failed", "integration_id", ir.id, "error", err)
			continue
		}

		encrypted, err := crypto.Encrypt(newCredJSON, w.encryptionKey)
		if err != nil {
			w.logger.Error("oauth refresh: encrypt credentials", "integration_id", ir.id, "error", err)
			continue
		}

		// Store encrypted creds as JSON string in JSONB column
		credsJSONB, _ := json.Marshal(encrypted)

		if _, err := w.pool.Exec(ctx,
			"UPDATE integrations SET credentials = $1::jsonb WHERE id = $2",
			credsJSONB, ir.id,
		); err != nil {
			w.logger.Error("worker: oauth credential update failed",
				"operation", "integration.oauth_refresh",
				"tenant_id", ir.tenantID,
				"entity_id", ir.id,
				"error", err)
			continue
		}

		refreshed++
		w.logger.Info("oauth: refreshed token",
			"operation", "integration.oauth_refresh",
			"provider", ir.provider,
			"tenant_id", ir.tenantID,
			"entity_id", ir.id)
	}

	w.logger.Info("oauth refresh completed", "checked", len(integrations), "refreshed", refreshed)
	return nil
}

func (w *OAuthRefresher) refreshAllegro(ctx context.Context, credJSON []byte, expiry time.Time) ([]byte, error) {
	var creds allegroIntegration.AllegroCredentials
	if err := json.Unmarshal(credJSON, &creds); err != nil {
		return nil, err
	}

	var opts []allegrosdk.Option
	opts = append(opts, allegrosdk.WithTokens(creds.AccessToken, creds.RefreshToken, expiry))
	if creds.Sandbox {
		opts = append(opts, allegrosdk.WithSandbox())
	}

	client := allegrosdk.NewClient(creds.ClientID, creds.ClientSecret, opts...)
	tok, err := client.RefreshAccessToken(ctx)
	client.Close()
	if err != nil {
		return nil, err
	}

	newExpiry := time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	newCreds := allegroIntegration.AllegroCredentials{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		TokenExpiry:  newExpiry.Format(time.RFC3339),
		Sandbox:      creds.Sandbox,
	}
	return json.Marshal(newCreds)
}

func (w *OAuthRefresher) refreshOLX(ctx context.Context, credJSON []byte, expiry time.Time) ([]byte, error) {
	var creds olxIntegration.OLXCredentials
	if err := json.Unmarshal(credJSON, &creds); err != nil {
		return nil, err
	}

	client := olxsdk.NewClient(creds.ClientID, creds.ClientSecret, "",
		olxsdk.WithTokens(creds.AccessToken, creds.RefreshToken, expiry),
	)
	tok, err := client.RefreshAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	newExpiry := time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	newCreds := olxIntegration.OLXCredentials{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		TokenExpiry:  newExpiry.Format(time.RFC3339),
	}
	return json.Marshal(newCreds)
}

func (w *OAuthRefresher) refreshAmazon(ctx context.Context, credJSON []byte, expiry time.Time) ([]byte, error) {
	var creds amazonIntegration.AmazonCredentials
	if err := json.Unmarshal(credJSON, &creds); err != nil {
		return nil, err
	}

	client := amazonsdk.NewClient(creds.ClientID, creds.ClientSecret,
		amazonsdk.WithTokens(creds.AccessToken, creds.RefreshToken, expiry),
	)
	tok, err := client.RefreshAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	newExpiry := time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	creds.AccessToken = tok.AccessToken
	if tok.RefreshToken != "" {
		creds.RefreshToken = tok.RefreshToken
	}
	creds.TokenExpiry = newExpiry.Format(time.RFC3339)
	return json.Marshal(creds)
}
