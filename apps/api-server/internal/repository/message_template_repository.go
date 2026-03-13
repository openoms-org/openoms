package repository

import (
	"context"
	"fmt"
	"strings"

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
	var whereClauses []string
	var args []any
	argIdx := 1

	if filter.Channel != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("channel = $%d", argIdx))
		args = append(args, *filter.Channel)
		argIdx++
	}
	if filter.Enabled != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("enabled = $%d", argIdx))
		args = append(args, *filter.Enabled)
		argIdx++
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM message_templates %s", whereSQL)
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count message templates: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at": "created_at",
		"name":       "name",
		"channel":    "channel",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	query := fmt.Sprintf(
		`SELECT id, tenant_id, name, channel, subject, body, variables,
		        is_autoresponder, trigger_event, enabled, created_at, updated_at
		 FROM message_templates %s %s LIMIT $%d OFFSET $%d`,
		whereSQL, orderByClause, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := tx.Query(ctx, query, args...)
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
	var setClauses []string
	var args []any
	argIdx := 1

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Channel != nil {
		setClauses = append(setClauses, fmt.Sprintf("channel = $%d", argIdx))
		args = append(args, *req.Channel)
		argIdx++
	}
	if req.Subject != nil {
		setClauses = append(setClauses, fmt.Sprintf("subject = $%d", argIdx))
		args = append(args, *req.Subject)
		argIdx++
	}
	if req.Body != nil {
		setClauses = append(setClauses, fmt.Sprintf("body = $%d", argIdx))
		args = append(args, *req.Body)
		argIdx++
	}
	if req.Variables != nil {
		setClauses = append(setClauses, fmt.Sprintf("variables = $%d", argIdx))
		args = append(args, req.Variables)
		argIdx++
	}
	if req.IsAutoresponder != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_autoresponder = $%d", argIdx))
		args = append(args, *req.IsAutoresponder)
		argIdx++
	}
	if req.TriggerEvent != nil {
		setClauses = append(setClauses, fmt.Sprintf("trigger_event = $%d", argIdx))
		args = append(args, *req.TriggerEvent)
		argIdx++
	}
	if req.Enabled != nil {
		setClauses = append(setClauses, fmt.Sprintf("enabled = $%d", argIdx))
		args = append(args, *req.Enabled)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")

	query := fmt.Sprintf("UPDATE message_templates SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

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
