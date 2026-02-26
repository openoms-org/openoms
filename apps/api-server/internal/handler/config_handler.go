package handler

import "net/http"

// ConfigHandler serves public (unauthenticated) configuration.
type ConfigHandler struct {
	registrationMode string
	licenseEnabled   bool
	billingEnabled   bool
	stripePublicKey  string
}

// NewConfigHandler creates a handler that serves public configuration.
func NewConfigHandler(registrationMode string, licenseEnabled, billingEnabled bool, stripePublicKey string) *ConfigHandler {
	return &ConfigHandler{
		registrationMode: registrationMode,
		licenseEnabled:   licenseEnabled,
		billingEnabled:   billingEnabled,
		stripePublicKey:  stripePublicKey,
	}
}

// PublicConfig returns non-sensitive configuration for the frontend.
// GET /v1/config/public — no auth required.
func (h *ConfigHandler) PublicConfig(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{
		"registration_mode": h.registrationMode,
		"license_enabled":   h.licenseEnabled,
		"billing_enabled":   h.billingEnabled,
	}
	if h.billingEnabled && h.stripePublicKey != "" {
		resp["stripe_public_key"] = h.stripePublicKey
	}
	writeJSON(w, http.StatusOK, resp)
}
