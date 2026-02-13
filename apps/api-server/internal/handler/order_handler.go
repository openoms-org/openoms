package handler

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

type OrderHandler struct {
	orderService *service.OrderService
	tenantRepo   repository.TenantRepo
	pool         *pgxpool.Pool
}

func NewOrderHandler(orderService *service.OrderService, tenantRepo repository.TenantRepo, pool *pgxpool.Pool) *OrderHandler {
	return &OrderHandler{orderService: orderService, tenantRepo: tenantRepo, pool: pool}
}

// parseOrderFilter extracts common order list filter parameters from the request.
func parseOrderFilter(r *http.Request) model.OrderListFilter {
	filter := model.OrderListFilter{}
	if s := r.URL.Query().Get("status"); s != "" {
		filter.Status = &s
	}
	if s := r.URL.Query().Get("source"); s != "" {
		filter.Source = &s
	}
	if s := r.URL.Query().Get("search"); s != "" {
		filter.Search = &s
	}
	if ps := r.URL.Query().Get("payment_status"); ps != "" {
		filter.PaymentStatus = &ps
	}
	if t := r.URL.Query().Get("tag"); t != "" {
		filter.Tag = &t
	}
	if p := r.URL.Query().Get("priority"); p != "" {
		filter.Priority = &p
	}
	return filter
}

func (h *OrderHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := parseOrderFilter(r)
	filter.PaginationParams = pagination

	resp, err := h.orderService.List(r.Context(), tenantID, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list orders")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *OrderHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	order, err := h.orderService.Get(r.Context(), tenantID, orderID)
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get order")
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	order, err := h.orderService.Create(r.Context(), tenantID, req, actorID, clientIP(r))
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, "failed to create order")
		}
		return
	}
	writeJSON(w, http.StatusCreated, order)
}

func (h *OrderHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	var req model.UpdateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	order, err := h.orderService.Update(r.Context(), tenantID, orderID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrderNotFound):
			writeError(w, http.StatusNotFound, "order not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "failed to update order")
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (h *OrderHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	err = h.orderService.Delete(r.Context(), tenantID, orderID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrderNotFound):
			writeError(w, http.StatusNotFound, "order not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to delete order")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *OrderHandler) TransitionStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	var req model.StatusTransitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	order, err := h.orderService.TransitionStatus(r.Context(), tenantID, orderID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrderNotFound):
			writeError(w, http.StatusNotFound, "order not found")
		case errors.Is(err, service.ErrInvalidTransition), errors.Is(err, service.ErrUnknownStatus):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "failed to transition order status")
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (h *OrderHandler) BulkTransitionStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.BulkStatusTransitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.orderService.BulkTransitionStatus(r.Context(), tenantID, req, actorID, clientIP(r))
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		slog.Error("bulk status transition failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to perform bulk status transition")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *OrderHandler) GetAudit(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	entries, err := h.orderService.GetAudit(r.Context(), tenantID, orderID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to retrieve audit log")
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (h *OrderHandler) DuplicateOrder(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	var newOrder *model.Order
	err = database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := h.orderService.OrderRepo().FindByID(r.Context(), tx, orderID)
		if err != nil {
			return err
		}
		if existing == nil {
			return errors.New("order not found")
		}

		newOrder = &model.Order{
			ID:              uuid.New(),
			TenantID:        existing.TenantID,
			ExternalID:      existing.ExternalID,
			Source:          existing.Source,
			IntegrationID:   existing.IntegrationID,
			Status:          "new",
			CustomerName:    existing.CustomerName,
			CustomerEmail:   existing.CustomerEmail,
			CustomerPhone:   existing.CustomerPhone,
			ShippingAddress: existing.ShippingAddress,
			BillingAddress:  existing.BillingAddress,
			Items:           existing.Items,
			TotalAmount:     existing.TotalAmount,
			Currency:        existing.Currency,
			Notes:           existing.Notes,
			Metadata:        existing.Metadata,
			Tags:            existing.Tags,
			OrderedAt:       existing.OrderedAt,
			DeliveryMethod:  existing.DeliveryMethod,
			PickupPointID:   existing.PickupPointID,
			PaymentStatus:   existing.PaymentStatus,
			PaymentMethod:   existing.PaymentMethod,
			CustomerID:      existing.CustomerID,
			InternalNotes:   existing.InternalNotes,
			Priority:        existing.Priority,
		}

		if err := h.orderService.OrderRepo().Create(r.Context(), tx, newOrder); err != nil {
			return fmt.Errorf("create duplicated order: %w", err)
		}

		return h.orderService.AuditRepo().Log(r.Context(), tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "order.duplicated",
			EntityType: "order",
			EntityID:   newOrder.ID,
			Changes:    map[string]string{"source_order_id": orderID.String()},
			IPAddress:  clientIP(r),
		})
	})
	if err != nil {
		if err.Error() == "order not found" {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		slog.Error("failed to duplicate order", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to duplicate order")
		return
	}

	writeJSON(w, http.StatusCreated, newOrder)
}

func (h *OrderHandler) loadCustomFieldsConfig(ctx context.Context, tenantID uuid.UUID) model.CustomFieldsConfig {
	var config model.CustomFieldsConfig
	if err := database.WithTenant(ctx, h.pool, tenantID, func(tx pgx.Tx) error {
		settings, err := h.tenantRepo.GetSettings(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		if settings != nil {
			var allSettings map[string]json.RawMessage
			if err := json.Unmarshal(settings, &allSettings); err == nil {
				if raw, ok := allSettings["custom_fields"]; ok {
					if err := json.Unmarshal(raw, &config); err != nil {
						slog.Warn("failed to unmarshal custom_fields config", "error", err, "tenant_id", tenantID)
					}
				}
			}
		}
		return nil
	}); err != nil {
		slog.Error("failed to load custom fields config", "error", err, "tenant_id", tenantID)
	}
	if config.Fields == nil {
		config.Fields = []model.CustomFieldDef{}
	}
	return config
}

func (h *OrderHandler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	filter := parseOrderFilter(r)

	// Load custom field definitions
	cfConfig := h.loadCustomFieldsConfig(r.Context(), tenantID)

	filename := fmt.Sprintf("zamowienia-%s.csv", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	// BOM for Excel UTF-8 compatibility
	w.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(w)
	defer writer.Flush()

	header := []string{
		"ID", "Klient", "Email", "Telefon", "Źródło", "Status",
		"Status płatności", "Metoda płatności", "Kwota", "Waluta",
		"Data zamówienia", "Data opłacenia", "Tagi",
	}
	for _, f := range cfConfig.Fields {
		header = append(header, f.Label)
	}
	if err := writer.Write(header); err != nil {
		slog.Error("csv export: failed to write header", "error", err)
		return
	}

	const batchSize = 500
	offset := 0

	for {
		filter.PaginationParams = model.PaginationParams{Limit: batchSize, Offset: offset}
		resp, err := h.orderService.List(r.Context(), tenantID, filter)
		if err != nil {
			slog.Error("csv export failed", "error", err, "offset", offset)
			break
		}

		if len(resp.Items) == 0 {
			break
		}

		for _, o := range resp.Items {
			email := ""
			if o.CustomerEmail != nil {
				email = *o.CustomerEmail
			}
			phone := ""
			if o.CustomerPhone != nil {
				phone = *o.CustomerPhone
			}
			method := ""
			if o.PaymentMethod != nil {
				method = *o.PaymentMethod
			}
			orderedAt := ""
			if o.OrderedAt != nil {
				orderedAt = o.OrderedAt.Format("2006-01-02 15:04")
			}
			paidAt := ""
			if o.PaidAt != nil {
				paidAt = o.PaidAt.Format("2006-01-02 15:04")
			}

			row := []string{
				o.ID.String(),
				o.CustomerName,
				email,
				phone,
				o.Source,
				o.Status,
				o.PaymentStatus,
				method,
				fmt.Sprintf("%.2f", o.TotalAmount),
				o.Currency,
				orderedAt,
				paidAt,
				strings.Join(o.Tags, ";"),
			}

			// Parse order metadata and append custom field values
			var metadata map[string]any
			if o.Metadata != nil {
				if err := json.Unmarshal(o.Metadata, &metadata); err != nil {
					slog.Warn("csv export: failed to unmarshal order metadata", "error", err, "order_id", o.ID)
				}
			}
			for _, f := range cfConfig.Fields {
				val := ""
				if metadata != nil {
					if v, ok := metadata[f.Key]; ok && v != nil {
						switch f.Type {
						case "checkbox":
							if b, ok := v.(bool); ok && b {
								val = "Tak"
							} else {
								val = "Nie"
							}
						default:
							val = fmt.Sprintf("%v", v)
						}
					}
				}
				row = append(row, val)
			}

			if err := writer.Write(row); err != nil {
				slog.Error("csv export: failed to write row", "error", err)
				return
			}
		}

		writer.Flush()
		if err := writer.Error(); err != nil {
			slog.Error("csv export: flush error", "error", err)
			return
		}

		if len(resp.Items) < batchSize {
			break
		}
		offset += batchSize
	}
}
