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

	assert.EqualValues(t, 3, worker.MaxConns)
	assert.EqualValues(t, 0, worker.MinConns)
	assert.Less(t, worker.MaxConns, app.MaxConns)
}
