package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

type WebhookDeliveryRepository struct{}

func NewWebhookDeliveryRepository() *WebhookDeliveryRepository {
	return &WebhookDeliveryRepository{}
}

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

func (r *WebhookDeliveryRepository) List(ctx context.Context, tx pgx.Tx, filter model.WebhookDeliveryFilter) ([]model.WebhookDelivery, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

	if filter.EventType != nil {
		conditions = append(conditions, fmt.Sprintf("event_type = $%d", argIdx))
		args = append(args, *filter.EventType)
		argIdx++
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM webhook_deliveries %s", where)
	var total int
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count webhook deliveries: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}

	query := fmt.Sprintf(
		`SELECT id, tenant_id, url, event_type, payload, status, response_code, error, created_at
		 FROM webhook_deliveries
		 %s
		 ORDER BY created_at DESC
		 LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
	)
	args = append(args, limit, filter.Offset)

	rows, err := tx.Query(ctx, query, args...)
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
