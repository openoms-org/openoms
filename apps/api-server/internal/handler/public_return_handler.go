package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// PublicReturnHandler handles public (no auth) return endpoints.
type PublicReturnHandler struct {
	pool    *pgxpool.Pool
	returns *service.ReturnService
}

// NewPublicReturnHandler creates a new PublicReturnHandler.
func NewPublicReturnHandler(pool *pgxpool.Pool, returns *service.ReturnService) *PublicReturnHandler {
	return &PublicReturnHandler{
		pool:    pool,
		returns: returns,
	}
}

// GetByToken returns a return by its public token.
func (h *PublicReturnHandler) GetByToken(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}

	ret, err := h.findReturnByToken(r.Context(), token)
	if err != nil {
		writeServerError(w, "failed to find return", err)
		return
	}
	if ret == nil {
		writeError(w, http.StatusNotFound, "return not found")
		return
	}

	// Return a safe public view (no tenant_id)
	writeJSON(w, http.StatusOK, map[string]any{
		"id":             ret.ID,
		"order_id":       ret.OrderID,
		"status":         ret.Status,
		"reason":         ret.Reason,
		"items":          ret.Items,
		"refund_amount":  ret.RefundAmount,
		"customer_email": ret.CustomerEmail,
		"customer_notes": ret.CustomerNotes,
		"created_at":     ret.CreatedAt,
		"updated_at":     ret.UpdatedAt,
	})
}

// GetStatusByToken returns public status data for the return.
func (h *PublicReturnHandler) GetStatusByToken(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}

	ret, err := h.findReturnByToken(r.Context(), token)
	if err != nil {
		writeServerError(w, "failed to find return", err)
		return
	}
	if ret == nil {
		writeError(w, http.StatusNotFound, "return not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":         ret.ID,
		"status":     ret.Status,
		"reason":     ret.Reason,
		"items":      ret.Items,
		"created_at": ret.CreatedAt,
		"updated_at": ret.UpdatedAt,
	})
}

// CreatePublicReturn creates a return request from a public form submission.
func (h *PublicReturnHandler) CreatePublicReturn(w http.ResponseWriter, r *http.Request) {
	var req model.PublicReturnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Sanitize
	req.Reason = model.StripHTMLTags(req.Reason)
	req.Notes = model.StripHTMLTags(req.Notes)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	orderID, err := uuid.Parse(req.OrderID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order_id format")
		return
	}

	// Look up the order's tenant_id using a SECURITY DEFINER function
	// that bypasses RLS (avoids chicken-and-egg problem).
	var tenantID uuid.UUID
	var customerEmail *string
	err = h.pool.QueryRow(r.Context(),
		`SELECT tenant_id, customer_email FROM find_order_tenant_id($1)`,
		orderID,
	).Scan(&tenantID, &customerEmail)
	if err == pgx.ErrNoRows {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	if err != nil {
		writeServerError(w, "failed to validate order", err)
		return
	}
	if customerEmail == nil || strings.ToLower(*customerEmail) != req.Email {
		writeError(w, http.StatusForbidden, "email does not match order")
		return
	}

	// Ownership is proven; the rest of the creation (row, audit, webhook, automation)
	// is the service's job, so a customer-submitted return behaves like any other.
	ret, err := h.returns.CreatePublic(r.Context(), tenantID, orderID, req)
	if err != nil {
		writeServerError(w, "failed to create return", err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":           ret.ID,
		"status":       ret.Status,
		"return_token": ret.ReturnToken,
		"created_at":   ret.CreatedAt,
	})
}

// findReturnByToken finds a return by token using a SECURITY DEFINER function
// that bypasses RLS (since the token is globally unique and this is a public endpoint).
func (h *PublicReturnHandler) findReturnByToken(ctx context.Context, token string) (*model.Return, error) {
	var r model.Return
	err := h.pool.QueryRow(ctx,
		`SELECT id, tenant_id, order_id, status, reason, items, refund_amount, notes,
		        return_token, customer_email, customer_notes,
		        created_at, updated_at
		 FROM find_return_by_token($1)`, token,
	).Scan(
		&r.ID, &r.TenantID, &r.OrderID, &r.Status, &r.Reason,
		&r.Items, &r.RefundAmount, &r.Notes,
		&r.ReturnToken, &r.CustomerEmail, &r.CustomerNotes,
		&r.CreatedAt, &r.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}
