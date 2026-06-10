package model

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// automationActionSpecs is the typed registry of automation action types and the string
// params each one REQUIRES. It mirrors the executor switch in internal/automation/actions.go
// (the runtime source of truth): a rule whose action type is not here, or is missing a
// required param, would fail at execution time, so we reject it at rule-save instead.
// Action types not listed (e.g. the documented-but-unimplemented remove_tag/set_field/
// create_shipment) are invalid until the executor implements them.
var automationActionSpecs = map[string][]string{
	"webhook":                  {"url"},
	"set_status":               {"status"},
	"add_tag":                  {"tag"},
	"send_email":               {},
	"create_invoice":           {},
	"activate_listing":         {},
	"deactivate_listing":       {},
	"send_marketplace_message": {"template_id"},
	"external_workflow":        {"integration_id"},
}

// IsValidActionType reports whether t is a known automation action type.
func IsValidActionType(t string) bool {
	_, ok := automationActionSpecs[t]
	return ok
}

// ValidActionTypes returns the known action types (sorted), for error messages / docs.
func ValidActionTypes() []string {
	out := make([]string, 0, len(automationActionSpecs))
	for t := range automationActionSpecs {
		out = append(out, t)
	}
	slices.Sort(out)
	return out
}

// validatedAction is the minimal shape needed to validate a rule's action at save time.
// It mirrors automation.Action's JSON without importing that package (model must not
// depend on automation — automation already depends on model).
type validatedAction struct {
	Type   string         `json:"type"`
	Params map[string]any `json:"params"`
}

// ValidateAutomationActions validates a rule's actions JSON at save time: it must be a
// JSON array of actions, each with a known type and every required param present + a
// non-blank string. Empty/nil input is allowed (a rule may have no actions). Returns a
// descriptive error naming the offending action index + type/param.
func ValidateAutomationActions(raw json.RawMessage) error {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "" || strings.TrimSpace(string(raw)) == "[]" {
		return nil
	}
	var actions []validatedAction
	if err := json.Unmarshal(raw, &actions); err != nil {
		return fmt.Errorf("actions must be a JSON array of {type, params}: %w", err)
	}
	for i, a := range actions {
		if strings.TrimSpace(a.Type) == "" {
			return fmt.Errorf("action %d: type is required", i)
		}
		required, ok := automationActionSpecs[a.Type]
		if !ok {
			return fmt.Errorf("action %d: unknown action type %q (valid: %s)", i, a.Type, strings.Join(ValidActionTypes(), ", "))
		}
		for _, key := range required {
			v, _ := a.Params[key].(string)
			if strings.TrimSpace(v) == "" {
				return fmt.Errorf("action %d (%s): param %q is required", i, a.Type, key)
			}
		}
	}
	return nil
}
