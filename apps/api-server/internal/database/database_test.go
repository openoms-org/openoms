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

	// Worker pool runs on the Supabase TRANSACTION pooler (port 6543, ~200 clients) via the
	// dedicated WORKER_DATABASE_URL secret (OPE-553), so it carries the throughput sizing
	// (MaxConns=10) rather than the session-cap-constrained 5. MinConns stays 0 (lazy) so idle
	// pods reserve nothing, and it stays below the app pool. See WorkerPoolOptions.
	assert.EqualValues(t, 10, worker.MaxConns)
	assert.EqualValues(t, 0, worker.MinConns)
	assert.Less(t, worker.MaxConns, app.MaxConns)
}
