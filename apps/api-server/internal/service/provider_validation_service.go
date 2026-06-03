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
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// Validation engine domain errors.
var (
	ErrInvalidProbe                 = errors.New("invalid validation probe")
	ErrInvalidValidationEnv         = errors.New("invalid validation environment")
	ErrInvalidResultStatus          = errors.New("invalid validation result status")
	ErrDestructiveProbeNotConfirmed = errors.New("destructive probes require explicit confirmation")
	ErrValidationRunNotFound        = errors.New("validation run not found")
	ErrValidationRunFinalized       = errors.New("validation run is finalized and immutable")
)

// ProviderValidationService owns provider-version validation: probe definitions,
// immutable validation runs/results, the run verdict, and auto-creation of
// integration gaps from failures. The actual probe execution against live
// providers is out of scope here; callers record already-redacted results.
type ProviderValidationService struct {
	pool *pgxpool.Pool
	vers *repository.ProviderVersionRepository
	caps *repository.ProviderCapabilityRepository
	val  *repository.ProviderValidationRepository
}

// NewProviderValidationService wires the service to the pool and repositories.
func NewProviderValidationService(
	pool *pgxpool.Pool,
	vers *repository.ProviderVersionRepository,
	caps *repository.ProviderCapabilityRepository,
	val *repository.ProviderValidationRepository,
) *ProviderValidationService {
	return &ProviderValidationService{pool: pool, vers: vers, caps: caps, val: val}
}

func (s *ProviderValidationService) inTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *ProviderValidationService) getVersion(ctx context.Context, versionID uuid.UUID) (*model.ProviderVersion, error) {
	v, err := s.vers.GetByID(ctx, versionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProviderVersionNotFound
		}
		return nil, err
	}
	return v, nil
}

// SetProbes replaces a version's probe set (frozen once published; validated).
func (s *ProviderValidationService) SetProbes(ctx context.Context, versionID uuid.UUID, probes []model.ProviderValidationProbe) ([]model.ProviderValidationProbe, error) {
	v, err := s.getVersion(ctx, versionID)
	if err != nil {
		return nil, err
	}
	if model.IsPublishedState(v.PublicationState) {
		return nil, ErrProviderVersionFrozen
	}
	if err := model.ValidateProbes(probes); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidProbe, err.Error())
	}
	if err := s.inTx(ctx, func(tx pgx.Tx) error {
		return s.val.ReplaceProbes(ctx, tx, versionID, probes)
	}); err != nil {
		return nil, err
	}
	return s.val.ListProbes(ctx, versionID)
}

// GetProbes returns a version's probes.
func (s *ProviderValidationService) GetProbes(ctx context.Context, versionID uuid.UUID) ([]model.ProviderValidationProbe, error) {
	if _, err := s.getVersion(ctx, versionID); err != nil {
		return nil, err
	}
	return s.val.ListProbes(ctx, versionID)
}

// StartRun opens a pending validation run. If any declared probe is destructive,
// allowDestructive must be true (safety gate).
func (s *ProviderValidationService) StartRun(ctx context.Context, versionID uuid.UUID, environment string, allowDestructive bool, actorID *uuid.UUID) (*model.ProviderValidationRun, error) {
	if _, err := s.getVersion(ctx, versionID); err != nil {
		return nil, err
	}
	if !model.IsValidValidationEnv(environment) {
		return nil, ErrInvalidValidationEnv
	}
	probes, err := s.val.ListProbes(ctx, versionID)
	if err != nil {
		return nil, err
	}
	if !allowDestructive {
		for _, p := range probes {
			if p.Destructive {
				return nil, ErrDestructiveProbeNotConfirmed
			}
		}
	}
	return s.val.CreateRun(ctx, versionID, environment, actorID)
}

// RecordResult records (or replaces) a probe result on a pending run.
func (s *ProviderValidationService) RecordResult(ctx context.Context, runID uuid.UUID, probeType, label, status, observation, payloadHash, findings string) (*model.ProviderValidationResult, error) {
	run, err := s.getRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run.Verdict != model.RunVerdictPending {
		return nil, ErrValidationRunFinalized
	}
	if !model.IsValidResultStatus(status) {
		return nil, ErrInvalidResultStatus
	}
	return s.val.UpsertResult(ctx, model.ProviderValidationResult{
		RunID: runID, ProbeType: probeType, Label: label, Status: status,
		Observation: observation, PayloadHash: payloadHash, Findings: findings,
	})
}

// CompleteRun finalizes a pending run: computes the verdict and, atomically,
// records the verdict + an integration gap for each failed/errored probe.
func (s *ProviderValidationService) CompleteRun(ctx context.Context, runID uuid.UUID) (*model.ProviderValidationRun, error) {
	run, err := s.getRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run.Verdict != model.RunVerdictPending {
		return nil, ErrValidationRunFinalized
	}
	results, err := s.val.ListResults(ctx, runID)
	if err != nil {
		return nil, err
	}
	verdict := model.VerdictFromResults(results)
	now := time.Now().UTC()
	if err := s.inTx(ctx, func(tx pgx.Tx) error {
		if err := s.val.FinalizeRun(ctx, tx, runID, verdict, now); err != nil {
			return err
		}
		for _, r := range results {
			gapType, severity := model.GapForFailedProbe(r.ProbeType, r.Status)
			if gapType == "" {
				continue
			}
			desc := fmt.Sprintf("validation probe %q (%s) %s", r.Label, r.ProbeType, r.Status)
			if _, err := s.caps.CreateGap(ctx, tx, run.ProviderVersionID, gapType, severity, desc); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return s.GetRunWithResults(ctx, runID)
}

func (s *ProviderValidationService) getRun(ctx context.Context, runID uuid.UUID) (*model.ProviderValidationRun, error) {
	run, err := s.val.GetRun(ctx, runID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrValidationRunNotFound
		}
		return nil, err
	}
	return run, nil
}

// ListRuns returns a version's validation runs.
func (s *ProviderValidationService) ListRuns(ctx context.Context, versionID uuid.UUID) ([]model.ProviderValidationRun, error) {
	if _, err := s.getVersion(ctx, versionID); err != nil {
		return nil, err
	}
	return s.val.ListRuns(ctx, versionID)
}

// GetRunWithResults returns a run header with its probe results.
func (s *ProviderValidationService) GetRunWithResults(ctx context.Context, runID uuid.UUID) (*model.ProviderValidationRun, error) {
	run, err := s.getRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	results, err := s.val.ListResults(ctx, runID)
	if err != nil {
		return nil, err
	}
	run.Results = results
	return run, nil
}
