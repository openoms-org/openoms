package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

type EmailService struct {
	tenantRepo *repository.TenantRepository
	pool       *pgxpool.Pool
}

func NewEmailService(tenantRepo *repository.TenantRepository, pool *pgxpool.Pool) *EmailService {
	return &EmailService{tenantRepo: tenantRepo, pool: pool}
}

func (s *EmailService) SendOrderStatusEmail(ctx context.Context, tenantID uuid.UUID, order *model.Order, oldStatus, newStatus string) {
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
	if err := json.Unmarshal(settings, &emailCfg); err != nil {
		slog.Debug("email: no email settings configured", "tenant_id", tenantID)
		return
	}

	if !emailCfg.Enabled {
		return
	}

	// Check if this status is in the notify list
	shouldNotify := false
	for _, status := range emailCfg.NotifyOn {
		if status == newStatus {
			shouldNotify = true
			break
		}
	}
	if !shouldNotify {
		return
	}

	subject, body := renderEmailTemplate(order, newStatus, emailCfg.FromName)
	if err := sendMail(emailCfg, *order.CustomerEmail, subject, body); err != nil {
		slog.Error("email: failed to send", "error", err, "to", *order.CustomerEmail, "status", newStatus, "order_id", order.ID)
	} else {
		slog.Info("email: sent successfully", "to", *order.CustomerEmail, "status", newStatus, "order_id", order.ID)
	}
}

func (s *EmailService) SendTestEmail(ctx context.Context, settings model.EmailSettings, toEmail string) error {
	subject := "OpenOMS — Testowy email"
	body := `<!DOCTYPE html>
<html><head><meta charset="utf-8"></head>
<body style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto;padding:20px;">
<h2 style="color:#1a1a1a;">Test polaczenia email</h2>
<p>Jesli widzisz ta wiadomosc, konfiguracja SMTP dziala poprawnie.</p>
<p style="color:#666;font-size:12px;margin-top:30px;">— OpenOMS</p>
</body></html>`

	return sendMail(settings, toEmail, subject, body)
}

var statusLabels = map[string]string{
	"confirmed":        "potwierdzone",
	"processing":       "w realizacji",
	"ready_to_ship":    "gotowe do wysylki",
	"shipped":          "wyslane",
	"in_transit":       "w transporcie",
	"out_for_delivery": "w dostawie",
	"delivered":        "dostarczone",
	"completed":        "zrealizowane",
	"cancelled":        "anulowane",
	"refunded":         "zwrocone",
	"on_hold":          "wstrzymane",
}

func renderEmailTemplate(order *model.Order, newStatus string, companyName string) (string, string) {
	orderShort := order.ID.String()[:8]
	customerName := order.CustomerName
	statusLabel := statusLabels[newStatus]
	if statusLabel == "" {
		statusLabel = newStatus
	}

	subject := fmt.Sprintf("Zamowienie #%s — %s", orderShort, statusLabel)

	var statusColor string
	switch newStatus {
	case "confirmed", "delivered", "completed":
		statusColor = "#16a34a"
	case "shipped", "in_transit", "out_for_delivery":
		statusColor = "#2563eb"
	case "cancelled":
		statusColor = "#dc2626"
	case "refunded":
		statusColor = "#d97706"
	default:
		statusColor = "#6b7280"
	}

	var extraInfo string
	if newStatus == "shipped" || newStatus == "in_transit" {
		extraInfo = `<p style="margin-top:15px;padding:12px;background:#f0f9ff;border-radius:6px;">Twoje zamowienie jest w drodze. Sledz przesylke u swojego kuriera.</p>`
	}
	if newStatus == "cancelled" {
		extraInfo = `<p style="margin-top:15px;padding:12px;background:#fef2f2;border-radius:6px;">Jesli masz pytania dotyczace anulowania, skontaktuj sie z nami.</p>`
	}
	if newStatus == "refunded" {
		extraInfo = `<p style="margin-top:15px;padding:12px;background:#fffbeb;border-radius:6px;">Zwrot srodkow zostal zainicjowany. Pieniadze pojawia sie na Twoim koncie w ciagu kilku dni roboczych.</p>`
	}

	totalAmount := fmt.Sprintf("%.2f %s", order.TotalAmount, order.Currency)

	body := fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="utf-8"></head>
<body style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto;padding:20px;background:#f9fafb;">
<div style="background:white;border-radius:8px;padding:30px;border:1px solid #e5e7eb;">
<h2 style="color:#1a1a1a;margin-top:0;">Zamowienie #%s</h2>
<p>Czesc %s,</p>
<p>Status Twojego zamowienia zostal zmieniony na:</p>
<div style="text-align:center;margin:20px 0;">
<span style="display:inline-block;padding:8px 20px;background:%s;color:white;border-radius:20px;font-weight:bold;font-size:16px;">%s</span>
</div>
<table style="width:100%%;border-collapse:collapse;margin-top:20px;">
<tr><td style="padding:8px 0;color:#666;">Numer zamowienia:</td><td style="padding:8px 0;font-weight:bold;">#%s</td></tr>
<tr><td style="padding:8px 0;color:#666;">Kwota:</td><td style="padding:8px 0;font-weight:bold;">%s</td></tr>
</table>
%s
</div>
<p style="color:#999;font-size:12px;text-align:center;margin-top:20px;">%s — Wiadomosc wygenerowana automatycznie przez OpenOMS</p>
</body></html>`,
		orderShort, customerName, statusColor, strings.ToUpper(statusLabel),
		orderShort, totalAmount, extraInfo, companyName)

	return subject, body
}

func sendMail(cfg model.EmailSettings, to, subject, htmlBody string) error {
	if cfg.SMTPHost == "" || cfg.FromEmail == "" {
		return fmt.Errorf("SMTP not configured")
	}

	from := cfg.FromEmail
	if cfg.FromName != "" {
		from = fmt.Sprintf("%s <%s>", cfg.FromName, cfg.FromEmail)
	}

	headers := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=utf-8\r\n\r\n",
		from, to, subject)
	msg := []byte(headers + htmlBody)

	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)

	var auth smtp.Auth
	if cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPHost)
	}

	return smtp.SendMail(addr, auth, cfg.FromEmail, []string{to}, msg)
}
