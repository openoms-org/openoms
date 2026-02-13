package model

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AutomationRule represents an automation rule record in the system.
type AutomationRule struct {
	ID           uuid.UUID       `json:"id"`
	TenantID     uuid.UUID       `json:"tenant_id"`
	Name         string          `json:"name"`
	Description  *string         `json:"description,omitempty"`
	Enabled      bool            `json:"enabled"`
	Priority     int             `json:"priority"`
	TriggerEvent string          `json:"trigger_event"`
	Conditions   json.RawMessage `json:"conditions"`
	Actions      json.RawMessage `json:"actions"`
	LastFiredAt  *time.Time      `json:"last_fired_at,omitempty"`
	FireCount    int             `json:"fire_count"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// ValidTriggerEvents is the set of supported trigger events.
var ValidTriggerEvents = map[string]bool{
	"order.created":           true,
	"order.status_changed":    true,
	"order.updated":           true,
	"shipment.created":        true,
	"shipment.status_changed": true,
	"return.created":          true,
	"return.status_changed":   true,
	"product.created":         true,
	"product.updated":         true,
}

// ValidConditionOperators is the set of supported condition operators.
var ValidConditionOperators = map[string]bool{
	"eq":           true,
	"neq":          true,
	"in":           true,
	"not_in":       true,
	"gt":           true,
	"gte":          true,
	"lt":           true,
	"lte":          true,
	"contains":     true,
	"not_contains": true,
	"starts_with":  true,
}

// ValidActionTypes is the set of supported action types.
var ValidActionTypes = map[string]bool{
	"set_status":     true,
	"add_tag":        true,
	"send_email":     true,
	"create_invoice": true,
	"webhook":        true,
}

// AutomationCondition represents a single condition in an automation rule.
type AutomationCondition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    any    `json:"value"`
}

// AutomationAction represents a single action in an automation rule.
type AutomationAction struct {
	Type         string         `json:"type"`
	Config       map[string]any `json:"config"`
	DelaySeconds int            `json:"delay_seconds,omitempty"`
}

// DelayedAction represents a pending delayed automation action.
type DelayedAction struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    uuid.UUID       `json:"tenant_id"`
	RuleID      uuid.UUID       `json:"rule_id"`
	ActionIndex int             `json:"action_index"`
	OrderID     *uuid.UUID      `json:"order_id,omitempty"`
	ExecuteAt   time.Time       `json:"execute_at"`
	Executed    bool            `json:"executed"`
	ExecutedAt  *time.Time      `json:"executed_at,omitempty"`
	Error       *string         `json:"error,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	ActionData  json.RawMessage `json:"action_data"`
	EventData   json.RawMessage `json:"event_data"`
}

// CreateAutomationRuleRequest is the payload for creating an automation rule.
type CreateAutomationRuleRequest struct {
	Name         string          `json:"name"`
	Description  *string         `json:"description,omitempty"`
	Enabled      *bool           `json:"enabled,omitempty"`
	Priority     *int            `json:"priority,omitempty"`
	TriggerEvent string          `json:"trigger_event"`
	Conditions   json.RawMessage `json:"conditions"`
	Actions      json.RawMessage `json:"actions"`
}

func (r *CreateAutomationRuleRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	if err := validateMaxLength("name", r.Name, 500); err != nil {
		return err
	}
	if r.Description != nil {
		if err := validateMaxLengthPtr("description", r.Description, 5000); err != nil {
			return err
		}
	}
	if strings.TrimSpace(r.TriggerEvent) == "" {
		return errors.New("trigger_event is required")
	}
	if !ValidTriggerEvents[r.TriggerEvent] {
		return errors.New("invalid trigger_event")
	}
	if r.Conditions == nil || string(r.Conditions) == "" {
		r.Conditions = json.RawMessage("[]")
	}
	if r.Actions == nil || string(r.Actions) == "" {
		r.Actions = json.RawMessage("[]")
	}
	return nil
}

// UpdateAutomationRuleRequest is the payload for updating an automation rule.
type UpdateAutomationRuleRequest struct {
	Name         *string         `json:"name,omitempty"`
	Description  *string         `json:"description,omitempty"`
	Enabled      *bool           `json:"enabled,omitempty"`
	Priority     *int            `json:"priority,omitempty"`
	TriggerEvent *string         `json:"trigger_event,omitempty"`
	Conditions   json.RawMessage `json:"conditions,omitempty"`
	Actions      json.RawMessage `json:"actions,omitempty"`
}

func (r *UpdateAutomationRuleRequest) Validate() error {
	if r.Name == nil && r.Description == nil && r.Enabled == nil &&
		r.Priority == nil && r.TriggerEvent == nil &&
		r.Conditions == nil && r.Actions == nil {
		return errors.New("at least one field must be provided")
	}
	if r.Name != nil {
		if strings.TrimSpace(*r.Name) == "" {
			return errors.New("name cannot be empty")
		}
		if err := validateMaxLength("name", *r.Name, 500); err != nil {
			return err
		}
	}
	if r.Description != nil {
		if err := validateMaxLengthPtr("description", r.Description, 5000); err != nil {
			return err
		}
	}
	if r.TriggerEvent != nil {
		if !ValidTriggerEvents[*r.TriggerEvent] {
			return errors.New("invalid trigger_event")
		}
	}
	return nil
}

// AutomationRuleLog represents a log entry for an automation rule execution.
type AutomationRuleLog struct {
	ID              uuid.UUID       `json:"id"`
	TenantID        uuid.UUID       `json:"tenant_id"`
	RuleID          uuid.UUID       `json:"rule_id"`
	TriggerEvent    string          `json:"trigger_event"`
	EntityType      string          `json:"entity_type"`
	EntityID        uuid.UUID       `json:"entity_id"`
	ConditionsMet   bool            `json:"conditions_met"`
	ActionsExecuted json.RawMessage `json:"actions_executed"`
	ErrorMessage    *string         `json:"error_message,omitempty"`
	ExecutedAt      time.Time       `json:"executed_at"`
}

// AutomationRuleListFilter defines the filter parameters for listing automation rules.
type AutomationRuleListFilter struct {
	TriggerEvent *string
	Enabled      *bool
	PaginationParams
}

// TestAutomationRuleRequest is the payload for dry-run testing an automation rule.
type TestAutomationRuleRequest struct {
	Data map[string]any `json:"data"`
}

// TestAutomationRuleResponse is the response from a dry-run test.
type TestAutomationRuleResponse struct {
	ConditionResults []ConditionResult  `json:"condition_results"`
	AllConditionsMet bool               `json:"all_conditions_met"`
	ActionsToExecute []AutomationAction `json:"actions_to_execute"`
}

// ConditionResult shows the result of evaluating a single condition.
type ConditionResult struct {
	Condition AutomationCondition `json:"condition"`
	Met       bool                `json:"met"`
}
