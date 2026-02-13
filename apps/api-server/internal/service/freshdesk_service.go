package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// FreshdeskSettings holds per-tenant Freshdesk configuration.
type FreshdeskSettings struct {
	Domain  string `json:"domain"`
	APIKey  string `json:"api_key"`
	Enabled bool   `json:"enabled"`
}

// FreshdeskTicket represents a Freshdesk ticket.
type FreshdeskTicket struct {
	ID          int64  `json:"id"`
	Subject     string `json:"subject"`
	Description string `json:"description,omitempty"`
	Status      int    `json:"status"`
	Priority    int    `json:"priority"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// FreshdeskService integrates with Freshdesk API v2.
type FreshdeskService struct {
	tenantRepo repository.TenantRepo
	orderRepo  repository.OrderRepo
	pool       *pgxpool.Pool
	httpClient *http.Client
	logger     *slog.Logger
}

func NewFreshdeskService(tenantRepo repository.TenantRepo, orderRepo repository.OrderRepo, pool *pgxpool.Pool, logger *slog.Logger) *FreshdeskService {
	return &FreshdeskService{
		tenantRepo: tenantRepo,
		orderRepo:  orderRepo,
		pool:       pool,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// GetSettings loads Freshdesk settings from tenant settings.
func (s *FreshdeskService) GetSettings(ctx context.Context, tenantID uuid.UUID) (*FreshdeskSettings, error) {
	var settings FreshdeskSettings
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
		if fd, ok := all["freshdesk"]; ok {
			json.Unmarshal(fd, &settings)
		}
		return nil
	})
	return &settings, err
}

func (s *FreshdeskService) doRequest(ctx context.Context, method, url, apiKey string, body any) ([]byte, int, error) {
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
	req.SetBasicAuth(apiKey, "X")
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("freshdesk request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}

	return respBytes, resp.StatusCode, nil
}

// CreateTicket creates a Freshdesk support ticket linked to an order.
func (s *FreshdeskService) CreateTicket(ctx context.Context, tenantID uuid.UUID, orderID uuid.UUID, subject, description, email string) (*FreshdeskTicket, error) {
	settings, err := s.GetSettings(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if !settings.Enabled || settings.APIKey == "" || settings.Domain == "" {
		return nil, ErrFreshdeskNotConfigured
	}

	url := fmt.Sprintf("https://%s.freshdesk.com/api/v2/tickets", settings.Domain)

	payload := map[string]any{
		"subject":     subject,
		"description": description,
		"email":       email,
		"priority":    1,
		"status":      2,
		"tags":        []string{"openoms", "order-" + orderID.String()},
		"custom_fields": map[string]any{
			"cf_order_id": orderID.String(),
		},
	}

	respBytes, statusCode, err := s.doRequest(ctx, http.MethodPost, url, settings.APIKey, payload)
	if err != nil {
		return nil, err
	}
	if statusCode >= 400 {
		return nil, fmt.Errorf("freshdesk create ticket failed (status %d): %s", statusCode, string(respBytes))
	}

	var ticket FreshdeskTicket
	if err := json.Unmarshal(respBytes, &ticket); err != nil {
		return nil, fmt.Errorf("parse ticket response: %w", err)
	}

	return &ticket, nil
}

// GetTickets lists Freshdesk tickets associated with an order.
func (s *FreshdeskService) GetTickets(ctx context.Context, tenantID uuid.UUID, orderID uuid.UUID) ([]FreshdeskTicket, error) {
	settings, err := s.GetSettings(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if !settings.Enabled || settings.APIKey == "" || settings.Domain == "" {
		return nil, ErrFreshdeskNotConfigured
	}

	// Search for tickets tagged with the order ID
	url := fmt.Sprintf("https://%s.freshdesk.com/api/v2/search/tickets?query=\"tag:'order-%s'\"", settings.Domain, orderID.String())

	respBytes, statusCode, err := s.doRequest(ctx, http.MethodGet, url, settings.APIKey, nil)
	if err != nil {
		return nil, err
	}
	if statusCode >= 400 {
		return nil, fmt.Errorf("freshdesk search tickets failed (status %d): %s", statusCode, string(respBytes))
	}

	var result struct {
		Results []FreshdeskTicket `json:"results"`
	}
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, fmt.Errorf("parse tickets response: %w", err)
	}

	return result.Results, nil
}

// ListAllTickets returns recent tickets from Freshdesk.
func (s *FreshdeskService) ListAllTickets(ctx context.Context, tenantID uuid.UUID) ([]FreshdeskTicket, error) {
	settings, err := s.GetSettings(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if !settings.Enabled || settings.APIKey == "" || settings.Domain == "" {
		return nil, ErrFreshdeskNotConfigured
	}

	url := fmt.Sprintf("https://%s.freshdesk.com/api/v2/tickets?per_page=30&order_by=created_at&order_type=desc", settings.Domain)

	respBytes, statusCode, err := s.doRequest(ctx, http.MethodGet, url, settings.APIKey, nil)
	if err != nil {
		return nil, err
	}
	if statusCode >= 400 {
		return nil, fmt.Errorf("freshdesk list tickets failed (status %d): %s", statusCode, string(respBytes))
	}

	var tickets []FreshdeskTicket
	if err := json.Unmarshal(respBytes, &tickets); err != nil {
		return nil, fmt.Errorf("parse tickets response: %w", err)
	}

	return tickets, nil
}
