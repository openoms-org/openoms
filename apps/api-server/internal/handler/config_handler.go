package handler

import "net/http"

// ConfigHandler serves public (unauthenticated) configuration.
type ConfigHandler struct {
	registrationMode string
	licenseEnabled   bool
}

func NewConfigHandler(registrationMode string, licenseEnabled bool) *ConfigHandler {
	return &ConfigHandler{registrationMode: registrationMode, licenseEnabled: licenseEnabled}
}

// PublicConfig returns non-sensitive configuration for the frontend.
// GET /v1/config/public — no auth required.
func (h *ConfigHandler) PublicConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"registration_mode": h.registrationMode,
		"license_enabled":   h.licenseEnabled,
	})
}
