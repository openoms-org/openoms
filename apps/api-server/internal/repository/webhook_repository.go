package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

type WebhookRepository struct{}

func NewWebhookRepository() *WebhookRepository {
	return &WebhookRepository{}
}

func (r *WebhookRepository) Create(ctx context.Context, tx pgx.Tx, event *model.WebhookEvent) error {
	return tx.QueryRow(ctx,
		`INSERT INTO webhook_events (id, tenant_id, provider, event_type, payload, status)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING created_at`,
		event.ID, event.TenantID, event.Provider, event.EventType, event.Payload, event.Status,
	).Scan(&event.CreatedAt)
}

func (r *WebhookRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.WebhookEvent, error) {
	var e model.WebhookEvent
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, provider, event_type, payload, status, created_at
		 FROM webhook_events WHERE id = $1`, id,
	).Scan(&e.ID, &e.TenantID, &e.Provider, &e.EventType, &e.Payload, &e.Status, &e.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find webhook event: %w", err)
	}
	return &e, nil
}
