// Package handler implements HTTP request handlers for the API server.
package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// AccountingHandler handles accounting provider configuration endpoints.
type AccountingHandler struct {
	tenantRepo repository.TenantRepo
	auditRepo  repository.AuditRepo
	pool       *pgxpool.Pool
}

// NewAccountingHandler creates a new AccountingHandler.
func NewAccountingHandler(tenantRepo repository.TenantRepo, auditRepo repository.AuditRepo, pool *pgxpool.Pool) *AccountingHandler {
	return &AccountingHandler{
		tenantRepo: tenantRepo,
		auditRepo:  auditRepo,
		pool:       pool,
	}
}

// AccountingConfig is the response for GET /v1/settings/accounting.
type AccountingConfig struct {
	Provider    string            `json:"provider"`
	Credentials map[string]string `json:"credentials"`
	Connected   bool              `json:"connected"`
}

// GetAccountingSettings returns the current accounting provider configuration.
func (h *AccountingHandler) GetAccountingSettings(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var invoicingCfg map[string]any
	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		return h.getSettingsSection(r.Context(), tx, tenantID, "invoicing", &invoicingCfg)
	})
	if err != nil {
		writeServerError(w, "failed to load accounting settings", err)
		return
	}

	config := AccountingConfig{
		Provider:    "",
		Credentials: map[string]string{},
		Connected:   false,
	}

	if invoicingCfg != nil {
		if p, ok := invoicingCfg["provider"].(string); ok {
			config.Provider = p
		}

		if creds, ok := invoicingCfg["credentials"].(map[string]any); ok {
			maskedCreds := make(map[string]string)
			for key, val := range creds {
				if s, ok := val.(string); ok && s != "" {
					maskedCreds[key] = "\u2022\u2022\u2022\u2022\u2022\u2022"
				} else {
					maskedCreds[key] = ""
				}
			}
			config.Credentials = maskedCreds
		}

		if config.Provider != "" {
			config.Connected = isKnownProvider(config.Provider)
		}
	}

	writeJSON(w, http.StatusOK, config)
}

// UpdateAccountingSettings updates the accounting provider configuration.
func (h *AccountingHandler) UpdateAccountingSettings(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req struct {
		Provider    string            `json:"provider"`
		Credentials map[string]string `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate provider name
	validProviders := map[string]bool{
		"":            true,
		"fakturownia": true,
		"wfirma":      true,
		"infakt":      true,
	}
	if !validProviders[req.Provider] {
		writeError(w, http.StatusBadRequest, "unknown provider: "+req.Provider)
		return
	}

	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		// Load existing invoicing settings
		var existingCfg map[string]any
		if err := h.getSettingsSection(r.Context(), tx, tenantID, "invoicing", &existingCfg); err != nil {
			return err
		}
		if existingCfg == nil {
			existingCfg = make(map[string]any)
		}

		// Handle masked credentials — preserve existing values
		if req.Credentials != nil {
			var oldCreds map[string]any
			if c, ok := existingCfg["credentials"].(map[string]any); ok {
				oldCreds = c
			}
			if oldCreds == nil {
				oldCreds = make(map[string]any)
			}

			for key, val := range req.Credentials {
				if val == "\u2022\u2022\u2022\u2022\u2022\u2022" {
					if oldVal, ok := oldCreds[key].(string); ok {
						req.Credentials[key] = oldVal
					}
				}
			}
		}

		// Update only provider and credentials; preserve other invoicing settings
		existingCfg["provider"] = req.Provider
		existingCfg["credentials"] = req.Credentials

		if err := h.updateSettingsSection(r.Context(), tx, tenantID, "invoicing", existingCfg); err != nil {
			return err
		}

		return h.auditRepo.Log(r.Context(), tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "settings.accounting_updated",
			EntityType: "settings",
			EntityID:   tenantID,
			Changes:    map[string]string{"provider": req.Provider},
			IPAddress:  clientIP(r),
		})
	})

	if err != nil {
		writeServerError(w, "failed to save accounting settings", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message":  "Accounting settings saved",
		"provider": req.Provider,
	})
}

// TestConnection tests the connection to the configured accounting provider.
func (h *AccountingHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var invoicingRaw json.RawMessage
	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		settings, err := h.tenantRepo.GetSettings(r.Context(), tx, tenantID)
		if err != nil {
			return err
		}
		if settings == nil {
			return nil
		}

		var allSettings map[string]json.RawMessage
		if err := json.Unmarshal(settings, &allSettings); err != nil {
			return err
		}

		if raw, ok := allSettings["invoicing"]; ok {
			invoicingRaw = raw
		}
		return nil
	})
	if err != nil {
		writeServerError(w, "failed to load settings", err)
		return
	}

	if invoicingRaw == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"success": false,
			"message": "No provider configured",
		})
		return
	}

	// Parse the invoicing config
	var cfgMap map[string]json.RawMessage
	if err := json.Unmarshal(invoicingRaw, &cfgMap); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"success": false,
			"message": "Invalid configuration",
		})
		return
	}

	// Extract provider name
	var providerName string
	if raw, ok := cfgMap["provider"]; ok {
		_ = json.Unmarshal(raw, &providerName)
	}
	if providerName == "" {
		writeJSON(w, http.StatusOK, map[string]any{
			"success": false,
			"message": "No provider selected",
		})
		return
	}

	// Extract credentials
	credsRaw, ok := cfgMap["credentials"]
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"success": false,
			"message": "Missing credentials",
		})
		return
	}

	// Create the provider and make a real API call to verify credentials.
	provider, provErr := integration.NewInvoicingProvider(providerName, credsRaw, invoicingRaw)
	if provErr != nil {
		slog.Error("accounting test connection failed", "provider", providerName, "error", provErr)
		writeJSON(w, http.StatusOK, map[string]any{
			"success": false,
			"message": "Failed to connect: " + provErr.Error(),
		})
		return
	}

	// Fetch a non-existent invoice to verify API credentials work.
	// Expected: "not found" error = credentials OK; auth error = credentials wrong.
	_, testErr := provider.GetInvoice(r.Context(), "test-connection-probe")
	if testErr != nil {
		errMsg := testErr.Error()
		// Auth errors indicate wrong credentials
		if strings.Contains(errMsg, "401") || strings.Contains(errMsg, "403") ||
			strings.Contains(errMsg, "unauthorized") || strings.Contains(errMsg, "forbidden") {
			writeJSON(w, http.StatusOK, map[string]any{
				"success": false,
				"message": "Authentication failed — check your API key",
			})
			return
		}
		// "not found" or other errors mean credentials are valid
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Connection to " + providerName + " successful",
	})
}

// getSettingsSection reads a specific section from the tenant's JSON settings blob.
func (h *AccountingHandler) getSettingsSection(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, key string, dest any) error {
	settings, err := h.tenantRepo.GetSettings(ctx, tx, tenantID)
	if err != nil {
		return err
	}

	if settings == nil {
		return nil
	}

	var allSettings map[string]json.RawMessage
	if err := json.Unmarshal(settings, &allSettings); err != nil {
		return err
	}

	raw, ok := allSettings[key]
	if !ok {
		return nil
	}

	if err := json.Unmarshal(raw, dest); err != nil {
		slog.Warn("failed to unmarshal settings section", "key", key, "error", err)
	}
	return nil
}

// updateSettingsSection merges a value into the tenant's JSON settings blob under the given key.
func (h *AccountingHandler) updateSettingsSection(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, key string, value any) error {
	// FOR UPDATE prevents concurrent read-modify-write races on the settings blob.
	existing, err := h.tenantRepo.GetSettingsForUpdate(ctx, tx, tenantID)
	if err != nil {
		return err
	}

	var allSettings map[string]json.RawMessage
	if err := json.Unmarshal(existing, &allSettings); err != nil {
		allSettings = make(map[string]json.RawMessage)
	}

	valueJSON, err := json.Marshal(value)
	if err != nil {
		return err
	}
	allSettings[key] = valueJSON

	newSettings, err := json.Marshal(allSettings)
	if err != nil {
		return err
	}

	return h.tenantRepo.UpdateSettings(ctx, tx, tenantID, newSettings)
}

// isKnownProvider checks if a provider name corresponds to a registered invoicing provider.
func isKnownProvider(provider string) bool {
	switch provider {
	case "fakturownia", "wfirma", "infakt":
		return true
	default:
		return false
	}
}
