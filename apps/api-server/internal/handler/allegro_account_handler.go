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
	allegroClientBase
}

// NewAllegroAccountHandler creates a new AllegroAccountHandler.
func NewAllegroAccountHandler(integrationService *service.IntegrationService, encryptionKey []byte) *AllegroAccountHandler {
	return &AllegroAccountHandler{allegroClientBase{integrationService: integrationService, encryptionKey: encryptionKey}}
}

// GetAccount retrieves the seller's account info and quality metrics.
// GET /v1/integrations/allegro/account
func (h *AllegroAccountHandler) GetAccount(w http.ResponseWriter, r *http.Request) {
	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro account: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Allegro integration not configured")
		return
	}
	defer client.Close()

	user, err := client.Account.GetMe(r.Context())
	if err != nil {
		slog.Error("allegro account: failed to get user", "error", err)
		writeAllegroError(w, "Failed to fetch Allegro account data", err)
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
		"sandbox": isAllegroSandbox(r, h.integrationService, h.encryptionKey),
	})
}

// GetBilling retrieves the seller's billing entries.
// GET /v1/integrations/allegro/billing
func (h *AllegroAccountHandler) GetBilling(w http.ResponseWriter, r *http.Request) {
	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro billing: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Allegro integration not configured")
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
		writeAllegroError(w, "Failed to fetch billing data", err)
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
		writeError(w, http.StatusBadRequest, "Allegro integration not configured")
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
		writeAllegroError(w, "Failed to fetch offers from Allegro", err)
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
		writeError(w, http.StatusBadRequest, "Allegro integration not configured")
		return
	}
	defer client.Close()

	if err := client.Offers.Deactivate(r.Context(), offerID); err != nil {
		slog.Error("allegro offers: failed to deactivate", "error", err, "offer_id", offerID) //nolint:gosec
		writeAllegroError(w, "Failed to deactivate offer", err)
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
		writeError(w, http.StatusBadRequest, "Allegro integration not configured")
		return
	}
	defer client.Close()

	if err := client.Offers.Activate(r.Context(), offerID); err != nil {
		slog.Error("allegro offers: failed to activate", "error", err, "offer_id", offerID) //nolint:gosec
		writeAllegroError(w, "Failed to activate offer", err)
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
		writeError(w, http.StatusBadRequest, "Allegro integration not configured")
		return
	}
	defer client.Close()

	if err := client.Offers.UpdateStock(r.Context(), offerID, body.Quantity); err != nil {
		slog.Error("allegro offers: failed to update stock", "error", err, "offer_id", offerID) //nolint:gosec
		writeAllegroError(w, "Failed to update stock", err)
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
		Price    *struct {
			Amount   any    `json:"amount"` // can be string or number
			Currency string `json:"currency"`
		} `json:"price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	// Support nested {"price":{"amount":"269.50","currency":"PLN"}} format (Allegro style)
	if body.Price != nil {
		switch v := body.Price.Amount.(type) {
		case float64:
			body.Amount = v
		case string:
			body.Amount, _ = strconv.ParseFloat(v, 64)
		}
		if body.Price.Currency != "" {
			body.Currency = body.Price.Currency
		}
	}
	if body.Currency == "" {
		body.Currency = "PLN"
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro offers: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Allegro integration not configured")
		return
	}
	defer client.Close()

	if err := client.Offers.UpdatePrice(r.Context(), offerID, body.Amount, body.Currency); err != nil {
		slog.Error("allegro offers: failed to update price", "error", err, "offer_id", offerID) //nolint:gosec
		writeAllegroError(w, "Failed to update price", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}
