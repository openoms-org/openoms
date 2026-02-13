package service

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"slices"
	"text/template"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	smsapi "github.com/openoms-org/openoms/packages/smsapi-go-sdk"
)

type SMSService struct {
	tenantRepo repository.TenantRepo
	pool       *pgxpool.Pool
}

func NewSMSService(tenantRepo repository.TenantRepo, pool *pgxpool.Pool) *SMSService {
	return &SMSService{tenantRepo: tenantRepo, pool: pool}
}

func (s *SMSService) loadSMSSettings(ctx context.Context, tenantID uuid.UUID) *model.SMSSettings {
	var cfg *model.SMSSettings
	if err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		settings, err := s.tenantRepo.GetSettings(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		if settings == nil {
			return nil
		}
		var allSettings map[string]json.RawMessage
		if err := json.Unmarshal(settings, &allSettings); err != nil {
			return nil
		}
		raw, ok := allSettings["sms"]
		if !ok {
			return nil
		}
		var parsed model.SMSSettings
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return nil
		}
		cfg = &parsed
		return nil
	}); err != nil {
		slog.Error("sms: failed to load tenant settings", "error", err, "tenant_id", tenantID)
	}
	return cfg
}

// SendOrderStatusSMS sends an SMS notification when order status changes.
func (s *SMSService) SendOrderStatusSMS(ctx context.Context, tenantID uuid.UUID, order *model.Order, oldStatus, newStatus string) {
	if order.CustomerPhone == nil || *order.CustomerPhone == "" {
		slog.Debug("sms: no customer phone number, skipping", "order_id", order.ID)
		return
	}

	cfg := s.loadSMSSettings(ctx, tenantID)
	if cfg == nil || !cfg.Enabled || cfg.APIToken == "" {
		return
	}

	// Check if this status is in the notify list
	shouldNotify := slices.Contains(cfg.NotifyOn, newStatus)
	if !shouldNotify {
		return
	}

	// Get template for this status
	tmplStr, ok := cfg.Templates[newStatus]
	if !ok || tmplStr == "" {
		slog.Debug("sms: no template for status", "status", newStatus, "tenant_id", tenantID)
		return
	}

	// Render template
	orderNumber := order.ID.String()[:8]
	data := map[string]string{
		"OrderNumber":    orderNumber,
		"Status":         newStatus,
		"CustomerName":   order.CustomerName,
		"TrackingNumber": "",
		"TrackingURL":    "",
	}

	message, err := renderSMSTemplate(tmplStr, data)
	if err != nil {
		slog.Error("sms: failed to render template", "error", err, "status", newStatus, "order_id", order.ID)
		return
	}

	// Send SMS
	client := smsapi.NewClient(cfg.APIToken, smsapi.WithFrom(cfg.From))
	_, err = client.SendSMS(ctx, smsapi.SendSMSRequest{
		To:      *order.CustomerPhone,
		Message: message,
	})
	if err != nil {
		slog.Error("sms: failed to send", "error", err, "to", *order.CustomerPhone, "status", newStatus, "order_id", order.ID)
	} else {
		slog.Info("sms: sent successfully", "to", *order.CustomerPhone, "status", newStatus, "order_id", order.ID)
	}
}

// SendShipmentStatusSMS sends an SMS when shipment status changes (tracking update).
func (s *SMSService) SendShipmentStatusSMS(ctx context.Context, tenantID uuid.UUID, shipment *model.Shipment, trackingURL string) {
	// Load the order to get the customer phone number
	var order *model.Order
	if err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// We need to get the order for the shipment
		// Use a simple query since we already have the order ID
		row := tx.QueryRow(ctx, "SELECT id, customer_name, customer_phone FROM orders WHERE id = $1", shipment.OrderID)
		var id uuid.UUID
		var customerName string
		var customerPhone *string
		if err := row.Scan(&id, &customerName, &customerPhone); err != nil {
			return err
		}
		order = &model.Order{
			ID:            id,
			CustomerName:  customerName,
			CustomerPhone: customerPhone,
		}
		return nil
	}); err != nil {
		slog.Error("sms: failed to load order for shipment", "error", err, "shipment_id", shipment.ID, "order_id", shipment.OrderID)
		return
	}

	if order.CustomerPhone == nil || *order.CustomerPhone == "" {
		slog.Debug("sms: no customer phone number for shipment", "shipment_id", shipment.ID)
		return
	}

	cfg := s.loadSMSSettings(ctx, tenantID)
	if cfg == nil || !cfg.Enabled || cfg.APIToken == "" {
		return
	}

	// Check if this shipment status is in notify list
	shouldNotify := slices.Contains(cfg.NotifyOn, shipment.Status)
	if !shouldNotify {
		return
	}

	// Get template for this status
	tmplStr, ok := cfg.Templates[shipment.Status]
	if !ok || tmplStr == "" {
		slog.Debug("sms: no template for shipment status", "status", shipment.Status, "tenant_id", tenantID)
		return
	}

	// Render template
	orderNumber := shipment.OrderID.String()[:8]
	trackingNumber := ""
	if shipment.TrackingNumber != nil {
		trackingNumber = *shipment.TrackingNumber
	}

	data := map[string]string{
		"OrderNumber":    orderNumber,
		"Status":         shipment.Status,
		"CustomerName":   order.CustomerName,
		"TrackingNumber": trackingNumber,
		"TrackingURL":    trackingURL,
	}

	message, err := renderSMSTemplate(tmplStr, data)
	if err != nil {
		slog.Error("sms: failed to render template", "error", err, "status", shipment.Status, "shipment_id", shipment.ID)
		return
	}

	// Send SMS
	client := smsapi.NewClient(cfg.APIToken, smsapi.WithFrom(cfg.From))
	_, err = client.SendSMS(ctx, smsapi.SendSMSRequest{
		To:      *order.CustomerPhone,
		Message: message,
	})
	if err != nil {
		slog.Error("sms: failed to send shipment SMS", "error", err, "to", *order.CustomerPhone, "status", shipment.Status, "shipment_id", shipment.ID)
	} else {
		slog.Info("sms: shipment SMS sent", "to", *order.CustomerPhone, "status", shipment.Status, "shipment_id", shipment.ID)
	}
}

// SendTestSMS sends a test SMS to the provided phone number.
func (s *SMSService) SendTestSMS(ctx context.Context, settings model.SMSSettings, phone string) error {
	client := smsapi.NewClient(settings.APIToken, smsapi.WithFrom(settings.From))
	_, err := client.SendSMS(ctx, smsapi.SendSMSRequest{
		To:      phone,
		Message: "OpenOMS — Testowy SMS. Jesli widzisz ta wiadomosc, konfiguracja SMSAPI dziala poprawnie.",
	})
	return err
}

func renderSMSTemplate(tmplStr string, data map[string]string) (string, error) {
	tmpl, err := template.New("sms").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
