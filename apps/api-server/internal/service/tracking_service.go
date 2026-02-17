package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	ErrTrackingNotFound = errors.New("order not found")
	ErrTrackingEmail    = errors.New("email does not match order")
)

// TrackingService provides public order tracking lookups.
type TrackingService struct {
	tenantRepo   repository.TenantRepo
	orderRepo    repository.OrderRepo
	shipmentRepo repository.ShipmentRepo
	auditRepo    repository.AuditRepo
	pool         *pgxpool.Pool
}

func NewTrackingService(
	tenantRepo repository.TenantRepo,
	orderRepo repository.OrderRepo,
	shipmentRepo repository.ShipmentRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
) *TrackingService {
	return &TrackingService{
		tenantRepo:   tenantRepo,
		orderRepo:    orderRepo,
		shipmentRepo: shipmentRepo,
		auditRepo:    auditRepo,
		pool:         pool,
	}
}

// TrackOrder looks up an order by tenant slug, order ID, and customer email.
// Returns a curated public-safe response.
func (s *TrackingService) TrackOrder(ctx context.Context, tenantSlug, orderID, email string) (*model.TrackingResponse, error) {
	// 1. Find tenant by slug (uses SECURITY DEFINER, bypasses RLS)
	tenant, err := s.tenantRepo.FindBySlug(ctx, tenantSlug)
	if err != nil {
		return nil, fmt.Errorf("find tenant: %w", err)
	}
	if tenant == nil {
		return nil, ErrTrackingNotFound
	}

	// 2. Parse order ID
	oid, err := uuid.Parse(orderID)
	if err != nil {
		return nil, ErrTrackingNotFound
	}

	var resp model.TrackingResponse

	// 3. Run within tenant context (RLS)
	err = database.WithTenant(ctx, s.pool, tenant.ID, func(tx pgx.Tx) error {
		// Find order by ID
		order, err := s.orderRepo.FindByID(ctx, tx, oid)
		if err != nil {
			return fmt.Errorf("find order: %w", err)
		}
		if order == nil {
			return ErrTrackingNotFound
		}

		// Verify customer email
		if order.CustomerEmail == nil || !strings.EqualFold(strings.TrimSpace(*order.CustomerEmail), strings.TrimSpace(email)) {
			return ErrTrackingEmail
		}

		// Build response
		resp.OrderNumber = order.ID.String()
		resp.Status = order.Status
		resp.CustomerName = order.CustomerName
		resp.CreatedAt = order.CreatedAt
		resp.UpdatedAt = order.UpdatedAt
		resp.TotalAmount = order.TotalAmount
		resp.Currency = order.Currency

		// Resolve status label from tenant settings
		resp.StatusLabel = s.resolveStatusLabel(tenant, order.Status)

		// Parse order items
		resp.Items = s.parseOrderItems(order.Items)

		// Get shipments for this order
		shipFilter := model.ShipmentListFilter{
			OrderID: &order.ID,
		}
		shipFilter.Limit = 20
		shipments, _, err := s.shipmentRepo.List(ctx, tx, shipFilter)
		if err != nil {
			slog.Error("failed to list shipments for tracking", "error", err, "order_id", order.ID)
		} else {
			resp.Shipments = make([]model.TrackingShipment, 0, len(shipments))
			for _, sh := range shipments {
				ts := model.TrackingShipment{
					Carrier: sh.Provider,
					Status:  sh.Status,
				}
				if sh.TrackingNumber != nil {
					ts.TrackingNumber = *sh.TrackingNumber
				}
				resp.Shipments = append(resp.Shipments, ts)
			}
		}

		// Get status change timeline from audit log
		auditEntries, err := s.auditRepo.ListByEntity(ctx, tx, "order", order.ID)
		if err != nil {
			slog.Error("failed to list audit entries for tracking", "error", err, "order_id", order.ID)
		} else {
			resp.Timeline = make([]model.TrackingEvent, 0)
			for _, entry := range auditEntries {
				if entry.Action == "status_change" || entry.Action == "create" {
					status := ""
					if entry.Action == "create" {
						status = "new"
					} else if newStatus, ok := entry.Changes["new_status"]; ok {
						status = newStatus
					} else if statusVal, ok := entry.Changes["status"]; ok {
						status = statusVal
					}
					if status != "" {
						resp.Timeline = append(resp.Timeline, model.TrackingEvent{
							Status:    status,
							Timestamp: entry.CreatedAt,
						})
					}
				}
			}
		}

		// Get company branding from tenant settings
		resp.CompanyName, resp.CompanyLogo = s.extractCompanyBranding(tenant)

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Ensure slices are never nil (JSON: [] not null)
	if resp.Items == nil {
		resp.Items = []model.TrackingItem{}
	}
	if resp.Shipments == nil {
		resp.Shipments = []model.TrackingShipment{}
	}
	if resp.Timeline == nil {
		resp.Timeline = []model.TrackingEvent{}
	}

	return &resp, nil
}

// resolveStatusLabel translates a status key into a human-readable label
// using the tenant's custom order status configuration.
func (s *TrackingService) resolveStatusLabel(tenant *model.Tenant, status string) string {
	cfg := s.getOrderStatusConfig(tenant)
	def := cfg.GetStatusDef(status)
	if def != nil {
		return def.Label
	}
	// Fallback: capitalize and replace underscores
	return strings.ReplaceAll(strings.Title(status), "_", " ") //nolint:staticcheck
}

// getOrderStatusConfig parses order status config from tenant settings.
func (s *TrackingService) getOrderStatusConfig(tenant *model.Tenant) model.OrderStatusConfig {
	if tenant.Settings == nil {
		return model.DefaultOrderStatusConfig()
	}

	var settings map[string]json.RawMessage
	if err := json.Unmarshal(tenant.Settings, &settings); err != nil {
		return model.DefaultOrderStatusConfig()
	}

	raw, ok := settings["order_statuses"]
	if !ok {
		return model.DefaultOrderStatusConfig()
	}

	var cfg model.OrderStatusConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return model.DefaultOrderStatusConfig()
	}
	if len(cfg.Statuses) == 0 {
		return model.DefaultOrderStatusConfig()
	}
	return cfg
}

// parseOrderItems converts the order's JSON items into public-safe TrackingItems.
func (s *TrackingService) parseOrderItems(items json.RawMessage) []model.TrackingItem {
	if items == nil {
		return []model.TrackingItem{}
	}

	// Try to parse as array of objects with name/quantity/price fields
	var rawItems []struct {
		Name     string  `json:"name"`
		Quantity int     `json:"quantity"`
		Price    float64 `json:"price"`
		// Support alternative field names
		ProductName string  `json:"product_name"`
		Qty         int     `json:"qty"`
		UnitPrice   float64 `json:"unit_price"`
	}
	if err := json.Unmarshal(items, &rawItems); err != nil {
		return []model.TrackingItem{}
	}

	result := make([]model.TrackingItem, 0, len(rawItems))
	for _, item := range rawItems {
		ti := model.TrackingItem{
			Name:     item.Name,
			Quantity: item.Quantity,
			Price:    item.Price,
		}
		// Fallback to alternative field names
		if ti.Name == "" {
			ti.Name = item.ProductName
		}
		if ti.Quantity == 0 {
			ti.Quantity = item.Qty
		}
		if ti.Price == 0 {
			ti.Price = item.UnitPrice
		}
		if ti.Name != "" {
			result = append(result, ti)
		}
	}
	return result
}

// extractCompanyBranding reads the company name and logo from tenant settings.
func (s *TrackingService) extractCompanyBranding(tenant *model.Tenant) (name, logo string) {
	if tenant.Settings == nil {
		return tenant.Name, ""
	}

	var settings map[string]json.RawMessage
	if err := json.Unmarshal(tenant.Settings, &settings); err != nil {
		return tenant.Name, ""
	}

	companyRaw, ok := settings["company"]
	if !ok {
		return tenant.Name, ""
	}

	var company struct {
		Name string `json:"name"`
		Logo string `json:"logo"`
	}
	if err := json.Unmarshal(companyRaw, &company); err != nil {
		return tenant.Name, ""
	}

	if company.Name == "" {
		company.Name = tenant.Name
	}
	return company.Name, company.Logo
}
