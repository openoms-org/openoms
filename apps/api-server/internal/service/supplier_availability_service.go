package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// SupplierAvailabilityService is the gated entry point (OPE-418) for resolving supplier
// availability and writing audited policies. When disabled it is a no-op: Enabled() is
// false and ResolveForProduct returns a zero decision so callers fall back to legacy stock.
type SupplierAvailabilityService struct {
	enabled bool
	pool    *pgxpool.Pool
	repo    *repository.SupplierAvailabilityRepository
	audit   repository.AuditRepo
}

// NewSupplierAvailabilityService constructs the service. enabled comes from
// cfg.SupplierAvailabilityEnabled; pool is the RLS-scoped app pool.
func NewSupplierAvailabilityService(enabled bool, pool *pgxpool.Pool, repo *repository.SupplierAvailabilityRepository, audit repository.AuditRepo) *SupplierAvailabilityService {
	return &SupplierAvailabilityService{enabled: enabled, pool: pool, repo: repo, audit: audit}
}

// Enabled reports whether the supplier-availability read-model is active. Nil-safe.
func (s *SupplierAvailabilityService) Enabled() bool { return s != nil && s.enabled }

// Repo returns the underlying repository so callers that already hold a tenant tx (e.g.
// the supplier sync) can upsert snapshots without a second pool round-trip. Nil-safe.
func (s *SupplierAvailabilityService) Repo() *repository.SupplierAvailabilityRepository {
	if s == nil {
		return nil
	}
	return s.repo
}

// ResolveForProduct loads the best snapshot + the policy chain for a (supplier, product,
// listing?, channel?) context and returns the resolved decision. When disabled it returns
// a zero decision with ok=false so callers keep the legacy behavior. Picks the snapshot
// with the most stock across warehouses (the order line is satisfied from any warehouse).
func (s *SupplierAvailabilityService) ResolveForProduct(ctx context.Context, tenantID, supplierID, productID uuid.UUID, listingID *uuid.UUID, channel *string, requestedQty int, preflightSupported bool, now time.Time) (model.AvailabilityDecision, bool, error) {
	if !s.Enabled() {
		return model.AvailabilityDecision{}, false, nil
	}
	var decision model.AvailabilityDecision
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		snaps, e := s.repo.ListSnapshotsByProduct(ctx, tx, productID)
		if e != nil {
			return e
		}
		if len(snaps) == 0 {
			// No snapshot recorded -> treat as unknown (untrusted) so routing is blocked.
			decision = model.AvailabilityDecision{Status: model.AvailabilityStatusUnknown}
			code := model.BlockerSupplierAvailabilityUnknown
			decision.BlockerCode = &code
			return nil
		}
		best := snaps[0]
		for _, sn := range snaps[1:] {
			if sn.SourceQuantity > best.SourceQuantity {
				best = sn
			}
		}
		policies, e := s.repo.ListPoliciesForContext(ctx, tx, supplierID, productID, listingID, channel)
		if e != nil {
			return e
		}
		eff := model.ResolvePolicyChain(sortPoliciesBySpecificity(policies))
		decision = model.ResolveAvailability(best, eff, requestedQty, preflightSupported, now)
		return nil
	})
	if err != nil {
		return model.AvailabilityDecision{}, false, fmt.Errorf("resolve supplier availability: %w", err)
	}
	return decision, true, nil
}

// scopeRank orders scopes least->most specific for the precedence fold.
func scopeRank(scope string) int {
	switch scope {
	case model.PolicyScopeSupplier:
		return 0
	case model.PolicyScopeProduct:
		return 1
	case model.PolicyScopeListing:
		return 2
	case model.PolicyScopeChannel:
		return 3
	default:
		return -1
	}
}

// sortPoliciesBySpecificity returns the policies ordered least->most specific so
// model.ResolvePolicyChain folds them with the most specific winning.
func sortPoliciesBySpecificity(in []model.SupplierAvailabilityPolicy) []model.SupplierAvailabilityPolicy {
	out := make([]model.SupplierAvailabilityPolicy, len(in))
	copy(out, in)
	sort.SliceStable(out, func(i, j int) bool { return scopeRank(out[i].Scope) < scopeRank(out[j].Scope) })
	return out
}

// SetPolicy upserts a scope policy and writes an audit entry when it changes the manual
// controls (override_quantity / mode) — the research requires an active override is never
// silently changed by automation.
func (s *SupplierAvailabilityService) SetPolicy(ctx context.Context, tenantID, actorID uuid.UUID, ip string, p model.SupplierAvailabilityPolicy) (*model.SupplierAvailabilityPolicy, error) {
	if !model.IsValidPolicyScope(p.Scope) || !model.IsValidPolicyMode(p.Mode) {
		return nil, NewValidationError(fmt.Errorf("invalid scope %q or mode %q", p.Scope, p.Mode))
	}
	p.TenantID = tenantID
	var out *model.SupplierAvailabilityPolicy
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		saved, e := s.repo.UpsertPolicy(ctx, tx, p)
		if e != nil {
			return e
		}
		out = saved
		if s.audit != nil && (p.OverrideQuantity != nil || p.Mode != model.PolicyModeAuto) {
			changes := map[string]string{"scope": p.Scope, "mode": p.Mode}
			if p.OverrideQuantity != nil {
				changes["override_quantity"] = fmt.Sprintf("%d", *p.OverrideQuantity)
			}
			return s.audit.Log(ctx, tx, model.AuditEntry{
				TenantID:   tenantID,
				UserID:     actorID,
				Action:     "supplier_availability.policy_override",
				EntityType: "supplier_availability_policy",
				EntityID:   saved.ID,
				Changes:    changes,
				IPAddress:  ip,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
