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

// AllegroAccountHandler handles Allegro account, billing, and offer management endpoints.
type AllegroAccountHandler struct {
	integrationService *service.IntegrationService
	encryptionKey      []byte
}

// NewAllegroAccountHandler creates a new AllegroAccountHandler.
func NewAllegroAccountHandler(integrationService *service.IntegrationService, encryptionKey []byte) *AllegroAccountHandler {
	return &AllegroAccountHandler{
		integrationService: integrationService,
		encryptionKey:      encryptionKey,
	}
}

// newAllegroClient creates an authenticated Allegro SDK client from the integration credentials.
func (h *AllegroAccountHandler) newAllegroClient(r *http.Request) (*allegrosdk.Client, error) {
	return buildAllegroClient(r, h.integrationService, h.encryptionKey)
}

// GetAccount retrieves the seller's account info and quality metrics.
// GET /v1/integrations/allegro/account
func (h *AllegroAccountHandler) GetAccount(w http.ResponseWriter, r *http.Request) {
	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro account: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	user, err := client.Account.GetMe(r.Context())
	if err != nil {
		slog.Error("allegro account: failed to get user", "error", err)
		writeError(w, http.StatusBadGateway, "Nie udało się pobrać danych konta Allegro")
		return
	}

	quality, err := client.Account.GetQuality(r.Context())
	if err != nil {
		slog.Warn("allegro account: failed to get quality", "error", err)
		// Quality is optional — return partial result
		quality = &allegrosdk.SellerQuality{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user":    user,
		"quality": quality,
	})
}

// GetBilling retrieves the seller's billing entries.
// GET /v1/integrations/allegro/billing
func (h *AllegroAccountHandler) GetBilling(w http.ResponseWriter, r *http.Request) {
	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro billing: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	params := &allegrosdk.BillingParams{}
	if v := r.URL.Query().Get("limit"); v != "" {
		params.Limit, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		params.Offset, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("type_group"); v != "" {
		params.TypeGroup = v
	}

	billing, err := client.Account.ListBilling(r.Context(), params)
	if err != nil {
		slog.Error("allegro billing: failed to list billing", "error", err)
		writeError(w, http.StatusBadGateway, "Nie udało się pobrać danych rozliczeniowych")
		return
	}

	writeJSON(w, http.StatusOK, billing)
}

// ListOffers retrieves the seller's offers from Allegro.
// GET /v1/integrations/allegro/offers
func (h *AllegroAccountHandler) ListOffers(w http.ResponseWriter, r *http.Request) {
	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro offers: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	params := &allegrosdk.ListOffersParams{}
	if v := r.URL.Query().Get("limit"); v != "" {
		params.Limit, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		params.Offset, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("name"); v != "" {
		params.Name = v
	}
	if v := r.URL.Query().Get("publication_status"); v != "" {
		params.PublicationStatus = v
	}

	offers, err := client.Offers.List(r.Context(), params)
	if err != nil {
		slog.Error("allegro offers: failed to list offers", "error", err)
		writeError(w, http.StatusBadGateway, "Nie udało się pobrać ofert z Allegro")
		return
	}

	writeJSON(w, http.StatusOK, offers)
}

// DeactivateOffer deactivates (ends) a single Allegro offer.
// POST /v1/integrations/allegro/offers/{offerId}/deactivate
func (h *AllegroAccountHandler) DeactivateOffer(w http.ResponseWriter, r *http.Request) {
	offerID := chi.URLParam(r, "offerId")
	if offerID == "" {
		writeError(w, http.StatusBadRequest, "offerId is required")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro offers: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	if err := client.Offers.Deactivate(r.Context(), offerID); err != nil {
		slog.Error("allegro offers: failed to deactivate", "error", err, "offer_id", offerID)
		writeError(w, http.StatusBadGateway, "Nie udało się dezaktywować oferty")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deactivated"})
}

// ActivateOffer activates a single Allegro offer.
// POST /v1/integrations/allegro/offers/{offerId}/activate
func (h *AllegroAccountHandler) ActivateOffer(w http.ResponseWriter, r *http.Request) {
	offerID := chi.URLParam(r, "offerId")
	if offerID == "" {
		writeError(w, http.StatusBadRequest, "offerId is required")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro offers: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	if err := client.Offers.Activate(r.Context(), offerID); err != nil {
		slog.Error("allegro offers: failed to activate", "error", err, "offer_id", offerID)
		writeError(w, http.StatusBadGateway, "Nie udało się aktywować oferty")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "activated"})
}

// UpdateOfferStock updates the stock quantity for a single Allegro offer.
// PATCH /v1/integrations/allegro/offers/{offerId}/stock
func (h *AllegroAccountHandler) UpdateOfferStock(w http.ResponseWriter, r *http.Request) {
	offerID := chi.URLParam(r, "offerId")
	if offerID == "" {
		writeError(w, http.StatusBadRequest, "offerId is required")
		return
	}

	var body struct {
		Quantity int `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro offers: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	if err := client.Offers.UpdateStock(r.Context(), offerID, body.Quantity); err != nil {
		slog.Error("allegro offers: failed to update stock", "error", err, "offer_id", offerID)
		writeError(w, http.StatusBadGateway, "Nie udało się zaktualizować stanu magazynowego")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// UpdateOfferPrice updates the price for a single Allegro offer.
// PATCH /v1/integrations/allegro/offers/{offerId}/price
func (h *AllegroAccountHandler) UpdateOfferPrice(w http.ResponseWriter, r *http.Request) {
	offerID := chi.URLParam(r, "offerId")
	if offerID == "" {
		writeError(w, http.StatusBadRequest, "offerId is required")
		return
	}

	var body struct {
		Amount   float64 `json:"amount"`
		Currency string  `json:"currency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Currency == "" {
		body.Currency = "PLN"
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro offers: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	if err := client.Offers.UpdatePrice(r.Context(), offerID, body.Amount, body.Currency); err != nil {
		slog.Error("allegro offers: failed to update price", "error", err, "offer_id", offerID)
		writeError(w, http.StatusBadGateway, "Nie udało się zaktualizować ceny")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}
