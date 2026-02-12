package handler

import (
	"encoding/json"
	"net/http"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// RateHandler handles shipping rate comparison requests.
type RateHandler struct {
	rateService *service.RateService
}

// NewRateHandler creates a new RateHandler.
func NewRateHandler(rateService *service.RateService) *RateHandler {
	return &RateHandler{rateService: rateService}
}

type getRatesRequest struct {
	FromPostalCode string  `json:"from_postal_code"`
	FromCountry    string  `json:"from_country"`
	ToPostalCode   string  `json:"to_postal_code"`
	ToCountry      string  `json:"to_country"`
	Weight         float64 `json:"weight"`
	Width          float64 `json:"width"`
	Height         float64 `json:"height"`
	Length         float64 `json:"length"`
	COD            float64 `json:"cod"`
}

type getRatesResponse struct {
	Rates []integration.Rate `json:"rates"`
}

// GetRates handles POST /v1/shipping/rates.
func (h *RateHandler) GetRates(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var req getRatesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Weight <= 0 {
		writeError(w, http.StatusBadRequest, "weight must be greater than 0")
		return
	}

	rateReq := integration.RateRequest{
		FromPostalCode: req.FromPostalCode,
		FromCountry:    req.FromCountry,
		ToPostalCode:   req.ToPostalCode,
		ToCountry:      req.ToCountry,
		Weight:         req.Weight,
		Width:          req.Width,
		Height:         req.Height,
		Length:         req.Length,
		COD:            req.COD,
	}

	rates, err := h.rateService.GetRates(r.Context(), tenantID, rateReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get shipping rates")
		return
	}

	writeJSON(w, http.StatusOK, getRatesResponse{Rates: rates})
}
