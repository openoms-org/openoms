package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDelayedActionRepositoryListPendingUsesCallerLimit(t *testing.T) {
	tx := &captureQueryTx{rows: emptyRows{}}
	repo := NewDelayedActionRepository()

	_, err := repo.ListPending(context.Background(), tx, 37)

	require.NoError(t, err)
	assert.Contains(t, tx.query, "ORDER BY execute_at ASC, id ASC")
	assert.Contains(t, tx.query, "LIMIT $1")
	require.Len(t, tx.args, 1)
	assert.Equal(t, 37, tx.args[0])
}

func TestDelayedActionRepositoryListPendingDefaultsInvalidLimit(t *testing.T) {
	tx := &captureQueryTx{rows: emptyRows{}}
	repo := NewDelayedActionRepository()

	_, err := repo.ListPending(context.Background(), tx, 0)

	require.NoError(t, err)
	require.Len(t, tx.args, 1)
	assert.Equal(t, defaultDelayedActionPendingLimit, tx.args[0])
}

type captureQueryTx struct {
	query string
	args  []any
	rows  pgx.Rows
}

func (t *captureQueryTx) Begin(context.Context) (pgx.Tx, error) { return nil, nil }
func (t *captureQueryTx) Commit(context.Context) error          { return nil }
func (t *captureQueryTx) Rollback(context.Context) error        { return nil }
func (t *captureQueryTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (t *captureQueryTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (t *captureQueryTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (t *captureQueryTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (t *captureQueryTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (t *captureQueryTx) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	t.query = sql
	t.args = args
	return t.rows, nil
}
func (t *captureQueryTx) QueryRow(context.Context, string, ...any) pgx.Row { return nil }
func (t *captureQueryTx) Conn() *pgx.Conn                                  { return nil }

type emptyRows struct{}

func (emptyRows) Close()                                       {}
func (emptyRows) Err() error                                   { return nil }
func (emptyRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (emptyRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (emptyRows) Next() bool                                   { return false }
func (emptyRows) Scan(...any) error                            { return nil }
func (emptyRows) Values() ([]any, error)                       { return nil, nil }
func (emptyRows) RawValues() [][]byte                          { return nil }
func (emptyRows) Conn() *pgx.Conn                              { return nil }
