package handler

import (
	"net/http"
	"strings"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
)

// OperationsHandler serves the tenant-scoped operations control-tower endpoints
// under /v1/operations. It is a read-only aggregate view over the canonical
// fulfillment model and shares the FulfillmentReader read service.
type OperationsHandler struct {
	svc FulfillmentReader
}

// NewOperationsHandler creates an OperationsHandler over the read service.
func NewOperationsHandler(svc FulfillmentReader) *OperationsHandler {
	return &OperationsHandler{svc: svc}
}

// Summary returns the operator control-tower bucket counts (ready / processing /
// stuck / blocked / provider_issue / missing_data) for the tenant.
func (h *OperationsHandler) Summary(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	summary, err := h.svc.OperationsSummary(r.Context(), tenantID)
	if err != nil {
		writeServerError(w, "failed to load operations summary", err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

// Exceptions returns the actionable processes (blocked / waiting_external /
// stuck) with their top open blocker, for triage. Query param: limit (caps the
// number of processes scanned).
func (h *OperationsHandler) Exceptions(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	limit := queryInt(r, "limit", 0)

	items, err := h.svc.OperationsExceptions(r.Context(), tenantID, limit)
	if err != nil {
		writeServerError(w, "failed to load operations exceptions", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}

// IntegrationCapabilitySummary returns automation-coverage metrics derived from
// the tenant's recorded blockers + provider attempts. Query param: scan_limit
// (caps the number of actionable processes scanned).
func (h *OperationsHandler) IntegrationCapabilitySummary(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	scanLimit := queryInt(r, "scan_limit", 0)

	summary, err := h.svc.IntegrationCapabilitySummary(r.Context(), tenantID, scanLimit)
	if err != nil {
		writeServerError(w, "failed to load integration capability summary", err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

// Parity returns the legacy-vs-process-backed parity / verification report for the
// tenant (OPE-423): non-terminal order coverage by fulfillment processes, the
// coverage gap, the legacy "needs attention" order count vs the process-backed
// exception total, and the rollout parity verdict. Read-only. Query param:
// coverage_threshold (0<t<=1; defaults to service.DefaultParityCoverageThreshold).
func (h *OperationsHandler) Parity(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	threshold := queryFloat(r, "coverage_threshold", 0)

	report, err := h.svc.ParityReport(r.Context(), tenantID, threshold)
	if err != nil {
		writeServerError(w, "failed to load parity report", err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

// multiValueQuery returns all values for a query parameter, supporting both
// repeated keys (?k=a&k=b) and comma-separated values (?k=a,b). Empty entries
// are dropped.
func multiValueQuery(r *http.Request, key string) []string {
	raw := r.URL.Query()[key]
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		for part := range strings.SplitSeq(v, ",") {
			if p := strings.TrimSpace(part); p != "" {
				out = append(out, p)
			}
		}
	}
	return out
}
