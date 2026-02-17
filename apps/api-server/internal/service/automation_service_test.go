package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// --- Create Automation Rule Tests ---

func TestAutomationService_Create_ValidationError_EmptyName(t *testing.T) {
	svc := NewAutomationService(nil, nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateAutomationRuleRequest{
		Name:         "",
		TriggerEvent: "order.created",
		Conditions:   json.RawMessage("[]"),
		Actions:      json.RawMessage("[]"),
	})

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "name")
}

func TestAutomationService_Create_ValidationError_WhitespaceName(t *testing.T) {
	svc := NewAutomationService(nil, nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateAutomationRuleRequest{
		Name:         "   ",
		TriggerEvent: "order.created",
		Conditions:   json.RawMessage("[]"),
		Actions:      json.RawMessage("[]"),
	})

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "name")
}

func TestAutomationService_Create_ValidationError_NameTooLong(t *testing.T) {
	svc := NewAutomationService(nil, nil, nil, nil, nil)

	longName := strings.Repeat("a", 501)
	_, err := svc.Create(context.Background(), uuid.New(), model.CreateAutomationRuleRequest{
		Name:         longName,
		TriggerEvent: "order.created",
		Conditions:   json.RawMessage("[]"),
		Actions:      json.RawMessage("[]"),
	})

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "name")
}

func TestAutomationService_Create_ValidationError_DescriptionTooLong(t *testing.T) {
	svc := NewAutomationService(nil, nil, nil, nil, nil)

	longDesc := strings.Repeat("d", 5001)
	_, err := svc.Create(context.Background(), uuid.New(), model.CreateAutomationRuleRequest{
		Name:         "Valid Rule",
		Description:  &longDesc,
		TriggerEvent: "order.created",
		Conditions:   json.RawMessage("[]"),
		Actions:      json.RawMessage("[]"),
	})

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "description")
}

func TestAutomationService_Create_ValidationError_EmptyTriggerEvent(t *testing.T) {
	svc := NewAutomationService(nil, nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateAutomationRuleRequest{
		Name:         "Valid Rule",
		TriggerEvent: "",
		Conditions:   json.RawMessage("[]"),
		Actions:      json.RawMessage("[]"),
	})

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "trigger_event")
}

func TestAutomationService_Create_ValidationError_InvalidTriggerEvent(t *testing.T) {
	svc := NewAutomationService(nil, nil, nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateAutomationRuleRequest{
		Name:         "Valid Rule",
		TriggerEvent: "invalid.event",
		Conditions:   json.RawMessage("[]"),
		Actions:      json.RawMessage("[]"),
	})

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "trigger_event")
}

// --- Update Automation Rule Tests ---

func TestAutomationService_Update_ValidationError_NoFieldsProvided(t *testing.T) {
	svc := NewAutomationService(nil, nil, nil, nil, nil)

	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateAutomationRuleRequest{})

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "at least one field")
}

func TestAutomationService_Update_ValidationError_EmptyName(t *testing.T) {
	svc := NewAutomationService(nil, nil, nil, nil, nil)

	emptyName := ""
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateAutomationRuleRequest{
		Name: &emptyName,
	})

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "name")
}

func TestAutomationService_Update_ValidationError_WhitespaceName(t *testing.T) {
	svc := NewAutomationService(nil, nil, nil, nil, nil)

	wsName := "   "
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateAutomationRuleRequest{
		Name: &wsName,
	})

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "name")
}

func TestAutomationService_Update_ValidationError_NameTooLong(t *testing.T) {
	svc := NewAutomationService(nil, nil, nil, nil, nil)

	longName := strings.Repeat("a", 501)
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateAutomationRuleRequest{
		Name: &longName,
	})

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "name")
}

func TestAutomationService_Update_ValidationError_DescriptionTooLong(t *testing.T) {
	svc := NewAutomationService(nil, nil, nil, nil, nil)

	longDesc := strings.Repeat("d", 5001)
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateAutomationRuleRequest{
		Description: &longDesc,
	})

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "description")
}

func TestAutomationService_Update_ValidationError_InvalidTriggerEvent(t *testing.T) {
	svc := NewAutomationService(nil, nil, nil, nil, nil)

	badEvent := "bad.event"
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateAutomationRuleRequest{
		TriggerEvent: &badEvent,
	})

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "trigger_event")
}
