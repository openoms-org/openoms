package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// KSeFHandler handles KSeF-related HTTP endpoints.
type KSeFHandler struct {
	ksefService *service.KSeFService
}

// NewKSeFHandler creates a new KSeF handler.
func NewKSeFHandler(ksefService *service.KSeFService) *KSeFHandler {
	return &KSeFHandler{ksefService: ksefService}
}

// GetSettings returns the KSeF configuration for the current tenant.
func (h *KSeFHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	cfg, err := h.ksefService.GetSettings(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load KSeF settings")
		return
	}

	if cfg == nil {
		cfg = &service.KSeFSettings{}
	}

	// Mask the token in the response
	resp := map[string]any{
		"enabled":         cfg.Enabled,
		"environment":     cfg.Environment,
		"nip":             cfg.NIP,
		"token":           maskToken(cfg.Token),
		"company_name":    cfg.CompanyName,
		"company_street":  cfg.CompanyStreet,
		"company_city":    cfg.CompanyCity,
		"company_postal":  cfg.CompanyPostal,
		"company_country": cfg.CompanyCountry,
	}

	writeJSON(w, http.StatusOK, resp)
}

// UpdateSettings saves the KSeF configuration.
func (h *KSeFHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var cfg service.KSeFSettings
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// If token is masked, preserve the existing one
	if cfg.Token == maskedValue {
		existing, err := h.ksefService.GetSettings(r.Context(), tenantID)
		if err == nil && existing != nil {
			cfg.Token = existing.Token
		}
	}

	if err := h.ksefService.UpdateSettings(r.Context(), tenantID, cfg, actorID, clientIP(r)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save KSeF settings")
		return
	}

	resp := map[string]any{
		"enabled":         cfg.Enabled,
		"environment":     cfg.Environment,
		"nip":             cfg.NIP,
		"token":           maskToken(cfg.Token),
		"company_name":    cfg.CompanyName,
		"company_street":  cfg.CompanyStreet,
		"company_city":    cfg.CompanyCity,
		"company_postal":  cfg.CompanyPostal,
		"company_country": cfg.CompanyCountry,
	}

	writeJSON(w, http.StatusOK, resp)
}

// TestConnection tests the KSeF API connection.
func (h *KSeFHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	result, err := h.ksefService.TestConnection(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to test KSeF connection")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// SendToKSeF sends a single invoice to KSeF.
func (h *KSeFHandler) SendToKSeF(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	invoiceID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	err = h.ksefService.SendToKSeF(r.Context(), tenantID, invoiceID, actorID, clientIP(r))
	if err != nil {
		if errors.Is(err, service.ErrInvoiceNotFound) {
			writeError(w, http.StatusNotFound, "invoice not found")
			return
		}
		if errors.Is(err, service.ErrKSeFNotConfigured) {
			writeError(w, http.StatusBadRequest, "KSeF nie jest skonfigurowany")
			return
		}
		if errors.Is(err, service.ErrKSeFAlreadySent) {
			writeError(w, http.StatusConflict, "Faktura została już wysłana do KSeF")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to send invoice to KSeF")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Faktura wysłana do KSeF"})
}

// CheckKSeFStatus checks the KSeF status of an invoice.
func (h *KSeFHandler) CheckKSeFStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	invoiceID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	inv, err := h.ksefService.CheckKSeFStatus(r.Context(), tenantID, invoiceID)
	if err != nil {
		if errors.Is(err, service.ErrInvoiceNotFound) {
			writeError(w, http.StatusNotFound, "invoice not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to check KSeF status")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ksef_status":   inv.KSeFStatus,
		"ksef_number":   inv.KSeFNumber,
		"ksef_sent_at":  inv.KSeFSentAt,
		"ksef_response": inv.KSeFResponse,
	})
}

// GetUPO downloads the UPO for an invoice.
func (h *KSeFHandler) GetUPO(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	invoiceID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	upoData, err := h.ksefService.GetUPO(r.Context(), tenantID, invoiceID)
	if err != nil {
		if errors.Is(err, service.ErrInvoiceNotFound) {
			writeError(w, http.StatusNotFound, "invoice not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to download UPO: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Content-Disposition", "attachment; filename=upo.xml")
	w.WriteHeader(http.StatusOK)
	w.Write(upoData)
}

// BulkSendToKSeF sends multiple invoices to KSeF.
func (h *KSeFHandler) BulkSendToKSeF(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req struct {
		InvoiceIDs []string `json:"invoice_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.InvoiceIDs) == 0 {
		writeError(w, http.StatusBadRequest, "invoice_ids is required")
		return
	}

	ids := make([]uuid.UUID, 0, len(req.InvoiceIDs))
	for _, idStr := range req.InvoiceIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid invoice ID: "+idStr)
			return
		}
		ids = append(ids, id)
	}

	sent, errs, err := h.ksefService.BulkSendToKSeF(r.Context(), tenantID, ids, actorID, clientIP(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to bulk send to KSeF")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"sent":   sent,
		"errors": errs,
		"total":  len(ids),
	})
}

const maskedValue = "******"

func maskToken(token string) string {
	if token == "" {
		return ""
	}
	return maskedValue
}
