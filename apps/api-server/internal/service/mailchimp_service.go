package service

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// MailchimpSettings holds per-tenant Mailchimp configuration.
type MailchimpSettings struct {
	APIKey  string `json:"api_key"`
	ListID  string `json:"list_id"`
	Enabled bool   `json:"enabled"`
}

// MailchimpService integrates with Mailchimp API v3.
type MailchimpService struct {
	tenantRepo   repository.TenantRepo
	customerRepo repository.CustomerRepo
	pool         *pgxpool.Pool
	httpClient   *http.Client
	logger       *slog.Logger
}

func NewMailchimpService(tenantRepo repository.TenantRepo, customerRepo repository.CustomerRepo, pool *pgxpool.Pool, logger *slog.Logger) *MailchimpService {
	return &MailchimpService{
		tenantRepo:   tenantRepo,
		customerRepo: customerRepo,
		pool:         pool,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// GetSettings loads Mailchimp settings from tenant settings.
func (s *MailchimpService) GetSettings(ctx context.Context, tenantID uuid.UUID) (*MailchimpSettings, error) {
	var settings MailchimpSettings
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		raw, err := s.tenantRepo.GetSettings(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		if raw == nil {
			return nil
		}
		var all map[string]json.RawMessage
		if err := json.Unmarshal(raw, &all); err != nil {
			return nil
		}
		if mc, ok := all["mailchimp"]; ok {
			json.Unmarshal(mc, &settings)
		}
		return nil
	})
	return &settings, err
}

func (s *MailchimpService) getDataCenter(apiKey string) string {
	parts := strings.Split(apiKey, "-")
	if len(parts) == 2 {
		return parts[1]
	}
	return "us1"
}

func (s *MailchimpService) doRequest(ctx context.Context, method, url, apiKey string, body any) ([]byte, int, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}
	req.SetBasicAuth("anystring", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("mailchimp request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}

	return respBytes, resp.StatusCode, nil
}

func emailHash(email string) string {
	h := md5.Sum([]byte(strings.ToLower(strings.TrimSpace(email))))
	return fmt.Sprintf("%x", h)
}

// SyncCustomer adds or updates a customer in the Mailchimp audience.
func (s *MailchimpService) SyncCustomer(ctx context.Context, tenantID uuid.UUID, customerEmail, customerName string) error {
	settings, err := s.GetSettings(ctx, tenantID)
	if err != nil {
		return err
	}
	if !settings.Enabled || settings.APIKey == "" || settings.ListID == "" {
		return fmt.Errorf("mailchimp nie jest skonfigurowany")
	}

	dc := s.getDataCenter(settings.APIKey)
	hash := emailHash(customerEmail)
	url := fmt.Sprintf("https://%s.api.mailchimp.com/3.0/lists/%s/members/%s", dc, settings.ListID, hash)

	names := strings.SplitN(customerName, " ", 2)
	firstName := customerName
	lastName := ""
	if len(names) == 2 {
		firstName = names[0]
		lastName = names[1]
	}

	payload := map[string]any{
		"email_address": customerEmail,
		"status_if_new": "subscribed",
		"merge_fields": map[string]string{
			"FNAME": firstName,
			"LNAME": lastName,
		},
	}

	_, statusCode, err := s.doRequest(ctx, http.MethodPut, url, settings.APIKey, payload)
	if err != nil {
		return err
	}
	if statusCode >= 400 {
		return fmt.Errorf("mailchimp sync failed (status %d)", statusCode)
	}

	return nil
}

// SyncAllCustomers syncs all customers with email addresses to Mailchimp.
func (s *MailchimpService) SyncAllCustomers(ctx context.Context, tenantID uuid.UUID) (int, int, error) {
	var synced, failed int

	settings, err := s.GetSettings(ctx, tenantID)
	if err != nil {
		return 0, 0, err
	}
	if !settings.Enabled || settings.APIKey == "" || settings.ListID == "" {
		return 0, 0, fmt.Errorf("mailchimp nie jest skonfigurowany")
	}

	// Fetch all customers
	type customerRow struct {
		Email string
		Name  string
	}
	var customers []customerRow

	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT email, name FROM customers WHERE email IS NOT NULL AND email != '' ORDER BY created_at`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var c customerRow
			if err := rows.Scan(&c.Email, &c.Name); err != nil {
				return err
			}
			customers = append(customers, c)
		}
		return rows.Err()
	})
	if err != nil {
		return 0, 0, err
	}

	for _, c := range customers {
		if err := s.SyncCustomer(ctx, tenantID, c.Email, c.Name); err != nil {
			s.logger.Warn("mailchimp sync failed for customer", "email", c.Email, "error", err)
			failed++
		} else {
			synced++
		}
	}

	return synced, failed, nil
}

// CreateCampaign creates a Mailchimp email campaign.
func (s *MailchimpService) CreateCampaign(ctx context.Context, tenantID uuid.UUID, name, subject, content string) (string, error) {
	settings, err := s.GetSettings(ctx, tenantID)
	if err != nil {
		return "", err
	}
	if !settings.Enabled || settings.APIKey == "" || settings.ListID == "" {
		return "", fmt.Errorf("mailchimp nie jest skonfigurowany")
	}

	dc := s.getDataCenter(settings.APIKey)

	// 1. Create campaign
	campaignPayload := map[string]any{
		"type": "regular",
		"recipients": map[string]any{
			"list_id": settings.ListID,
		},
		"settings": map[string]any{
			"subject_line": subject,
			"title":        name,
			"from_name":    name,
			"reply_to":     "noreply@example.com",
		},
	}

	respBytes, statusCode, err := s.doRequest(ctx, http.MethodPost,
		fmt.Sprintf("https://%s.api.mailchimp.com/3.0/campaigns", dc),
		settings.APIKey, campaignPayload)
	if err != nil {
		return "", err
	}
	if statusCode >= 400 {
		return "", fmt.Errorf("mailchimp create campaign failed (status %d): %s", statusCode, string(respBytes))
	}

	var campaign struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBytes, &campaign); err != nil {
		return "", fmt.Errorf("parse campaign response: %w", err)
	}

	// 2. Set campaign content
	contentPayload := map[string]any{
		"html": content,
	}
	_, statusCode, err = s.doRequest(ctx, http.MethodPut,
		fmt.Sprintf("https://%s.api.mailchimp.com/3.0/campaigns/%s/content", dc, campaign.ID),
		settings.APIKey, contentPayload)
	if err != nil {
		return campaign.ID, err
	}
	if statusCode >= 400 {
		return campaign.ID, fmt.Errorf("mailchimp set content failed (status %d)", statusCode)
	}

	return campaign.ID, nil
}
