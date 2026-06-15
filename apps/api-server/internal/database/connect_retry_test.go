package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

func TestConnectWithRetrySucceedsAfterTransientFailures(t *testing.T) {
	calls := 0
	pool, err := connectWithRetry(context.Background(), 5, time.Millisecond, func(context.Context) (*pgxpool.Pool, error) {
		calls++
		if calls < 3 {
			return nil, errors.New("transient pooler blip")
		}
		return nil, nil // success (a real pool is not needed to exercise the retry loop)
	})
	assert.NoError(t, err)
	assert.Nil(t, pool)
	assert.Equal(t, 3, calls, "should stop retrying once connect succeeds")
}

func TestConnectWithRetryGivesUpAfterAttempts(t *testing.T) {
	calls := 0
	_, err := connectWithRetry(context.Background(), 3, time.Millisecond, func(context.Context) (*pgxpool.Pool, error) {
		calls++
		return nil, errors.New("database down")
	})
	assert.Error(t, err)
	assert.Equal(t, 3, calls, "should attempt exactly the configured number of times")
}

func TestConnectWithRetryRespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	_, err := connectWithRetry(ctx, 5, time.Hour, func(context.Context) (*pgxpool.Pool, error) {
		calls++
		cancel() // cancel during the first backoff
		return nil, errors.New("transient")
	})
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, calls, "must not keep retrying after the context is cancelled")
}

func TestConnectWithRetryClampsAttempts(t *testing.T) {
	calls := 0
	_, err := connectWithRetry(context.Background(), 0, time.Millisecond, func(context.Context) (*pgxpool.Pool, error) {
		calls++
		return nil, errors.New("down")
	})
	assert.Error(t, err)
	assert.Equal(t, 1, calls, "attempts < 1 is clamped to a single try")
}
