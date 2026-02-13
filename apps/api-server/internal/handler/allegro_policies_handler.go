package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// AllegroPoliciesHandler handles Allegro after-sales policies: return policies,
// implied warranties (rekojmia), and size tables.
type AllegroPoliciesHandler struct {
	integrationService *service.IntegrationService
	encryptionKey      []byte
}

// NewAllegroPoliciesHandler creates a new AllegroPoliciesHandler.
func NewAllegroPoliciesHandler(integrationService *service.IntegrationService, encryptionKey []byte) *AllegroPoliciesHandler {
	return &AllegroPoliciesHandler{
		integrationService: integrationService,
		encryptionKey:      encryptionKey,
	}
}

// newAllegroClient creates an authenticated Allegro SDK client from the integration credentials.
func (h *AllegroPoliciesHandler) newAllegroClient(r *http.Request) (*allegrosdk.Client, error) {
	return buildAllegroClient(r, h.integrationService, h.encryptionKey)
}

// --- Return Policies ---

// ListReturnPolicies lists the seller's return policies.
// GET /v1/integrations/allegro/return-policies
func (h *AllegroPoliciesHandler) ListReturnPolicies(w http.ResponseWriter, r *http.Request) {
	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro policies: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	result, err := client.AfterSales.ListReturnPolicies(r.Context())
	if err != nil {
		slog.Error("allegro policies: failed to list return policies", "error", err)
		writeError(w, http.StatusBadGateway, "Nie udalo sie pobrac polityk zwrotow")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetReturnPolicy gets a single return policy.
// GET /v1/integrations/allegro/return-policies/{policyId}
func (h *AllegroPoliciesHandler) GetReturnPolicy(w http.ResponseWriter, r *http.Request) {
	policyID := chi.URLParam(r, "policyId")
	if policyID == "" {
		writeError(w, http.StatusBadRequest, "policyId jest wymagane")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro policies: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	result, err := client.AfterSales.GetReturnPolicy(r.Context(), policyID)
	if err != nil {
		slog.Error("allegro policies: failed to get return policy", "error", err, "policy_id", policyID)
		writeError(w, http.StatusBadGateway, "Nie udalo sie pobrac polityki zwrotow")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// CreateReturnPolicy creates a return policy.
// POST /v1/integrations/allegro/return-policies
func (h *AllegroPoliciesHandler) CreateReturnPolicy(w http.ResponseWriter, r *http.Request) {
	var body allegrosdk.CreateReturnPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidlowe dane wejsciowe")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro policies: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	result, err := client.AfterSales.CreateReturnPolicy(r.Context(), body)
	if err != nil {
		slog.Error("allegro policies: failed to create return policy", "error", err)
		writeError(w, http.StatusBadGateway, allegroErrorMessage("Nie udalo sie utworzyc polityki zwrotow", err))
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

// UpdateReturnPolicy updates a return policy.
// PUT /v1/integrations/allegro/return-policies/{policyId}
func (h *AllegroPoliciesHandler) UpdateReturnPolicy(w http.ResponseWriter, r *http.Request) {
	policyID := chi.URLParam(r, "policyId")
	if policyID == "" {
		writeError(w, http.StatusBadRequest, "policyId jest wymagane")
		return
	}

	var body allegrosdk.CreateReturnPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidlowe dane wejsciowe")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro policies: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	result, err := client.AfterSales.UpdateReturnPolicy(r.Context(), policyID, body)
	if err != nil {
		slog.Error("allegro policies: failed to update return policy", "error", err, "policy_id", policyID)
		writeError(w, http.StatusBadGateway, "Nie udalo sie zaktualizowac polityki zwrotow")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// --- Implied Warranties ---

// ListWarranties lists the seller's implied warranty policies.
// GET /v1/integrations/allegro/warranties
func (h *AllegroPoliciesHandler) ListWarranties(w http.ResponseWriter, r *http.Request) {
	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro policies: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	result, err := client.AfterSales.ListWarranties(r.Context())
	if err != nil {
		slog.Error("allegro policies: failed to list warranties", "error", err)
		writeError(w, http.StatusBadGateway, "Nie udalo sie pobrac rekojmi")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetWarranty gets a single implied warranty.
// GET /v1/integrations/allegro/warranties/{warrantyId}
func (h *AllegroPoliciesHandler) GetWarranty(w http.ResponseWriter, r *http.Request) {
	warrantyID := chi.URLParam(r, "warrantyId")
	if warrantyID == "" {
		writeError(w, http.StatusBadRequest, "warrantyId jest wymagane")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro policies: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	result, err := client.AfterSales.GetWarranty(r.Context(), warrantyID)
	if err != nil {
		slog.Error("allegro policies: failed to get warranty", "error", err, "warranty_id", warrantyID)
		writeError(w, http.StatusBadGateway, "Nie udalo sie pobrac rekojmi")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// CreateWarranty creates an implied warranty.
// POST /v1/integrations/allegro/warranties
func (h *AllegroPoliciesHandler) CreateWarranty(w http.ResponseWriter, r *http.Request) {
	var body allegrosdk.CreateWarrantyRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidlowe dane wejsciowe")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro policies: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	result, err := client.AfterSales.CreateWarranty(r.Context(), body)
	if err != nil {
		slog.Error("allegro policies: failed to create warranty", "error", err)
		writeError(w, http.StatusBadGateway, allegroErrorMessage("Nie udalo sie utworzyc rekojmi", err))
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

// UpdateWarranty updates an implied warranty.
// PUT /v1/integrations/allegro/warranties/{warrantyId}
func (h *AllegroPoliciesHandler) UpdateWarranty(w http.ResponseWriter, r *http.Request) {
	warrantyID := chi.URLParam(r, "warrantyId")
	if warrantyID == "" {
		writeError(w, http.StatusBadRequest, "warrantyId jest wymagane")
		return
	}

	var body allegrosdk.CreateWarrantyRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidlowe dane wejsciowe")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro policies: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	result, err := client.AfterSales.UpdateWarranty(r.Context(), warrantyID, body)
	if err != nil {
		slog.Error("allegro policies: failed to update warranty", "error", err, "warranty_id", warrantyID)
		writeError(w, http.StatusBadGateway, "Nie udalo sie zaktualizowac rekojmi")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// --- Size Tables ---

// ListSizeTables lists the seller's size tables.
// GET /v1/integrations/allegro/size-tables
func (h *AllegroPoliciesHandler) ListSizeTables(w http.ResponseWriter, r *http.Request) {
	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro policies: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	result, err := client.SizeTables.List(r.Context())
	if err != nil {
		slog.Error("allegro policies: failed to list size tables", "error", err)
		writeError(w, http.StatusBadGateway, "Nie udalo sie pobrac tabel rozmiarow")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetSizeTable gets a single size table.
// GET /v1/integrations/allegro/size-tables/{tableId}
func (h *AllegroPoliciesHandler) GetSizeTable(w http.ResponseWriter, r *http.Request) {
	tableID := chi.URLParam(r, "tableId")
	if tableID == "" {
		writeError(w, http.StatusBadRequest, "tableId jest wymagane")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro policies: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	result, err := client.SizeTables.Get(r.Context(), tableID)
	if err != nil {
		slog.Error("allegro policies: failed to get size table", "error", err, "table_id", tableID)
		writeError(w, http.StatusBadGateway, "Nie udalo sie pobrac tabeli rozmiarow")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// CreateSizeTable creates a new size table.
// POST /v1/integrations/allegro/size-tables
func (h *AllegroPoliciesHandler) CreateSizeTable(w http.ResponseWriter, r *http.Request) {
	var body allegrosdk.CreateSizeTableRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidlowe dane wejsciowe")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro policies: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	result, err := client.SizeTables.Create(r.Context(), body)
	if err != nil {
		slog.Error("allegro policies: failed to create size table", "error", err)
		writeError(w, http.StatusBadGateway, "Nie udalo sie utworzyc tabeli rozmiarow")
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

// UpdateSizeTable updates a size table.
// PUT /v1/integrations/allegro/size-tables/{tableId}
func (h *AllegroPoliciesHandler) UpdateSizeTable(w http.ResponseWriter, r *http.Request) {
	tableID := chi.URLParam(r, "tableId")
	if tableID == "" {
		writeError(w, http.StatusBadRequest, "tableId jest wymagane")
		return
	}

	var body allegrosdk.CreateSizeTableRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidlowe dane wejsciowe")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro policies: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	result, err := client.SizeTables.Update(r.Context(), tableID, body)
	if err != nil {
		slog.Error("allegro policies: failed to update size table", "error", err, "table_id", tableID)
		writeError(w, http.StatusBadGateway, "Nie udalo sie zaktualizowac tabeli rozmiarow")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// DeleteSizeTable deletes a size table.
// DELETE /v1/integrations/allegro/size-tables/{tableId}
func (h *AllegroPoliciesHandler) DeleteSizeTable(w http.ResponseWriter, r *http.Request) {
	tableID := chi.URLParam(r, "tableId")
	if tableID == "" {
		writeError(w, http.StatusBadRequest, "tableId jest wymagane")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro policies: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	if err := client.SizeTables.Delete(r.Context(), tableID); err != nil {
		slog.Error("allegro policies: failed to delete size table", "error", err, "table_id", tableID)
		writeError(w, http.StatusBadGateway, "Nie udalo sie usunac tabeli rozmiarow")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// allegroErrorMessage extracts the detailed error message from Allegro API errors.
func allegroErrorMessage(fallback string, err error) string {
	var apiErr *allegrosdk.APIError
	if errors.As(err, &apiErr) {
		if len(apiErr.Details) > 0 {
			msgs := make([]string, 0, len(apiErr.Details))
			for _, d := range apiErr.Details {
				msg := d.UserMessage
				if msg == "" {
					msg = d.Message
				}
				if d.Details != nil {
					msg += fmt.Sprintf(" (%v)", d.Details)
				}
				if d.Path != "" {
					msg += " [" + d.Path + "]"
				}
				msgs = append(msgs, msg)
			}
			return fmt.Sprintf("%s: %s", fallback, strings.Join(msgs, "; "))
		}
		if apiErr.Message != "" {
			return fmt.Sprintf("%s: %s", fallback, apiErr.Message)
		}
	}
	return fallback
}
