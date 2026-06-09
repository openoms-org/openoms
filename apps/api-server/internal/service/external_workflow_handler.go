package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// DeferUntil signals the orchestration worker to leave the event PENDING with
// next_attempt_at = At (the callback deadline) instead of marking it succeeded. The
// external-workflow dispatch handler returns it after a successful outbound POST so the event
// is parked pending-callback: the worker will not re-pick it before the deadline, the callback
// resolves it first in the normal case, and if the deadline passes unresolved the worker
// re-dispatches (Attempts>0) and the handler fails it permanently (timeout blocker).
type DeferUntil struct{ At time.Time }

func (e *DeferUntil) Error() string { return "deferred until external workflow callback deadline" }

// externalWorkflowConfigLoader decrypts the per-integration external-workflow config.
type externalWorkflowConfigLoader func(ctx context.Context, tenantID, integrationID uuid.UUID) (ExternalWorkflowConfig, error)

// ExternalWorkflowHandler implements OrchestrationHandler for EventExternalWorkflow. On first
// dispatch it POSTs the signed payload and parks the event with next_attempt_at = now+timeout
// (via DeferUntil). If the callback resolves it first, the worker never re-dispatches. If the
// deadline passes unresolved, the worker re-dispatches and this handler — seeing Attempts>0 —
// fails it permanently so the worker opens an external_workflow_timeout blocker.
type ExternalWorkflowHandler struct {
	httpClient *http.Client // SSRF-safe client (noPrivateDialer), shared with webhook dispatch
	loadConfig externalWorkflowConfigLoader
}

// NewExternalWorkflowHandler constructs the dispatch handler.
func NewExternalWorkflowHandler(httpClient *http.Client, loadConfig externalWorkflowConfigLoader) *ExternalWorkflowHandler {
	return &ExternalWorkflowHandler{httpClient: httpClient, loadConfig: loadConfig}
}

// Handle dispatches or times out one external-workflow event.
func (h *ExternalWorkflowHandler) Handle(ctx context.Context, event model.OrchestrationOutboxEvent) error {
	payload, _ := event.Payload.(map[string]any)
	integrationIDStr, _ := payload["integration_id"].(string)
	nonce, _ := payload["correlation_nonce"].(string)

	// Re-dispatch past the deadline (the event was dispatched once and its deadline passed
	// with no callback) -> Attempts>0 -> timeout. Permanent so the worker opens a blocker.
	if event.Attempts > 0 {
		return model.Permanent(fmt.Errorf("external workflow callback timed out (nonce %s)", nonce))
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
