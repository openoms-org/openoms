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

type ExchangeRateHandler struct {
	exchangeRateService *service.ExchangeRateService
}

func NewExchangeRateHandler(exchangeRateService *service.ExchangeRateService) *ExchangeRateHandler {
	return &ExchangeRateHandler{exchangeRateService: exchangeRateService}
}

func (h *ExchangeRateHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.ExchangeRateListFilter{
		PaginationParams: pagination,
	}
	if base := r.URL.Query().Get("base_currency"); base != "" {
		filter.BaseCurrency = &base
	}
	if target := r.URL.Query().Get("target_currency"); target != "" {
		filter.TargetCurrency = &target
	}

	rates, total, err := h.exchangeRateService.List(r.Context(), tenantID, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list exchange rates")
		return
	}
	if rates == nil {
		rates = []model.ExchangeRate{}
	}
	writeJSON(w, http.StatusOK, model.ListResponse[model.ExchangeRate]{
		Items:  rates,
		Total:  total,
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	})
}

func (h *ExchangeRateHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid exchange rate ID")
		return
	}

	rate, err := h.exchangeRateService.Get(r.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, service.ErrExchangeRateNotFound) {
			writeError(w, http.StatusNotFound, "exchange rate not found")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to get exchange rate")
		}
		return
	}
	writeJSON(w, http.StatusOK, rate)
}

func (h *ExchangeRateHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.CreateExchangeRateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rate, err := h.exchangeRateService.Create(r.Context(), tenantID, req, actorID, clientIP(r))
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, "failed to create exchange rate")
		}
		return
	}
	writeJSON(w, http.StatusCreated, rate)
}

func (h *ExchangeRateHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid exchange rate ID")
		return
	}

	var req model.UpdateExchangeRateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rate, err := h.exchangeRateService.Update(r.Context(), tenantID, id, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrExchangeRateNotFound):
			writeError(w, http.StatusNotFound, "exchange rate not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "failed to update exchange rate")
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, rate)
}

func (h *ExchangeRateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid exchange rate ID")
		return
	}

	err = h.exchangeRateService.Delete(r.Context(), tenantID, id, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrExchangeRateNotFound):
			writeError(w, http.StatusNotFound, "exchange rate not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to delete exchange rate")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ExchangeRateHandler) FetchNBP(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	count, err := h.exchangeRateService.FetchNBPRates(r.Context(), tenantID, actorID, clientIP(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch NBP rates: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"fetched": count,
		"source":  "nbp",
	})
}

func (h *ExchangeRateHandler) Convert(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var req model.ConvertAmountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.exchangeRateService.ConvertAmount(r.Context(), tenantID, req.Amount, req.From, req.To)
	if err != nil {
		if errors.Is(err, service.ErrRateNotAvailable) {
			writeError(w, http.StatusNotFound, "exchange rate not available for this currency pair")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to convert amount")
		}
		return
	}

	writeJSON(w, http.StatusOK, result)
}
