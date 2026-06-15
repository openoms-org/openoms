package automation

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Dashboard-created rules store action params under "config"; the executor reads
// Action.Params, so the tolerant unmarshal must surface them.
func TestActionUnmarshal_ConfigKeyPopulatesParams(t *testing.T) {
	var a Action
	err := json.Unmarshal([]byte(`{"type":"set_status","config":{"status":"shipped"}}`), &a)
	assert.NoError(t, err)
	assert.Equal(t, "set_status", a.Type)
	assert.Equal(t, "shipped", a.Params["status"])
}

func TestActionUnmarshal_ParamsKeyStillWorks(t *testing.T) {
	var a Action
	err := json.Unmarshal([]byte(`{"type":"add_tag","params":{"tag":"vip"}}`), &a)
	assert.NoError(t, err)
	assert.Equal(t, "vip", a.Params["tag"])
}

func TestActionUnmarshal_ParamsWinsOverConfig(t *testing.T) {
	var a Action
	err := json.Unmarshal([]byte(`{"type":"x","params":{"k":"p"},"config":{"k":"c"}}`), &a)
	assert.NoError(t, err)
	assert.Equal(t, "p", a.Params["k"])
}

// Snapshots (delayed-action ActionData) must stay canonical: marshalling emits
// "params", never "config".
func TestActionMarshal_EmitsParamsNotConfig(t *testing.T) {
	a := Action{Type: "set_status", Params: map[string]any{"status": "shipped"}}
	b, err := json.Marshal(a)
	assert.NoError(t, err)
	assert.Contains(t, string(b), `"params"`)
	assert.NotContains(t, string(b), `"config"`)
}
