package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// WebhookDeliveryRepository handles persistence for webhook delivery records.
type WebhookDeliveryRepository struct{}

// NewWebhookDeliveryRepository creates a new WebhookDeliveryRepository.
func NewWebhookDeliveryRepository() *WebhookDeliveryRepository {
	return &WebhookDeliveryRepository{}
}

// Create inserts a new webhook delivery record.
func (r *WebhookDeliveryRepository) Create(ctx context.Context, tx pgx.Tx, delivery *model.WebhookDelivery) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO webhook_deliveries (id, tenant_id, url, event_type, payload, status, response_code, error, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		delivery.ID, delivery.TenantID, delivery.URL, delivery.EventType, delivery.Payload,
		delivery.Status, delivery.ResponseCode, delivery.Error, delivery.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create webhook delivery: %w", err)
	}
	return nil
}

// List returns a paginated list of webhook deliveries matching the filter.
func (r *WebhookDeliveryRepository) List(ctx context.Context, tx pgx.Tx, filter model.WebhookDeliveryFilter) ([]model.WebhookDelivery, int, error) {
	qb := NewQueryBuilder()

	if filter.EventType != nil {
		qb.Add("event_type = $%d", *filter.EventType)
	}
	if filter.Status != nil {
		qb.Add("status = $%d", *filter.Status)
	}

	where := qb.WhereClause()

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM webhook_deliveries %s", where)
	var total int
	if err := tx.QueryRow(ctx, countQuery, qb.Args()...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count webhook deliveries: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}

	argIdx := qb.AddArgs(limit, filter.Offset)
	query := fmt.Sprintf(
		`SELECT id, tenant_id, url, event_type, payload, status, response_code, error, created_at
		 FROM webhook_deliveries
		 %s
		 ORDER BY created_at DESC
		 LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
	)

	rows, err := tx.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, 0, fmt.Errorf("list webhook deliveries: %w", err)
	}
	defer rows.Close()

	var deliveries []model.WebhookDelivery
	for rows.Next() {
		var d model.WebhookDelivery
		if err := rows.Scan(&d.ID, &d.TenantID, &d.URL, &d.EventType, &d.Payload, &d.Status, &d.ResponseCode, &d.Error, &d.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan webhook delivery: %w", err)
		}
		deliveries = append(deliveries, d)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return deliveries, total, nil
}
