package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// AllegroPromotionsHandler handles Allegro promotion/campaign management endpoints.
type AllegroPromotionsHandler struct {
	integrationService *service.IntegrationService
	encryptionKey      []byte
}

// NewAllegroPromotionsHandler creates a new AllegroPromotionsHandler.
func NewAllegroPromotionsHandler(integrationService *service.IntegrationService, encryptionKey []byte) *AllegroPromotionsHandler {
	return &AllegroPromotionsHandler{
		integrationService: integrationService,
		encryptionKey:      encryptionKey,
	}
}

// newAllegroClient creates an authenticated Allegro SDK client from the integration credentials.
func (h *AllegroPromotionsHandler) newAllegroClient(r *http.Request) (*allegrosdk.Client, error) {
	return buildAllegroClient(r, h.integrationService, h.encryptionKey)
}

// ListPromotions retrieves the seller's promotions.
// GET /v1/integrations/allegro/promotions
func (h *AllegroPromotionsHandler) ListPromotions(w http.ResponseWriter, r *http.Request) {
	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro promotions: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	params := &allegrosdk.ListPromotionsParams{}
	if v := r.URL.Query().Get("limit"); v != "" {
		params.Limit, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		params.Offset, _ = strconv.Atoi(v)
	}

	promotions, err := client.Promotions.List(r.Context(), params)
	if err != nil {
		slog.Error("allegro promotions: failed to list promotions", "error", err)
		writeError(w, http.StatusBadGateway, "Nie udalo sie pobrac promocji z Allegro")
		return
	}

	writeJSON(w, http.StatusOK, promotions)
}

// GetPromotion retrieves a single promotion by ID.
// GET /v1/integrations/allegro/promotions/{promotionId}
func (h *AllegroPromotionsHandler) GetPromotion(w http.ResponseWriter, r *http.Request) {
	promotionID := chi.URLParam(r, "promotionId")
	if promotionID == "" {
		writeError(w, http.StatusBadRequest, "promotionId jest wymagane")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro promotions: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	promotion, err := client.Promotions.Get(r.Context(), promotionID)
	if err != nil {
		slog.Error("allegro promotions: failed to get promotion", "error", err, "promotion_id", promotionID)
		writeError(w, http.StatusBadGateway, "Nie udalo sie pobrac promocji")
		return
	}

	writeJSON(w, http.StatusOK, promotion)
}

// CreatePromotion creates a new promotion campaign.
// POST /v1/integrations/allegro/promotions
func (h *AllegroPromotionsHandler) CreatePromotion(w http.ResponseWriter, r *http.Request) {
	var body allegrosdk.CreatePromotionRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidlowe dane wejsciowe")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro promotions: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	promotion, err := client.Promotions.Create(r.Context(), body)
	if err != nil {
		slog.Error("allegro promotions: failed to create promotion", "error", err)
		writeError(w, http.StatusBadGateway, "Nie udalo sie utworzyc promocji")
		return
	}

	writeJSON(w, http.StatusCreated, promotion)
}

// UpdatePromotion updates an existing promotion.
// PUT /v1/integrations/allegro/promotions/{promotionId}
func (h *AllegroPromotionsHandler) UpdatePromotion(w http.ResponseWriter, r *http.Request) {
	promotionID := chi.URLParam(r, "promotionId")
	if promotionID == "" {
		writeError(w, http.StatusBadRequest, "promotionId jest wymagane")
		return
	}

	var body allegrosdk.CreatePromotionRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidlowe dane wejsciowe")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro promotions: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	promotion, err := client.Promotions.Update(r.Context(), promotionID, body)
	if err != nil {
		slog.Error("allegro promotions: failed to update promotion", "error", err, "promotion_id", promotionID)
		writeError(w, http.StatusBadGateway, "Nie udalo sie zaktualizowac promocji")
		return
	}

	writeJSON(w, http.StatusOK, promotion)
}

// DeletePromotion deletes a promotion.
// DELETE /v1/integrations/allegro/promotions/{promotionId}
func (h *AllegroPromotionsHandler) DeletePromotion(w http.ResponseWriter, r *http.Request) {
	promotionID := chi.URLParam(r, "promotionId")
	if promotionID == "" {
		writeError(w, http.StatusBadRequest, "promotionId jest wymagane")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro promotions: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	if err := client.Promotions.Delete(r.Context(), promotionID); err != nil {
		slog.Error("allegro promotions: failed to delete promotion", "error", err, "promotion_id", promotionID)
		writeError(w, http.StatusBadGateway, "Nie udalo sie usunac promocji")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ListBadges lists available promotion badge packages.
// GET /v1/integrations/allegro/promotion-badges
func (h *AllegroPromotionsHandler) ListBadges(w http.ResponseWriter, r *http.Request) {
	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro promotions: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	badges, err := client.Promotions.ListBadges(r.Context())
	if err != nil {
		slog.Error("allegro promotions: failed to list badges", "error", err)
		writeError(w, http.StatusBadGateway, "Nie udalo sie pobrac pakietow promocyjnych")
		return
	}

	writeJSON(w, http.StatusOK, badges)
}
