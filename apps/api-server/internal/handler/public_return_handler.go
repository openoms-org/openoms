package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// PublicReturnHandler handles public (no auth) return endpoints.
type PublicReturnHandler struct {
	pool       *pgxpool.Pool
	returnRepo repository.ReturnRepo
	orderRepo  repository.OrderRepo
}

func NewPublicReturnHandler(pool *pgxpool.Pool, returnRepo repository.ReturnRepo, orderRepo repository.OrderRepo) *PublicReturnHandler {
	return &PublicReturnHandler{
		pool:       pool,
		returnRepo: returnRepo,
		orderRepo:  orderRepo,
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
		writeError(w, http.StatusInternalServerError, "failed to find return")
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
		writeError(w, http.StatusInternalServerError, "failed to find return")
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
		writeError(w, http.StatusInternalServerError, "failed to validate order")
		return
	}
	if customerEmail == nil || strings.ToLower(*customerEmail) != req.Email {
		writeError(w, http.StatusForbidden, "email does not match order")
		return
	}

	// Generate random return token
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}
	returnToken := hex.EncodeToString(tokenBytes)

	items := req.Items
	if items == nil {
		items = json.RawMessage("[]")
	}

	var notes *string
	if req.Notes != "" {
		notes = &req.Notes
	}

	ret := &model.Return{
		ID:            uuid.New(),
		TenantID:      tenantID,
		OrderID:       orderID,
		Status:        "requested",
		Reason:        req.Reason,
		Items:         items,
		RefundAmount:  0,
		Notes:         notes,
		ReturnToken:   &returnToken,
		CustomerEmail: &req.Email,
		CustomerNotes: notes,
	}

	// Create the return with tenant context
	err = h.withTenant(r.Context(), tenantID, func(tx pgx.Tx) error {
		return h.returnRepo.Create(r.Context(), tx, ret)
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create return")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":           ret.ID,
		"status":       ret.Status,
		"return_token": returnToken,
		"created_at":   ret.CreatedAt,
	})
}

// withTenant runs a function in a transaction with RLS set to the given tenant.
func (h *PublicReturnHandler) withTenant(ctx context.Context, tenantID uuid.UUID, fn func(tx pgx.Tx) error) error {
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx,
		"SELECT set_config('app.current_tenant_id', $1, true)",
		tenantID.String(),
	); err != nil {
		return fmt.Errorf("set tenant context: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
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
