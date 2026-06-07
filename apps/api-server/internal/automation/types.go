package automation

import (
	"context"

	"github.com/google/uuid"
)

// Event represents an automation trigger event.
type Event struct {
	Type       string // "order.created", "order.status_changed", etc.
	TenantID   uuid.UUID
	EntityType string // "order", "shipment", "return"
	EntityID   uuid.UUID
	Data       map[string]any // event-specific data
}

// Condition defines a single condition to evaluate against event data.
type Condition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    any    `json:"value"`
}

// Action defines an action to execute when conditions are met.
//
// RuleID and ActionIndex identify the owning rule + the action's position within
// that rule. The engine populates them before dispatch (and persists them onto
// the delayed-action snapshot), so executors — including the orchestration-routed
// set_status path (OPE-421) — can build a stable idempotency key from
// (rule, order, target). They are omitempty so existing persisted actions and
// hand-built test actions remain compatible.
type Action struct {
	Type         string         `json:"type"`
	Params       map[string]any `json:"params"`
	DelaySeconds int            `json:"delay_seconds,omitempty"`
	RuleID       uuid.UUID      `json:"rule_id,omitempty"`
	ActionIndex  int            `json:"action_index,omitempty"`
}

// EventProcessor is the interface that services use to fire automation events.
// This avoids circular imports between service and automation packages.
type EventProcessor interface {
	ProcessEvent(ctx context.Context, event Event)
}
