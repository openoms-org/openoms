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
	AutoSend    bool   `json:"auto_send"` // Auto-send to KSeF on invoice creation
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
			Message: "KSeF not configured. Set NIP and token.",
		}, nil
	}

	client := s.createClient(cfg)

	// Test by requesting an authorisation challenge
	resp, err := client.Session.AuthorisationChallenge(ctx, cfg.NIP)
	if err != nil {
		return &KSeFTestResult{
			Success: false,
			Message: fmt.Sprintf("KSeF connection error: %v", err),
		}, nil
	}

	return &KSeFTestResult{
		Success:   true,
		Message:   "KSeF connection OK.",
		Timestamp: resp.Timestamp,
		Challenge: resp.Challenge,
	}, nil
}

// SendToKSeF sends a single invoice to KSeF.
// Uses a three-phase approach to avoid holding a DB connection during external API calls:
// Phase 1: short DB transaction to read invoice data and validate status.
// Phase 2: external KSeF API calls (no DB connection held).
// Phase 3: short DB transaction to update invoice with results.
func (s *KSeFService) SendToKSeF(ctx context.Context, tenantID, invoiceID, actorID uuid.UUID, ip string) error {
	// Phase 1: Read invoice, settings, and order data in a short transaction.
	var inv *model.Invoice
	var order *model.Order
	var cfg KSeFSettings
	var xmlBytes []byte
	var existingRetryCount int

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		inv, err = s.invoiceRepo.FindByID(ctx, tx, invoiceID)
		if err != nil {
			return err
		}
		if inv == nil {
			return ErrInvoiceNotFound
		}

		if inv.KSeFStatus != "not_sent" && inv.KSeFStatus != "retrying" {
			return ErrKSeFAlreadySent
		}

		cfg, err = s.loadKSeFSettings(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		if !cfg.Enabled {
			return ErrKSeFNotConfigured
		}

		// Load order for buyer details
		order, err = s.orderRepo.FindByID(ctx, tx, inv.OrderID)
		if err != nil {
			return fmt.Errorf("load order: %w", err)
		}

		// Preserve existing retry_count from ksef_response
		if inv.KSeFResponse != nil {
			var existing map[string]any
			if jsonErr := json.Unmarshal(inv.KSeFResponse, &existing); jsonErr == nil {
				if rc, ok := existing["retry_count"].(float64); ok {
					existingRetryCount = int(rc)
				}
			}
		}

		// Build the invoice XML while we still have the data
		invoiceData := s.buildInvoiceData(inv, order, cfg)
		xmlBytes, err = ksef.BuildInvoiceXML(invoiceData)
		if err != nil {
			return fmt.Errorf("build invoice XML: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Phase 2: External KSeF API calls — no DB connection held.
	client := s.createClient(cfg)
	now := time.Now()

	// Step 1: Get authorisation challenge
	challenge, err := client.Session.AuthorisationChallenge(ctx, cfg.NIP)
	if err != nil {
		ksefErr := fmt.Errorf("authorisation challenge: %w", err)
		s.markKSeFErrorOutsideTx(ctx, tenantID, inv, existingRetryCount, ksefErr)
		return ksefErr
	}

	// Step 2: Init session with token
	session, err := client.Session.InitToken(ctx, cfg.NIP, cfg.Token, challenge.Challenge)
	if err != nil {
		ksefErr := fmt.Errorf("init session: %w", err)
		s.markKSeFErrorOutsideTx(ctx, tenantID, inv, existingRetryCount, ksefErr)
		return ksefErr
	}
	// Ensure session cleanup on any failure path (context cancellation, panic, etc.).
	// Uses background context since the caller's context may already be cancelled.
	defer func() {
		_, _ = client.Session.Terminate(context.Background(), session.SessionToken.Token)
	}()

	// Step 3: Send the invoice
	sendResp, err := client.Invoice.Send(ctx, session.SessionToken.Token, xmlBytes)
	if err != nil {
		ksefErr := fmt.Errorf("send invoice: %w", err)
		s.markKSeFErrorOutsideTx(ctx, tenantID, inv, existingRetryCount, ksefErr)
		return ksefErr
	}

	// Phase 3: Short DB transaction to update invoice with results.
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		inv.KSeFStatus = "pending"
		inv.KSeFSentAt = &now

		responseJSON, _ := json.Marshal(map[string]any{
			"element_reference_number": sendResp.ElementReferenceNumber,
			"reference_number":         sendResp.ReferenceNumber,
			"processing_code":          sendResp.ProcessingCode,
			"timestamp":                sendResp.Timestamp,
			"retry_count":              existingRetryCount,
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

// markKSeFErrorOutsideTx persists a KSeF error status using its own short transaction.
// Used when the error occurs outside the main DB transaction (during external API calls).
func (s *KSeFService) markKSeFErrorOutsideTx(ctx context.Context, tenantID uuid.UUID, inv *model.Invoice, retryCount int, ksefErr error) {
	updateErr := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		inv.KSeFStatus = "error"
		errMsg := ksefErr.Error()
		responseJSON, _ := json.Marshal(map[string]any{
			"error":         errMsg,
			"retry_count":   retryCount,
			"last_error_at": time.Now().UTC().Format(time.RFC3339),
		})
		inv.KSeFResponse = responseJSON
		return s.invoiceRepo.Update(ctx, tx, inv)
	})
	if updateErr != nil {
		slog.Error("ksef: failed to persist error status", "invoice_id", inv.ID, "update_error", updateErr, "ksef_error", ksefErr)
	}
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
// Uses a two-phase approach to avoid holding a DB connection during external API calls:
// Phase 1: gather pending invoices + KSeF settings inside a short transaction.
// Phase 2: call KSeF API for each invoice, persisting updates in short per-invoice transactions.
// Batch size is limited to 100 invoices per run (enforced by FindPendingKSeF SQL LIMIT).
func (s *KSeFService) SyncPendingStatuses(ctx context.Context, tenantID uuid.UUID) (int, error) {
	// Phase 1: gather data inside short transaction
	var pending []model.Invoice
	var cfg KSeFSettings
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		pending, err = s.invoiceRepo.FindPendingKSeF(ctx, tx)
		if err != nil {
			return err
		}
		if len(pending) == 0 {
			return nil
		}
		cfg, err = s.loadKSeFSettings(ctx, tx, tenantID)
		return err
	})
	if err != nil || len(pending) == 0 || !cfg.Enabled {
		return 0, err
	}

	// Phase 2: external API calls + short DB updates outside the main transaction
	client := s.createClient(cfg)
	synced := 0

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
			slog.Warn("ksef: failed to check status", "invoice_id", inv.ID, "error", err)
			continue
		}

		if upo.ProcessingCode == 200 {
			responseJSON, _ := json.Marshal(map[string]any{
				"reference_number":       upo.ReferenceNumber,
				"processing_code":        upo.ProcessingCode,
				"processing_description": upo.ProcessingDescription,
			})
			if updateErr := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
				return s.invoiceRepo.UpdateKSeFStatus(ctx, tx, inv.ID, &upo.ReferenceNumber, "accepted", responseJSON)
			}); updateErr != nil {
				slog.Error("ksef: update status failed", "invoice_id", inv.ID, "error", updateErr)
				continue
			}
			synced++
		} else if upo.ProcessingCode >= 400 {
			responseJSON, _ := json.Marshal(map[string]any{
				"reference_number":       upo.ReferenceNumber,
				"processing_code":        upo.ProcessingCode,
				"processing_description": upo.ProcessingDescription,
			})
			if updateErr := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
				return s.invoiceRepo.UpdateKSeFStatus(ctx, tx, inv.ID, nil, "rejected", responseJSON)
			}); updateErr != nil {
				slog.Error("ksef: update status failed", "invoice_id", inv.ID, "error", updateErr)
				continue
			}
			synced++
		}
	}

	return synced, nil
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
				Name:       "Order",
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
				Name:       "Order",
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
				Name:       "Order",
				Quantity:   1,
				Unit:       "szt.",
				VATRate:    fmt.Sprintf("%d", taxRate),
			},
		}
	}

	return items
}

// markKSeFError updates an invoice with a KSeF error status.
// Uses "error" status (retryable) instead of "rejected" (terminal, only for UPO rejections).
func (s *KSeFService) markKSeFError(ctx context.Context, tx pgx.Tx, inv *model.Invoice, err error) error {
	// Preserve existing retry_count from ksef_response
	retryCount := 0
	if inv.KSeFResponse != nil {
		var existing map[string]any
		if jsonErr := json.Unmarshal(inv.KSeFResponse, &existing); jsonErr == nil {
			if rc, ok := existing["retry_count"].(float64); ok {
				retryCount = int(rc)
			}
		}
	}

	inv.KSeFStatus = "error"
	errMsg := err.Error()
	responseJSON, _ := json.Marshal(map[string]any{
		"error":         errMsg,
		"retry_count":   retryCount,
		"last_error_at": time.Now().UTC().Format(time.RFC3339),
	})
	inv.KSeFResponse = responseJSON
	if updateErr := s.invoiceRepo.Update(ctx, tx, inv); updateErr != nil {
		slog.Error("ksef: failed to persist error status", "invoice_id", inv.ID, "update_error", updateErr, "ksef_error", err)
	}
	return err
}

// RetryErroredInvoices retries sending invoices with ksef_status = 'error'.
// Max 3 retries with exponential backoff (5min, 15min, 45min).
// After max retries, status is set to "rejected" (terminal).
// Batch size is limited to 100 invoices per run (enforced by FindErrorKSeF SQL LIMIT).
func (s *KSeFService) RetryErroredInvoices(ctx context.Context, tenantID uuid.UUID) (int, error) {
	// Phase 1: Inside a single transaction, evaluate errored invoices and either
	// mark them as "rejected" (terminal) or set them to "retrying" (intermediate).
	// Collect IDs of invoices that were set to "retrying" for the send phase.
	var retryIDs []uuid.UUID
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		errored, err := s.invoiceRepo.FindErrorKSeF(ctx, tx)
		if err != nil {
			return err
		}

		if len(errored) == 0 {
			return nil
		}

		cfg, err := s.loadKSeFSettings(ctx, tx, tenantID)
		if err != nil || !cfg.Enabled {
			return nil
		}

		now := time.Now()
		for _, inv := range errored {
			retryCount := 0
			var lastErrorAt time.Time

			if inv.KSeFResponse != nil {
				var respData map[string]any
				if jsonErr := json.Unmarshal(inv.KSeFResponse, &respData); jsonErr == nil {
					if rc, ok := respData["retry_count"].(float64); ok {
						retryCount = int(rc)
					}
					if lea, ok := respData["last_error_at"].(string); ok {
						lastErrorAt, _ = time.Parse(time.RFC3339, lea)
					}
				}
			}

			// Max 3 retries — after that, mark as "rejected" (terminal)
			const maxRetries = 3
			if retryCount >= maxRetries {
				responseJSON, _ := json.Marshal(map[string]any{
					"error":       "max retries exceeded",
					"retry_count": retryCount,
				})
				if err := s.invoiceRepo.UpdateKSeFStatus(ctx, tx, inv.ID, nil, "rejected", responseJSON); err != nil {
					slog.Error("ksef: failed to mark rejected", "invoice_id", inv.ID, "error", err)
				}
				continue
			}

			// Exponential backoff: 5min, 15min, 45min
			backoff := 5 * time.Minute * time.Duration(pow3(retryCount))
			if !lastErrorAt.IsZero() && now.Before(lastErrorAt.Add(backoff)) {
				continue // Too early to retry
			}

			// Reset status to "retrying" so SendToKSeF accepts it, and bump retry_count
			responseJSON, _ := json.Marshal(map[string]any{
				"retry_count":   retryCount + 1,
				"last_error_at": now.UTC().Format(time.RFC3339),
			})
			if err := s.invoiceRepo.UpdateKSeFStatus(ctx, tx, inv.ID, nil, "retrying", responseJSON); err != nil {
				slog.Error("ksef: failed to reset for retry", "invoice_id", inv.ID, "error", err)
				continue
			}
			retryIDs = append(retryIDs, inv.ID)
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	// Phase 2: Send each "retrying" invoice outside the transaction.
	// SendToKSeF opens its own WithTenant — no nesting risk.
	// Track actual send successes, not just resets.
	sent := 0
	for _, id := range retryIDs {
		if sendErr := s.SendToKSeF(ctx, tenantID, id, uuid.Nil, "system"); sendErr != nil {
			slog.Warn("ksef: retry send failed", "invoice_id", id, "error", sendErr)
		} else {
			sent++
		}
	}

	return sent, nil
}

// pow3 returns 3^n (used for exponential backoff multiplier).
func pow3(n int) int {
	result := 1
	for range n {
		result *= 3
	}
	return result
}
