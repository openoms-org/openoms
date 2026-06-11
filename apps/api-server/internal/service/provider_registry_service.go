package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/obsmetrics"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// Provider registry domain errors.
var (
	ErrProviderDefinitionNotFound  = errors.New("provider definition not found")
	ErrProviderVersionNotFound     = errors.New("provider version not found")
	ErrIllegalProviderTransition   = errors.New("illegal provider lifecycle transition")
	ErrProviderVersionFrozen       = errors.New("published provider version is immutable")
	ErrProviderEnableStateInvalid  = errors.New("version must be in private_beta or available to enable a tenant")
	ErrProviderDisableStateInvalid = errors.New("only available or private_beta versions can be emergency-disabled")
	ErrInvalidProviderType         = errors.New("invalid provider type")
	ErrInvalidPublicationState     = errors.New("invalid publication state")
	ErrInvalidFieldSchema          = errors.New("invalid provider field schema")
	ErrInvalidCapability           = errors.New("invalid provider capability profile")
	ErrInvalidStatusMapping        = errors.New("invalid provider status mapping")
	ErrInvalidGap                  = errors.New("invalid provider integration gap")
	ErrProviderGapNotFound         = errors.New("provider integration gap not found")
)

// ProviderRegistryService owns the provider registry, lifecycle transitions,
// publication audit, and tenant private-beta enablement. It enforces the
// lifecycle graph and the immutability of published versions. Lifecycle
// mutations run in a transaction that locks the owning definition row, so the
// version-state change, the latest-published pointer, and the audit event
// commit atomically and concurrent transitions on one provider serialize.
type ProviderRegistryService struct {
	pool    *pgxpool.Pool
	defs    *repository.ProviderDefinitionRepository
	vers    *repository.ProviderVersionRepository
	pub     *repository.ProviderPublicationRepository
	schemas *repository.ProviderSchemaRepository
	caps    *repository.ProviderCapabilityRepository
	// metrics is an OPTIONAL, best-effort observability handle (OPE-422). nil-receiver
	// safe, so every record call is a no-op when it is not wired and can never affect
	// a lifecycle transition.
	metrics *obsmetrics.FulfillmentMetrics
}

// NewProviderRegistryService wires the service to the pool and its repositories.
func NewProviderRegistryService(
	pool *pgxpool.Pool,
	defs *repository.ProviderDefinitionRepository,
	vers *repository.ProviderVersionRepository,
	pub *repository.ProviderPublicationRepository,
	schemas *repository.ProviderSchemaRepository,
	caps *repository.ProviderCapabilityRepository,
) *ProviderRegistryService {
	return &ProviderRegistryService{pool: pool, defs: defs, vers: vers, pub: pub, schemas: schemas, caps: caps}
}

// WithMetrics wires the OPTIONAL metrics collector (OPE-422). Returns the service
// for chaining. Safe to omit: the collector is nil-receiver safe.
func (s *ProviderRegistryService) WithMetrics(m *obsmetrics.FulfillmentMetrics) *ProviderRegistryService {
	if s == nil {
		return nil
	}
	s.metrics = m
	return s
}

// SetCapabilities replaces the capability profile set of a version (frozen once
// published; structurally validated). Atomic DELETE+INSERT; the frozen check is
// re-run under the definition-row lock so it cannot race a publish transition.
func (s *ProviderRegistryService) SetCapabilities(ctx context.Context, versionID uuid.UUID, caps []model.ProviderCapability) ([]model.ProviderCapability, error) {
	// Fast pre-check for a friendly error; re-checked authoritatively under lock.
	v, err := s.GetVersion(ctx, versionID)
	if err != nil {
		return nil, err
	}
	if model.IsPublishedState(v.PublicationState) {
		return nil, ErrProviderVersionFrozen
	}
	if err := model.ValidateCapabilities(caps); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidCapability, err.Error())
	}
	if err := s.inTx(ctx, func(tx pgx.Tx) error {
		if err := requireUnpublishedLocked(ctx, tx, versionID); err != nil {
			return err
		}
		return s.caps.ReplaceCapabilities(ctx, tx, versionID, caps)
	}); err != nil {
		return nil, err
	}
	return s.caps.ListCapabilities(ctx, versionID)
}

// GetCapabilities returns a version's capability profiles.
func (s *ProviderRegistryService) GetCapabilities(ctx context.Context, versionID uuid.UUID) ([]model.ProviderCapability, error) {
	if _, err := s.GetVersion(ctx, versionID); err != nil {
		return nil, err
	}
	return s.caps.ListCapabilities(ctx, versionID)
}

// SetStatusMappings replaces the status-mapping set of a version (frozen once
// published; structurally validated). Atomic DELETE+INSERT; the frozen check is
// re-run under the definition-row lock so it cannot race a publish transition.
func (s *ProviderRegistryService) SetStatusMappings(ctx context.Context, versionID uuid.UUID, mappings []model.ProviderStatusMapping) ([]model.ProviderStatusMapping, error) {
	// Fast pre-check for a friendly error; re-checked authoritatively under lock.
	v, err := s.GetVersion(ctx, versionID)
	if err != nil {
		return nil, err
	}
	if model.IsPublishedState(v.PublicationState) {
		return nil, ErrProviderVersionFrozen
	}
	if err := model.ValidateStatusMappings(mappings); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidStatusMapping, err.Error())
	}
	if err := s.inTx(ctx, func(tx pgx.Tx) error {
		if err := requireUnpublishedLocked(ctx, tx, versionID); err != nil {
			return err
		}
		return s.caps.ReplaceStatusMappings(ctx, tx, versionID, mappings)
	}); err != nil {
		return nil, err
	}
	return s.caps.ListStatusMappings(ctx, versionID)
}

// GetStatusMappings returns a version's status mappings.
func (s *ProviderRegistryService) GetStatusMappings(ctx context.Context, versionID uuid.UUID) ([]model.ProviderStatusMapping, error) {
	if _, err := s.GetVersion(ctx, versionID); err != nil {
		return nil, err
	}
	return s.caps.ListStatusMappings(ctx, versionID)
}

// CreateGap records an integration gap for a version.
func (s *ProviderRegistryService) CreateGap(ctx context.Context, versionID uuid.UUID, gapType, severity, description string) (*model.ProviderIntegrationGap, error) {
	if _, err := s.GetVersion(ctx, versionID); err != nil {
		return nil, err
	}
	if !model.IsValidGapType(gapType) || !model.IsValidGapSeverity(severity) {
		return nil, ErrInvalidGap
	}
	return s.caps.CreateGap(ctx, s.pool, versionID, gapType, severity, description)
}

// ListGaps returns the gaps for a version.
func (s *ProviderRegistryService) ListGaps(ctx context.Context, versionID uuid.UUID) ([]model.ProviderIntegrationGap, error) {
	if _, err := s.GetVersion(ctx, versionID); err != nil {
		return nil, err
	}
	return s.caps.ListGaps(ctx, versionID)
}

// UpdateGapStatus changes a gap's lifecycle status.
func (s *ProviderRegistryService) UpdateGapStatus(ctx context.Context, gapID uuid.UUID, status string) (*model.ProviderIntegrationGap, error) {
	if !model.IsValidGapStatus(status) {
		return nil, ErrInvalidGap
	}
	g, err := s.caps.UpdateGapStatus(ctx, gapID, status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProviderGapNotFound
		}
		return nil, err
	}
	return g, nil
}

// SetSchema stores the credential/settings field schema for a version. Rejected
// once the version is published (frozen), or when the schema is structurally
// invalid. The frozen check is re-run under the definition-row lock in the same
// transaction as the write, so it cannot race a publish transition.
func (s *ProviderRegistryService) SetSchema(ctx context.Context, versionID uuid.UUID, groups []model.ProviderFieldGroup) (*model.ProviderFieldSchema, error) {
	// Fast pre-check for a friendly error; re-checked authoritatively under lock.
	v, err := s.GetVersion(ctx, versionID)
	if err != nil {
		return nil, err
	}
	if model.IsPublishedState(v.PublicationState) {
		return nil, ErrProviderVersionFrozen
	}
	if err := model.ValidateFieldSchema(groups); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidFieldSchema, err.Error())
	}
	var saved *model.ProviderFieldSchema
	if err := s.inTx(ctx, func(tx pgx.Tx) error {
		if err := requireUnpublishedLocked(ctx, tx, versionID); err != nil {
			return err
		}
		saved, err = s.schemas.Upsert(ctx, tx, versionID, groups)
		return err
	}); err != nil {
		return nil, err
	}
	return saved, nil
}

// GetSchema returns the field schema for a version. If the version exists but
// has no schema yet, an empty schema is returned (not an error).
func (s *ProviderRegistryService) GetSchema(ctx context.Context, versionID uuid.UUID) (*model.ProviderFieldSchema, error) {
	if _, err := s.GetVersion(ctx, versionID); err != nil {
		return nil, err
	}
	schema, err := s.schemas.GetByVersion(ctx, versionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &model.ProviderFieldSchema{ProviderVersionID: versionID, Groups: []model.ProviderFieldGroup{}}, nil
		}
		return nil, err
	}
	return schema, nil
}

// runInTx runs fn inside a transaction on pool, committing on success and
// rolling back on error.
func runInTx(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *ProviderRegistryService) inTx(ctx context.Context, fn func(pgx.Tx) error) error {
	return runInTx(ctx, s.pool, fn)
}

// CreateProviderDefinitionInput is the payload for creating a provider family.
type CreateProviderDefinitionInput struct {
	ProviderKey     string
	DisplayName     string
	ProviderType    string
	Regions         []string
	BusinessDomains []string
	Owner           string
	SourceLinks     []string
	Notes           string
}

// CreateDefinition creates a provider family definition.
func (s *ProviderRegistryService) CreateDefinition(ctx context.Context, in CreateProviderDefinitionInput) (*model.ProviderDefinition, error) {
	if !model.IsValidProviderType(in.ProviderType) {
		return nil, ErrInvalidProviderType
	}
	return s.defs.Create(ctx, model.ProviderDefinition{
		ProviderKey:     in.ProviderKey,
		DisplayName:     in.DisplayName,
		ProviderType:    in.ProviderType,
		Regions:         in.Regions,
		BusinessDomains: in.BusinessDomains,
		Owner:           in.Owner,
		SourceLinks:     in.SourceLinks,
		Notes:           in.Notes,
	})
}

// ListDefinitions returns all provider definitions.
func (s *ProviderRegistryService) ListDefinitions(ctx context.Context) ([]model.ProviderDefinition, error) {
	return s.defs.List(ctx)
}

// GetDefinition returns a definition by id.
func (s *ProviderRegistryService) GetDefinition(ctx context.Context, id uuid.UUID) (*model.ProviderDefinition, error) {
	d, err := s.defs.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProviderDefinitionNotFound
		}
		return nil, err
	}
	return d, nil
}

// GetDefinitionByKey returns a definition by its provider_key, or
// ErrProviderDefinitionNotFound.
func (s *ProviderRegistryService) GetDefinitionByKey(ctx context.Context, key string) (*model.ProviderDefinition, error) {
	d, err := s.defs.GetByKey(ctx, key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProviderDefinitionNotFound
		}
		return nil, err
	}
	return d, nil
}

// UpdateDefinitionMetadata updates editable definition fields.
func (s *ProviderRegistryService) UpdateDefinitionMetadata(ctx context.Context, in model.ProviderDefinition) (*model.ProviderDefinition, error) {
	if !model.IsValidProviderType(in.ProviderType) {
		return nil, ErrInvalidProviderType
	}
	if _, err := s.GetDefinition(ctx, in.ID); err != nil {
		return nil, err
	}
	return s.defs.UpdateMetadata(ctx, in)
}

// CreateVersion creates a new draft (research) version under a definition and
// records a 'create' publication event, atomically.
func (s *ProviderRegistryService) CreateVersion(ctx context.Context, defID uuid.UUID, version, changelog, compat string, actorID *uuid.UUID) (*model.ProviderVersion, error) {
	if _, err := s.GetDefinition(ctx, defID); err != nil {
		return nil, err
	}
	var created *model.ProviderVersion
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		v, err := s.vers.Create(ctx, tx, model.ProviderVersion{
			ProviderDefinitionID: defID,
			Version:              version,
			PublicationState:     model.ProviderStateResearch,
			Changelog:            changelog,
			CompatibilityNotes:   compat,
			CreatedBy:            actorID,
		})
		if err != nil {
			return err
		}
		created = v
		return s.pub.LogEvent(ctx, tx, v.ID, nil, model.ProviderStateResearch, model.ProviderEventCreate, actorID, "", nil)
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

// GetVersion returns a version by id.
func (s *ProviderRegistryService) GetVersion(ctx context.Context, id uuid.UUID) (*model.ProviderVersion, error) {
	v, err := s.vers.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProviderVersionNotFound
		}
		return nil, err
	}
	return v, nil
}

// ListVersions returns the versions of a definition (404 if the definition is absent).
func (s *ProviderRegistryService) ListVersions(ctx context.Context, defID uuid.UUID) ([]model.ProviderVersion, error) {
	if _, err := s.GetDefinition(ctx, defID); err != nil {
		return nil, err
	}
	return s.vers.ListByDefinition(ctx, defID)
}

// UpdateVersionMetadata edits changelog/compatibility notes; rejected once the
// version is in a published (frozen) state. The frozen check is re-run under
// the definition-row lock in the same transaction as the write, so it cannot
// race a publish transition.
func (s *ProviderRegistryService) UpdateVersionMetadata(ctx context.Context, versionID uuid.UUID, changelog, compat string) (*model.ProviderVersion, error) {
	// Fast pre-check for a friendly error; re-checked authoritatively under lock.
	v, err := s.GetVersion(ctx, versionID)
	if err != nil {
		return nil, err
	}
	if model.IsPublishedState(v.PublicationState) {
		return nil, ErrProviderVersionFrozen
	}
	var updated *model.ProviderVersion
	if err := s.inTx(ctx, func(tx pgx.Tx) error {
		if err := requireUnpublishedLocked(ctx, tx, versionID); err != nil {
			return err
		}
		updated, err = s.vers.UpdateMetadata(ctx, tx, versionID, changelog, compat)
		return err
	}); err != nil {
		return nil, err
	}
	return updated, nil
}

// lockProviderVersionState locks the version's owning definition row
// (serializing all lifecycle operations and frozen-checked mutations for that
// provider family) and returns the version's authoritative current state +
// definition id read under that lock. Shared by the registry and validation
// services so every published-version freeze check serializes with Transition.
//
// The lock and the state read are two statements on purpose: under READ
// COMMITTED a single blocked `SELECT ... FOR UPDATE OF d` would, after the
// conflicting transaction commits, still return the version row from its
// pre-wait snapshot (EvalPlanQual only re-reads the locked definition row,
// which the lifecycle code locks but never modifies). The second statement
// starts after the lock is held and therefore sees the committed state.
func lockProviderVersionState(ctx context.Context, tx pgx.Tx, versionID uuid.UUID) (string, uuid.UUID, error) {
	var defID uuid.UUID
	err := tx.QueryRow(ctx,
		`SELECT d.id
		   FROM provider_versions v
		   JOIN provider_definitions d ON d.id = v.provider_definition_id
		  WHERE v.id = $1
		  FOR UPDATE OF d`, versionID).Scan(&defID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", uuid.Nil, ErrProviderVersionNotFound
		}
		return "", uuid.Nil, err
	}
	var state string
	err = tx.QueryRow(ctx,
		`SELECT publication_state FROM provider_versions WHERE id = $1`, versionID).Scan(&state)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", uuid.Nil, ErrProviderVersionNotFound
		}
		return "", uuid.Nil, err
	}
	return state, defID, nil
}

// requireUnpublishedLocked locks the version's owning definition row and
// returns ErrProviderVersionFrozen when the version is already in a published
// (frozen) state. It is the authoritative re-check that frozen-gated mutations
// run inside their transaction, so check+write are atomic and serialized with
// Transition (OPE-515).
func requireUnpublishedLocked(ctx context.Context, tx pgx.Tx, versionID uuid.UUID) error {
	state, _, err := lockProviderVersionState(ctx, tx, versionID)
	if err != nil {
		return err
	}
	if model.IsPublishedState(state) {
		return ErrProviderVersionFrozen
	}
	return nil
}

// Transition moves a version along the lifecycle graph. The authoritative
// transition check, state write, latest-published reconciliation, and audit
// event all happen under the definition row lock, atomically.
func (s *ProviderRegistryService) Transition(ctx context.Context, versionID uuid.UUID, toState string, actorID *uuid.UUID, reason string) (*model.ProviderVersion, error) {
	if !model.IsValidPublicationState(toState) {
		return nil, ErrInvalidPublicationState
	}
	// Fast pre-check for a friendly error; re-checked authoritatively under lock.
	pre, err := s.GetVersion(ctx, versionID)
	if err != nil {
		return nil, err
	}
	if !model.CanTransition(pre.PublicationState, toState) {
		return nil, fmt.Errorf("%w: %s -> %s", ErrIllegalProviderTransition, pre.PublicationState, toState)
	}
	err = s.inTx(ctx, func(tx pgx.Tx) error {
		from, defID, err := lockProviderVersionState(ctx, tx, versionID)
		if err != nil {
			return err
		}
		if !model.CanTransition(from, toState) {
			return fmt.Errorf("%w: %s -> %s", ErrIllegalProviderTransition, from, toState)
		}
		if err := s.applyTransitionTx(ctx, tx, versionID, defID, from, toState, actorID); err != nil {
			return err
		}
		return s.pub.LogEvent(ctx, tx, versionID, &from, toState, model.ProviderEventTransition, actorID, reason, nil)
	})
	if err != nil {
		return nil, err
	}
	// Best-effort metric AFTER the transition commits (OPE-422). toState is a bounded
	// publication-state enum (validated above); the collector coerces anything else to
	// "other". version id / actor id are NEVER used as labels. nil-safe.
	s.metrics.RecordPublicationTransition(toState)
	return s.vers.GetByID(ctx, versionID)
}

// applyTransitionTx writes the version state and reconciles the definition's
// latest-published pointer within the caller's transaction.
func (s *ProviderRegistryService) applyTransitionTx(ctx context.Context, tx pgx.Tx, versionID, defID uuid.UUID, from, to string, actorID *uuid.UUID) error {
	var publishedBy *uuid.UUID
	var publishedAt *time.Time
	if to == model.ProviderStateAvailable {
		publishedBy = actorID
		now := time.Now().UTC()
		publishedAt = &now
	}
	if _, err := s.vers.UpdateState(ctx, tx, versionID, to, publishedBy, publishedAt); err != nil {
		return err
	}
	return s.reconcileLatestPublished(ctx, tx, defID, versionID, from, to)
}

// reconcileLatestPublished keeps provider_definitions.latest_published_version_id
// in sync: set to this version when it becomes available, repoint to the next
// available version (or nil) when it leaves available.
func (s *ProviderRegistryService) reconcileLatestPublished(ctx context.Context, tx pgx.Tx, defID, versionID uuid.UUID, from, to string) error {
	switch {
	case to == model.ProviderStateAvailable:
		return s.defs.SetLatestPublishedVersion(ctx, tx, defID, &versionID)
	case from == model.ProviderStateAvailable && to != model.ProviderStateAvailable:
		prev, err := s.vers.LatestAvailableExcluding(ctx, tx, defID, versionID)
		if err != nil {
			return err
		}
		var ptr *uuid.UUID
		if prev != nil {
			ptr = &prev.ID
		}
		return s.defs.SetLatestPublishedVersion(ctx, tx, defID, ptr)
	default:
		return nil
	}
}

// EmergencyDisable immediately pulls an available/private_beta version back to
// internal_validation (removing customer visibility) and rolls the definition's
// latest-published pointer back to the previous available version. Atomic.
func (s *ProviderRegistryService) EmergencyDisable(ctx context.Context, versionID uuid.UUID, actorID *uuid.UUID, reason string) (*model.ProviderVersion, error) {
	pre, err := s.GetVersion(ctx, versionID)
	if err != nil {
		return nil, err
	}
	if pre.PublicationState != model.ProviderStateAvailable && pre.PublicationState != model.ProviderStatePrivateBeta {
		return nil, ErrProviderDisableStateInvalid
	}
	err = s.inTx(ctx, func(tx pgx.Tx) error {
		from, defID, err := lockProviderVersionState(ctx, tx, versionID)
		if err != nil {
			return err
		}
		if from != model.ProviderStateAvailable && from != model.ProviderStatePrivateBeta {
			return ErrProviderDisableStateInvalid
		}
		if err := s.applyTransitionTx(ctx, tx, versionID, defID, from, model.ProviderStateInternalValidation, actorID); err != nil {
			return err
		}
		return s.pub.LogEvent(ctx, tx, versionID, &from, model.ProviderStateInternalValidation, model.ProviderEventEmergencyDisable, actorID, reason, nil)
	})
	if err != nil {
		return nil, err
	}
	return s.vers.GetByID(ctx, versionID)
}

// EnableTenant adds a tenant to a version's private-beta allowlist. Only valid
// for versions in private_beta or available.
func (s *ProviderRegistryService) EnableTenant(ctx context.Context, versionID, tenantID uuid.UUID, actorID *uuid.UUID) (*model.ProviderTenantEnable, error) {
	v, err := s.GetVersion(ctx, versionID)
	if err != nil {
		return nil, err
	}
	if v.PublicationState != model.ProviderStatePrivateBeta && v.PublicationState != model.ProviderStateAvailable {
		return nil, ErrProviderEnableStateInvalid
	}
	return s.pub.EnableTenant(ctx, versionID, tenantID, actorID)
}

// ListPublicationEvents returns the publication audit trail for a version.
func (s *ProviderRegistryService) ListPublicationEvents(ctx context.Context, versionID uuid.UUID) ([]model.ProviderPublicationEvent, error) {
	return s.pub.ListEvents(ctx, versionID)
}
