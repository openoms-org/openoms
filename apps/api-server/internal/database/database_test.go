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

	assert.EqualValues(t, 10, worker.MaxConns)
	assert.EqualValues(t, 2, worker.MinConns)
	// Worker pool stays smaller than the per-replica app pool: the aggregate
	// client-connection budget across all replicas must remain within the pooler cap.
	assert.Less(t, worker.MaxConns, app.MaxConns)
}
