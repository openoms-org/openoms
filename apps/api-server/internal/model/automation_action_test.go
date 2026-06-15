package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// API/runtime-written rules store action params under "params"; the workflow-builder
// and rule-test paths read AutomationAction.Config, so the tolerant unmarshal must
// surface them.
func TestAutomationActionUnmarshal_ParamsKeyPopulatesConfig(t *testing.T) {
	var a AutomationAction
	err := json.Unmarshal([]byte(`{"type":"set_status","params":{"status":"shipped"}}`), &a)
	assert.NoError(t, err)
	assert.Equal(t, "set_status", a.Type)
	assert.Equal(t, "shipped", a.Config["status"])
}

func TestAutomationActionUnmarshal_ConfigKeyStillWorks(t *testing.T) {
	var a AutomationAction
	err := json.Unmarshal([]byte(`{"type":"add_tag","config":{"tag":"vip"}}`), &a)
	assert.NoError(t, err)
	assert.Equal(t, "vip", a.Config["tag"])
}

func TestAutomationActionMarshal_EmitsConfig(t *testing.T) {
	a := AutomationAction{Type: "set_status", Config: map[string]any{"status": "shipped"}}
	b, err := json.Marshal(a)
	assert.NoError(t, err)
	assert.Contains(t, string(b), `"config"`)
}

// Save-time validation must accept dashboard "config"-keyed required-param actions
// (the same tolerance as execution), else such rules could never be created.
func TestValidateAutomationActions_AcceptsConfigKey(t *testing.T) {
	err := ValidateAutomationActions(json.RawMessage(`[{"type":"set_status","config":{"status":"shipped"}}]`))
	assert.NoError(t, err)
}

func TestValidateAutomationActions_AcceptsParamsKey(t *testing.T) {
	err := ValidateAutomationActions(json.RawMessage(`[{"type":"set_status","params":{"status":"shipped"}}]`))
	assert.NoError(t, err)
}

func TestValidateAutomationActions_StillRejectsMissingRequired(t *testing.T) {
	err := ValidateAutomationActions(json.RawMessage(`[{"type":"set_status","config":{}}]`))
	assert.Error(t, err)
}
