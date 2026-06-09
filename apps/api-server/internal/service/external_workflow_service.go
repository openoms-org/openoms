package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// EventExternalWorkflow is the orchestration outbox event type for an external-workflow
// dispatch. The ExternalWorkflowHandler (dispatcher) consumes it.
const EventExternalWorkflow = "automation.external_workflow"

// EventExternalWorkflowCommand is the follow-on command enqueued from a callback. It is
// applied by the handler registered for this type (set_status/add_tag/add_note), through the
// audited orchestration outbox; the external engine never writes durable state directly.
const EventExternalWorkflowCommand = "automation.external_workflow_command"

// ErrCorrelationNotResolvable is returned when a callback's nonce does not match an in-flight,
// unresolved event for the token's integration (unknown, already resolved, or wrong integration).
var ErrCorrelationNotResolvable = errors.New("external workflow correlation not resolvable")

// ErrExternalWorkflowBadSignature is returned when the callback body's HMAC signature does not
// verify against the integration's signing secret.
var ErrExternalWorkflowBadSignature = errors.New("external workflow bad signature")

// ExternalWorkflowService is gated (Enabled): when off, EnqueueDispatch is a no-op and the
// callback path is never reached (the route is unregistered). It owns the correlation enqueue
// and the callback resolution; the actual outbound POST is the ExternalWorkflowHandler.
type ExternalWorkflowService struct {
	enabled       bool
	pool          *pgxpool.Pool
	workerPool    *pgxpool.Pool // privileged, for the cross-tenant token lookup
	fulfillment   *FulfillmentService
	orchestration *repository.OrchestrationRepository
	tokens        *repository.ExternalWorkflowTokenRepository
	audit         repository.AuditRepo

	// orderLoader loads an order (RLS-scoped, inside a WithTenant tx) so the action can
	// build the redacted payload from real order fields. Wired via SetOrderLoader.
	orderLoader func(ctx context.Context, tx pgx.Tx, orderID uuid.UUID) (*model.Order, error)
	// configLoader decrypts the integration's external-workflow config (allowlist + secret).
	// Wired via SetConfigLoader.
	configLoader func(ctx context.Context, tenantID, integrationID uuid.UUID) (ExternalWorkflowConfig, error)
}

// ExternalWorkflowConfig is the per-integration dispatch config decrypted from the integration
// credentials JSONB. Shared by the action (allowlist), the callback (signing secret), and the
// dispatch handler (outbound URL / timeout / criticality).
type ExternalWorkflowConfig struct {
	OutboundURL            string   `json:"outbound_url"`
	SigningSecret          string   `json:"signing_secret"`
	TimeoutSecs            int      `json:"timeout_seconds"`
	Criticality            string   `json:"criticality"` // "warning" | "blocker"
	OutboundFieldAllowlist []string `json:"outbound_field_allowlist"`
}

// NewExternalWorkflowService constructs the service.
func NewExternalWorkflowService(enabled bool, pool, workerPool *pgxpool.Pool, fulfillment *FulfillmentService,
	orchestration *repository.OrchestrationRepository, tokens *repository.ExternalWorkflowTokenRepository, audit repository.AuditRepo) *ExternalWorkflowService {
	return &ExternalWorkflowService{enabled: enabled, pool: pool, workerPool: workerPool,
		fulfillment: fulfillment, orchestration: orchestration, tokens: tokens, audit: audit}
}

// SetOrderLoader wires the RLS-scoped order loader used to build the redacted payload.
func (s *ExternalWorkflowService) SetOrderLoader(fn func(ctx context.Context, tx pgx.Tx, orderID uuid.UUID) (*model.Order, error)) {
	if s != nil {
		s.orderLoader = fn
	}
}

// SetConfigLoader wires the integration-config decrypt function.
func (s *ExternalWorkflowService) SetConfigLoader(fn func(ctx context.Context, tenantID, integrationID uuid.UUID) (ExternalWorkflowConfig, error)) {
	if s != nil {
		s.configLoader = fn
	}
}

// Enabled reports whether the connector is active. Nil-safe.
func (s *ExternalWorkflowService) Enabled() bool { return s != nil && s.enabled }

// DispatchForOrder is the automation-action entry point: it opens its own tenant tx, loads the
// integration config (allowlist) + the order, builds the redacted payload, and enqueues a
// correlated external-workflow event. No-op when disabled. Returns the correlation nonce.
func (s *ExternalWorkflowService) DispatchForOrder(ctx context.Context, tenantID, orderID, integrationID uuid.UUID) error {
	if !s.Enabled() {
		return nil
	}
	if s.configLoader == nil || s.orderLoader == nil {
		return errors.New("external workflow: loaders not wired")
	}
	cfg, err := s.configLoader(ctx, tenantID, integrationID)
	if err != nil {
		return fmt.Errorf("external workflow: load integration config: %w", err)
	}
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		order, oerr := s.orderLoader(ctx, tx, orderID)
		if oerr != nil {
			return fmt.Errorf("external workflow: load order: %w", oerr)
		}
		if order == nil {
			return fmt.Errorf("external workflow: order %s not found", orderID)
		}
		redacted := model.BuildRedactedPayload(orderToMap(order), cfg.OutboundFieldAllowlist)
		_, eerr := s.enqueueDispatch(ctx, tx, tenantID, orderID, integrationID, redacted)
		return eerr
	})
}

// enqueueDispatch enqueues a correlated external-workflow event for an order, inside the
// caller's tenant tx. Ensures the order has a fulfillment process (the outbox process_id FK),
// then enqueues the event carrying the integration, a single-use correlation nonce, and the
// redacted payload. Returns the nonce.
func (s *ExternalWorkflowService) enqueueDispatch(ctx context.Context, tx pgx.Tx, tenantID, orderID, integrationID uuid.UUID, redacted map[string]any) (string, error) {
	proc, err := s.fulfillment.EnsureProcessForOrderUnconditional(ctx, tx, tenantID, orderID)
	if err != nil {
		return "", fmt.Errorf("ensure process for external workflow: %w", err)
	}
	nonce := uuid.NewString()
	payload := map[string]any{
		"integration_id":    integrationID.String(),
		"correlation_nonce": nonce,
		"order_id":          orderID.String(),
		"redacted_payload":  redacted,
	}
	if _, _, err := s.orchestration.EnqueueEvent(ctx, tx, model.OrchestrationOutboxEvent{
		TenantID:       tenantID,
		ProcessID:      proc.ID,
		EventType:      EventExternalWorkflow,
		IdempotencyKey: EventExternalWorkflow + ":" + orderID.String() + ":" + nonce,
		Payload:        payload,
	}); err != nil {
		return "", fmt.Errorf("enqueue external workflow event: %w", err)
	}
	return nonce, nil
}

// ResolveCallbackHTTP is the HTTP-facing entry point: it resolves the token cross-tenant, loads
// the integration signing secret, verifies the HMAC body signature (constant-time), and then
// delegates to the resolution logic. Returns the resolved tenant for the handler.
func (s *ExternalWorkflowService) ResolveCallbackHTTP(ctx context.Context, tokenHash, sigHeader string, body []byte, req model.ExternalWorkflowCallbackRequest, ip string, now time.Time) (uuid.UUID, error) {
	tok, err := s.tokens.FindByHashCrossTenant(ctx, s.workerPool, tokenHash)
	if err != nil {
		return uuid.Nil, err // ErrExternalWorkflowTokenNotFound -> handler maps to 401
	}
	if tok.IsExpired(now) {
		return uuid.Nil, repository.ErrExternalWorkflowTokenNotFound
	}
	if s.configLoader == nil {
		return uuid.Nil, errors.New("external workflow: config loader not wired")
	}
	cfg, err := s.configLoader(ctx, tok.TenantID, tok.IntegrationID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("external workflow: load integration config: %w", err)
	}
	if !model.VerifyExternalWorkflowSignature(body, sigHeader, cfg.SigningSecret) {
		return uuid.Nil, ErrExternalWorkflowBadSignature
	}
	return tok.TenantID, s.resolveCallback(ctx, tok, req, ip)
}

// resolveCallback re-scopes via WithTenant to find the in-flight correlated event, marks it
// succeeded/failed, optionally enqueues one in-scope follow-on command, records the external
// exec id, touches last_used_at, and audits. The token is already authenticated.
func (s *ExternalWorkflowService) resolveCallback(ctx context.Context, tok *model.ExternalWorkflowToken, req model.ExternalWorkflowCallbackRequest, ip string) error {
	tenantID := tok.TenantID
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		event, e := s.orchestration.FindPendingByEventAndNonce(ctx, tx, EventExternalWorkflow, req.CorrelationNonce, tok.IntegrationID)
		if e != nil {
			return ErrCorrelationNotResolvable
		}
		// Resolve the correlated event (single-use: it is no longer pending after this).
		success := req.Status == "succeeded"
		if success {
			if e := s.orchestration.MarkSucceeded(ctx, tx, event.ID); e != nil {
				return e
			}
		} else {
			if e := s.orchestration.MarkFailedPermanent(ctx, tx, event.ID, "external workflow reported failure"); e != nil {
				return e
			}
		}
		if req.ExternalExecID != "" {
			_ = s.orchestration.SetLatestAttemptExternalExecID(ctx, tx, event.ID, req.ExternalExecID)
		}
		// Optional single follow-on command (only if whitelisted AND in the token scope).
		if success && req.Command != "" {
			if !model.IsWhitelistedCallbackCommand(req.Command) || !tok.ScopeAllows(req.Command) {
				return fmt.Errorf("callback command %q not permitted", req.Command) // -> 403
			}
			cmdPayload := map[string]any{
				"order_id": payloadOrderID(event),
				"command":  req.Command,
				"value":    req.CommandValue,
			}
			if _, _, e := s.orchestration.EnqueueEvent(ctx, tx, model.OrchestrationOutboxEvent{
				TenantID:       tenantID,
				ProcessID:      event.ProcessID,
				EventType:      EventExternalWorkflowCommand,
				IdempotencyKey: EventExternalWorkflowCommand + ":" + req.CorrelationNonce,
				Payload:        cmdPayload,
			}); e != nil {
				return e
			}
		}
		if e := s.tokens.TouchLastUsed(ctx, tx, tok.ID); e != nil {
			return e
		}
		if s.audit != nil {
			_ = s.audit.Log(ctx, tx, model.AuditEntry{
				TenantID: tenantID, UserID: uuid.Nil, Action: "external_workflow.callback",
				EntityType: "orchestration_outbox", EntityID: event.ID,
				Changes: map[string]string{
					"status":                req.Status,
					"command":               req.Command,
					"external_execution_id": req.ExternalExecID,
				},
				IPAddress: ip,
			})
		}
		return nil
	})
}

// orderToMap converts an order to a generic map keyed by JSON field names so the redaction
// allowlist can project a stable, documented set of fields.
func orderToMap(order *model.Order) map[string]any {
	b, err := json.Marshal(order)
	if err != nil {
		return map[string]any{}
	}
	out := map[string]any{}
	_ = json.Unmarshal(b, &out)
	return out
}

// payloadOrderID extracts the order_id from an external-workflow event payload (scanned as any).
func payloadOrderID(event *model.OrchestrationOutboxEvent) string {
	if m, ok := event.Payload.(map[string]any); ok {
		if v, ok := m["order_id"].(string); ok {
			return v
		}
	}
	return ""
}
