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

// AllegroDisputesHandler handles Allegro post-purchase dispute API routes.
type AllegroDisputesHandler struct {
	integrationService *service.IntegrationService
	encryptionKey      []byte
}

// NewAllegroDisputesHandler creates a new AllegroDisputesHandler.
func NewAllegroDisputesHandler(integrationService *service.IntegrationService, encryptionKey []byte) *AllegroDisputesHandler {
	return &AllegroDisputesHandler{
		integrationService: integrationService,
		encryptionKey:      encryptionKey,
	}
}

// newAllegroClient creates an authenticated Allegro SDK client from the integration credentials.
func (h *AllegroDisputesHandler) newAllegroClient(r *http.Request) (*allegrosdk.Client, error) {
	return buildAllegroClient(r, h.integrationService, h.encryptionKey)
}

// ListDisputes returns the list of disputes from Allegro.
// GET /v1/integrations/allegro/disputes
func (h *AllegroDisputesHandler) ListDisputes(w http.ResponseWriter, r *http.Request) {
	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro disputes: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	params := &allegrosdk.ListDisputesParams{}
	if v := r.URL.Query().Get("limit"); v != "" {
		params.Limit, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		params.Offset, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("status"); v != "" {
		params.Status = v
	}

	result, err := client.Disputes.List(r.Context(), params)
	if err != nil {
		slog.Error("allegro disputes: failed to list disputes", "error", err)
		writeError(w, http.StatusBadGateway, "Nie udalo sie pobrac sporow z Allegro")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetDispute returns a single dispute from Allegro.
// GET /v1/integrations/allegro/disputes/{disputeId}
func (h *AllegroDisputesHandler) GetDispute(w http.ResponseWriter, r *http.Request) {
	disputeID := chi.URLParam(r, "disputeId")
	if disputeID == "" {
		writeError(w, http.StatusBadRequest, "ID sporu jest wymagane")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro disputes: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	result, err := client.Disputes.Get(r.Context(), disputeID)
	if err != nil {
		slog.Error("allegro disputes: failed to get dispute", "error", err, "disputeId", disputeID)
		writeError(w, http.StatusBadGateway, "Nie udalo sie pobrac sporu z Allegro")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// ListDisputeMessages returns messages for a specific dispute.
// GET /v1/integrations/allegro/disputes/{disputeId}/messages
func (h *AllegroDisputesHandler) ListDisputeMessages(w http.ResponseWriter, r *http.Request) {
	disputeID := chi.URLParam(r, "disputeId")
	if disputeID == "" {
		writeError(w, http.StatusBadRequest, "ID sporu jest wymagane")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro disputes: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	result, err := client.Disputes.ListMessages(r.Context(), disputeID)
	if err != nil {
		slog.Error("allegro disputes: failed to list dispute messages", "error", err, "disputeId", disputeID)
		writeError(w, http.StatusBadGateway, "Nie udalo sie pobrac wiadomosci sporu z Allegro")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// SendDisputeMessage sends a message in a dispute.
// POST /v1/integrations/allegro/disputes/{disputeId}/messages
func (h *AllegroDisputesHandler) SendDisputeMessage(w http.ResponseWriter, r *http.Request) {
	disputeID := chi.URLParam(r, "disputeId")
	if disputeID == "" {
		writeError(w, http.StatusBadRequest, "ID sporu jest wymagane")
		return
	}

	var body struct {
		Text string `json:"text"`
		Type string `json:"type,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidlowe dane zadania")
		return
	}
	if body.Text == "" {
		writeError(w, http.StatusBadRequest, "Tresc wiadomosci jest wymagana")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro disputes: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	msg, err := client.Disputes.SendMessage(r.Context(), disputeID, allegrosdk.DisputeMessageRequest{
		Text: body.Text,
		Type: body.Type,
	})
	if err != nil {
		slog.Error("allegro disputes: failed to send message", "error", err, "disputeId", disputeID)
		writeError(w, http.StatusBadGateway, "Nie udalo sie wyslac wiadomosci w sporze")
		return
	}

	writeJSON(w, http.StatusOK, msg)
}
