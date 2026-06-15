package automation

import (
	"context"
	"encoding/json"

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

// UnmarshalJSON tolerates both the canonical "params" key and the legacy "config"
// key for action parameters. The dashboard automation form stores action params
// under "config" (its own field name) while the executor, engine and delayed-action
// snapshots use "params"; reading both keeps every stored rule executable regardless
// of which path wrote it. Marshalling is unchanged (still emits "params"), so
// snapshots stay canonical.
func (a *Action) UnmarshalJSON(data []byte) error {
	type actionAlias Action // shed UnmarshalJSON to avoid recursion
	aux := struct {
		*actionAlias
		Config map[string]any `json:"config"`
	}{actionAlias: (*actionAlias)(a)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if len(a.Params) == 0 && len(aux.Config) > 0 {
		a.Params = aux.Config
	}
	return nil
}

// EventProcessor is the interface that services use to fire automation events.
// This avoids circular imports between service and automation packages.
type EventProcessor interface {
	ProcessEvent(ctx context.Context, event Event)
}
