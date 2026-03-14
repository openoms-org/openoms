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

// AllegroCommsHandler handles Allegro messaging, returns, and refund API routes.
type AllegroCommsHandler struct {
	integrationService *service.IntegrationService
	encryptionKey      []byte
}

// NewAllegroCommsHandler creates a new AllegroCommsHandler.
func NewAllegroCommsHandler(integrationService *service.IntegrationService, encryptionKey []byte) *AllegroCommsHandler {
	return &AllegroCommsHandler{
		integrationService: integrationService,
		encryptionKey:      encryptionKey,
	}
}

// newAllegroClient creates an authenticated Allegro SDK client from the integration credentials.
func (h *AllegroCommsHandler) newAllegroClient(r *http.Request) (*allegrosdk.Client, error) {
	return buildAllegroClient(r, h.integrationService, h.encryptionKey)
}

// --- Messaging ---

// ListThreads returns the list of messaging threads from Allegro.
func (h *AllegroCommsHandler) ListThreads(w http.ResponseWriter, r *http.Request) {
	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro comms: failed to get provider", "error", err)
		writeError(w, http.StatusBadRequest, "No active Allegro integration")
		return
	}
	defer client.Close()

	params := &allegrosdk.ListThreadsParams{}
	if v := r.URL.Query().Get("limit"); v != "" {
		params.Limit, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		params.Offset, _ = strconv.Atoi(v)
	}

	result, err := client.Messages.ListThreads(r.Context(), params)
	if err != nil {
		slog.Error("allegro comms: failed to list threads", "error", err)
		writeAllegroError(w, "Failed to fetch threads from Allegro", err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetMessages returns messages for a specific thread.
func (h *AllegroCommsHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	threadID := chi.URLParam(r, "threadId")

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro comms: failed to get provider", "error", err)
		writeError(w, http.StatusBadRequest, "No active Allegro integration")
		return
	}
	defer client.Close()

	params := &allegrosdk.ListMessagesParams{}
	if v := r.URL.Query().Get("limit"); v != "" {
		params.Limit, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		params.Offset, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("before"); v != "" {
		params.Before = v
	}

	result, err := client.Messages.ListMessages(r.Context(), threadID, params)
	if err != nil {
		slog.Error("allegro comms: failed to list messages", "error", err, "threadId", threadID) //nolint:gosec
		writeAllegroError(w, "Failed to fetch messages from Allegro", err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// SendMessage sends a message in a specific thread.
func (h *AllegroCommsHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	threadID := chi.URLParam(r, "threadId")

	var body struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if body.Text == "" {
		writeError(w, http.StatusBadRequest, "Message text is required")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro comms: failed to get provider", "error", err)
		writeError(w, http.StatusBadRequest, "No active Allegro integration")
		return
	}
	defer client.Close()

	msg, err := client.Messages.SendMessage(r.Context(), threadID, allegrosdk.SendMessageRequest{
		Text: body.Text,
	})
	if err != nil {
		slog.Error("allegro comms: failed to send message", "error", err, "threadId", threadID) //nolint:gosec
		writeAllegroError(w, "Failed to send message", err)
		return
	}

	writeJSON(w, http.StatusOK, msg)
}

// --- Returns ---

// ListAllegroReturns returns the list of customer returns from Allegro.
func (h *AllegroCommsHandler) ListAllegroReturns(w http.ResponseWriter, r *http.Request) {
	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro comms: failed to get provider", "error", err)
		writeError(w, http.StatusBadRequest, "No active Allegro integration")
		return
	}
	defer client.Close()

	params := &allegrosdk.ListReturnsParams{}
	if v := r.URL.Query().Get("limit"); v != "" {
		params.Limit, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		params.Offset, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("status"); v != "" {
		params.Status = v
	}

	result, err := client.Returns.ListReturns(r.Context(), params)
	if err != nil {
		slog.Error("allegro comms: failed to list returns", "error", err)
		writeAllegroError(w, "Failed to fetch returns from Allegro", err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetAllegroReturn returns a single customer return from Allegro.
func (h *AllegroCommsHandler) GetAllegroReturn(w http.ResponseWriter, r *http.Request) {
	returnID := chi.URLParam(r, "returnId")

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro comms: failed to get provider", "error", err)
		writeError(w, http.StatusBadRequest, "No active Allegro integration")
		return
	}
	defer client.Close()

	result, err := client.Returns.GetReturn(r.Context(), returnID)
	if err != nil {
		slog.Error("allegro comms: failed to get return", "error", err, "returnId", returnID) //nolint:gosec
		writeAllegroError(w, "Failed to fetch return from Allegro", err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// RejectAllegroReturn rejects a customer return on Allegro.
func (h *AllegroCommsHandler) RejectAllegroReturn(w http.ResponseWriter, r *http.Request) {
	returnID := chi.URLParam(r, "returnId")

	var body struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if body.Reason == "" {
		writeError(w, http.StatusBadRequest, "Rejection reason is required")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro comms: failed to get provider", "error", err)
		writeError(w, http.StatusBadRequest, "No active Allegro integration")
		return
	}
	defer client.Close()

	err = client.Returns.RejectReturn(r.Context(), returnID, allegrosdk.ReturnRejection{
		Reason: body.Reason,
	})
	if err != nil {
		slog.Error("allegro comms: failed to reject return", "error", err, "returnId", returnID) //nolint:gosec
		writeAllegroError(w, "Failed to reject return", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

// --- Refunds ---

// CreateRefund creates a new refund on Allegro.
func (h *AllegroCommsHandler) CreateRefund(w http.ResponseWriter, r *http.Request) {
	var body allegrosdk.CreateRefundRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if body.Payment.ID == "" {
		writeError(w, http.StatusBadRequest, "Payment ID is required")
		return
	}
	if body.Reason == "" {
		writeError(w, http.StatusBadRequest, "Refund reason is required")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro comms: failed to get provider", "error", err)
		writeError(w, http.StatusBadRequest, "No active Allegro integration")
		return
	}
	defer client.Close()

	result, err := client.Payments.CreateRefund(r.Context(), body)
	if err != nil {
		slog.Error("allegro comms: failed to create refund", "error", err)
		writeAllegroError(w, "Failed to create refund", err)
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

// ListRefunds returns the list of refunds from Allegro.
func (h *AllegroCommsHandler) ListRefunds(w http.ResponseWriter, r *http.Request) {
	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro comms: failed to get provider", "error", err)
		writeError(w, http.StatusBadRequest, "No active Allegro integration")
		return
	}
	defer client.Close()

	params := &allegrosdk.ListRefundsParams{}
	if v := r.URL.Query().Get("limit"); v != "" {
		params.Limit, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		params.Offset, _ = strconv.Atoi(v)
	}

	result, err := client.Payments.ListRefunds(r.Context(), params)
	if err != nil {
		slog.Error("allegro comms: failed to list refunds", "error", err)
		writeAllegroError(w, "Failed to fetch refunds from Allegro", err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}
