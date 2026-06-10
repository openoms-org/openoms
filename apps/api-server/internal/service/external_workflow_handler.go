package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// DeferUntil signals the orchestration worker to leave the event PENDING with
// next_attempt_at = At (the callback deadline) instead of marking it succeeded. The
// external-workflow dispatch handler returns it after a successful outbound POST so the event
// is parked pending-callback: the worker records a succeeded attempt (the dispatch evidence)
// in the same transaction as the park and will not re-pick the event before the deadline. The
// callback resolves it first in the normal case; if the deadline passes unresolved the worker
// re-dispatches and the handler — seeing the dispatch evidence — fails it permanently
// (timeout blocker).
type DeferUntil struct{ At time.Time }

func (e *DeferUntil) Error() string { return "deferred until external workflow callback deadline" }

// externalWorkflowConfigLoader decrypts the per-integration external-workflow config.
type externalWorkflowConfigLoader func(ctx context.Context, tenantID, integrationID uuid.UUID) (ExternalWorkflowConfig, error)

// ExternalWorkflowHandler implements OrchestrationHandler for EventExternalWorkflow. On first
// dispatch it POSTs the signed payload and parks the event with next_attempt_at = now+timeout
// (via DeferUntil). If the callback resolves it first, the worker never re-dispatches. If the
// deadline passes unresolved, the worker re-dispatches and this handler — seeing the explicit
// dispatch evidence recorded at park time — fails it permanently so the worker opens an
// external_workflow_timeout blocker (OPE-514).
type ExternalWorkflowHandler struct {
	httpClient *http.Client // SSRF-safe client (noPrivateDialer), shared with webhook dispatch
	loadConfig externalWorkflowConfigLoader
	// pool is the privileged worker pool: the handler runs inside the cross-tenant
	// orchestration worker, and the evidence query filters by the event's tenant_id itself.
	pool          *pgxpool.Pool
	orchestration *repository.OrchestrationRepository
}

// NewExternalWorkflowHandler constructs the dispatch handler. pool MUST be the privileged
// worker pool (the orchestration worker invokes the handler across tenants).
func NewExternalWorkflowHandler(httpClient *http.Client, loadConfig externalWorkflowConfigLoader,
	pool *pgxpool.Pool, orchestration *repository.OrchestrationRepository) *ExternalWorkflowHandler {
	return &ExternalWorkflowHandler{httpClient: httpClient, loadConfig: loadConfig, pool: pool, orchestration: orchestration}
}

// Handle dispatches or times out one external-workflow event.
func (h *ExternalWorkflowHandler) Handle(ctx context.Context, event model.OrchestrationOutboxEvent) error {
	payload, _ := event.Payload.(map[string]any)
	integrationIDStr, _ := payload["integration_id"].(string)
	nonce, _ := payload["correlation_nonce"].(string)

	// OPE-514: classify "awaiting callback" from EXPLICIT dispatch evidence, never from
	// Attempts alone — the worker also increments Attempts on retryable failures, so a
	// transient transport error or 5xx on the very first POST must stay a normal retry, not
	// become a premature timeout blocker. The only path that records a SUCCEEDED attempt
	// while the event stays pending is the pending-callback park (written transactionally
	// with the park), so its presence proves an earlier POST succeeded and the deadline —
	// next_attempt_at, set to now+timeout at dispatch time — passed with no callback.
	if event.Attempts > 0 {
		dispatched, derr := h.orchestration.HasDispatchedAttempt(ctx, h.pool, event.TenantID, event.ID)
		if derr != nil {
			return fmt.Errorf("external workflow: check dispatch evidence: %w", derr) // retryable
		}
		if dispatched {
			if !time.Now().UTC().Before(event.NextAttemptAt) {
				// Past the deadline with no callback -> timeout. Permanent so the worker
				// opens an external_workflow_timeout blocker.
				return model.Permanent(fmt.Errorf("external workflow callback timed out (nonce %s)", nonce))
			}
			// Defensive (the worker only claims due events): dispatched but the deadline
			// has not passed yet — keep waiting for the callback, do not re-POST.
			return &DeferUntil{At: event.NextAttemptAt}
		}
		// No evidence: every earlier attempt failed before reaching the engine — this is a
		// retry of the dispatch itself, so fall through and POST normally.
	}

	integrationID, err := uuid.Parse(integrationIDStr)
	if err != nil {
		return model.Permanent(fmt.Errorf("external workflow: invalid integration_id %q: %w", integrationIDStr, err))
	}

	cfg, err := h.loadConfig(ctx, event.TenantID, integrationID)
	if err != nil {
		return fmt.Errorf("load external workflow config: %w", err) // retryable
	}
	if cfg.OutboundURL == "" {
		return model.Permanent(fmt.Errorf("external workflow: integration %s has no outbound_url", integrationID))
	}
	if cfg.SigningSecret == "" {
		// Refuse to dispatch unsigned (an empty key would produce a forgeable signature).
		return model.Permanent(fmt.Errorf("external workflow: integration %s has no signing_secret", integrationID))
	}

	body, _ := json.Marshal(payload["redacted_payload"])
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.OutboundURL, bytes.NewReader(body))
	if err != nil {
		return model.Permanent(fmt.Errorf("external workflow: build request: %w", err))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature-256", model.SignExternalWorkflowBody(body, cfg.SigningSecret))
	req.Header.Set("X-OpenOMS-Event", EventExternalWorkflow)
	req.Header.Set("X-OpenOMS-Correlation", nonce)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("dispatch external workflow: %w", err) // retryable transport error
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("external workflow returned %d", resp.StatusCode) // retryable
	}

	// Success: park pending-callback with next_attempt_at = now + timeout (the deadline).
	timeout := time.Duration(cfg.TimeoutSecs) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Minute
	}
	return &DeferUntil{At: time.Now().UTC().Add(timeout)}
}

// Compile-time assertion that the dispatch handler satisfies OrchestrationHandler.
var _ OrchestrationHandler = (*ExternalWorkflowHandler)(nil)
