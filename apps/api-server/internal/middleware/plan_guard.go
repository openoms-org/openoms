package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// PlanSettings represents the limits portion of tenant.settings JSONB.
type PlanSettings struct {
	Limits *PlanLimits `json:"limits,omitempty"`
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
// pool can be nil (for testing) — if nil and cache misses, request passes through.
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

			if pool != nil {
				plan, settings, err = cache.GetOrLoad(r.Context(), pool, tenantID)
			} else {
				plan, settings, _ = cache.Get(tenantID)
			}

			if err != nil || plan == "" {
				// Can't determine plan — fail open (allow request)
				next.ServeHTTP(w, r)
				return
			}

			switch plan {
			case "suspended":
				writePlanError(w, http.StatusPaymentRequired, "subscription_suspended",
					"Subskrypcja została zawieszona. Odnów płatność aby kontynuować.")
				return

			case "past_due":
				if isMutation(r.Method) {
					writePlanError(w, http.StatusPaymentRequired, "payment_past_due",
						"Płatność zaległa. Nowe operacje zablokowane do czasu uregulowania należności.")
					return
				}
			}

			// Store parsed settings in context for downstream limit checks
			if settings != nil {
				var ps PlanSettings
				if json.Unmarshal(settings, &ps) == nil && ps.Limits != nil {
					ctx := context.WithValue(r.Context(), planLimitsKey, ps.Limits)
					r = r.WithContext(ctx)
				}
			}

			next.ServeHTTP(w, r)
		})
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

type planContextKey string

const planLimitsKey planContextKey = "plan_limits"

// PlanLimitsFromContext returns the plan limits from the request context (set by TenantPlanGuard).
func PlanLimitsFromContext(ctx context.Context) *PlanLimits {
	if limits, ok := ctx.Value(planLimitsKey).(*PlanLimits); ok {
		return limits
	}
	return nil
}

func writePlanError(w http.ResponseWriter, status int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   errorCode,
		"message": message,
	})
}
