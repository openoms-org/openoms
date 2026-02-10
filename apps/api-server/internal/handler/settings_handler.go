package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

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

func (h *SettingsHandler) GetEmailSettings(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var settings json.RawMessage
	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		settings, err = h.tenantRepo.GetSettings(r.Context(), tx, tenantID)
		return err
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load settings")
		return
	}

	// Parse full settings, extract email part
	var allSettings map[string]json.RawMessage
	if err := json.Unmarshal(settings, &allSettings); err != nil {
		// Settings is empty or not a map — return defaults
		writeJSON(w, http.StatusOK, model.EmailSettings{
			SMTPPort: 587,
			NotifyOn: []string{"confirmed", "shipped", "delivered", "cancelled", "refunded"},
		})
		return
	}

	emailRaw, ok := allSettings["email"]
	if !ok {
		writeJSON(w, http.StatusOK, model.EmailSettings{
			SMTPPort: 587,
			NotifyOn: []string{"confirmed", "shipped", "delivered", "cancelled", "refunded"},
		})
		return
	}

	var emailCfg model.EmailSettings
	if err := json.Unmarshal(emailRaw, &emailCfg); err != nil {
		writeJSON(w, http.StatusOK, model.EmailSettings{
			SMTPPort: 587,
			NotifyOn: []string{"confirmed", "shipped", "delivered", "cancelled", "refunded"},
		})
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
		// Load existing settings
		existing, err := h.tenantRepo.GetSettings(r.Context(), tx, tenantID)
		if err != nil {
			return err
		}

		var allSettings map[string]json.RawMessage
		if err := json.Unmarshal(existing, &allSettings); err != nil {
			allSettings = make(map[string]json.RawMessage)
		}

		// If password is masked, keep the existing one
		if emailCfg.SMTPPass == "••••••" {
			var oldEmail model.EmailSettings
			if oldRaw, ok := allSettings["email"]; ok {
				json.Unmarshal(oldRaw, &oldEmail)
				emailCfg.SMTPPass = oldEmail.SMTPPass
			}
		}

		emailJSON, err := json.Marshal(emailCfg)
		if err != nil {
			return err
		}
		allSettings["email"] = emailJSON

		newSettings, err := json.Marshal(allSettings)
		if err != nil {
			return err
		}

		if err := h.tenantRepo.UpdateSettings(r.Context(), tx, tenantID, newSettings); err != nil {
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

	var settings json.RawMessage
	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		settings, err = h.tenantRepo.GetSettings(r.Context(), tx, tenantID)
		return err
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load settings")
		return
	}

	// Parse full settings, extract company part
	var allSettings map[string]json.RawMessage
	if err := json.Unmarshal(settings, &allSettings); err != nil {
		writeJSON(w, http.StatusOK, model.CompanySettings{})
		return
	}

	companyRaw, ok := allSettings["company"]
	if !ok {
		writeJSON(w, http.StatusOK, model.CompanySettings{})
		return
	}

	var companyCfg model.CompanySettings
	if err := json.Unmarshal(companyRaw, &companyCfg); err != nil {
		writeJSON(w, http.StatusOK, model.CompanySettings{})
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
		// Load existing settings
		existing, err := h.tenantRepo.GetSettings(r.Context(), tx, tenantID)
		if err != nil {
			return err
		}

		var allSettings map[string]json.RawMessage
		if err := json.Unmarshal(existing, &allSettings); err != nil {
			allSettings = make(map[string]json.RawMessage)
		}

		companyJSON, err := json.Marshal(companyCfg)
		if err != nil {
			return err
		}
		allSettings["company"] = companyJSON

		newSettings, err := json.Marshal(allSettings)
		if err != nil {
			return err
		}

		if err := h.tenantRepo.UpdateSettings(r.Context(), tx, tenantID, newSettings); err != nil {
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

	var config model.OrderStatusConfig
	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		settings, err := h.tenantRepo.GetSettings(r.Context(), tx, tenantID)
		if err != nil {
			return err
		}

		if settings != nil {
			var allSettings map[string]json.RawMessage
			if err := json.Unmarshal(settings, &allSettings); err == nil {
				if raw, ok := allSettings["order_statuses"]; ok {
					if err := json.Unmarshal(raw, &config); err == nil && len(config.Statuses) > 0 {
						return nil
					}
				}
			}
		}

		config = model.DefaultOrderStatusConfig()
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
		existing, err := h.tenantRepo.GetSettings(r.Context(), tx, tenantID)
		if err != nil {
			return err
		}

		var allSettings map[string]json.RawMessage
		if err := json.Unmarshal(existing, &allSettings); err != nil {
			allSettings = make(map[string]json.RawMessage)
		}

		configJSON, err := json.Marshal(config)
		if err != nil {
			return err
		}
		allSettings["order_statuses"] = configJSON

		newSettings, err := json.Marshal(allSettings)
		if err != nil {
			return err
		}

		if err := h.tenantRepo.UpdateSettings(r.Context(), tx, tenantID, newSettings); err != nil {
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

	var config model.CustomFieldsConfig
	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		settings, err := h.tenantRepo.GetSettings(r.Context(), tx, tenantID)
		if err != nil {
			return err
		}

		if settings != nil {
			var allSettings map[string]json.RawMessage
			if err := json.Unmarshal(settings, &allSettings); err == nil {
				if raw, ok := allSettings["custom_fields"]; ok {
					if err := json.Unmarshal(raw, &config); err == nil && len(config.Fields) > 0 {
						return nil
					}
				}
			}
		}

		config = model.CustomFieldsConfig{Fields: []model.CustomFieldDef{}}
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
		existing, err := h.tenantRepo.GetSettings(r.Context(), tx, tenantID)
		if err != nil {
			return err
		}

		var allSettings map[string]json.RawMessage
		if err := json.Unmarshal(existing, &allSettings); err != nil {
			allSettings = make(map[string]json.RawMessage)
		}

		configJSON, err := json.Marshal(config)
		if err != nil {
			return err
		}
		allSettings["custom_fields"] = configJSON

		newSettings, err := json.Marshal(allSettings)
		if err != nil {
			return err
		}

		if err := h.tenantRepo.UpdateSettings(r.Context(), tx, tenantID, newSettings); err != nil {
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

	var config model.ProductCategoriesConfig
	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		settings, err := h.tenantRepo.GetSettings(r.Context(), tx, tenantID)
		if err != nil {
			return err
		}

		if settings != nil {
			var allSettings map[string]json.RawMessage
			if err := json.Unmarshal(settings, &allSettings); err == nil {
				if raw, ok := allSettings["product_categories"]; ok {
					if err := json.Unmarshal(raw, &config); err == nil && len(config.Categories) > 0 {
						return nil
					}
				}
			}
		}

		config = model.DefaultProductCategoriesConfig()
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
		existing, err := h.tenantRepo.GetSettings(r.Context(), tx, tenantID)
		if err != nil {
			return err
		}

		var allSettings map[string]json.RawMessage
		if err := json.Unmarshal(existing, &allSettings); err != nil {
			allSettings = make(map[string]json.RawMessage)
		}

		configJSON, err := json.Marshal(config)
		if err != nil {
			return err
		}
		allSettings["product_categories"] = configJSON

		newSettings, err := json.Marshal(allSettings)
		if err != nil {
			return err
		}

		if err := h.tenantRepo.UpdateSettings(r.Context(), tx, tenantID, newSettings); err != nil {
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

	var config model.WebhookConfig
	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		settings, err := h.tenantRepo.GetSettings(r.Context(), tx, tenantID)
		if err != nil {
			return err
		}

		if settings != nil {
			var allSettings map[string]json.RawMessage
			if err := json.Unmarshal(settings, &allSettings); err == nil {
				if raw, ok := allSettings["webhooks"]; ok {
					if err := json.Unmarshal(raw, &config); err == nil && len(config.Endpoints) > 0 {
						return nil
					}
				}
			}
		}

		config = model.DefaultWebhookConfig()
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
	}

	actorID := middleware.UserIDFromContext(r.Context())
	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := h.tenantRepo.GetSettings(r.Context(), tx, tenantID)
		if err != nil {
			return err
		}

		var allSettings map[string]json.RawMessage
		if err := json.Unmarshal(existing, &allSettings); err != nil {
			allSettings = make(map[string]json.RawMessage)
		}

		configJSON, err := json.Marshal(config)
		if err != nil {
			return err
		}
		allSettings["webhooks"] = configJSON

		newSettings, err := json.Marshal(allSettings)
		if err != nil {
			return err
		}

		if err := h.tenantRepo.UpdateSettings(r.Context(), tx, tenantID, newSettings); err != nil {
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
		settings, err := h.tenantRepo.GetSettings(r.Context(), tx, tenantID)
		if err != nil {
			return err
		}

		var allSettings map[string]json.RawMessage
		if err := json.Unmarshal(settings, &allSettings); err != nil {
			return err
		}

		emailRaw, ok := allSettings["email"]
		if !ok {
			return fmt.Errorf("email settings not configured")
		}
		return json.Unmarshal(emailRaw, &emailCfg)
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
