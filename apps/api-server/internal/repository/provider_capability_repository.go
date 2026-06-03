package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// ProviderCapabilityRepository persists per-version capability profiles, status
// mappings, and integration gaps. Platform-managed, no RLS.
type ProviderCapabilityRepository struct {
	pool *pgxpool.Pool
}

// NewProviderCapabilityRepository creates the repository backed by the pool.
func NewProviderCapabilityRepository(pool *pgxpool.Pool) *ProviderCapabilityRepository {
	return &ProviderCapabilityRepository{pool: pool}
}

const providerCapabilityColumns = `id, provider_version_id, capability_key, support_status, channel, mode,
	freshness, required_inputs, provided_outputs, latency_sla_seconds, notes, created_at, updated_at`

// ReplaceCapabilities atomically replaces the capability set of a version
// (DELETE then INSERT) using the caller's transaction.
func (r *ProviderCapabilityRepository) ReplaceCapabilities(ctx context.Context, q Querier, versionID uuid.UUID, caps []model.ProviderCapability) error {
	if _, err := q.Exec(ctx, `DELETE FROM provider_capability_profiles WHERE provider_version_id = $1`, versionID); err != nil {
		return fmt.Errorf("clear capabilities: %w", err)
	}
	for _, c := range caps {
		_, err := q.Exec(ctx,
			`INSERT INTO provider_capability_profiles
			 (provider_version_id, capability_key, support_status, channel, mode, freshness, required_inputs, provided_outputs, latency_sla_seconds, notes)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			versionID, c.CapabilityKey, c.SupportStatus, c.Channel, c.Mode, c.Freshness,
			orEmptySlice(c.RequiredInputs), orEmptySlice(c.ProvidedOutputs), c.LatencySLASeconds, c.Notes)
		if err != nil {
			return fmt.Errorf("insert capability %q: %w", c.CapabilityKey, err)
		}
	}
	return nil
}

// ListCapabilities returns a version's capabilities.
func (r *ProviderCapabilityRepository) ListCapabilities(ctx context.Context, versionID uuid.UUID) ([]model.ProviderCapability, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+providerCapabilityColumns+` FROM provider_capability_profiles WHERE provider_version_id = $1 ORDER BY capability_key`, versionID)
	if err != nil {
		return nil, fmt.Errorf("list capabilities: %w", err)
	}
	defer rows.Close()
	result := []model.ProviderCapability{}
	for rows.Next() {
		var c model.ProviderCapability
		if err := rows.Scan(&c.ID, &c.ProviderVersionID, &c.CapabilityKey, &c.SupportStatus, &c.Channel, &c.Mode,
			&c.Freshness, &c.RequiredInputs, &c.ProvidedOutputs, &c.LatencySLASeconds, &c.Notes, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan capability: %w", err)
		}
		if c.RequiredInputs == nil {
			c.RequiredInputs = []string{}
		}
		if c.ProvidedOutputs == nil {
			c.ProvidedOutputs = []string{}
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

const providerStatusMappingColumns = `id, provider_version_id, status_domain, raw_status, canonical_status,
	canonical_event_type, canonical_step_key, confidence, is_terminal, notes, created_at, updated_at`

// ReplaceStatusMappings atomically replaces the status-mapping set of a version.
func (r *ProviderCapabilityRepository) ReplaceStatusMappings(ctx context.Context, q Querier, versionID uuid.UUID, mappings []model.ProviderStatusMapping) error {
	if _, err := q.Exec(ctx, `DELETE FROM provider_status_mappings WHERE provider_version_id = $1`, versionID); err != nil {
		return fmt.Errorf("clear status mappings: %w", err)
	}
	for _, m := range mappings {
		_, err := q.Exec(ctx,
			`INSERT INTO provider_status_mappings
			 (provider_version_id, status_domain, raw_status, canonical_status, canonical_event_type, canonical_step_key, confidence, is_terminal, notes)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			versionID, m.StatusDomain, m.RawStatus, m.CanonicalStatus, m.CanonicalEventType, m.CanonicalStepKey, m.Confidence, m.IsTerminal, m.Notes)
		if err != nil {
			return fmt.Errorf("insert status mapping %q/%q: %w", m.StatusDomain, m.RawStatus, err)
		}
	}
	return nil
}

// ListStatusMappings returns a version's status mappings.
func (r *ProviderCapabilityRepository) ListStatusMappings(ctx context.Context, versionID uuid.UUID) ([]model.ProviderStatusMapping, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+providerStatusMappingColumns+` FROM provider_status_mappings WHERE provider_version_id = $1 ORDER BY status_domain, raw_status`, versionID)
	if err != nil {
		return nil, fmt.Errorf("list status mappings: %w", err)
	}
	defer rows.Close()
	result := []model.ProviderStatusMapping{}
	for rows.Next() {
		var m model.ProviderStatusMapping
		if err := rows.Scan(&m.ID, &m.ProviderVersionID, &m.StatusDomain, &m.RawStatus, &m.CanonicalStatus,
			&m.CanonicalEventType, &m.CanonicalStepKey, &m.Confidence, &m.IsTerminal, &m.Notes, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan status mapping: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

const providerGapColumns = `id, provider_version_id, gap_type, severity, status, description, created_at, updated_at, resolved_at`

// CreateGap records an integration gap for a version.
func (r *ProviderCapabilityRepository) CreateGap(ctx context.Context, versionID uuid.UUID, gapType, severity, description string) (*model.ProviderIntegrationGap, error) {
	var g model.ProviderIntegrationGap
	err := r.pool.QueryRow(ctx,
		`INSERT INTO provider_integration_gaps (provider_version_id, gap_type, severity, description)
		 VALUES ($1,$2,$3,$4) RETURNING `+providerGapColumns,
		versionID, gapType, severity, description).
		Scan(&g.ID, &g.ProviderVersionID, &g.GapType, &g.Severity, &g.Status, &g.Description, &g.CreatedAt, &g.UpdatedAt, &g.ResolvedAt)
	if err != nil {
		return nil, fmt.Errorf("create gap: %w", err)
	}
	return &g, nil
}

// ListGaps returns the gaps for a version, newest first.
func (r *ProviderCapabilityRepository) ListGaps(ctx context.Context, versionID uuid.UUID) ([]model.ProviderIntegrationGap, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+providerGapColumns+` FROM provider_integration_gaps WHERE provider_version_id = $1 ORDER BY created_at DESC`, versionID)
	if err != nil {
		return nil, fmt.Errorf("list gaps: %w", err)
	}
	defer rows.Close()
	result := []model.ProviderIntegrationGap{}
	for rows.Next() {
		var g model.ProviderIntegrationGap
		if err := rows.Scan(&g.ID, &g.ProviderVersionID, &g.GapType, &g.Severity, &g.Status, &g.Description, &g.CreatedAt, &g.UpdatedAt, &g.ResolvedAt); err != nil {
			return nil, fmt.Errorf("scan gap: %w", err)
		}
		result = append(result, g)
	}
	return result, rows.Err()
}

// UpdateGapStatus updates a gap's lifecycle status (setting resolved_at when resolved).
func (r *ProviderCapabilityRepository) UpdateGapStatus(ctx context.Context, gapID uuid.UUID, status string) (*model.ProviderIntegrationGap, error) {
	var resolvedAt *time.Time
	if status == model.GapStatusResolved {
		now := time.Now().UTC()
		resolvedAt = &now
	}
	var g model.ProviderIntegrationGap
	err := r.pool.QueryRow(ctx,
		`UPDATE provider_integration_gaps
		    SET status = $2, resolved_at = $3::timestamptz, updated_at = now()
		  WHERE id = $1 RETURNING `+providerGapColumns,
		gapID, status, resolvedAt).
		Scan(&g.ID, &g.ProviderVersionID, &g.GapType, &g.Severity, &g.Status, &g.Description, &g.CreatedAt, &g.UpdatedAt, &g.ResolvedAt)
	if err != nil {
		return nil, err
	}
	return &g, nil
}
