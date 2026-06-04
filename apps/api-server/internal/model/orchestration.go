package model

import (
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
)

// Orchestration outbox row status.
const (
	OutboxStatusPending   = "pending"
	OutboxStatusClaimed   = "claimed"
	OutboxStatusSucceeded = "succeeded"
	OutboxStatusFailed    = "failed"
)

// Orchestration attempt status.
const (
	AttemptStatusRunning   = "running"
	AttemptStatusSucceeded = "succeeded"
	AttemptStatusFailed    = "failed"
)

var validOutboxStatuses = []string{OutboxStatusPending, OutboxStatusClaimed, OutboxStatusSucceeded, OutboxStatusFailed}
var validAttemptStatuses = []string{AttemptStatusRunning, AttemptStatusSucceeded, AttemptStatusFailed}

// IsValidOutboxStatus reports whether s is a known outbox status.
func IsValidOutboxStatus(s string) bool { return slices.Contains(validOutboxStatuses, s) }

// IsValidAttemptStatus reports whether s is a known attempt status.
func IsValidAttemptStatus(s string) bool { return slices.Contains(validAttemptStatuses, s) }

const (
	outboxBackoffBase = 30 * time.Second
	outboxBackoffCap  = time.Hour
)

// NextOutboxBackoff returns the delay before the next retry given the number of
// attempts already made: base * 2^attempts, capped. attempts<=0 returns base.
func NextOutboxBackoff(attempts int) time.Duration {
	if attempts <= 0 {
		return outboxBackoffBase
	}
	d := outboxBackoffBase
	for range attempts {
		d *= 2
		if d >= outboxBackoffCap {
			return outboxBackoffCap
		}
	}
	return d
}

// PermanentError marks a handler failure that must not be retried (the outbox
// row goes straight to failed + a blocker, instead of being re-queued).
type PermanentError struct{ Err error }

func (e *PermanentError) Error() string {
	if e.Err == nil {
		return "permanent orchestration error"
	}
	return e.Err.Error()
}

func (e *PermanentError) Unwrap() error { return e.Err }

// Permanent wraps an error as non-retryable.
func Permanent(err error) error { return &PermanentError{Err: err} }

// IsPermanent reports whether err (or anything it wraps) is a PermanentError.
func IsPermanent(err error) bool {
	var pe *PermanentError
	return errors.As(err, &pe)
}

// OrchestrationOutboxEvent is a durable, claimable fulfillment side effect.
type OrchestrationOutboxEvent struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	ProcessID      uuid.UUID  `json:"process_id"`
	UnitID         *uuid.UUID `json:"unit_id,omitempty"`
	EventType      string     `json:"event_type"`
	Payload        any        `json:"payload,omitempty"`
	Status         string     `json:"status"`
	IdempotencyKey string     `json:"idempotency_key"`
	Attempts       int        `json:"attempts"`
	MaxAttempts    int        `json:"max_attempts"`
	NextAttemptAt  time.Time  `json:"next_attempt_at"`
	ClaimedAt      *time.Time `json:"claimed_at,omitempty"`
	LastError      string     `json:"last_error"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// OrchestrationAttempt records a single execution attempt of an outbox event.
type OrchestrationAttempt struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	OutboxID      uuid.UUID  `json:"outbox_id"`
	AttemptNumber int        `json:"attempt_number"`
	Status        string     `json:"status"`
	Error         string     `json:"error"`
	StartedAt     time.Time  `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
}
