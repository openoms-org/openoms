package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditRepository_Log_InsertsNullUserForSystemActor(t *testing.T) {
	tx := &captureExecTx{}
	repo := NewAuditRepository()

	err := repo.Log(context.Background(), tx, model.AuditEntry{
		TenantID:   uuid.New(),
		UserID:     uuid.Nil,
		Action:     "exchange_rate.nbp_fetched",
		EntityType: "exchange_rate",
		EntityID:   uuid.Nil,
		Changes:    map[string]string{"count": "1"},
		IPAddress:  "0.0.0.0",
	})

	require.NoError(t, err)
	require.Len(t, tx.args, 7)
	assert.Nil(t, tx.args[1])
}

func TestAuditRepository_Log_InsertsUserIDForHumanActor(t *testing.T) {
	tx := &captureExecTx{}
	repo := NewAuditRepository()
	userID := uuid.New()

	err := repo.Log(context.Background(), tx, model.AuditEntry{
		TenantID:   uuid.New(),
		UserID:     userID,
		Action:     "exchange_rate.created",
		EntityType: "exchange_rate",
		EntityID:   uuid.New(),
	})

	require.NoError(t, err)
	require.Len(t, tx.args, 7)
	assert.Equal(t, userID, tx.args[1])
}

type captureExecTx struct {
	args []any
}

func (t *captureExecTx) Begin(context.Context) (pgx.Tx, error) { return nil, nil }
func (t *captureExecTx) Commit(context.Context) error          { return nil }
func (t *captureExecTx) Rollback(context.Context) error        { return nil }
func (t *captureExecTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (t *captureExecTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (t *captureExecTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (t *captureExecTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (t *captureExecTx) Exec(_ context.Context, _ string, arguments ...any) (pgconn.CommandTag, error) {
	t.args = arguments
	return pgconn.CommandTag{}, nil
}
func (t *captureExecTx) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }
func (t *captureExecTx) QueryRow(context.Context, string, ...any) pgx.Row        { return nil }
func (t *captureExecTx) Conn() *pgx.Conn                                         { return nil }
