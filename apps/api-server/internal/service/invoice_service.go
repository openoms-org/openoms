package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/asyncutil"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	// ErrInvoiceNotFound is returned when an invoice does not exist.
	ErrInvoiceNotFound = errors.New("invoice not found")
)

// InvoicingSettings represents the invoicing section of tenant settings.
type InvoicingSettings struct {
	Provider           string   `json:"provider"`
	AutoCreateOnStatus []string `json:"auto_create_on_status"`
	DefaultTaxRate     int      `json:"default_tax_rate"`
	PaymentDays        int      `json:"payment_days"`
}

// InvoiceService handles business logic for invoicing.
type InvoiceService struct {
	invoiceRepo repository.InvoiceRepo
	orderRepo   repository.OrderRepo
	tenantRepo  repository.TenantRepo
	auditRepo   repository.AuditRepo
	pool        *pgxpool.Pool
	encKey      []byte
	ksefService *KSeFService
}

// SetKSeFService sets the KSeF service for automatic sending on invoice creation.
func (s *InvoiceService) SetKSeFService(ksefSvc *KSeFService) {
	s.ksefService = ksefSvc
}

// NewInvoiceService creates a new InvoiceService.
func NewInvoiceService(
	invoiceRepo repository.InvoiceRepo,
	orderRepo repository.OrderRepo,
	tenantRepo repository.TenantRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
	encKey []byte,
) *InvoiceService {
	return &InvoiceService{
		invoiceRepo: invoiceRepo,
		orderRepo:   orderRepo,
		tenantRepo:  tenantRepo,
		auditRepo:   auditRepo,
		pool:        pool,
		encKey:      encKey,
	}
}

// List returns a paginated list of invoices for a tenant.
func (s *InvoiceService) List(ctx context.Context, tenantID uuid.UUID, filter model.InvoiceListFilter) (model.ListResponse[model.Invoice], error) {
	var resp model.ListResponse[model.Invoice]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		invoices, total, err := s.invoiceRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}
		resp = model.NewListResponse(invoices, total, filter.Limit, filter.Offset)
		return nil
	})
	return resp, err
}

// Get returns a single invoice by ID.
func (s *InvoiceService) Get(ctx context.Context, tenantID, invoiceID uuid.UUID) (*model.Invoice, error) {
	var inv *model.Invoice
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		inv, err = s.invoiceRepo.FindByID(ctx, tx, invoiceID)
		return err
	})
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, ErrInvoiceNotFound
	}
	return inv, nil
}

// ListByOrderID returns all invoices linked to a specific order.
func (s *InvoiceService) ListByOrderID(ctx context.Context, tenantID, orderID uuid.UUID) ([]model.Invoice, error) {
	var invoices []model.Invoice
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		invoices, err = s.invoiceRepo.FindByOrderID(ctx, tx, orderID)
		return err
	})
	if invoices == nil {
		invoices = []model.Invoice{}
	}
	return invoices, err
}

// Create generates a new invoice, calling the configured invoicing provider.
func (s *InvoiceService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateInvoiceRequest, actorID uuid.UUID, ip string) (*model.Invoice, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var inv *model.Invoice
	var savedProviderName string
	var savedItems []integration.InvoiceItem
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Verify order exists
		order, err := s.orderRepo.FindByID(ctx, tx, req.OrderID)
		if err != nil {
			return err
		}
		if order == nil {
			return NewValidationError(errors.New("order not found"))
		}

		// Load invoicing settings for provider credentials
		invoicingCfg, err := s.loadInvoicingSettings(ctx, tx, tenantID)
		if err != nil {
			return fmt.Errorf("load invoicing settings: %w", err)
		}

		providerName := req.Provider
		if providerName == "" && invoicingCfg != nil {
			providerName = invoicingCfg.Provider
		}
		if providerName == "" {
			return NewValidationError(errors.New("no invoicing provider configured"))
		}

		now := time.Now()
		issueDate := now
		taxRate := 23
		paymentDays := 14
		if invoicingCfg != nil {
			if invoicingCfg.DefaultTaxRate > 0 {
				taxRate = invoicingCfg.DefaultTaxRate
			}
			if invoicingCfg.PaymentDays > 0 {
				paymentDays = invoicingCfg.PaymentDays
			}
		}
		dueDate := issueDate.AddDate(0, 0, paymentDays)

		// Build items from order
		items := s.buildInvoiceItems(order, taxRate)
		totalNet := order.TotalAmount / (1 + float64(taxRate)/100)
		totalGross := order.TotalAmount

		invoice := &model.Invoice{
			ID:          uuid.New(),
			TenantID:    tenantID,
			OrderID:     req.OrderID,
			Provider:    providerName,
			Status:      "draft",
			InvoiceType: req.InvoiceType,
			TotalNet:    &totalNet,
			TotalGross:  &totalGross,
			Currency:    order.Currency,
			IssueDate:   &issueDate,
			DueDate:     &dueDate,
			Metadata:    json.RawMessage("{}"),
		}

		if err := s.invoiceRepo.Create(ctx, tx, invoice); err != nil {
			return err
		}

		inv = invoice
		savedProviderName = providerName
		savedItems = items

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "invoice.created",
			EntityType: "invoice",
			EntityID:   invoice.ID,
			Changes:    map[string]string{"order_id": req.OrderID.String(), "provider": providerName},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return inv, err
	}

	// Phase 2: Call invoicing provider OUTSIDE the main transaction.
	// This prevents holding a DB connection during a potentially slow HTTP call.
	var provider integration.InvoicingProvider
	var provErr error
	_ = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		provider, provErr = s.getProvider(ctx, tx, tenantID, savedProviderName)
		return nil
	})
	if provErr == nil {
		customerName := req.CustomerName
		if customerName == "" && inv != nil {
			customerName = req.CustomerName
		}

		invoiceReq := integration.InvoiceRequest{
			OrderID:       req.OrderID.String(),
			CustomerName:  customerName,
			CustomerEmail: req.CustomerEmail,
			NIP:           req.NIP,
			Items:         savedItems,
			TotalNet:      *inv.TotalNet,
			TotalGross:    *inv.TotalGross,
			Currency:      inv.Currency,
			IssueDate:     *inv.IssueDate,
			DueDate:       *inv.DueDate,
			PaymentMethod: req.PaymentMethod,
			Notes:         req.Notes,
		}

		result, createErr := provider.CreateInvoice(ctx, invoiceReq)

		// Phase 3: Update invoice with provider result in a new short transaction.
		if updateErr := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
			if createErr != nil {
				errMsg := createErr.Error()
				inv.ErrorMessage = &errMsg
				inv.Status = "error"
			} else {
				inv.ExternalID = &result.ExternalID
				inv.ExternalNumber = &result.ExternalNumber
				inv.PDFURL = &result.PDFURL
				inv.Status = "issued"
			}
			return s.invoiceRepo.Update(ctx, tx, inv)
		}); updateErr != nil {
			slog.Error("failed to update invoice after provider call", "invoice_id", inv.ID, "error", updateErr)
		}
	} else {
		// Provider creation failed — mark invoice as error
		if updateErr := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
			errMsg := provErr.Error()
			inv.ErrorMessage = &errMsg
			inv.Status = "error"
			return s.invoiceRepo.Update(ctx, tx, inv)
		}); updateErr != nil {
			slog.Error("failed to mark invoice as error", "invoice_id", inv.ID, "error", updateErr)
		}
	}

	// Trigger KSeF auto-send after successful creation
	if inv != nil && inv.Status == "issued" {
		s.triggerKSeFAutoSend(tenantID, inv.ID)
	}

	return inv, nil
}

// Cancel voids an existing invoice via the invoicing provider.
func (s *InvoiceService) Cancel(ctx context.Context, tenantID, invoiceID, actorID uuid.UUID, ip string) error {
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		inv, err := s.invoiceRepo.FindByID(ctx, tx, invoiceID)
		if err != nil {
			return err
		}
		if inv == nil {
			return ErrInvoiceNotFound
		}

		// Try cancelling with provider
		if inv.ExternalID != nil && *inv.ExternalID != "" {
			provider, provErr := s.getProvider(ctx, tx, tenantID, inv.Provider)
			if provErr == nil {
				if cancelErr := provider.CancelInvoice(ctx, *inv.ExternalID); cancelErr != nil {
					slog.Error("failed to cancel invoice with provider", "invoice_id", invoiceID, "error", cancelErr)
				}
			}
		}

		inv.Status = "cancelled"
		if err := s.invoiceRepo.Update(ctx, tx, inv); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "invoice.cancelled",
			EntityType: "invoice",
			EntityID:   invoiceID,
			IPAddress:  ip,
		})
	})
	return err
}

// GetPDF retrieves the PDF binary for an invoice.
func (s *InvoiceService) GetPDF(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]byte, error) {
	var pdfData []byte
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		inv, err := s.invoiceRepo.FindByID(ctx, tx, invoiceID)
		if err != nil {
			return err
		}
		if inv == nil {
			return ErrInvoiceNotFound
		}
		if inv.ExternalID == nil || *inv.ExternalID == "" {
			return errors.New("invoice has no external ID")
		}

		provider, err := s.getProvider(ctx, tx, tenantID, inv.Provider)
		if err != nil {
			return fmt.Errorf("get provider: %w", err)
		}

		pdfData, err = provider.GetPDF(ctx, *inv.ExternalID)
		return err
	})
	return pdfData, err
}

// HandleOrderStatusChange is called when an order's status changes. If auto-invoicing
// is configured for this status, it creates an invoice automatically.
func (s *InvoiceService) HandleOrderStatusChange(ctx context.Context, tenantID uuid.UUID, order *model.Order) {
	var createdInvoice *model.Invoice
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		invoicingCfg, err := s.loadInvoicingSettings(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		if invoicingCfg == nil || invoicingCfg.Provider == "" {
			return nil // No invoicing configured
		}

		// Check if auto-create is enabled for this status
		shouldCreate := slices.Contains(invoicingCfg.AutoCreateOnStatus, order.Status)
		if !shouldCreate {
			return nil
		}

		// Check if an invoice already exists for this order
		existing, err := s.invoiceRepo.FindByOrderID(ctx, tx, order.ID)
		if err != nil {
			return err
		}
		// Only auto-create if there are no existing non-cancelled invoices
		for _, inv := range existing {
			if inv.Status != "cancelled" && inv.Status != "error" {
				return nil // Invoice already exists
			}
		}

		now := time.Now()
		taxRate := 23
		paymentDays := 14
		if invoicingCfg.DefaultTaxRate > 0 {
			taxRate = invoicingCfg.DefaultTaxRate
		}
		if invoicingCfg.PaymentDays > 0 {
			paymentDays = invoicingCfg.PaymentDays
		}
		dueDate := now.AddDate(0, 0, paymentDays)

		items := s.buildInvoiceItems(order, taxRate)
		totalNet := order.TotalAmount / (1 + float64(taxRate)/100)
		totalGross := order.TotalAmount

		invoice := &model.Invoice{
			ID:          uuid.New(),
			TenantID:    tenantID,
			OrderID:     order.ID,
			Provider:    invoicingCfg.Provider,
			Status:      "draft",
			InvoiceType: "vat",
			TotalNet:    &totalNet,
			TotalGross:  &totalGross,
			Currency:    order.Currency,
			IssueDate:   &now,
			DueDate:     &dueDate,
			Metadata:    json.RawMessage("{}"),
		}

		if err := s.invoiceRepo.Create(ctx, tx, invoice); err != nil {
			return err
		}

		provider, provErr := s.getProvider(ctx, tx, tenantID, invoicingCfg.Provider)
		if provErr != nil {
			errMsg := provErr.Error()
			invoice.ErrorMessage = &errMsg
			invoice.Status = "error"
			_ = s.invoiceRepo.Update(ctx, tx, invoice)
			return nil //nolint:nilerr // provider error recorded in invoice.ErrorMessage; returning nil commits the invoice with error status
		}

		customerEmail := ""
		if order.CustomerEmail != nil {
			customerEmail = *order.CustomerEmail
		}

		invoiceReq := integration.InvoiceRequest{
			OrderID:       order.ID.String(),
			CustomerName:  order.CustomerName,
			CustomerEmail: customerEmail,
			Items:         items,
			TotalNet:      totalNet,
			TotalGross:    totalGross,
			Currency:      order.Currency,
			IssueDate:     now,
			DueDate:       dueDate,
		}

		result, createErr := provider.CreateInvoice(ctx, invoiceReq)
		if createErr != nil {
			errMsg := createErr.Error()
			invoice.ErrorMessage = &errMsg
			invoice.Status = "error"
		} else {
			invoice.ExternalID = &result.ExternalID
			invoice.ExternalNumber = &result.ExternalNumber
			invoice.PDFURL = &result.PDFURL
			invoice.Status = "issued"
		}
		_ = s.invoiceRepo.Update(ctx, tx, invoice)

		createdInvoice = invoice

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     uuid.Nil,
			Action:     "invoice.auto_created",
			EntityType: "invoice",
			EntityID:   invoice.ID,
			Changes:    map[string]string{"order_id": order.ID.String(), "trigger": order.Status},
		})
	})
	if err != nil {
		slog.Error("auto-invoice failed", "tenant_id", tenantID, "order_id", order.ID, "error", err)
		return
	}

	// Trigger KSeF auto-send after successful auto-creation (outside the transaction)
	if createdInvoice != nil && createdInvoice.Status == "issued" {
		s.triggerKSeFAutoSend(tenantID, createdInvoice.ID)
	}
}

// triggerKSeFAutoSend sends the invoice to KSeF in a background goroutine.
// The settings check is performed inside the goroutine to avoid synchronous DB calls on the hot path.
func (s *InvoiceService) triggerKSeFAutoSend(tenantID, invoiceID uuid.UUID) {
	if s.ksefService == nil {
		return
	}

	asyncutil.SafeGo(func() {
		bgCtx := context.Background()
		cfg, err := s.ksefService.GetSettings(bgCtx, tenantID)
		if err != nil || cfg == nil || !cfg.AutoSend || !cfg.Enabled {
			return
		}
		if err := s.ksefService.SendToKSeF(bgCtx, tenantID, invoiceID, uuid.Nil, "system"); err != nil {
			slog.Error("ksef auto-send failed", "invoice_id", invoiceID, "error", err)
		}
	})
}

// loadInvoicingSettings reads the "invoicing" section from tenant settings.
func (s *InvoiceService) loadInvoicingSettings(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) (*InvoicingSettings, error) {
	settings, err := s.tenantRepo.GetSettings(ctx, tx, tenantID)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		return nil, nil
	}

	var allSettings map[string]json.RawMessage
	if err := json.Unmarshal(settings, &allSettings); err != nil {
		return nil, err
	}

	raw, ok := allSettings["invoicing"]
	if !ok {
		return nil, nil
	}

	var cfg InvoicingSettings
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// getProvider loads integration credentials and creates an invoicing provider.
func (s *InvoiceService) getProvider(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, providerName string) (integration.InvoicingProvider, error) {
	// Load invoicing credentials from tenant settings
	settings, err := s.tenantRepo.GetSettings(ctx, tx, tenantID)
	if err != nil {
		return nil, err
	}

	if settings == nil {
		return nil, fmt.Errorf("no settings found for tenant")
	}

	var allSettings map[string]json.RawMessage
	if err := json.Unmarshal(settings, &allSettings); err != nil {
		return nil, fmt.Errorf("invalid tenant settings: %w", err)
	}

	raw, ok := allSettings["invoicing"]
	if !ok {
		return nil, fmt.Errorf("no invoicing settings found")
	}

	// The credentials are embedded in the invoicing settings
	var cfgMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &cfgMap); err != nil {
		return nil, fmt.Errorf("invalid invoicing settings: %w", err)
	}

	credsRaw, ok := cfgMap["credentials"]
	if !ok {
		return nil, fmt.Errorf("no invoicing credentials found")
	}

	return integration.NewInvoicingProvider(providerName, credsRaw, raw)
}

// buildInvoiceItems extracts order items and builds invoice line items.
func (s *InvoiceService) buildInvoiceItems(order *model.Order, taxRate int) []integration.InvoiceItem {
	if order.Items == nil || string(order.Items) == "[]" || string(order.Items) == "null" {
		// Fallback: single line item from order total
		return fallbackInvoiceItems(order, taxRate)
	}

	type orderItem struct {
		Name     string  `json:"name"`
		SKU      string  `json:"sku"`
		Quantity int     `json:"quantity"`
		Price    float64 `json:"price"`
	}

	var orderItems []orderItem
	if err := json.Unmarshal(order.Items, &orderItems); err != nil {
		return fallbackInvoiceItems(order, taxRate)
	}

	items := make([]integration.InvoiceItem, 0, len(orderItems))
	for _, oi := range orderItems {
		qty := oi.Quantity
		if qty <= 0 {
			qty = 1
		}
		netPrice := oi.Price / (1 + float64(taxRate)/100)
		items = append(items, integration.InvoiceItem{
			Name:     oi.Name,
			Quantity: qty,
			NetPrice: netPrice,
			TaxRate:  taxRate,
			Unit:     "szt.",
		})
	}

	if len(items) == 0 {
		return fallbackInvoiceItems(order, taxRate)
	}

	return items
}

// fallbackInvoiceItems returns a single invoice line item derived from the order
// total, used when the order has no parseable line items.
func fallbackInvoiceItems(order *model.Order, taxRate int) []integration.InvoiceItem {
	totalNet := order.TotalAmount / (1 + float64(taxRate)/100)
	return []integration.InvoiceItem{
		{
			Name:     "Order " + order.ID.String()[:8],
			Quantity: 1,
			NetPrice: totalNet,
			TaxRate:  taxRate,
			Unit:     "szt.",
		},
	}
}
