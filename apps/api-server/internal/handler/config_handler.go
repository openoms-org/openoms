package handler

import "net/http"

// ConfigHandler serves public (unauthenticated) configuration.
type ConfigHandler struct {
	registrationMode string
}

func NewConfigHandler(registrationMode string) *ConfigHandler {
	return &ConfigHandler{registrationMode: registrationMode}
}

// PublicConfig returns non-sensitive configuration for the frontend.
// GET /v1/config/public — no auth required.
func (h *ConfigHandler) PublicConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"registration_mode": h.registrationMode,
	})
}
