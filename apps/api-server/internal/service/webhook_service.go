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
	ErrInvalidSignature = errors.New("invalid webhook signature")
	ErrUnknownProvider  = errors.New("unknown webhook provider")
)

type WebhookService struct {
	webhookRepo          *repository.WebhookRepository
	pool                 *pgxpool.Pool
	allegroWebhookSecret string
	inpostWebhookSecret  string
}

func NewWebhookService(
	webhookRepo *repository.WebhookRepository,
	pool *pgxpool.Pool,
	allegroWebhookSecret string,
	inpostWebhookSecret string,
) *WebhookService {
	return &WebhookService{
		webhookRepo:          webhookRepo,
		pool:                 pool,
		allegroWebhookSecret: allegroWebhookSecret,
		inpostWebhookSecret:  inpostWebhookSecret,
	}
}

func (s *WebhookService) Receive(ctx context.Context, tenantID uuid.UUID, provider string, signature string, body []byte) (*model.WebhookEvent, error) {
	secret, err := s.secretForProvider(provider)
	if err != nil {
		return nil, err
	}

	if secret != "" {
		if !verifyHMAC(body, signature, secret) {
			return nil, ErrInvalidSignature
		}
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
		return s.webhookRepo.Create(ctx, tx, event)
	})
	if err != nil {
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
