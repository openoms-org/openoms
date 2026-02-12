package automation

import (
	"context"

	"github.com/google/uuid"
)

// Event represents an automation trigger event.
type Event struct {
	Type       string         // "order.created", "order.status_changed", etc.
	TenantID   uuid.UUID
	EntityType string         // "order", "shipment", "return"
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
type Action struct {
	Type         string         `json:"type"`
	Params       map[string]any `json:"params"`
	DelaySeconds int            `json:"delay_seconds,omitempty"`
}

// EventProcessor is the interface that services use to fire automation events.
// This avoids circular imports between service and automation packages.
type EventProcessor interface {
	ProcessEvent(ctx context.Context, event Event)
}

// ValidOperators is the set of supported condition operators.
var ValidOperators = map[string]bool{
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
