package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPoolOptions(t *testing.T) {
	app := DefaultPoolOptions()
	worker := WorkerPoolOptions()

	assert.EqualValues(t, 20, app.MaxConns)
	assert.EqualValues(t, 2, app.MinConns)

	// Worker pool uses the Supabase SESSION-mode pooler (15-client cap shared across pods),
	// so it must stay small and hold no eager idle connections (MinConns=0), or blue-green
	// deploys hit FATAL (EMAXCONNSESSION). See WorkerPoolOptions.
	assert.EqualValues(t, 5, worker.MaxConns)
	assert.EqualValues(t, 0, worker.MinConns)
	assert.Less(t, worker.MaxConns, app.MaxConns)
}
