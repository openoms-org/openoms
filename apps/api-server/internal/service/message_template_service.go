package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var ErrMessageTemplateNotFound = errors.New("message template not found")

type MessageTemplateService struct {
	repo repository.MessageTemplateRepo
	pool *pgxpool.Pool
}

func NewMessageTemplateService(repo repository.MessageTemplateRepo, pool *pgxpool.Pool) *MessageTemplateService {
	return &MessageTemplateService{repo: repo, pool: pool}
}

func (s *MessageTemplateService) List(ctx context.Context, tenantID uuid.UUID, filter model.MessageTemplateListFilter) ([]model.MessageTemplate, int, error) {
	var templates []model.MessageTemplate
	var total int
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		templates, total, err = s.repo.List(ctx, tx, filter)
		return err
	})
	return templates, total, err
}

func (s *MessageTemplateService) Get(ctx context.Context, tenantID, id uuid.UUID) (*model.MessageTemplate, error) {
	var t *model.MessageTemplate
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		t, err = s.repo.FindByID(ctx, tx, id)
		return err
	})
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrMessageTemplateNotFound
	}
	return t, nil
}

func (s *MessageTemplateService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateMessageTemplateRequest) (*model.MessageTemplate, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	variables := req.Variables
	if variables == nil {
		variables = []string{}
	}

	t := &model.MessageTemplate{
		ID:              uuid.New(),
		TenantID:        tenantID,
		Name:            req.Name,
		Channel:         req.Channel,
		Subject:         req.Subject,
		Body:            req.Body,
		Variables:       variables,
		IsAutoresponder: req.IsAutoresponder,
		TriggerEvent:    req.TriggerEvent,
		Enabled:         enabled,
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		return s.repo.Create(ctx, tx, t)
	})
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (s *MessageTemplateService) Update(ctx context.Context, tenantID, id uuid.UUID, req model.UpdateMessageTemplateRequest) (*model.MessageTemplate, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var t *model.MessageTemplate
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		t, err = s.repo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if t == nil {
			return ErrMessageTemplateNotFound
		}

		if err := s.repo.Update(ctx, tx, id, req); err != nil {
			return err
		}

		t, err = s.repo.FindByID(ctx, tx, id)
		return err
	})
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (s *MessageTemplateService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		t, err := s.repo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if t == nil {
			return ErrMessageTemplateNotFound
		}
		return s.repo.Delete(ctx, tx, id)
	})
}
