package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/automation"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// ErrAutomationRuleNotFound is returned when an automation rule does not exist.
var ErrAutomationRuleNotFound = errors.New("automation rule not found")

// AutomationService handles business logic for automation rules and logs.
type AutomationService struct {
	ruleRepo    repository.AutomationRuleRepo
	logRepo     repository.AutomationRuleLogRepo
	delayedRepo repository.DelayedActionRepo
	pool        *pgxpool.Pool
	engine      *automation.Engine
	logger      *slog.Logger
}

// NewAutomationService creates a new AutomationService.
func NewAutomationService(
	ruleRepo repository.AutomationRuleRepo,
	logRepo repository.AutomationRuleLogRepo,
	pool *pgxpool.Pool,
	engine *automation.Engine,
	logger *slog.Logger,
) *AutomationService {
	return &AutomationService{
		ruleRepo: ruleRepo,
		logRepo:  logRepo,
		pool:     pool,
		engine:   engine,
		logger:   logger,
	}
}

// SetDelayedActionRepo sets the delayed action repository.
func (s *AutomationService) SetDelayedActionRepo(repo repository.DelayedActionRepo) {
	s.delayedRepo = repo
}

// List returns a paginated list of automation rules.
func (s *AutomationService) List(ctx context.Context, tenantID uuid.UUID, filter model.AutomationRuleListFilter) (model.ListResponse[model.AutomationRule], error) {
	var resp model.ListResponse[model.AutomationRule]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		rules, total, err := s.ruleRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}
		resp = model.NewListResponse(rules, total, filter.Limit, filter.Offset)
		return nil
	})
	return resp, err
}

// Get returns a single automation rule by ID.
func (s *AutomationService) Get(ctx context.Context, tenantID, ruleID uuid.UUID) (*model.AutomationRule, error) {
	var rule *model.AutomationRule
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		rule, err = s.ruleRepo.FindByID(ctx, tx, ruleID)
		return err
	})
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, ErrAutomationRuleNotFound
	}
	return rule, nil
}

// Create inserts a new automation rule.
func (s *AutomationService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateAutomationRuleRequest) (*model.AutomationRule, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	req.Name = model.StripHTMLTags(req.Name)
	if req.Description != nil {
		sanitized := model.StripHTMLTags(*req.Description)
		req.Description = &sanitized
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	priority := 0
	if req.Priority != nil {
		priority = *req.Priority
	}

	rule := &model.AutomationRule{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Name:         req.Name,
		Description:  req.Description,
		Enabled:      enabled,
		Priority:     priority,
		TriggerEvent: req.TriggerEvent,
		Conditions:   req.Conditions,
		Actions:      req.Actions,
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		return s.ruleRepo.Create(ctx, tx, rule)
	})
	if err != nil {
		return nil, err
	}
	return rule, nil
}

// Update modifies an existing automation rule.
func (s *AutomationService) Update(ctx context.Context, tenantID, ruleID uuid.UUID, req model.UpdateAutomationRuleRequest) (*model.AutomationRule, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	if req.Name != nil {
		sanitized := model.StripHTMLTags(*req.Name)
		req.Name = &sanitized
	}
	if req.Description != nil {
		sanitized := model.StripHTMLTags(*req.Description)
		req.Description = &sanitized
	}

	var rule *model.AutomationRule
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.ruleRepo.FindByID(ctx, tx, ruleID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrAutomationRuleNotFound
		}

		if err := s.ruleRepo.Update(ctx, tx, ruleID, req); err != nil {
			return err
		}

		rule, err = s.ruleRepo.FindByID(ctx, tx, ruleID)
		return err
	})
	return rule, err
}

// Delete removes an automation rule by ID.
func (s *AutomationService) Delete(ctx context.Context, tenantID, ruleID uuid.UUID) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.ruleRepo.FindByID(ctx, tx, ruleID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrAutomationRuleNotFound
		}
		return s.ruleRepo.Delete(ctx, tx, ruleID)
	})
}

// GetLogs returns execution logs for an automation rule.
func (s *AutomationService) GetLogs(ctx context.Context, tenantID, ruleID uuid.UUID, limit, offset int) (model.ListResponse[model.AutomationRuleLog], error) {
	var resp model.ListResponse[model.AutomationRuleLog]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.ruleRepo.FindByID(ctx, tx, ruleID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrAutomationRuleNotFound
		}

		logs, total, err := s.logRepo.ListByRuleID(ctx, tx, ruleID, limit, offset)
		if err != nil {
			return err
		}
		resp = model.NewListResponse(logs, total, limit, offset)
		return nil
	})
	return resp, err
}

// TestRule evaluates a rule's conditions against provided test data without executing actions.
func (s *AutomationService) TestRule(ctx context.Context, tenantID, ruleID uuid.UUID, testData map[string]any) (*model.TestAutomationRuleResponse, error) {
	var rule *model.AutomationRule
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		rule, err = s.ruleRepo.FindByID(ctx, tx, ruleID)
		return err
	})
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, ErrAutomationRuleNotFound
	}

	condResults, allMet, actions := s.engine.TestRule(*rule, testData)

	resp := &model.TestAutomationRuleResponse{
		AllConditionsMet: allMet,
		ConditionResults: []model.ConditionResult{},
		ActionsToExecute: []model.AutomationAction{},
	}

	for _, cr := range condResults {
		resp.ConditionResults = append(resp.ConditionResults, model.ConditionResult{
			Condition: model.AutomationCondition{
				Field:    cr.Condition.Field,
				Operator: cr.Condition.Operator,
				Value:    cr.Condition.Value,
			},
			Met: cr.Met,
		})
	}

	for _, a := range actions {
		resp.ActionsToExecute = append(resp.ActionsToExecute, model.AutomationAction{
			Type:   a.Type,
			Config: a.Params,
		})
	}

	return resp, nil
}

// ProcessEvent delegates to the automation engine.
func (s *AutomationService) ProcessEvent(ctx context.Context, event automation.Event) {
	s.engine.ProcessEvent(ctx, event)
}

// ListDelayed returns pending delayed actions for admin visibility.
func (s *AutomationService) ListDelayed(ctx context.Context, tenantID uuid.UUID) ([]model.DelayedAction, error) {
	if s.delayedRepo == nil {
		return []model.DelayedAction{}, nil
	}

	var actions []model.DelayedAction
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		actions, err = s.delayedRepo.ListPendingByTenant(ctx, tx)
		return err
	})
	if err != nil {
		return nil, err
	}
	if actions == nil {
		actions = []model.DelayedAction{}
	}
	return actions, nil
}
