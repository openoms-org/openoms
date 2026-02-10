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
	tenantRepo   *repository.TenantRepository
	emailService *service.EmailService
	pool         *pgxpool.Pool
}

func NewSettingsHandler(tenantRepo *repository.TenantRepository, emailService *service.EmailService, pool *pgxpool.Pool) *SettingsHandler {
	return &SettingsHandler{tenantRepo: tenantRepo, emailService: emailService, pool: pool}
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

		return h.tenantRepo.UpdateSettings(r.Context(), tx, tenantID, newSettings)
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

		return h.tenantRepo.UpdateSettings(r.Context(), tx, tenantID, newSettings)
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save settings")
		return
	}

	writeJSON(w, http.StatusOK, companyCfg)
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
