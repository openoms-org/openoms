package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// MessageTemplateRepository implements MessageTemplateRepo.
type MessageTemplateRepository struct{}

// NewMessageTemplateRepository creates a new MessageTemplateRepository.
func NewMessageTemplateRepository() *MessageTemplateRepository {
	return &MessageTemplateRepository{}
}

// List returns a paginated list of message templates matching the filter.
func (r *MessageTemplateRepository) List(ctx context.Context, tx pgx.Tx, filter model.MessageTemplateListFilter) ([]model.MessageTemplate, int, error) {
	qb := NewQueryBuilder()
	if filter.Channel != nil {
		qb.Add("channel = $%d", *filter.Channel)
	}
	if filter.Enabled != nil {
		qb.Add("enabled = $%d", *filter.Enabled)
	}
	whereSQL := qb.WhereClause()

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM message_templates %s", whereSQL)
	if err := tx.QueryRow(ctx, countQuery, qb.Args()...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count message templates: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at": "created_at",
		"name":       "name",
		"channel":    "channel",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	limitIdx := qb.AddArgs(filter.Limit, filter.Offset)
	query := fmt.Sprintf(
		`SELECT id, tenant_id, name, channel, subject, body, variables,
		        is_autoresponder, trigger_event, enabled, created_at, updated_at
		 FROM message_templates %s %s LIMIT $%d OFFSET $%d`,
		whereSQL, orderByClause, limitIdx, limitIdx+1,
	)

	rows, err := tx.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, 0, fmt.Errorf("list message templates: %w", err)
	}
	defer rows.Close()

	var templates []model.MessageTemplate
	for rows.Next() {
		var t model.MessageTemplate
		if err := rows.Scan(
			&t.ID, &t.TenantID, &t.Name, &t.Channel, &t.Subject, &t.Body, &t.Variables,
			&t.IsAutoresponder, &t.TriggerEvent, &t.Enabled, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan message template: %w", err)
		}
		templates = append(templates, t)
	}
	return templates, total, rows.Err()
}

// FindByID returns a message template by its ID.
func (r *MessageTemplateRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.MessageTemplate, error) {
	var t model.MessageTemplate
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, name, channel, subject, body, variables,
		        is_autoresponder, trigger_event, enabled, created_at, updated_at
		 FROM message_templates WHERE id = $1`, id,
	).Scan(
		&t.ID, &t.TenantID, &t.Name, &t.Channel, &t.Subject, &t.Body, &t.Variables,
		&t.IsAutoresponder, &t.TriggerEvent, &t.Enabled, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find message template by id: %w", err)
	}
	return &t, nil
}

// Create inserts a new message template.
func (r *MessageTemplateRepository) Create(ctx context.Context, tx pgx.Tx, template *model.MessageTemplate) error {
	return tx.QueryRow(ctx,
		`INSERT INTO message_templates (id, tenant_id, name, channel, subject, body, variables,
		                                is_autoresponder, trigger_event, enabled)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING created_at, updated_at`,
		template.ID, template.TenantID, template.Name, template.Channel, template.Subject,
		template.Body, template.Variables, template.IsAutoresponder, template.TriggerEvent, template.Enabled,
	).Scan(&template.CreatedAt, &template.UpdatedAt)
}

// Update applies partial updates to a message template.
func (r *MessageTemplateRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateMessageTemplateRequest) error {
	ub := NewUpdateBuilder()
	SetPtr(ub, "name", req.Name)
	SetPtr(ub, "channel", req.Channel)
	SetPtr(ub, "subject", req.Subject)
	SetPtr(ub, "body", req.Body)
	if req.Variables != nil {
		ub.Set("variables", req.Variables)
	}
	SetPtr(ub, "is_autoresponder", req.IsAutoresponder)
	SetPtr(ub, "trigger_event", req.TriggerEvent)
	SetPtr(ub, "enabled", req.Enabled)

	if ub.IsEmpty() {
		return nil
	}

	ub.SetRaw("updated_at = NOW()")

	query := fmt.Sprintf("UPDATE message_templates SET %s WHERE id = $%d",
		ub.SetClause(), ub.NextArgIdx())
	args := append(ub.Args(), id)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update message template: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("message template not found")
	}
	return nil
}

// Delete removes a message template by its ID.
func (r *MessageTemplateRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM message_templates WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete message template: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("message template not found")
	}
	return nil
}
