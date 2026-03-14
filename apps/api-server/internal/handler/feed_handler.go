package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// FeedHandler handles product feed endpoints (Ceneo, Google Shopping).
type FeedHandler struct {
	feedService *service.FeedService
}

// NewFeedHandler creates a new FeedHandler.
func NewFeedHandler(feedService *service.FeedService) *FeedHandler {
	return &FeedHandler{feedService: feedService}
}

// GetConfig returns the current feed configuration for the authenticated tenant.
// GET /v1/feeds/config
func (h *FeedHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	cfg, err := h.feedService.GetFeedConfig(r.Context(), tenantID)
	if err != nil {
		writeServerError(w, "failed to load feed config", err)
		return
	}

	writeJSON(w, http.StatusOK, cfg)
}

// UpdateConfig updates the feed configuration for the authenticated tenant.
// PUT /v1/feeds/config
func (h *FeedHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var cfg model.ProductFeedConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.feedService.UpdateFeedConfig(r.Context(), tenantID, &cfg); err != nil {
		writeServerError(w, "failed to save feed config", err)
		return
	}

	writeJSON(w, http.StatusOK, cfg)
}

// RegenerateToken generates a new token for the specified feed type.
// POST /v1/feeds/regenerate-token
func (h *FeedHandler) RegenerateToken(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var req struct {
		FeedType string `json:"feed_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || (req.FeedType != "ceneo" && req.FeedType != "google") {
		writeError(w, http.StatusBadRequest, "feed_type must be 'ceneo' or 'google'")
		return
	}

	cfg, err := h.feedService.RegenerateToken(r.Context(), tenantID, req.FeedType)
	if err != nil {
		writeServerError(w, "failed to regenerate token", err)
		return
	}

	writeJSON(w, http.StatusOK, cfg)
}

// ServeCeneoFeed serves the Ceneo XML feed (public, token-authenticated).
// GET /v1/feeds/ceneo/{tenant_id}/{token}
func (h *FeedHandler) ServeCeneoFeed(w http.ResponseWriter, r *http.Request) {
	tenantID, token, ok := h.parseFeedParams(w, r)
	if !ok {
		return
	}

	valid, err := h.feedService.ValidateFeedToken(r.Context(), tenantID, "ceneo", token)
	if err != nil {
		writeServerError(w, "internal error", err)
		return
	}
	if !valid {
		writeError(w, http.StatusForbidden, "invalid or disabled feed")
		return
	}

	data, err := h.feedService.GenerateCeneoFeed(r.Context(), tenantID)
	if err != nil {
		writeServerError(w, "failed to generate feed", err)
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=900") // 15 min
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data) //nolint:gosec
}

// ServeGoogleFeed serves the Google Shopping XML feed (public, token-authenticated).
// GET /v1/feeds/google/{tenant_id}/{token}
func (h *FeedHandler) ServeGoogleFeed(w http.ResponseWriter, r *http.Request) {
	tenantID, token, ok := h.parseFeedParams(w, r)
	if !ok {
		return
	}

	valid, err := h.feedService.ValidateFeedToken(r.Context(), tenantID, "google", token)
	if err != nil {
		writeServerError(w, "internal error", err)
		return
	}
	if !valid {
		writeError(w, http.StatusForbidden, "invalid or disabled feed")
		return
	}

	data, err := h.feedService.GenerateGoogleFeed(r.Context(), tenantID)
	if err != nil {
		writeServerError(w, "failed to generate feed", err)
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=900") // 15 min
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data) //nolint:gosec
}

// parseFeedParams extracts and validates tenant_id and token from URL params.
func (h *FeedHandler) parseFeedParams(w http.ResponseWriter, r *http.Request) (uuid.UUID, string, bool) {
	tenantIDStr := chi.URLParam(r, "tenant_id")
	token := chi.URLParam(r, "token")

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant_id")
		return uuid.Nil, "", false
	}

	if token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return uuid.Nil, "", false
	}

	return tenantID, token, true
}
