package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// WebhookRepository handles persistence for webhook events.
type WebhookRepository struct{}

// NewWebhookRepository creates a new WebhookRepository.
func NewWebhookRepository() *WebhookRepository {
	return &WebhookRepository{}
}

// Create inserts a new webhook event.
func (r *WebhookRepository) Create(ctx context.Context, tx pgx.Tx, event *model.WebhookEvent) error {
	return tx.QueryRow(ctx,
		`INSERT INTO webhook_events (id, tenant_id, provider, event_type, payload, status)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING created_at`,
		event.ID, event.TenantID, event.Provider, event.EventType, event.Payload, event.Status,
	).Scan(&event.CreatedAt)
}
