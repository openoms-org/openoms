package automation

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// AutomationRuleRepo is the interface needed by the engine for rule persistence.
type AutomationRuleRepo interface {
	FindByTenantAndEvent(ctx context.Context, tx pgx.Tx, event string) ([]model.AutomationRule, error)
	IncrementFireCount(ctx context.Context, tx pgx.Tx, id uuid.UUID, firedAt time.Time) error
}

// AutomationRuleLogRepo is the interface needed by the engine for log persistence.
type AutomationRuleLogRepo interface {
	Create(ctx context.Context, tx pgx.Tx, log *model.AutomationRuleLog) error
}

// Engine is the automation rules engine that processes events.
type Engine struct {
	ruleRepo AutomationRuleRepo
	logRepo  AutomationRuleLogRepo
	pool     *pgxpool.Pool
	executor ActionExecutor
	logger   *slog.Logger
}

// NewEngine creates a new automation engine.
func NewEngine(
	ruleRepo AutomationRuleRepo,
	logRepo AutomationRuleLogRepo,
	pool *pgxpool.Pool,
	executor ActionExecutor,
	logger *slog.Logger,
) *Engine {
	return &Engine{
		ruleRepo: ruleRepo,
		logRepo:  logRepo,
		pool:     pool,
		executor: executor,
		logger:   logger,
	}
}

// ProcessEvent processes an automation event by loading matching rules,
// evaluating conditions, and executing actions.
// This runs asynchronously and should not block the caller.
func (e *Engine) ProcessEvent(ctx context.Context, event Event) {
	go e.processEventAsync(ctx, event)
}

func (e *Engine) processEventAsync(ctx context.Context, event Event) {
	err := database.WithTenant(ctx, e.pool, event.TenantID, func(tx pgx.Tx) error {
		rules, err := e.ruleRepo.FindByTenantAndEvent(ctx, tx, event.Type)
		if err != nil {
			return err
		}

		for _, rule := range rules {
			e.processRule(ctx, tx, rule, event)
		}

		return nil
	})

	if err != nil {
		e.logger.Error("automation engine: failed to process event",
			"event_type", event.Type,
			"tenant_id", event.TenantID,
			"error", err,
		)
	}
}

// TestRule performs a dry-run evaluation of a rule's conditions against the given data.
// It does NOT execute any actions.
func (e *Engine) TestRule(rule model.AutomationRule, data map[string]any) (conditionResults []struct {
	Condition Condition
	Met       bool
}, allMet bool, actions []Action) {
	var conditions []Condition
	if err := json.Unmarshal(rule.Conditions, &conditions); err != nil {
		return nil, false, nil
	}

	allMet = true
	for _, cond := range conditions {
		met := evaluateCondition(cond, data)
		conditionResults = append(conditionResults, struct {
			Condition Condition
			Met       bool
		}{Condition: cond, Met: met})
		if !met {
			allMet = false
		}
	}

	if len(conditions) == 0 {
		allMet = true
	}

	if err := json.Unmarshal(rule.Actions, &actions); err != nil {
		actions = nil
	}

	return conditionResults, allMet, actions
}

func (e *Engine) processRule(ctx context.Context, tx pgx.Tx, rule model.AutomationRule, event Event) {
	// Parse conditions
	var conditions []Condition
	if err := json.Unmarshal(rule.Conditions, &conditions); err != nil {
		e.logger.Error("automation engine: failed to parse conditions",
			"rule_id", rule.ID,
			"error", err,
		)
		return
	}

	// Evaluate conditions
	conditionsMet := EvaluateConditions(conditions, event.Data)

	// Parse actions
	var actions []Action
	if err := json.Unmarshal(rule.Actions, &actions); err != nil {
		e.logger.Error("automation engine: failed to parse actions",
			"rule_id", rule.ID,
			"error", err,
		)
		return
	}

	// Execute actions if conditions met
	var actionsExecuted []map[string]any
	var errorMessage *string

	if conditionsMet {
		for _, action := range actions {
			actionResult := map[string]any{
				"type":   action.Type,
				"params": action.Params,
			}

			if err := e.executor.ExecuteAction(ctx, event.TenantID, action, event); err != nil {
				e.logger.Error("automation engine: action failed",
					"rule_id", rule.ID,
					"action_type", action.Type,
					"error", err,
				)
				actionResult["error"] = err.Error()
				errMsg := err.Error()
				errorMessage = &errMsg
			} else {
				actionResult["success"] = true
			}

			actionsExecuted = append(actionsExecuted, actionResult)
		}

		// Update fire count
		now := time.Now()
		if err := e.ruleRepo.IncrementFireCount(ctx, tx, rule.ID, now); err != nil {
			e.logger.Error("automation engine: failed to update fire count",
				"rule_id", rule.ID,
				"error", err,
			)
		}
	}

	// Log execution
	actionsJSON, _ := json.Marshal(actionsExecuted)
	if actionsJSON == nil {
		actionsJSON = json.RawMessage("[]")
	}

	logEntry := &model.AutomationRuleLog{
		ID:              uuid.New(),
		TenantID:        event.TenantID,
		RuleID:          rule.ID,
		TriggerEvent:    event.Type,
		EntityType:      event.EntityType,
		EntityID:        event.EntityID,
		ConditionsMet:   conditionsMet,
		ActionsExecuted: actionsJSON,
		ErrorMessage:    errorMessage,
		ExecutedAt:      time.Now(),
	}

	if err := e.logRepo.Create(ctx, tx, logEntry); err != nil {
		e.logger.Error("automation engine: failed to create log",
			"rule_id", rule.ID,
			"error", err,
		)
	}
}
