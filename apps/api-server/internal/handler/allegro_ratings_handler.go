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

// AllegroRatingsHandler handles Allegro seller rating management API routes.
type AllegroRatingsHandler struct {
	integrationService *service.IntegrationService
	encryptionKey      []byte
}

// NewAllegroRatingsHandler creates a new AllegroRatingsHandler.
func NewAllegroRatingsHandler(integrationService *service.IntegrationService, encryptionKey []byte) *AllegroRatingsHandler {
	return &AllegroRatingsHandler{
		integrationService: integrationService,
		encryptionKey:      encryptionKey,
	}
}

// newAllegroClient creates an authenticated Allegro SDK client from the integration credentials.
func (h *AllegroRatingsHandler) newAllegroClient(r *http.Request) (*allegrosdk.Client, error) {
	return buildAllegroClient(r, h.integrationService, h.encryptionKey)
}

// ListRatings returns the list of seller ratings from Allegro.
// GET /v1/integrations/allegro/ratings
func (h *AllegroRatingsHandler) ListRatings(w http.ResponseWriter, r *http.Request) {
	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro ratings: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	params := &allegrosdk.RatingsParams{}
	if v := r.URL.Query().Get("limit"); v != "" {
		params.Limit, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		params.Offset, _ = strconv.Atoi(v)
	}

	result, err := client.Ratings.List(r.Context(), params)
	if err != nil {
		slog.Error("allegro ratings: failed to list ratings", "error", err)
		writeError(w, http.StatusBadGateway, "Nie udalo sie pobrac ocen z Allegro")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetAnswer returns the seller's answer to a rating.
// GET /v1/integrations/allegro/ratings/{ratingId}/answer
func (h *AllegroRatingsHandler) GetAnswer(w http.ResponseWriter, r *http.Request) {
	ratingID := chi.URLParam(r, "ratingId")
	if ratingID == "" {
		writeError(w, http.StatusBadRequest, "ID oceny jest wymagane")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro ratings: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	result, err := client.Ratings.GetAnswer(r.Context(), ratingID)
	if err != nil {
		slog.Error("allegro ratings: failed to get answer", "error", err, "ratingId", ratingID)
		writeError(w, http.StatusBadGateway, "Nie udalo sie pobrac odpowiedzi na ocene")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// CreateAnswer creates or updates the seller's answer to a rating.
// PUT /v1/integrations/allegro/ratings/{ratingId}/answer
func (h *AllegroRatingsHandler) CreateAnswer(w http.ResponseWriter, r *http.Request) {
	ratingID := chi.URLParam(r, "ratingId")
	if ratingID == "" {
		writeError(w, http.StatusBadRequest, "ID oceny jest wymagane")
		return
	}

	var body struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidlowe dane zadania")
		return
	}
	if body.Text == "" {
		writeError(w, http.StatusBadRequest, "Tresc odpowiedzi jest wymagana")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro ratings: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	result, err := client.Ratings.CreateAnswer(r.Context(), ratingID, allegrosdk.RatingAnswerRequest{
		Text: body.Text,
	})
	if err != nil {
		slog.Error("allegro ratings: failed to create answer", "error", err, "ratingId", ratingID)
		writeError(w, http.StatusBadGateway, "Nie udalo sie zapisac odpowiedzi na ocene")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// DeleteAnswer removes the seller's answer to a rating.
// DELETE /v1/integrations/allegro/ratings/{ratingId}/answer
func (h *AllegroRatingsHandler) DeleteAnswer(w http.ResponseWriter, r *http.Request) {
	ratingID := chi.URLParam(r, "ratingId")
	if ratingID == "" {
		writeError(w, http.StatusBadRequest, "ID oceny jest wymagane")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro ratings: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	err = client.Ratings.DeleteAnswer(r.Context(), ratingID)
	if err != nil {
		slog.Error("allegro ratings: failed to delete answer", "error", err, "ratingId", ratingID)
		writeError(w, http.StatusBadGateway, "Nie udalo sie usunac odpowiedzi na ocene")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// RequestRemoval requests removal of a rating.
// POST /v1/integrations/allegro/ratings/{ratingId}/removal
func (h *AllegroRatingsHandler) RequestRemoval(w http.ResponseWriter, r *http.Request) {
	ratingID := chi.URLParam(r, "ratingId")
	if ratingID == "" {
		writeError(w, http.StatusBadRequest, "ID oceny jest wymagane")
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidlowe dane zadania")
		return
	}
	if body.Reason == "" {
		writeError(w, http.StatusBadRequest, "Powod zgloszenia jest wymagany")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro ratings: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	err = client.Ratings.RequestRemoval(r.Context(), ratingID, body.Reason)
	if err != nil {
		slog.Error("allegro ratings: failed to request removal", "error", err, "ratingId", ratingID)
		writeError(w, http.StatusBadGateway, "Nie udalo sie zglosic oceny do usuniecia")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "removal_requested"})
}
