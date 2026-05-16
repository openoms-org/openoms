package worker

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"
	amazonsdk "github.com/openoms-org/openoms/packages/amazon-sp-sdk"
	ebaysdk "github.com/openoms-org/openoms/packages/ebay-go-sdk"
	olxsdk "github.com/openoms-org/openoms/packages/olx-go-sdk"
)

const olxReauthRequiredMessage = "OLX authorization expired or was revoked. Reconnect OLX to resume synchronization."

func isTerminalOAuthCredentialError(provider string, err error) bool {
	if err == nil {
		return false
	}
	switch strings.ToLower(provider) {
	case "allegro":
		return errors.Is(err, allegrosdk.ErrUnauthorized) ||
			errors.Is(err, allegrosdk.ErrForbidden) ||
			hasTerminalOAuthErrorText(err)
	case "amazon":
		var apiErr *amazonsdk.APIError
		if errors.As(err, &apiErr) && (apiErr.StatusCode == 401 || apiErr.StatusCode == 403) {
			return true
		}
		return hasTerminalOAuthErrorText(err)
	case "ebay":
		return errors.Is(err, ebaysdk.ErrUnauthorized) ||
			errors.Is(err, ebaysdk.ErrForbidden) ||
			hasTerminalOAuthErrorText(err)
	case "olx":
		return errors.Is(err, olxsdk.ErrInvalidGrant)
	default:
		return false
	}
}

func hasTerminalOAuthErrorText(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "invalid_grant") ||
		strings.Contains(message, "http 401") ||
		strings.Contains(message, "http 403")
}

func terminalOAuthCredentialMessage(provider string) string {
	switch strings.ToLower(provider) {
	case "olx":
		return olxReauthRequiredMessage
	default:
		return "Marketplace authorization expired or was revoked. Reconnect the integration to resume synchronization."
	}
}

func markIntegrationRequiresReauth(ctx context.Context, pool *pgxpool.Pool, provider, tenantID, integrationID string, logger *slog.Logger) {
	if pool == nil {
		return
	}
	if logger == nil {
		logger = slog.Default()
	}
	message := terminalOAuthCredentialMessage(provider)
	if _, err := pool.Exec(ctx,
		`UPDATE integrations
		    SET status = 'error', error_message = $1, updated_at = NOW()
		  WHERE id = $2 AND status = 'active'`,
		message, integrationID,
	); err != nil {
		logger.Error("worker: failed to mark integration as requiring reauthorization",
			"operation", "integration.reauth_required",
			"tenant_id", tenantID,
			"entity_id", integrationID,
			"provider", provider,
			"error", err,
		)
		return
	}
	logger.Warn("worker: integration requires reauthorization",
		"operation", "integration.reauth_required",
		"tenant_id", tenantID,
		"entity_id", integrationID,
		"provider", provider,
	)
}
