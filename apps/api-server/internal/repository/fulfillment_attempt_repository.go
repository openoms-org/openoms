package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// FulfillmentAttemptRepository persists carrier/provider attempt records
// (OPE-417). Like the other fulfillment tables this is tenant-scoped with RLS, so
// every method takes a pgx.Tx obtained from database.WithTenant — the tenant
// context (app.current_tenant_id) both filters reads and constrains writes.
type FulfillmentAttemptRepository struct{}

// NewFulfillmentAttemptRepository creates a FulfillmentAttemptRepository.
func NewFulfillmentAttemptRepository() *FulfillmentAttemptRepository {
	return &FulfillmentAttemptRepository{}
}

const fulfillmentAttemptColumns = `id, tenant_id, process_id, provider, operation, status,
	request_id, correlation_id, raw_provider_status, result_redacted, payload_hash, error_code, created_at`

func scanProviderAttempt(row interface{ Scan(...any) error }) (*model.ProviderAttempt, error) {
	var a model.ProviderAttempt
	var requestID, correlationID, rawStatus, payloadHash, errorCode *string
	var result []byte
	if err := row.Scan(
		&a.ID, &a.TenantID, &a.ProcessID, &a.Provider, &a.Operation, &a.Status,
		&requestID, &correlationID, &rawStatus, &result, &payloadHash, &errorCode, &a.CreatedAt,
	); err != nil {
		return nil, err
	}
	if requestID != nil {
		a.RequestID = *requestID
	}
	if correlationID != nil {
		a.CorrelationID = *correlationID
	}
	if rawStatus != nil {
		a.RawProviderStatus = *rawStatus
	}
	if payloadHash != nil {
		a.PayloadHash = *payloadHash
	}
	if errorCode != nil {
		a.ErrorCode = *errorCode
	}
	unmarshalMeta(result, &a.ResultRedacted)
	return &a, nil
}

// resultRedactedJSON marshals the redacted result for the jsonb column, or NULL
// when none was supplied. Reuses marshalMeta's safe-fallback behavior.
func resultRedactedJSON(v any) *string {
	if v == nil {
		return nil
	}
	s := marshalMeta(v)
	return &s
}

// RecordAttempt inserts a provider attempt row. Status/operation default and are
// not validated here (the caller and the DB CHECK constraint are authoritative);
// the row is written in the caller's tenant transaction (RLS-scoped).
func (r *FulfillmentAttemptRepository) RecordAttempt(ctx context.Context, tx pgx.Tx, a model.ProviderAttempt) (*model.ProviderAttempt, error) {
	if a.Status == "" {
		a.Status = model.ProviderAttemptPending
	}
	out, err := scanProviderAttempt(tx.QueryRow(ctx,
		`INSERT INTO fulfillment_provider_attempts
		   (tenant_id, process_id, provider, operation, status, request_id, correlation_id,
		    raw_provider_status, result_redacted, payload_hash, error_code)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10,$11)
		 RETURNING `+fulfillmentAttemptColumns,
		a.TenantID, a.ProcessID, a.Provider, a.Operation, a.Status,
		nilIfEmpty(a.RequestID), nilIfEmpty(a.CorrelationID), nilIfEmpty(a.RawProviderStatus),
		resultRedactedJSON(a.ResultRedacted), nilIfEmpty(a.PayloadHash), nilIfEmpty(a.ErrorCode)))
	if err != nil {
		return nil, fmt.Errorf("record provider attempt: %w", err)
	}
	return out, nil
}

// ListByProcess returns a process's provider attempts, newest first (RLS-scoped).
func (r *FulfillmentAttemptRepository) ListByProcess(ctx context.Context, tx pgx.Tx, processID uuid.UUID) ([]model.ProviderAttempt, error) {
	rows, err := tx.Query(ctx,
		`SELECT `+fulfillmentAttemptColumns+` FROM fulfillment_provider_attempts WHERE process_id = $1 ORDER BY created_at DESC`,
		processID)
	if err != nil {
		return nil, fmt.Errorf("list provider attempts by process: %w", err)
	}
	defer rows.Close()
	result := []model.ProviderAttempt{}
	for rows.Next() {
		a, err := scanProviderAttempt(rows)
		if err != nil {
			return nil, fmt.Errorf("scan provider attempt: %w", err)
		}
		result = append(result, *a)
	}
	return result, rows.Err()
}

// ListByCorrelation returns attempts for a correlation id (e.g. shipment id),
// newest first (RLS-scoped). Useful for idempotency / dedup lookups when no
// fulfillment process exists yet.
func (r *FulfillmentAttemptRepository) ListByCorrelation(ctx context.Context, tx pgx.Tx, correlationID, provider, operation string) ([]model.ProviderAttempt, error) {
	rows, err := tx.Query(ctx,
		`SELECT `+fulfillmentAttemptColumns+`
		   FROM fulfillment_provider_attempts
		  WHERE correlation_id = $1 AND provider = $2 AND operation = $3
		  ORDER BY created_at DESC`,
		correlationID, provider, operation)
	if err != nil {
		return nil, fmt.Errorf("list provider attempts by correlation: %w", err)
	}
	defer rows.Close()
	result := []model.ProviderAttempt{}
	for rows.Next() {
		a, err := scanProviderAttempt(rows)
		if err != nil {
			return nil, fmt.Errorf("scan provider attempt: %w", err)
		}
		result = append(result, *a)
	}
	return result, rows.Err()
}
