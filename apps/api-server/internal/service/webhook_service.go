package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	// ErrInvalidSignature is returned when a webhook HMAC signature does not match.
	ErrInvalidSignature = errors.New("invalid webhook signature")
	// ErrUnknownProvider is returned when a webhook provider name is not recognised.
	ErrUnknownProvider = errors.New("unknown webhook provider")
	// ErrWebhookSecretNotConfigured is returned when a provider requires signatures but has no configured secret.
	ErrWebhookSecretNotConfigured = errors.New("webhook secret not configured")
	// ErrUnknownTenant is returned when the tenant_id in a legacy webhook URL does not exist.
	ErrUnknownTenant = errors.New("unknown tenant")
)

// WebhookService handles inbound webhook event verification and storage.
type WebhookService struct {
	webhookRepo          repository.WebhookRepo
	tenantRepo           repository.TenantRepo
	pool                 *pgxpool.Pool
	allegroWebhookSecret string
	inpostWebhookSecret  string
}

// NewWebhookService creates a new WebhookService.
func NewWebhookService(
	webhookRepo repository.WebhookRepo,
	tenantRepo repository.TenantRepo,
	pool *pgxpool.Pool,
	allegroWebhookSecret string,
	inpostWebhookSecret string,
) *WebhookService {
	return &WebhookService{
		webhookRepo:          webhookRepo,
		tenantRepo:           tenantRepo,
		pool:                 pool,
		allegroWebhookSecret: allegroWebhookSecret,
		inpostWebhookSecret:  inpostWebhookSecret,
	}
}

// ensureTenantExists verifies the tenant_id taken from the (legacy, shared-secret)
// webhook URL refers to a real tenant. Without this an attacker holding the
// deployment secret could persist events under an arbitrary tenant_id
// (wrong-tenant data injection). Runs inside a WithTenant transaction so RLS
// scopes the lookup to the claimed tenant.
func (s *WebhookService) ensureTenantExists(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) error {
	tenant, err := s.tenantRepo.FindByID(ctx, tx, tenantID)
	if err != nil {
		return err
	}
	if tenant == nil {
		return ErrUnknownTenant
	}
	return nil
}

// Receive verifies the signature of an inbound webhook and persists the event.
func (s *WebhookService) Receive(ctx context.Context, tenantID uuid.UUID, provider string, signature string, body []byte) (*model.WebhookEvent, error) {
	secret, err := s.secretForProvider(provider)
	if err != nil {
		return nil, err
	}
	if secret == "" {
		return nil, ErrWebhookSecretNotConfigured
	}

	if !verifyHMAC(body, signature, secret) {
		return nil, ErrInvalidSignature
	}

	eventType := extractEventType(provider, body)

	event := &model.WebhookEvent{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Provider:  provider,
		EventType: eventType,
		Payload:   json.RawMessage(body),
		Status:    "received",
	}

	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.ensureTenantExists(ctx, tx, tenantID); err != nil {
			return err
		}
		return s.webhookRepo.Create(ctx, tx, event)
	})
	if err != nil {
		if errors.Is(err, ErrUnknownTenant) {
			return nil, ErrUnknownTenant
		}
		return nil, fmt.Errorf("store webhook event: %w", err)
	}

	return event, nil
}

func (s *WebhookService) secretForProvider(provider string) (string, error) {
	switch provider {
	case "allegro":
		return s.allegroWebhookSecret, nil
	case "inpost":
		return s.inpostWebhookSecret, nil
	default:
		return "", ErrUnknownProvider
	}
}

func verifyHMAC(body []byte, signature string, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func extractEventType(provider string, body []byte) string {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return "unknown"
	}

	switch provider {
	case "allegro":
		if t, ok := payload["type"]; ok {
			var eventType string
			if err := json.Unmarshal(t, &eventType); err == nil {
				return eventType
			}
		}
	case "inpost":
		if t, ok := payload["event"]; ok {
			var eventType string
			if err := json.Unmarshal(t, &eventType); err == nil {
				return eventType
			}
		}
	}

	return "unknown"
}
