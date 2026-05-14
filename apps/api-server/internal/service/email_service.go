package service

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log/slog"
	"net/smtp"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// EmailService sends transactional email notifications for order events.
type EmailService struct {
	tenantRepo repository.TenantRepo
	pool       *pgxpool.Pool
}

// NewEmailService creates a new EmailService.
func NewEmailService(tenantRepo repository.TenantRepo, pool *pgxpool.Pool) *EmailService {
	return &EmailService{tenantRepo: tenantRepo, pool: pool}
}

func (s *EmailService) loadStatusConfig(ctx context.Context, tenantID uuid.UUID) *model.OrderStatusConfig {
	var config *model.OrderStatusConfig
	if err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		settings, err := s.tenantRepo.GetSettings(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		if settings != nil {
			var allSettings map[string]json.RawMessage
			if err := json.Unmarshal(settings, &allSettings); err == nil {
				if raw, ok := allSettings["order_statuses"]; ok {
					var cfg model.OrderStatusConfig
					if err := json.Unmarshal(raw, &cfg); err == nil && len(cfg.Statuses) > 0 {
						config = &cfg
						return nil
					}
				}
			}
		}
		cfg := model.DefaultOrderStatusConfig()
		config = &cfg
		return nil
	}); err != nil {
		slog.Error("email: failed to load status config", "error", err, "tenant_id", tenantID)
	}
	return config
}

// SendOrderStatusEmail sends a status-change notification email to the customer.
func (s *EmailService) SendOrderStatusEmail(ctx context.Context, tenantID uuid.UUID, order *model.Order, _ string, newStatus string) {
	if order.CustomerEmail == nil || *order.CustomerEmail == "" {
		return
	}

	var settings json.RawMessage
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		settings, err = s.tenantRepo.GetSettings(ctx, tx, tenantID)
		return err
	})
	if err != nil {
		slog.Error("email: failed to load tenant settings", "error", err, "tenant_id", tenantID)
		return
	}

	var emailCfg model.EmailSettings
	var allSettings map[string]json.RawMessage
	if err := json.Unmarshal(settings, &allSettings); err != nil {
		slog.Debug("email: no email settings configured", "tenant_id", tenantID)
		return
	}
	emailRaw, ok := allSettings["email"]
	if !ok {
		slog.Debug("email: no email settings configured", "tenant_id", tenantID)
		return
	}
	if err := json.Unmarshal(emailRaw, &emailCfg); err != nil {
		slog.Debug("email: invalid email settings", "tenant_id", tenantID)
		return
	}

	if !emailCfg.Enabled {
		return
	}

	// Check if this status is in the notify list
	shouldNotify := slices.Contains(emailCfg.NotifyOn, newStatus)
	if !shouldNotify {
		return
	}

	statusCfg := s.loadStatusConfig(ctx, tenantID)
	subject, body := renderEmailTemplate(order, newStatus, emailCfg.FromName, statusCfg)
	if err := sendMail(emailCfg, *order.CustomerEmail, subject, body); err != nil {
		slog.Error("email: failed to send", "error", err, "status", newStatus, "order_id", order.ID, "tenant_id", tenantID)
	} else {
		slog.Info("email: sent successfully", "status", newStatus, "order_id", order.ID, "tenant_id", tenantID)
	}
}

// SendTestEmail sends a test message to verify email configuration.
func (s *EmailService) SendTestEmail(_ context.Context, settings model.EmailSettings, toEmail string) error {
	subject := "OpenOMS — Email connection test"
	body := `<!DOCTYPE html>
<html><head><meta charset="utf-8"></head>
<body style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto;padding:20px;">
<h2 style="color:#1a1a1a;">Email connection test</h2>
<p>If you see this message, SMTP configuration is working correctly.</p>
<p style="color:#666;font-size:12px;margin-top:30px;">— OpenOMS</p>
</body></html>`

	return sendMail(settings, toEmail, subject, body)
}

func renderEmailTemplate(order *model.Order, newStatus string, companyName string, statusCfg *model.OrderStatusConfig) (string, string) {
	orderShort := order.ID.String()[:8]
	customerName := html.EscapeString(order.CustomerName)

	// Dynamic label lookup
	statusLabel := newStatus
	if statusCfg != nil {
		if def := statusCfg.GetStatusDef(newStatus); def != nil {
			statusLabel = def.Label
		}
	}

	subject := fmt.Sprintf("Order #%s — %s", orderShort, statusLabel)

	// Escape all tenant-controlled values before HTML interpolation
	statusLabel = html.EscapeString(statusLabel)
	companyName = html.EscapeString(companyName)

	// Dynamic color lookup
	statusColor := "#6b7280" // default gray
	if statusCfg != nil {
		if def := statusCfg.GetStatusDef(newStatus); def != nil {
			if hex, ok := model.ColorPresetHex[def.Color]; ok {
				statusColor = hex
			}
		}
	}

	var extraInfo string
	if newStatus == "shipped" || newStatus == "in_transit" {
		extraInfo = `<p style="margin-top:15px;padding:12px;background:#f0f9ff;border-radius:6px;">Your order is on its way. Track your shipment with your carrier.</p>`
	}
	if newStatus == "cancelled" {
		extraInfo = `<p style="margin-top:15px;padding:12px;background:#fef2f2;border-radius:6px;">If you have questions about the cancellation, please contact us.</p>`
	}
	if newStatus == "refunded" {
		extraInfo = `<p style="margin-top:15px;padding:12px;background:#fffbeb;border-radius:6px;">A refund has been initiated. The funds will appear in your account within a few business days.</p>`
	}

	totalAmount := fmt.Sprintf("%.2f %s", order.TotalAmount, html.EscapeString(order.Currency))

	body := fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="utf-8"></head>
<body style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto;padding:20px;background:#f9fafb;">
<div style="background:white;border-radius:8px;padding:30px;border:1px solid #e5e7eb;">
<h2 style="color:#1a1a1a;margin-top:0;">Order #%s</h2>
<p>Hi %s,</p>
<p>Your order status has been updated to:</p>
<div style="text-align:center;margin:20px 0;">
<span style="display:inline-block;padding:8px 20px;background:%s;color:white;border-radius:20px;font-weight:bold;font-size:16px;">%s</span>
</div>
<table style="width:100%%;border-collapse:collapse;margin-top:20px;">
<tr><td style="padding:8px 0;color:#666;">Order number:</td><td style="padding:8px 0;font-weight:bold;">#%s</td></tr>
<tr><td style="padding:8px 0;color:#666;">Amount:</td><td style="padding:8px 0;font-weight:bold;">%s</td></tr>
</table>
%s
</div>
<p style="color:#999;font-size:12px;text-align:center;margin-top:20px;">%s — Automated message from OpenOMS</p>
</body></html>`,
		orderShort, customerName, statusColor, strings.ToUpper(statusLabel),
		orderShort, totalAmount, extraInfo, companyName)

	return subject, body
}

func sendMail(cfg model.EmailSettings, to, subject, htmlBody string) error {
	if cfg.SMTPHost == "" || cfg.FromEmail == "" {
		return fmt.Errorf("SMTP not configured")
	}

	// Sanitize header values to prevent header injection (CRLF)
	sanitizer := strings.NewReplacer("\r", "", "\n", "")
	to = sanitizer.Replace(to)
	fromName := sanitizer.Replace(cfg.FromName)
	subject = sanitizer.Replace(subject)

	fromEmail := sanitizer.Replace(cfg.FromEmail)
	from := fromEmail
	if fromName != "" {
		from = fmt.Sprintf("%s <%s>", fromName, fromEmail)
	}

	headers := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=utf-8\r\n\r\n",
		from, to, subject)
	msg := []byte(headers + htmlBody)

	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)

	var auth smtp.Auth
	if cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPHost)
	}

	return smtp.SendMail(addr, auth, fromEmail, []string{to}, msg)
}
