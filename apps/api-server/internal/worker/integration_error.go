package worker

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	olxsdk "github.com/openoms-org/openoms/packages/olx-go-sdk"
)

const olxReauthRequiredMessage = "OLX authorization expired or was revoked. Reconnect OLX to resume synchronization."

func isTerminalOAuthCredentialError(provider string, err error) bool {
	if err == nil {
		return false
	}
	switch strings.ToLower(provider) {
	case "olx":
		return errors.Is(err, olxsdk.ErrInvalidGrant)
	default:
		return false
	}
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
