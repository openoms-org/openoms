package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsValidActionType(t *testing.T) {
	for _, ty := range []string{
		"webhook", "set_status", "add_tag", "send_email", "create_invoice",
		"activate_listing", "deactivate_listing", "send_marketplace_message",
		"external_workflow",
	} {
		assert.Truef(t, IsValidActionType(ty), "%q should be a valid action type", ty)
	}
	// Documented-but-unimplemented / unknown types are NOT valid (the executor would
	// fail them at runtime — we reject them at save instead).
	assert.False(t, IsValidActionType("remove_tag"))
	assert.False(t, IsValidActionType("teleport_order"))
	assert.False(t, IsValidActionType(""))
}

func TestValidateAutomationActions_OK(t *testing.T) {
	raw := json.RawMessage(`[
		{"type":"set_status","params":{"status":"shipped"}},
		{"type":"add_tag","params":{"tag":"vip"}},
		{"type":"webhook","params":{"url":"https://example.com/hook"}},
		{"type":"send_email","params":{}},
		{"type":"send_marketplace_message","params":{"template_id":"abc"}},
		{"type":"set_status","params":{"status":"processing"},"delay_seconds":3600}
	]`)
	assert.NoError(t, ValidateAutomationActions(raw))
}

func TestValidateAutomationActions_EmptyOrNil_OK(t *testing.T) {
	assert.NoError(t, ValidateAutomationActions(nil))
	assert.NoError(t, ValidateAutomationActions(json.RawMessage("")))
	assert.NoError(t, ValidateAutomationActions(json.RawMessage("[]")))
}

func TestValidateAutomationActions_UnknownType(t *testing.T) {
	err := ValidateAutomationActions(json.RawMessage(`[{"type":"remove_tag","params":{"tag":"x"}}]`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "remove_tag")
}

func TestValidateAutomationActions_MissingRequiredParam(t *testing.T) {
	// set_status without status
	err := ValidateAutomationActions(json.RawMessage(`[{"type":"set_status","params":{}}]`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status")

	// webhook without url
	err = ValidateAutomationActions(json.RawMessage(`[{"type":"webhook","params":{}}]`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "url")

	// add_tag with a blank tag (present but empty) is rejected too
	err = ValidateAutomationActions(json.RawMessage(`[{"type":"add_tag","params":{"tag":"   "}}]`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tag")
}

func TestValidateAutomationActions_MalformedJSON(t *testing.T) {
	err := ValidateAutomationActions(json.RawMessage(`{not an array`))
	require.Error(t, err)
}

func TestValidateAutomationActions_MissingType(t *testing.T) {
	err := ValidateAutomationActions(json.RawMessage(`[{"params":{"status":"shipped"}}]`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "type")
}
