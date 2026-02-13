package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	ksef "github.com/openoms-org/openoms/packages/ksef-go-sdk"
)

var (
	ErrKSeFNotConfigured = errors.New("KSeF is not configured for this tenant")
	ErrKSeFAlreadySent   = errors.New("invoice has already been sent to KSeF")
)

// KSeFSettings holds the KSeF configuration from tenant settings.
type KSeFSettings struct {
	Enabled     bool   `json:"enabled"`
	Environment string `json:"environment"` // "test" or "production"
	NIP         string `json:"nip"`
	Token       string `json:"token"`
	// Company details for XML generation
	CompanyName    string `json:"company_name"`
	CompanyStreet  string `json:"company_street"`
	CompanyCity    string `json:"company_city"`
	CompanyPostal  string `json:"company_postal"`
	CompanyCountry string `json:"company_country"`
}

// KSeFTestResult holds the result of a KSeF connection test.
type KSeFTestResult struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp,omitempty"`
	Challenge string `json:"challenge,omitempty"`
}

// KSeFService handles KSeF (Krajowy System e-Faktur) operations.
type KSeFService struct {
	invoiceRepo repository.InvoiceRepo
	orderRepo   repository.OrderRepo
	tenantRepo  repository.TenantRepo
	auditRepo   repository.AuditRepo
	pool        *pgxpool.Pool
}

// NewKSeFService creates a new KSeF service.
func NewKSeFService(
	invoiceRepo repository.InvoiceRepo,
	orderRepo repository.OrderRepo,
	tenantRepo repository.TenantRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
) *KSeFService {
	return &KSeFService{
		invoiceRepo: invoiceRepo,
		orderRepo:   orderRepo,
		tenantRepo:  tenantRepo,
		auditRepo:   auditRepo,
		pool:        pool,
	}
}

// GetSettings loads the KSeF settings for a tenant.
func (s *KSeFService) GetSettings(ctx context.Context, tenantID uuid.UUID) (*KSeFSettings, error) {
	var cfg KSeFSettings
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var loadErr error
		cfg, loadErr = s.loadKSeFSettings(ctx, tx, tenantID)
		return loadErr
	})
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// UpdateSettings saves the KSeF settings for a tenant.
func (s *KSeFService) UpdateSettings(ctx context.Context, tenantID uuid.UUID, cfg KSeFSettings, actorID uuid.UUID, ip string) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.saveKSeFSettings(ctx, tx, tenantID, cfg); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "settings.ksef_updated",
			EntityType: "settings",
			EntityID:   tenantID,
			IPAddress:  ip,
		})
	})
}

// TestConnection tests the KSeF API connection using the configured credentials.
func (s *KSeFService) TestConnection(ctx context.Context, tenantID uuid.UUID) (*KSeFTestResult, error) {
	var cfg KSeFSettings
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var loadErr error
		cfg, loadErr = s.loadKSeFSettings(ctx, tx, tenantID)
		return loadErr
	})
	if err != nil {
		return nil, err
	}

	if !cfg.Enabled || cfg.NIP == "" || cfg.Token == "" {
		return &KSeFTestResult{
			Success: false,
			Message: "KSeF nie jest skonfigurowany. Uzupełnij NIP i token.",
		}, nil
	}

	client := s.createClient(cfg)

	// Test by requesting an authorisation challenge
	resp, err := client.Session.AuthorisationChallenge(ctx, cfg.NIP)
	if err != nil {
		return &KSeFTestResult{
			Success: false,
			Message: fmt.Sprintf("Błąd połączenia z KSeF: %v", err),
		}, nil
	}

	return &KSeFTestResult{
		Success:   true,
		Message:   "Połączenie z KSeF działa poprawnie.",
		Timestamp: resp.Timestamp,
		Challenge: resp.Challenge,
	}, nil
}

// SendToKSeF sends a single invoice to KSeF.
func (s *KSeFService) SendToKSeF(ctx context.Context, tenantID, invoiceID, actorID uuid.UUID, ip string) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		inv, err := s.invoiceRepo.FindByID(ctx, tx, invoiceID)
		if err != nil {
			return err
		}
		if inv == nil {
			return ErrInvoiceNotFound
		}

		if inv.KSeFStatus != "not_sent" {
			return ErrKSeFAlreadySent
		}

		cfg, err := s.loadKSeFSettings(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		if !cfg.Enabled {
			return ErrKSeFNotConfigured
		}

		// Load order for buyer details
		order, err := s.orderRepo.FindByID(ctx, tx, inv.OrderID)
		if err != nil {
			return fmt.Errorf("load order: %w", err)
		}

		// Build the invoice XML
		invoiceData := s.buildInvoiceData(inv, order, cfg)
		xmlBytes, err := ksef.BuildInvoiceXML(invoiceData)
		if err != nil {
			return fmt.Errorf("build invoice XML: %w", err)
		}

		// Initialize session, send invoice, terminate session
		client := s.createClient(cfg)
		now := time.Now()

		// Step 1: Get authorisation challenge
		challenge, err := client.Session.AuthorisationChallenge(ctx, cfg.NIP)
		if err != nil {
			return s.markKSeFError(ctx, tx, inv, fmt.Errorf("authorisation challenge: %w", err))
		}

		// Step 2: Init session with token
		session, err := client.Session.InitToken(ctx, cfg.NIP, cfg.Token, challenge.Challenge)
		if err != nil {
			return s.markKSeFError(ctx, tx, inv, fmt.Errorf("init session: %w", err))
		}

		// Step 3: Send the invoice
		sendResp, err := client.Invoice.Send(ctx, session.SessionToken.Token, xmlBytes)
		if err != nil {
			// Try to terminate session even if send failed
			_, _ = client.Session.Terminate(ctx, session.SessionToken.Token)
			return s.markKSeFError(ctx, tx, inv, fmt.Errorf("send invoice: %w", err))
		}

		// Step 4: Terminate session
		_, _ = client.Session.Terminate(ctx, session.SessionToken.Token)

		// Update invoice with KSeF tracking info
		inv.KSeFStatus = "pending"
		inv.KSeFSentAt = &now

		responseJSON, _ := json.Marshal(map[string]any{
			"element_reference_number": sendResp.ElementReferenceNumber,
			"reference_number":         sendResp.ReferenceNumber,
			"processing_code":          sendResp.ProcessingCode,
			"timestamp":                sendResp.Timestamp,
		})
		inv.KSeFResponse = responseJSON

		if err := s.invoiceRepo.Update(ctx, tx, inv); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "invoice.ksef_sent",
			EntityType: "invoice",
			EntityID:   invoiceID,
			Changes:    map[string]string{"reference": sendResp.ReferenceNumber},
			IPAddress:  ip,
		})
	})
}

// CheckKSeFStatus checks the KSeF status of a submitted invoice.
func (s *KSeFService) CheckKSeFStatus(ctx context.Context, tenantID, invoiceID uuid.UUID) (*model.Invoice, error) {
	var inv *model.Invoice
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		inv, err = s.invoiceRepo.FindByID(ctx, tx, invoiceID)
		if err != nil {
			return err
		}
		if inv == nil {
			return ErrInvoiceNotFound
		}

		if inv.KSeFStatus != "pending" {
			return nil // Nothing to check
		}

		cfg, err := s.loadKSeFSettings(ctx, tx, tenantID)
		if err != nil || !cfg.Enabled {
			return nil
		}

		// Extract reference number from response
		var respData map[string]any
		if inv.KSeFResponse != nil {
			_ = json.Unmarshal(inv.KSeFResponse, &respData)
		}
		refNum, _ := respData["reference_number"].(string)
		if refNum == "" {
			return nil
		}

		client := s.createClient(cfg)
		upo, err := client.Invoice.GetUPO(ctx, refNum)
		if err != nil {
			slog.Warn("ksef: failed to check status", "invoice_id", invoiceID, "error", err)
			return nil // Don't fail, just leave as pending
		}

		if upo.ProcessingCode == 200 {
			inv.KSeFStatus = "accepted"
			inv.KSeFNumber = &upo.ReferenceNumber
			responseJSON, _ := json.Marshal(map[string]any{
				"reference_number":       upo.ReferenceNumber,
				"processing_code":        upo.ProcessingCode,
				"processing_description": upo.ProcessingDescription,
				"timestamp":              upo.Timestamp,
			})
			inv.KSeFResponse = responseJSON
			return s.invoiceRepo.Update(ctx, tx, inv)
		} else if upo.ProcessingCode >= 400 {
			inv.KSeFStatus = "rejected"
			responseJSON, _ := json.Marshal(map[string]any{
				"reference_number":       upo.ReferenceNumber,
				"processing_code":        upo.ProcessingCode,
				"processing_description": upo.ProcessingDescription,
				"timestamp":              upo.Timestamp,
			})
			inv.KSeFResponse = responseJSON
			return s.invoiceRepo.Update(ctx, tx, inv)
		}

		return nil
	})
	return inv, err
}

// GetUPO downloads the UPO for an invoice.
func (s *KSeFService) GetUPO(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]byte, error) {
	var upoBytes []byte
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		inv, err := s.invoiceRepo.FindByID(ctx, tx, invoiceID)
		if err != nil {
			return err
		}
		if inv == nil {
			return ErrInvoiceNotFound
		}

		if inv.KSeFStatus != "accepted" {
			return errors.New("UPO is only available for accepted invoices")
		}

		var respData map[string]any
		if inv.KSeFResponse != nil {
			_ = json.Unmarshal(inv.KSeFResponse, &respData)
		}
		refNum, _ := respData["reference_number"].(string)
		if refNum == "" {
			return errors.New("no reference number found")
		}

		cfg, err := s.loadKSeFSettings(ctx, tx, tenantID)
		if err != nil || !cfg.Enabled {
			return ErrKSeFNotConfigured
		}

		client := s.createClient(cfg)
		upoBytes, err = client.Invoice.GetUPOBytes(ctx, refNum)
		return err
	})
	return upoBytes, err
}

// BulkSendToKSeF sends multiple invoices to KSeF.
func (s *KSeFService) BulkSendToKSeF(ctx context.Context, tenantID uuid.UUID, invoiceIDs []uuid.UUID, actorID uuid.UUID, ip string) (int, []string, error) {
	sent := 0
	var errs []string

	for _, id := range invoiceIDs {
		if err := s.SendToKSeF(ctx, tenantID, id, actorID, ip); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", id.String()[:8], err))
		} else {
			sent++
		}
	}

	return sent, errs, nil
}

// SyncPendingStatuses checks status of all pending KSeF invoices for a tenant.
func (s *KSeFService) SyncPendingStatuses(ctx context.Context, tenantID uuid.UUID) (int, error) {
	synced := 0
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		pending, err := s.invoiceRepo.FindPendingKSeF(ctx, tx)
		if err != nil {
			return err
		}

		if len(pending) == 0 {
			return nil
		}

		cfg, err := s.loadKSeFSettings(ctx, tx, tenantID)
		if err != nil || !cfg.Enabled {
			return nil
		}

		client := s.createClient(cfg)

		for _, inv := range pending {
			var respData map[string]any
			if inv.KSeFResponse != nil {
				_ = json.Unmarshal(inv.KSeFResponse, &respData)
			}
			refNum, _ := respData["reference_number"].(string)
			if refNum == "" {
				continue
			}

			upo, err := client.Invoice.GetUPO(ctx, refNum)
			if err != nil {
				slog.Warn("ksef: sync status failed", "invoice_id", inv.ID, "error", err)
				continue
			}

			if upo.ProcessingCode == 200 {
				responseJSON, _ := json.Marshal(map[string]any{
					"reference_number":       upo.ReferenceNumber,
					"processing_code":        upo.ProcessingCode,
					"processing_description": upo.ProcessingDescription,
				})
				if err := s.invoiceRepo.UpdateKSeFStatus(ctx, tx, inv.ID, &upo.ReferenceNumber, "accepted", responseJSON); err != nil {
					slog.Error("ksef: update status failed", "invoice_id", inv.ID, "error", err)
					continue
				}
				synced++
			} else if upo.ProcessingCode >= 400 {
				responseJSON, _ := json.Marshal(map[string]any{
					"reference_number":       upo.ReferenceNumber,
					"processing_code":        upo.ProcessingCode,
					"processing_description": upo.ProcessingDescription,
				})
				if err := s.invoiceRepo.UpdateKSeFStatus(ctx, tx, inv.ID, nil, "rejected", responseJSON); err != nil {
					slog.Error("ksef: update status failed", "invoice_id", inv.ID, "error", err)
					continue
				}
				synced++
			}
		}
		return nil
	})
	return synced, err
}

// loadKSeFSettings reads the "ksef" section from tenant settings.
func (s *KSeFService) loadKSeFSettings(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) (KSeFSettings, error) {
	settings, err := s.tenantRepo.GetSettings(ctx, tx, tenantID)
	if err != nil {
		return KSeFSettings{}, err
	}
	if settings == nil {
		return KSeFSettings{}, nil
	}

	var allSettings map[string]json.RawMessage
	if err := json.Unmarshal(settings, &allSettings); err != nil {
		return KSeFSettings{}, nil
	}

	raw, ok := allSettings["ksef"]
	if !ok {
		return KSeFSettings{}, nil
	}

	var cfg KSeFSettings
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return KSeFSettings{}, nil
	}
	return cfg, nil
}

// saveKSeFSettings writes the "ksef" section to tenant settings.
func (s *KSeFService) saveKSeFSettings(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, cfg KSeFSettings) error {
	existing, err := s.tenantRepo.GetSettings(ctx, tx, tenantID)
	if err != nil {
		return err
	}

	var allSettings map[string]json.RawMessage
	if err := json.Unmarshal(existing, &allSettings); err != nil {
		allSettings = make(map[string]json.RawMessage)
	}

	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	allSettings["ksef"] = cfgJSON

	newSettings, err := json.Marshal(allSettings)
	if err != nil {
		return err
	}

	return s.tenantRepo.UpdateSettings(ctx, tx, tenantID, newSettings)
}

// createClient creates a KSeF API client based on the settings.
func (s *KSeFService) createClient(cfg KSeFSettings) *ksef.Client {
	env := ksef.EnvironmentTest
	if cfg.Environment == "production" {
		env = ksef.EnvironmentProduction
	}
	return ksef.NewClient(env)
}

// buildInvoiceData converts an invoice and order into KSeF invoice data.
func (s *KSeFService) buildInvoiceData(inv *model.Invoice, order *model.Order, cfg KSeFSettings) ksef.InvoiceData {
	data := ksef.InvoiceData{
		InvoiceNumber: "",
		Currency:      inv.Currency,
		SellerNIP:     cfg.NIP,
		SellerName:    cfg.CompanyName,
		SellerStreet:  cfg.CompanyStreet,
		SellerCity:    cfg.CompanyCity,
		SellerPostal:  cfg.CompanyPostal,
		SellerCountry: cfg.CompanyCountry,
	}

	if inv.ExternalNumber != nil {
		data.InvoiceNumber = *inv.ExternalNumber
	}
	if inv.IssueDate != nil {
		data.InvoiceDate = *inv.IssueDate
	} else {
		data.InvoiceDate = time.Now()
	}
	if inv.DueDate != nil {
		data.PaymentDate = *inv.DueDate
	}
	if inv.TotalNet != nil {
		data.TotalNet = *inv.TotalNet
	}
	if inv.TotalGross != nil {
		data.TotalGross = *inv.TotalGross
	}
	data.TotalVAT = data.TotalGross - data.TotalNet

	if order != nil {
		data.BuyerName = order.CustomerName
		if order.CustomerEmail != nil {
			data.Notes = "Email: " + *order.CustomerEmail
		}
		// Try to extract NIP and address from order metadata
		if order.ShippingAddress != nil {
			var addr map[string]string
			if err := json.Unmarshal(order.ShippingAddress, &addr); err == nil {
				data.BuyerStreet = addr["street"]
				data.BuyerCity = addr["city"]
				data.BuyerPostal = addr["postal_code"]
				data.BuyerCountry = addr["country"]
			}
		}
	}

	// Build line items from order items
	taxRate := 23 // Default Polish VAT rate
	if inv.TotalNet != nil && inv.TotalGross != nil && *inv.TotalNet > 0 {
		effectiveRate := (*inv.TotalGross / *inv.TotalNet - 1) * 100
		if effectiveRate > 20 && effectiveRate < 26 {
			taxRate = 23
		} else if effectiveRate > 6 && effectiveRate < 10 {
			taxRate = 8
		} else if effectiveRate > 3 && effectiveRate < 7 {
			taxRate = 5
		}
	}

	data.Items = s.buildLineItems(order, taxRate)

	return data
}

// buildLineItems extracts line items from order data.
func (s *KSeFService) buildLineItems(order *model.Order, taxRate int) []ksef.InvoiceLineItem {
	if order == nil || order.Items == nil || string(order.Items) == "[]" || string(order.Items) == "null" {
		return []ksef.InvoiceLineItem{
			{
				LineNumber: 1,
				Name:       "Zamówienie",
				Quantity:   1,
				Unit:       "szt.",
				VATRate:    fmt.Sprintf("%d", taxRate),
			},
		}
	}

	type orderItem struct {
		Name     string  `json:"name"`
		SKU      string  `json:"sku"`
		Quantity int     `json:"quantity"`
		Price    float64 `json:"price"`
	}

	var orderItems []orderItem
	if err := json.Unmarshal(order.Items, &orderItems); err != nil {
		return []ksef.InvoiceLineItem{
			{
				LineNumber: 1,
				Name:       "Zamówienie",
				Quantity:   1,
				Unit:       "szt.",
				VATRate:    fmt.Sprintf("%d", taxRate),
			},
		}
	}

	items := make([]ksef.InvoiceLineItem, 0, len(orderItems))
	for i, oi := range orderItems {
		qty := oi.Quantity
		if qty <= 0 {
			qty = 1
		}
		netPrice := oi.Price / (1 + float64(taxRate)/100)
		netAmount := netPrice * float64(qty)
		vatAmount := (oi.Price - netPrice) * float64(qty)
		grossAmount := oi.Price * float64(qty)

		items = append(items, ksef.InvoiceLineItem{
			LineNumber:  i + 1,
			Name:        oi.Name,
			Quantity:    float64(qty),
			Unit:        "szt.",
			NetPrice:    netPrice,
			NetAmount:   netAmount,
			VATRate:     fmt.Sprintf("%d", taxRate),
			VATAmount:   vatAmount,
			GrossAmount: grossAmount,
		})
	}

	if len(items) == 0 {
		return []ksef.InvoiceLineItem{
			{
				LineNumber: 1,
				Name:       "Zamówienie",
				Quantity:   1,
				Unit:       "szt.",
				VATRate:    fmt.Sprintf("%d", taxRate),
			},
		}
	}

	return items
}

// markKSeFError updates an invoice with a KSeF error status.
func (s *KSeFService) markKSeFError(ctx context.Context, tx pgx.Tx, inv *model.Invoice, err error) error {
	inv.KSeFStatus = "rejected"
	errMsg := err.Error()
	responseJSON, _ := json.Marshal(map[string]any{
		"error": errMsg,
	})
	inv.KSeFResponse = responseJSON
	_ = s.invoiceRepo.Update(ctx, tx, inv)
	return err
}
