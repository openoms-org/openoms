package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// PlanSettings represents the limits portion of tenant.settings JSONB.
type PlanSettings struct {
	Limits             *PlanLimits `json:"limits,omitempty"`
	SubscriptionStatus string      `json:"subscription_status,omitempty"`
}

// PlanLimits defines per-plan resource limits.
type PlanLimits struct {
	MaxUsers         int `json:"max_users,omitempty"`
	MaxOrdersMonthly int `json:"max_orders_monthly,omitempty"`
	MaxIntegrations  int `json:"max_integrations,omitempty"`
}

// TenantPlanGuard checks tenant.plan and blocks requests for inactive plans.
//   - "suspended": 402 on ALL requests
//   - "past_due": 402 on mutations (POST/PUT/PATCH/DELETE), allows GET/HEAD/OPTIONS
//   - active plans: pass through
//
// When plan data cannot be resolved (DB/cache lookup fails or returns no plan),
// the guard fails CLOSED: state-changing requests (POST/PUT/PATCH/DELETE) are
// blocked while safe reads (GET/HEAD/OPTIONS) are still allowed. The short-TTL
// PlanCache absorbs brief DB blips so a transient outage does not block writes
// for tenants whose plan was recently resolved.
//
// pool can be nil (for testing) — if nil and the cache misses, the request
// passes through unguarded (test-only path; production always supplies a pool).
func TenantPlanGuard(cache *service.PlanCache, pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := TenantIDFromContext(r.Context())
			if tenantID == uuid.Nil {
				next.ServeHTTP(w, r)
				return
			}

			var plan string
			var settings json.RawMessage
			var err error

			if pool == nil {
				// Test-only path: no DB available. Only the cache can answer.
				plan, settings, _ = cache.Get(tenantID)
				if plan == "" {
					next.ServeHTTP(w, r)
					return
				}
			} else {
				plan, settings, err = cache.GetOrLoad(r.Context(), pool, tenantID)
				if err != nil || plan == "" {
					// Can't resolve plan from cache or DB — fail CLOSED.
					if failClosedOnUnavailablePlan(w, r) {
						return
					}
					next.ServeHTTP(w, r)
					return
				}
			}

			var ps PlanSettings
			if settings != nil {
				_ = json.Unmarshal(settings, &ps)
			}

			if blockForSubscriptionStatus(w, r, ps.SubscriptionStatus) {
				return
			}

			switch plan {
			case "suspended":
				writePlanError(w, "subscription_suspended",
					"subscription has been suspended")
				return

			case "past_due":
				if isMutation(r.Method) {
					writePlanError(w, "payment_past_due",
						"payment past due, write operations are blocked")
					return
				}
			}

			// Store parsed settings in context for downstream limit checks
			if ps.Limits != nil {
				ctx := context.WithValue(r.Context(), planLimitsKey, ps.Limits)
				r = r.WithContext(ctx)
			}

			next.ServeHTTP(w, r)
		})
	}
}

func blockForSubscriptionStatus(w http.ResponseWriter, r *http.Request, status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "active", "trialing":
		return false
	case "past_due", "unpaid", "incomplete":
		if isMutation(r.Method) {
			writePlanError(w, "payment_past_due",
				"payment past due, write operations are blocked")
			return true
		}
		return false
	case "canceled", "incomplete_expired", "paused":
		if isMutation(r.Method) {
			writePlanError(w, "subscription_inactive",
				"subscription is not active")
			return true
		}
		return false
	case "suspended":
		writePlanError(w, "subscription_suspended",
			"subscription has been suspended")
		return true
	default:
		if isMutation(r.Method) {
			writePlanError(w, "subscription_inactive",
				"subscription status is not active")
			return true
		}
		return false
	}
}

func isMutation(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

// failClosedOnUnavailablePlan blocks state-changing requests when the tenant's
// plan/subscription state cannot be resolved (cache miss AND DB lookup failed),
// so a suspended/past_due tenant cannot gain full write access during an outage.
// Safe reads (GET/HEAD/OPTIONS) are still allowed so the app degrades gracefully.
// Returns true if the request was blocked (caller must stop), false to proceed.
func failClosedOnUnavailablePlan(w http.ResponseWriter, r *http.Request) bool {
	if !isMutation(r.Method) {
		return false
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   "plan_check_unavailable",
		"message": "unable to verify subscription status, write operations are temporarily blocked",
	})
	return true
}

type planContextKey string

const planLimitsKey planContextKey = "plan_limits"

// PlanLimitsFromContext returns the plan limits from the request context (set by TenantPlanGuard).
func PlanLimitsFromContext(ctx context.Context) *PlanLimits {
	if limits, ok := ctx.Value(planLimitsKey).(*PlanLimits); ok {
		return limits
	}
	return nil
}

func writePlanError(w http.ResponseWriter, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusPaymentRequired)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   errorCode,
		"message": message,
	})
}
