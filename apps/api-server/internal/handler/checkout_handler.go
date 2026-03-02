package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// CheckoutHandler handles billing checkout endpoints.
type CheckoutHandler struct {
	checkoutSvc *service.CheckoutService
	planCache   *service.PlanCache
	pool        *pgxpool.Pool
	frontendURL string
}

// NewCheckoutHandler creates a new CheckoutHandler.
func NewCheckoutHandler(checkoutSvc *service.CheckoutService, planCache *service.PlanCache, pool *pgxpool.Pool, frontendURL string) *CheckoutHandler {
	return &CheckoutHandler{checkoutSvc: checkoutSvc, planCache: planCache, pool: pool, frontendURL: frontendURL}
}

// ListPlans returns available plans without Stripe-sensitive data.
// GET /v1/billing/plans
func (h *CheckoutHandler) ListPlans(w http.ResponseWriter, _ *http.Request) {
	plans := h.checkoutSvc.ListPlans()
	writeJSON(w, http.StatusOK, plans)
}

// CreateCheckoutSession creates a Stripe Checkout Session.
// POST /v1/billing/checkout
func (h *CheckoutHandler) CreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	var req model.CheckoutSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	successURL := h.frontendURL + "/register/complete?session_id={CHECKOUT_SESSION_ID}"
	cancelURL := h.frontendURL + "/register"

	resp, err := h.checkoutSvc.CreateCheckoutSession(r.Context(), req.PlanID, req.Interval, successURL, cancelURL)
	if err != nil {
		switch err {
		case service.ErrPlanNotFound:
			writeError(w, http.StatusBadRequest, "invalid plan_id")
		default:
			writeServerError(w, "failed to create checkout session", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetCheckoutSessionStatus returns the status of a checkout session.
// GET /v1/billing/checkout/{session_id}
func (h *CheckoutHandler) GetCheckoutSessionStatus(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "session_id")
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "session_id is required")
		return
	}

	status, err := h.checkoutSvc.GetSessionStatus(r.Context(), sessionID)
	if err != nil {
		writeServerError(w, "failed to get checkout session status", err)
		return
	}
	if status == nil {
		writeError(w, http.StatusNotFound, "checkout session not found")
		return
	}

	writeJSON(w, http.StatusOK, status)
}

// GetSubscription returns the current subscription status for the authenticated tenant.
// GET /v1/billing/subscription
func (h *CheckoutHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var tenantPlan string
	var tenantSettings json.RawMessage
	if h.planCache != nil && h.pool != nil {
		tenantPlan, tenantSettings, _ = h.planCache.GetOrLoad(r.Context(), h.pool, tenantID)
	}
	if tenantPlan == "" {
		tenantPlan = "free"
	}

	sub, err := h.checkoutSvc.GetSubscription(r.Context(), tenantID, tenantPlan, tenantSettings)
	if err != nil {
		writeServerError(w, "failed to get subscription", err)
		return
	}

	writeJSON(w, http.StatusOK, sub)
}
