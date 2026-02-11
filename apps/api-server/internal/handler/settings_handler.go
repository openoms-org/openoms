package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

type SettingsHandler struct {
	tenantRepo   repository.TenantRepo
	auditRepo    repository.AuditRepo
	emailService *service.EmailService
	pool         *pgxpool.Pool
}

func NewSettingsHandler(tenantRepo repository.TenantRepo, auditRepo repository.AuditRepo, emailService *service.EmailService, pool *pgxpool.Pool) *SettingsHandler {
	return &SettingsHandler{tenantRepo: tenantRepo, auditRepo: auditRepo, emailService: emailService, pool: pool}
}

// getSettingsSection reads a specific section from the tenant's JSON settings blob.
// If the section or settings don't exist, dest is left at its zero value.
func (h *SettingsHandler) getSettingsSection(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, key string, dest interface{}) error {
	settings, err := h.tenantRepo.GetSettings(ctx, tx, tenantID)
	if err != nil {
		return err
	}

	if settings == nil {
		return nil
	}

	var allSettings map[string]json.RawMessage
	if err := json.Unmarshal(settings, &allSettings); err != nil {
		return nil // settings is empty or not a map
	}

	raw, ok := allSettings[key]
	if !ok {
		return nil
	}

	// Ignore unmarshal errors — return zero value of dest
	json.Unmarshal(raw, dest)
	return nil
}

// updateSettingsSection merges a value into the tenant's JSON settings blob under the given key,
// persists it, and writes an audit log entry.
func (h *SettingsHandler) updateSettingsSection(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, key string, value interface{}) error {
	existing, err := h.tenantRepo.GetSettings(ctx, tx, tenantID)
	if err != nil {
		return err
	}

	var allSettings map[string]json.RawMessage
	if err := json.Unmarshal(existing, &allSettings); err != nil {
		allSettings = make(map[string]json.RawMessage)
	}

	valueJSON, err := json.Marshal(value)
	if err != nil {
		return err
	}
	allSettings[key] = valueJSON

	newSettings, err := json.Marshal(allSettings)
	if err != nil {
		return err
	}

	return h.tenantRepo.UpdateSettings(ctx, tx, tenantID, newSettings)
}

func (h *SettingsHandler) GetEmailSettings(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	emailCfg := model.EmailSettings{
		SMTPPort: 587,
		NotifyOn: []string{"confirmed", "shipped", "delivered", "cancelled", "refunded"},
	}

	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		return h.getSettingsSection(r.Context(), tx, tenantID, "email", &emailCfg)
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load settings")
		return
	}

	// Mask password
	if emailCfg.SMTPPass != "" {
		emailCfg.SMTPPass = "••••••"
	}

	writeJSON(w, http.StatusOK, emailCfg)
}

func (h *SettingsHandler) UpdateEmailSettings(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var emailCfg model.EmailSettings
	if err := json.NewDecoder(r.Body).Decode(&emailCfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	actorID := middleware.UserIDFromContext(r.Context())
	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		// If password is masked, keep the existing one
		if emailCfg.SMTPPass == "••••••" {
			var oldEmail model.EmailSettings
			if err := h.getSettingsSection(r.Context(), tx, tenantID, "email", &oldEmail); err != nil {
				slog.Error("failed to load existing email settings for password preservation", "error", err, "tenant_id", tenantID)
			}
			emailCfg.SMTPPass = oldEmail.SMTPPass
		}

		if err := h.updateSettingsSection(r.Context(), tx, tenantID, "email", emailCfg); err != nil {
			return err
		}
		return h.auditRepo.Log(r.Context(), tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "settings.email_updated",
			EntityType: "settings",
			EntityID:   tenantID,
			IPAddress:  clientIP(r),
		})
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save settings")
		return
	}

	// Mask password in response
	if emailCfg.SMTPPass != "" {
		emailCfg.SMTPPass = "••••••"
	}
	writeJSON(w, http.StatusOK, emailCfg)
}

func (h *SettingsHandler) GetCompanySettings(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var companyCfg model.CompanySettings
	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		return h.getSettingsSection(r.Context(), tx, tenantID, "company", &companyCfg)
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load settings")
		return
	}

	writeJSON(w, http.StatusOK, companyCfg)
}

func (h *SettingsHandler) UpdateCompanySettings(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var companyCfg model.CompanySettings
	if err := json.NewDecoder(r.Body).Decode(&companyCfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		if err := h.updateSettingsSection(r.Context(), tx, tenantID, "company", companyCfg); err != nil {
			return err
		}
		return h.auditRepo.Log(r.Context(), tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "settings.company_updated",
			EntityType: "settings",
			EntityID:   tenantID,
			IPAddress:  clientIP(r),
		})
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save settings")
		return
	}

	writeJSON(w, http.StatusOK, companyCfg)
}

func (h *SettingsHandler) GetOrderStatuses(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	config := model.DefaultOrderStatusConfig()
	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		var loaded model.OrderStatusConfig
		if err := h.getSettingsSection(r.Context(), tx, tenantID, "order_statuses", &loaded); err != nil {
			return err
		}
		if len(loaded.Statuses) > 0 {
			config = loaded
		}
		return nil
	})

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load order statuses")
		return
	}

	writeJSON(w, http.StatusOK, config)
}

func (h *SettingsHandler) UpdateOrderStatuses(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var config model.OrderStatusConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate: no empty keys, no duplicate keys
	keys := make(map[string]bool)
	for _, s := range config.Statuses {
		if s.Key == "" || s.Label == "" {
			writeError(w, http.StatusBadRequest, "status key and label are required")
			return
		}
		if keys[s.Key] {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("duplicate status key: %s", s.Key))
			return
		}
		keys[s.Key] = true
	}

	// Validate: all transition targets reference existing statuses
	for from, targets := range config.Transitions {
		if !keys[from] {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("transition from unknown status: %s", from))
			return
		}
		for _, to := range targets {
			if !keys[to] {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("transition to unknown status: %s", to))
				return
			}
		}
	}

	actorID := middleware.UserIDFromContext(r.Context())
	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		if err := h.updateSettingsSection(r.Context(), tx, tenantID, "order_statuses", config); err != nil {
			return err
		}
		return h.auditRepo.Log(r.Context(), tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "settings.order_statuses_updated",
			EntityType: "settings",
			EntityID:   tenantID,
			IPAddress:  clientIP(r),
		})
	})

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save order statuses")
		return
	}

	writeJSON(w, http.StatusOK, config)
}

func (h *SettingsHandler) GetCustomFields(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	config := model.CustomFieldsConfig{Fields: []model.CustomFieldDef{}}
	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		var loaded model.CustomFieldsConfig
		if err := h.getSettingsSection(r.Context(), tx, tenantID, "custom_fields", &loaded); err != nil {
			return err
		}
		if len(loaded.Fields) > 0 {
			config = loaded
		}
		return nil
	})

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load custom fields")
		return
	}

	writeJSON(w, http.StatusOK, config)
}

func (h *SettingsHandler) UpdateCustomFields(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var config model.CustomFieldsConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate: no empty keys or labels, no duplicate keys, valid types
	keys := make(map[string]bool)
	for _, f := range config.Fields {
		if f.Key == "" || f.Label == "" {
			writeError(w, http.StatusBadRequest, "field key and label are required")
			return
		}
		if keys[f.Key] {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("duplicate field key: %s", f.Key))
			return
		}
		keys[f.Key] = true
		if !model.IsValidFieldType(f.Type) {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid field type: %s", f.Type))
			return
		}
		if f.Type == "select" && len(f.Options) == 0 {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("select field %q must have at least 1 option", f.Key))
			return
		}
	}

	actorID := middleware.UserIDFromContext(r.Context())
	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		if err := h.updateSettingsSection(r.Context(), tx, tenantID, "custom_fields", config); err != nil {
			return err
		}
		return h.auditRepo.Log(r.Context(), tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "settings.custom_fields_updated",
			EntityType: "settings",
			EntityID:   tenantID,
			IPAddress:  clientIP(r),
		})
	})

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save custom fields")
		return
	}

	writeJSON(w, http.StatusOK, config)
}

func (h *SettingsHandler) GetProductCategories(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	config := model.DefaultProductCategoriesConfig()
	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		var loaded model.ProductCategoriesConfig
		if err := h.getSettingsSection(r.Context(), tx, tenantID, "product_categories", &loaded); err != nil {
			return err
		}
		if len(loaded.Categories) > 0 {
			config = loaded
		}
		return nil
	})

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load product categories")
		return
	}

	writeJSON(w, http.StatusOK, config)
}

func (h *SettingsHandler) UpdateProductCategories(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var config model.ProductCategoriesConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate: no empty keys, no empty labels, no duplicate keys
	keys := make(map[string]bool)
	for _, c := range config.Categories {
		if c.Key == "" || c.Label == "" {
			writeError(w, http.StatusBadRequest, "category key and label are required")
			return
		}
		if keys[c.Key] {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("duplicate category key: %s", c.Key))
			return
		}
		keys[c.Key] = true
	}

	actorID := middleware.UserIDFromContext(r.Context())
	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		if err := h.updateSettingsSection(r.Context(), tx, tenantID, "product_categories", config); err != nil {
			return err
		}
		return h.auditRepo.Log(r.Context(), tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "settings.product_categories_updated",
			EntityType: "settings",
			EntityID:   tenantID,
			IPAddress:  clientIP(r),
		})
	})

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save product categories")
		return
	}

	writeJSON(w, http.StatusOK, config)
}

func (h *SettingsHandler) GetWebhooks(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	config := model.DefaultWebhookConfig()
	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		var loaded model.WebhookConfig
		if err := h.getSettingsSection(r.Context(), tx, tenantID, "webhooks", &loaded); err != nil {
			return err
		}
		if len(loaded.Endpoints) > 0 {
			config = loaded
		}
		return nil
	})

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load webhook settings")
		return
	}

	writeJSON(w, http.StatusOK, config)
}

func (h *SettingsHandler) UpdateWebhooks(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var config model.WebhookConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate: each endpoint must have non-empty name, URL, at least one event; no duplicate IDs
	ids := make(map[string]bool)
	for _, ep := range config.Endpoints {
		if ep.Name == "" {
			writeError(w, http.StatusBadRequest, "endpoint name is required")
			return
		}
		if ep.URL == "" {
			writeError(w, http.StatusBadRequest, "endpoint URL is required")
			return
		}
		if len(ep.Events) == 0 {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("endpoint %q must have at least one event", ep.Name))
			return
		}
		if ep.ID != "" {
			if ids[ep.ID] {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("duplicate endpoint ID: %s", ep.ID))
				return
			}
			ids[ep.ID] = true
		}

		// SSRF protection: reject private/internal webhook URLs
		if isPrivateWebhookURL(ep.URL) {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("endpoint URL %q resolves to a private/internal address", ep.URL))
			return
		}
	}

	actorID := middleware.UserIDFromContext(r.Context())
	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		if err := h.updateSettingsSection(r.Context(), tx, tenantID, "webhooks", config); err != nil {
			return err
		}
		return h.auditRepo.Log(r.Context(), tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "settings.webhooks_updated",
			EntityType: "settings",
			EntityID:   tenantID,
			IPAddress:  clientIP(r),
		})
	})

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save webhook settings")
		return
	}

	writeJSON(w, http.StatusOK, config)
}

func (h *SettingsHandler) SendTestEmail(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var req struct {
		ToEmail string `json:"to_email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ToEmail == "" {
		writeError(w, http.StatusBadRequest, "to_email is required")
		return
	}

	// Load email settings
	var emailCfg model.EmailSettings
	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		if err := h.getSettingsSection(r.Context(), tx, tenantID, "email", &emailCfg); err != nil {
			return err
		}
		if emailCfg.SMTPHost == "" {
			return fmt.Errorf("email settings not configured")
		}
		return nil
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.emailService.SendTestEmail(r.Context(), emailCfg, req.ToEmail); err != nil {
		writeError(w, http.StatusBadGateway, "failed to send test email: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Test email sent successfully"})
}

func (h *SettingsHandler) GetInvoicingSettings(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var invoicingCfg map[string]any
	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		return h.getSettingsSection(r.Context(), tx, tenantID, "invoicing", &invoicingCfg)
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load invoicing settings")
		return
	}

	if invoicingCfg == nil {
		invoicingCfg = map[string]any{
			"provider":               "",
			"auto_create_on_status":  []string{},
			"default_tax_rate":       23,
			"payment_days":           14,
			"credentials":            map[string]any{},
		}
	}

	writeJSON(w, http.StatusOK, invoicingCfg)
}

func (h *SettingsHandler) UpdateInvoicingSettings(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var invoicingCfg map[string]any
	if err := json.NewDecoder(r.Body).Decode(&invoicingCfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		if err := h.updateSettingsSection(r.Context(), tx, tenantID, "invoicing", invoicingCfg); err != nil {
			return err
		}
		return h.auditRepo.Log(r.Context(), tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "settings.invoicing_updated",
			EntityType: "settings",
			EntityID:   tenantID,
			IPAddress:  clientIP(r),
		})
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save invoicing settings")
		return
	}

	writeJSON(w, http.StatusOK, invoicingCfg)
}

// isPrivateWebhookURL checks whether a URL resolves to a private/internal IP address.
func isPrivateWebhookURL(urlStr string) bool {
	u, err := url.Parse(urlStr)
	if err != nil {
		return true
	}

	hostname := u.Hostname()
	if hostname == "" {
		return true
	}

	ips, err := net.LookupHost(hostname)
	if err != nil {
		return true
	}

	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	}

	var cidrs []*net.IPNet
	for _, cidr := range privateRanges {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		cidrs = append(cidrs, ipNet)
	}

	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		for _, cidr := range cidrs {
			if cidr.Contains(ip) {
				return true
			}
		}
	}

	return false
}
