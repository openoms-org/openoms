package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

type ReconciliationHandler struct {
	reconciliationService *service.ReconciliationService
}

func NewReconciliationHandler(reconciliationService *service.ReconciliationService) *ReconciliationHandler {
	return &ReconciliationHandler{reconciliationService: reconciliationService}
}

// CreateSettlement handles POST /v1/reconciliation/settlements
func (h *ReconciliationHandler) CreateSettlement(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var req model.CreateSettlementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Provider == "" {
		writeError(w, http.StatusBadRequest, "provider is required")
		return
	}
	if req.SettlementDate == "" {
		writeError(w, http.StatusBadRequest, "settlement_date is required")
		return
	}

	settlement, err := h.reconciliationService.ImportSettlement(r.Context(), tenantID, req)
	if err != nil {
		writeServerError(w, "failed to create settlement", err)
		return
	}

	writeJSON(w, http.StatusCreated, settlement)
}

// ImportCSV handles POST /v1/reconciliation/import-csv
func (h *ReconciliationHandler) ImportCSV(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	// Limit file size to 10MB
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "file too large or invalid form data (max 10MB)")
		return
	}

	provider := r.FormValue("provider")
	if provider == "" {
		writeError(w, http.StatusBadRequest, "provider is required")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	settlement, err := h.reconciliationService.ImportCSV(r.Context(), tenantID, provider, file)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, settlement)
}

// ListSettlements handles GET /v1/reconciliation/settlements
func (h *ReconciliationHandler) ListSettlements(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.SettlementListFilter{
		PaginationParams: pagination,
	}
	if s := r.URL.Query().Get("provider"); s != "" {
		filter.Provider = &s
	}
	if s := r.URL.Query().Get("status"); s != "" {
		filter.Status = &s
	}
	if s := r.URL.Query().Get("date_from"); s != "" {
		filter.DateFrom = &s
	}
	if s := r.URL.Query().Get("date_to"); s != "" {
		filter.DateTo = &s
	}

	resp, err := h.reconciliationService.ListSettlements(r.Context(), tenantID, filter)
	if err != nil {
		writeServerError(w, "failed to list settlements", err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetSettlement handles GET /v1/reconciliation/settlements/{id}
func (h *ReconciliationHandler) GetSettlement(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	settlementID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid settlement ID")
		return
	}

	result, err := h.reconciliationService.GetSettlement(r.Context(), tenantID, settlementID)
	if err != nil {
		if errors.Is(err, service.ErrSettlementNotFound) {
			writeError(w, http.StatusNotFound, "settlement not found")
			return
		}
		writeServerError(w, "failed to get settlement", err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// AutoMatch handles POST /v1/reconciliation/settlements/{id}/auto-match
func (h *ReconciliationHandler) AutoMatch(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	settlementID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid settlement ID")
		return
	}

	result, err := h.reconciliationService.AutoMatch(r.Context(), tenantID, settlementID)
	if err != nil {
		if errors.Is(err, service.ErrSettlementNotFound) {
			writeError(w, http.StatusNotFound, "settlement not found")
			return
		}
		writeServerError(w, "failed to auto-match", err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// ListTransactions handles GET /v1/reconciliation/transactions
func (h *ReconciliationHandler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.TransactionListFilter{
		PaginationParams: pagination,
	}
	if s := r.URL.Query().Get("settlement_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid settlement_id filter")
			return
		}
		filter.SettlementID = &id
	}
	if s := r.URL.Query().Get("match_status"); s != "" {
		filter.MatchStatus = &s
	}
	if s := r.URL.Query().Get("provider"); s != "" {
		filter.Provider = &s
	}
	if s := r.URL.Query().Get("transaction_type"); s != "" {
		filter.TransactionType = &s
	}
	if s := r.URL.Query().Get("date_from"); s != "" {
		filter.DateFrom = &s
	}
	if s := r.URL.Query().Get("date_to"); s != "" {
		filter.DateTo = &s
	}

	resp, err := h.reconciliationService.ListTransactions(r.Context(), tenantID, filter)
	if err != nil {
		writeServerError(w, "failed to list transactions", err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// ManualMatch handles POST /v1/reconciliation/transactions/{id}/match
func (h *ReconciliationHandler) ManualMatch(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	transactionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid transaction ID")
		return
	}

	var req model.ManualMatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.OrderID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "order_id is required")
		return
	}

	err = h.reconciliationService.ManualMatch(r.Context(), tenantID, transactionID, req)
	if err != nil {
		if errors.Is(err, service.ErrTransactionNotFound) {
			writeError(w, http.StatusNotFound, "transaction not found")
			return
		}
		if errors.Is(err, service.ErrOrderNotFound) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeServerError(w, "failed to match transaction", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Unmatch handles POST /v1/reconciliation/transactions/{id}/unmatch
func (h *ReconciliationHandler) Unmatch(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	transactionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid transaction ID")
		return
	}

	err = h.reconciliationService.Unmatch(r.Context(), tenantID, transactionID)
	if err != nil {
		if errors.Is(err, service.ErrTransactionNotFound) {
			writeError(w, http.StatusNotFound, "transaction not found")
			return
		}
		writeServerError(w, "failed to unmatch transaction", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetSummary handles GET /v1/reconciliation/summary
func (h *ReconciliationHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var dateFrom, dateTo *string
	if s := r.URL.Query().Get("date_from"); s != "" {
		dateFrom = &s
	}
	if s := r.URL.Query().Get("date_to"); s != "" {
		dateTo = &s
	}

	summary, err := h.reconciliationService.GetSummary(r.Context(), tenantID, dateFrom, dateTo)
	if err != nil {
		writeServerError(w, "failed to get reconciliation summary", err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}
