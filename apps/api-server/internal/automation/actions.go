// Package automation implements the rule engine and action executors for workflow automation.
package automation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/crypto"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	allegroIntegration "github.com/openoms-org/openoms/apps/api-server/internal/integration/allegro"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/netutil"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"
)

// EventAutomationSetStatus is the orchestration outbox event type emitted when a
// set_status automation action is routed through the orchestration outbox (OPE-421)
// instead of calling OrderService.TransitionStatus directly. It is consumed by
// service.AutomationStatusTransitionHandler. Defined here (the producer) and
// referenced by the handler/registration to keep producer and consumer in sync.
const EventAutomationSetStatus = "automation.set_status"

// ActionExecutor executes a single automation action within the automation engine.
// Implementations may discard the caller context for actions that call external APIs
// (marketplace, email, webhooks) to ensure those calls complete regardless of the
// original request timeout or cancellation.
type ActionExecutor interface {
	// ExecuteAction dispatches and runs the given action for the specified tenant.
	// The action type determines which handler is invoked (e.g. "set_status", "webhook").
	ExecuteAction(ctx context.Context, tenantID uuid.UUID, action Action, event Event) error
}

// OrderStatusTransitioner transitions an order's status.
// Implemented by service.OrderService.
type OrderStatusTransitioner interface {
	TransitionStatus(ctx context.Context, tenantID, orderID uuid.UUID, req model.StatusTransitionRequest, actorID uuid.UUID, ip string) (*model.Order, error)
}

// OrderGetter retrieves an order by ID.
// Implemented by service.OrderService.
type OrderGetter interface {
	Get(ctx context.Context, tenantID, orderID uuid.UUID) (*model.Order, error)
}

// OrderTagUpdater updates order tags via the repository layer (within a tenant transaction).
// We define a narrow interface for the repo method we need.
type OrderTagUpdater interface {
	Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateOrderRequest) error
	FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Order, error)
}

// EmailSender sends order status emails.
// Implemented by service.EmailService.
type EmailSender interface {
	SendOrderStatusEmail(ctx context.Context, tenantID uuid.UUID, order *model.Order, oldStatus, newStatus string)
}

// OrchestrationEnqueuer enqueues an orchestration outbox event within the caller's
// tenant transaction (RLS-scoped, idempotent on tenant + idempotency key).
// Implemented by *repository.OrchestrationRepository.
type OrchestrationEnqueuer interface {
	EnqueueEvent(ctx context.Context, q repository.Querier, e model.OrchestrationOutboxEvent) (bool, *model.OrchestrationOutboxEvent, error)
}

// FulfillmentProcessEnsurer looks up / creates the fulfillment process anchoring an
// order's orchestration timeline. Implemented by *repository.FulfillmentRepository.
type FulfillmentProcessEnsurer interface {
	GetProcessByOrder(ctx context.Context, tx pgx.Tx, orderID uuid.UUID) (*model.FulfillmentProcess, error)
	CreateProcess(ctx context.Context, tx pgx.Tx, p model.FulfillmentProcess) (*model.FulfillmentProcess, error)
}

// InvoiceCreator handles auto-invoice creation on order status change.
// Implemented by service.InvoiceService.
type InvoiceCreator interface {
	HandleOrderStatusChange(ctx context.Context, tenantID uuid.UUID, order *model.Order)
}

// ExternalWorkflowDispatcher hands an order event to an external workflow engine via the
// OPE-421/Phase-13 connector (gated). Implemented by *service.ExternalWorkflowService.
// Enabled() is nil-safe and false unless the connector is wired AND EXTERNAL_WORKFLOW_ENABLED.
type ExternalWorkflowDispatcher interface {
	Enabled() bool
	DispatchForOrder(ctx context.Context, tenantID, orderID, integrationID uuid.UUID) error
}

// ListingActivatorDeps holds the dependencies needed by the activate_listing action.
type ListingActivatorDeps struct {
	ListingRepo     ListingActionRepo
	IntegrationRepo IntegrationActionRepo
	EncryptionKey   []byte
	// ProviderFactory creates a marketplace provider from decrypted credentials.
	// Returns (provider, needsClose, error). The caller must close the provider if needsClose is true.
	ProviderFactory func(provider string, credentials json.RawMessage, settings json.RawMessage) (ListingActivatorProvider, error)
}

// ListingActionRepo is the subset of repository.ProductListingRepo needed here.
type ListingActionRepo interface {
	ListByProduct(ctx context.Context, tx pgx.Tx, productID uuid.UUID) ([]*model.ProductListing, error)
	Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req *model.UpdateProductListingRequest) error
}

// IntegrationActionRepo is the subset of repository.IntegrationRepo needed here.
type IntegrationActionRepo interface {
	FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.IntegrationWithCreds, error)
}

// ListingActivatorProvider is the interface a marketplace provider must implement
// to support offer activation/relisting.
type ListingActivatorProvider interface {
	ActivateOffer(ctx context.Context, externalOfferID string) error
}

// ListingDeactivatorProvider is the interface a marketplace provider must implement
// to support offer deactivation.
type ListingDeactivatorProvider interface {
	DeactivateOffer(ctx context.Context, externalOfferID string) error
}

// ListingDeactivatorDeps holds the dependencies needed by the deactivate_listing action.
type ListingDeactivatorDeps struct {
	ListingRepo     ListingActionRepo
	IntegrationRepo IntegrationActionRepo
	EncryptionKey   []byte
	ProviderFactory func(provider string, credentials json.RawMessage, settings json.RawMessage) (ListingDeactivatorProvider, error)
}

// MarketplaceMessageDeps holds the dependencies needed by the send_marketplace_message action.
type MarketplaceMessageDeps struct {
	TemplateRepo interface {
		FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.MessageTemplate, error)
	}
	OrderRepo interface {
		FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Order, error)
	}
	Pool           *pgxpool.Pool
	IntegrationSvc interface {
		GetDecryptedCredentialsByProvider(ctx context.Context, tenantID uuid.UUID, provider string) ([]byte, *model.Integration, error)
	}
}

// DefaultActionExecutor is a basic action executor that handles webhook actions
// and executes set_status, send_email, add_tag, create_invoice, activate_listing,
// and send_marketplace_message actions via injected service dependencies.
type DefaultActionExecutor struct {
	logger     *slog.Logger
	httpClient *http.Client

	// Service dependencies (set via setters to avoid changing constructor for DelayedActionWorker).
	orderTransitioner      OrderStatusTransitioner
	orderGetter            OrderGetter
	orderRepo              OrderTagUpdater
	emailSender            EmailSender
	invoiceCreator         InvoiceCreator
	listingDeps            *ListingActivatorDeps
	listingDeactivatorDeps *ListingDeactivatorDeps
	marketplaceMessageDeps *MarketplaceMessageDeps
	pool                   *pgxpool.Pool

	// Orchestration routing (OPE-421). When orchestrationEnabled is true AND both
	// repositories are wired, executeSetStatus enqueues an automation.set_status
	// outbox event (anchored to the order's fulfillment process) instead of calling
	// TransitionStatus directly. The zero value is DISABLED: orchestrationEnabled is
	// false and the repos are nil, so the direct path runs unchanged.
	orchestration        OrchestrationEnqueuer
	fulfillment          FulfillmentProcessEnsurer
	orchestrationEnabled bool

	// externalWorkflow is the OPE-421/Phase-13 connector (gated). The zero value is nil,
	// and executeExternalWorkflow is a no-op unless it is wired AND its Enabled() is true.
	externalWorkflow ExternalWorkflowDispatcher
}

// NewDefaultActionExecutor creates a new DefaultActionExecutor with a safe HTTP client.
func NewDefaultActionExecutor(logger *slog.Logger) *DefaultActionExecutor {
	return &DefaultActionExecutor{
		logger:     logger,
		httpClient: netutil.SafeHTTPClient(10 * time.Second),
	}
}

// SetOrderServices wires the order service dependencies needed by set_status and add_tag actions.
func (e *DefaultActionExecutor) SetOrderServices(transitioner OrderStatusTransitioner, getter OrderGetter, repo OrderTagUpdater, pool *pgxpool.Pool) {
	e.orderTransitioner = transitioner
	e.orderGetter = getter
	e.orderRepo = repo
	e.pool = pool
}

// SetEmailSender wires the email service dependency needed by the send_email action.
func (e *DefaultActionExecutor) SetEmailSender(sender EmailSender) {
	e.emailSender = sender
}

// SetInvoiceCreator wires the invoice service dependency needed by the create_invoice action.
func (e *DefaultActionExecutor) SetInvoiceCreator(creator InvoiceCreator) {
	e.invoiceCreator = creator
}

// SetListingActivatorDeps wires the dependencies needed by the activate_listing action.
func (e *DefaultActionExecutor) SetListingActivatorDeps(deps *ListingActivatorDeps) {
	e.listingDeps = deps
}

// SetListingDeactivatorDeps wires the dependencies needed by the deactivate_listing action.
func (e *DefaultActionExecutor) SetListingDeactivatorDeps(deps *ListingDeactivatorDeps) {
	e.listingDeactivatorDeps = deps
}

// SetMarketplaceMessageDeps wires the dependencies needed by the send_marketplace_message action.
func (e *DefaultActionExecutor) SetMarketplaceMessageDeps(deps *MarketplaceMessageDeps) {
	e.marketplaceMessageDeps = deps
}

// SetOrchestration enables routing set_status actions through the orchestration
// outbox (OPE-421). When enabled is true and both repositories are non-nil,
// executeSetStatus enqueues an automation.set_status event (ensuring the order's
// fulfillment process first) rather than calling TransitionStatus directly. Passing
// enabled=false (or leaving this unset) keeps the byte-for-byte unchanged direct
// path — the zero-value executor is always in the off state. The pool used for the
// enqueue transaction is the one already wired via SetOrderServices.
func (e *DefaultActionExecutor) SetOrchestration(enabled bool, orchestration OrchestrationEnqueuer, fulfillment FulfillmentProcessEnsurer) {
	e.orchestrationEnabled = enabled
	e.orchestration = orchestration
	e.fulfillment = fulfillment
}

// SetExternalWorkflow wires the OPE-421/Phase-13 external-workflow connector. When unset (or
// when its Enabled() is false) the external_workflow action is a no-op — the default path is
// byte-for-byte unchanged. Mirrors the SetOrchestration gating pattern.
func (e *DefaultActionExecutor) SetExternalWorkflow(dispatcher ExternalWorkflowDispatcher) {
	e.externalWorkflow = dispatcher
}

// ExecuteAction dispatches an automation action based on its type.
func (e *DefaultActionExecutor) ExecuteAction(ctx context.Context, tenantID uuid.UUID, action Action, event Event) error {
	switch action.Type {
	case "webhook":
		return e.executeWebhook(ctx, tenantID, action, event)
	case "set_status":
		return e.executeSetStatus(ctx, tenantID, action, event)
	case "add_tag":
		return e.executeAddTag(ctx, tenantID, action, event)
	case "send_email":
		return e.executeSendEmail(ctx, tenantID, action, event)
	case "create_invoice":
		return e.executeCreateInvoice(ctx, tenantID, action, event)
	case "activate_listing":
		return e.executeActivateListing(ctx, tenantID, action, event)
	case "deactivate_listing":
		return e.executeDeactivateListing(ctx, tenantID, action, event)
	case "send_marketplace_message":
		return e.executeSendMarketplaceMessage(ctx, tenantID, action, event)
	case "external_workflow":
		return e.executeExternalWorkflow(ctx, tenantID, action, event)
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

// executeExternalWorkflow hands the event to an external workflow engine via the
// external-workflow connector (OPE-421/Phase-13). Gated/best-effort: a no-op when the
// connector is nil/disabled, so the default build is byte-for-byte unchanged.
func (e *DefaultActionExecutor) executeExternalWorkflow(_ context.Context, tenantID uuid.UUID, action Action, event Event) error {
	if e.externalWorkflow == nil || !e.externalWorkflow.Enabled() {
		return nil
	}
	if event.EntityType != "order" {
		return fmt.Errorf("external_workflow action only supports order entities, got %q", event.EntityType)
	}
	integrationIDStr, _ := action.Params["integration_id"].(string)
	integrationID, err := uuid.Parse(integrationIDStr)
	if err != nil {
		return fmt.Errorf("external_workflow: invalid integration_id: %w", err)
	}
	// Use a fresh context: the dispatch enqueue must complete regardless of the original
	// request timeout/cancellation (mirrors the other external-effect actions).
	return e.externalWorkflow.DispatchForOrder(context.Background(), tenantID, event.EntityID, integrationID)
}

// executeSetStatus transitions an order to the target status specified in action params.
func (e *DefaultActionExecutor) executeSetStatus(_ context.Context, tenantID uuid.UUID, action Action, event Event) error {
	e.logger.Info("automation action: set_status",
		"tenant_id", tenantID,
		"entity_type", event.EntityType,
		"entity_id", event.EntityID,
	)

	if e.orderTransitioner == nil {
		e.logger.Warn("automation action set_status: order service not wired, skipping")
		return nil
	}

	if event.EntityType != "order" {
		return fmt.Errorf("set_status action only supports order entities, got %q", event.EntityType)
	}

	targetStatus, _ := action.Params["status"].(string)
	if targetStatus == "" {
		return fmt.Errorf("set_status action missing 'status' parameter")
	}

	// Automation is system-driven but not omniscient: a rule that targets a status the
	// tenant's graph cannot reach from the order's current one is a misconfiguration,
	// and applying it anyway skips the states whose side effects carry the stock and
	// money (e.g. new -> completed never passes through shipped). Validate by default;
	// a rule that genuinely needs to jump opts in with "force".
	force := paramBool(action.Params["force"])

	// Use a fresh context in case the original is cancelled.
	bgCtx := context.Background()

	// OPE-421: when orchestration routing is enabled, the transition is performed
	// asynchronously and idempotently by AutomationStatusTransitionHandler. We
	// ensure the order's fulfillment process exists (the outbox FK requires a
	// process), enqueue an automation.set_status event, and return WITHOUT calling
	// TransitionStatus here. The enqueue is deduped on its idempotency key so a
	// re-fire of the same (rule, order, target) is a no-op. When routing is disabled
	// (the default) this whole block is skipped and the direct path below runs
	// byte-for-byte unchanged.
	if e.orchestrationEnabled && e.orchestration != nil && e.fulfillment != nil && e.pool != nil {
		return e.enqueueSetStatus(bgCtx, tenantID, action, event, targetStatus, force)
	}

	req := model.StatusTransitionRequest{
		Status: targetStatus,
		Force:  force,
	}

	_, err := e.orderTransitioner.TransitionStatus(bgCtx, tenantID, event.EntityID, req, uuid.Nil, "automation")
	if err != nil {
		e.logger.Error("automation action set_status failed",
			"tenant_id", tenantID,
			"order_id", event.EntityID,
			"target_status", targetStatus,
			"error", err,
		)
		return fmt.Errorf("set_status: %w", err)
	}

	e.logger.Info("automation action set_status completed",
		"tenant_id", tenantID,
		"order_id", event.EntityID,
		"target_status", targetStatus,
	)
	return nil
}

// setStatusIdempotencyKey builds the stable outbox dedup key for an orchestrated
// set_status action. Keying on (rule, order, target) means a repeated fire of the
// same rule on the same order toward the same status enqueues at most one event.
// The action index is intentionally excluded so two set_status actions in one rule
// targeting the same status still collapse to one transition.
func setStatusIdempotencyKey(ruleID, orderID uuid.UUID, targetStatus string) string {
	return EventAutomationSetStatus + ":" + ruleID.String() + ":" + orderID.String() + ":" + targetStatus
}

// setStatusPayload builds the outbox payload for an orchestration-routed set_status.
// "force" travels with the event so the async handler applies the same graph policy
// the rule asked for; it is deliberately NOT part of the idempotency key, so flipping
// the flag on a rule does not mint a second transition for the same target status.
func setStatusPayload(orderID uuid.UUID, targetStatus string, ruleID uuid.UUID, actionIndex int, force bool) map[string]any {
	return map[string]any{
		"order_id":      orderID.String(),
		"target_status": targetStatus,
		"rule_id":       ruleID.String(),
		"action_index":  actionIndex,
		"force":         force,
	}
}

// paramBool interprets an action parameter as a boolean opt-in. Action params arrive
// as free-form JSON (jsonb, or the dashboard's form state), so a flag can show up as
// a real boolean or as its string form; anything else counts as not set.
func paramBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return strings.EqualFold(strings.TrimSpace(t), "true")
	default:
		return false
	}
}

// enqueueSetStatus is the OPE-421 orchestration-routed path for set_status. It runs
// a single tenant transaction that (1) ensures the order's fulfillment process
// exists — the outbox process_id FK is NOT NULL — and (2) enqueues an
// automation.set_status event idempotently. It performs NO direct transition; the
// OrchestrationWorker + AutomationStatusTransitionHandler apply it asynchronously
// and idempotently. On error it returns to the engine (which records it in
// automation_rule_logs); it never panics and never silently drops the action.
func (e *DefaultActionExecutor) enqueueSetStatus(ctx context.Context, tenantID uuid.UUID, action Action, event Event, targetStatus string, force bool) error {
	payload := setStatusPayload(event.EntityID, targetStatus, action.RuleID, action.ActionIndex, force)
	idemKey := setStatusIdempotencyKey(action.RuleID, event.EntityID, targetStatus)

	err := database.WithTenant(ctx, e.pool, tenantID, func(tx pgx.Tx) error {
		return e.ensureProcessAndEnqueueSetStatus(ctx, tx, tenantID, event.EntityID, idemKey, payload)
	})
	if err != nil {
		e.logger.Error("automation action set_status: enqueue failed",
			"tenant_id", tenantID,
			"order_id", event.EntityID,
			"target_status", targetStatus,
			"error", err,
		)
		return fmt.Errorf("set_status enqueue: %w", err)
	}

	e.logger.Info("automation action set_status enqueued via orchestration outbox",
		"tenant_id", tenantID,
		"order_id", event.EntityID,
		"target_status", targetStatus,
		"idempotency_key", idemKey,
	)
	return nil
}

// ensureProcessAndEnqueueSetStatus runs the tenant-tx-bound work for the
// orchestration-routed set_status: it ensures a fulfillment process exists for the
// order (idempotent — reuse on hit, create only on pgx.ErrNoRows) and enqueues the
// automation.set_status outbox event (idempotent on idemKey). Split out of
// enqueueSetStatus so the DB-bound logic is unit-testable with fakes; the tx is
// passed straight through to the repositories.
func (e *DefaultActionExecutor) ensureProcessAndEnqueueSetStatus(ctx context.Context, tx pgx.Tx, tenantID, orderID uuid.UUID, idemKey string, payload map[string]any) error {
	proc, perr := e.fulfillment.GetProcessByOrder(ctx, tx, orderID)
	if perr != nil {
		if !errors.Is(perr, pgx.ErrNoRows) {
			return fmt.Errorf("lookup fulfillment process: %w", perr)
		}
		proc, perr = e.fulfillment.CreateProcess(ctx, tx, model.FulfillmentProcess{
			TenantID: tenantID,
			OrderID:  orderID,
		})
		if perr != nil {
			return fmt.Errorf("create fulfillment process: %w", perr)
		}
	}

	if _, _, eerr := e.orchestration.EnqueueEvent(ctx, tx, model.OrchestrationOutboxEvent{
		TenantID:       tenantID,
		ProcessID:      proc.ID,
		EventType:      EventAutomationSetStatus,
		IdempotencyKey: idemKey,
		Payload:        payload,
	}); eerr != nil {
		return fmt.Errorf("enqueue automation.set_status event: %w", eerr)
	}
	return nil
}

// executeAddTag appends a tag to an order's tags array, skipping duplicates.
func (e *DefaultActionExecutor) executeAddTag(_ context.Context, tenantID uuid.UUID, action Action, event Event) error {
	e.logger.Info("automation action: add_tag",
		"tenant_id", tenantID,
		"entity_type", event.EntityType,
		"entity_id", event.EntityID,
	)

	if e.orderRepo == nil || e.pool == nil {
		e.logger.Warn("automation action add_tag: order repo not wired, skipping")
		return nil
	}

	if event.EntityType != "order" {
		return fmt.Errorf("add_tag action only supports order entities, got %q", event.EntityType)
	}

	tag, _ := action.Params["tag"].(string)
	if tag == "" {
		return fmt.Errorf("add_tag action missing 'tag' parameter")
	}

	bgCtx := context.Background()

	err := database.WithTenant(bgCtx, e.pool, tenantID, func(tx pgx.Tx) error {
		order, err := e.orderRepo.FindByID(bgCtx, tx, event.EntityID)
		if err != nil {
			return fmt.Errorf("find order: %w", err)
		}
		if order == nil {
			return fmt.Errorf("order %s not found", event.EntityID)
		}

		// Check for duplicate tag
		if slices.Contains(order.Tags, tag) {
			e.logger.Info("automation action add_tag: tag already exists, skipping",
				"order_id", event.EntityID,
				"tag", tag,
			)
			return nil
		}

		newTags := append(order.Tags, tag) //nolint:gocritic // intentionally creating new slice for update
		updateReq := model.UpdateOrderRequest{
			Tags: &newTags,
		}

		return e.orderRepo.Update(bgCtx, tx, event.EntityID, updateReq)
	})

	if err != nil {
		e.logger.Error("automation action add_tag failed",
			"tenant_id", tenantID,
			"order_id", event.EntityID,
			"tag", tag,
			"error", err,
		)
		return fmt.Errorf("add_tag: %w", err)
	}

	e.logger.Info("automation action add_tag completed",
		"tenant_id", tenantID,
		"order_id", event.EntityID,
		"tag", tag,
	)
	return nil
}

// executeSendEmail sends a status notification email for the order.
func (e *DefaultActionExecutor) executeSendEmail(_ context.Context, tenantID uuid.UUID, action Action, event Event) error {
	e.logger.Info("automation action: send_email",
		"tenant_id", tenantID,
		"entity_type", event.EntityType,
		"entity_id", event.EntityID,
	)

	if e.emailSender == nil || e.orderGetter == nil {
		e.logger.Warn("automation action send_email: email service or order service not wired, skipping")
		return nil
	}

	if event.EntityType != "order" {
		return fmt.Errorf("send_email action only supports order entities, got %q", event.EntityType)
	}

	bgCtx := context.Background()

	// Retrieve the full order to have customer email, amounts, etc.
	order, err := e.orderGetter.Get(bgCtx, tenantID, event.EntityID)
	if err != nil {
		e.logger.Error("automation action send_email: failed to get order",
			"order_id", event.EntityID,
			"error", err,
		)
		return fmt.Errorf("send_email: get order: %w", err)
	}

	// Determine old/new status from event data or action params.
	oldStatus, _ := event.Data["old_status"].(string)
	newStatus, _ := action.Params["status"].(string)
	if newStatus == "" {
		// Fallback: try new_status from event data, then current order status.
		newStatus, _ = event.Data["new_status"].(string)
		if newStatus == "" {
			newStatus = order.Status
		}
	}

	// SendOrderStatusEmail is a fire-and-forget method that handles its own errors.
	e.emailSender.SendOrderStatusEmail(bgCtx, tenantID, order, oldStatus, newStatus)

	e.logger.Info("automation action send_email completed",
		"tenant_id", tenantID,
		"order_id", event.EntityID,
		"new_status", newStatus,
	)
	return nil
}

// executeCreateInvoice triggers invoice creation for an order.
func (e *DefaultActionExecutor) executeCreateInvoice(_ context.Context, tenantID uuid.UUID, _ Action, event Event) error {
	e.logger.Info("automation action: create_invoice",
		"tenant_id", tenantID,
		"entity_type", event.EntityType,
		"entity_id", event.EntityID,
	)

	if e.invoiceCreator == nil || e.orderGetter == nil {
		e.logger.Warn("automation action create_invoice: invoice service or order service not wired, skipping")
		return nil
	}

	if event.EntityType != "order" {
		return fmt.Errorf("create_invoice action only supports order entities, got %q", event.EntityType)
	}

	bgCtx := context.Background()

	order, err := e.orderGetter.Get(bgCtx, tenantID, event.EntityID)
	if err != nil {
		e.logger.Error("automation action create_invoice: failed to get order",
			"order_id", event.EntityID,
			"error", err,
		)
		return fmt.Errorf("create_invoice: get order: %w", err)
	}

	// HandleOrderStatusChange checks invoicing settings, auto-create conditions,
	// and whether an invoice already exists. It handles all errors internally.
	e.invoiceCreator.HandleOrderStatusChange(bgCtx, tenantID, order)

	e.logger.Info("automation action create_invoice completed",
		"tenant_id", tenantID,
		"order_id", event.EntityID,
	)
	return nil
}

// executeActivateListing reactivates marketplace listings for a product.
// Triggered by product.stock_restored when stock goes from 0 → >0.
//
// NOTE: Similar logic exists in StockSyncService.reactivateListings (service package).
// That method serves as a direct fallback when the automation engine is not wired,
// while this one is the automation-rule-driven path. They are kept separate to avoid
// a circular dependency and because they differ in provider creation and interface widths.
func (e *DefaultActionExecutor) executeActivateListing(_ context.Context, tenantID uuid.UUID, _ Action, event Event) error {
	e.logger.Info("automation action: activate_listing",
		"tenant_id", tenantID,
		"entity_type", event.EntityType,
		"entity_id", event.EntityID,
	)

	if event.EntityType != "product" {
		return fmt.Errorf("activate_listing action only supports product entities, got %q", event.EntityType)
	}

	if e.listingDeps == nil || e.pool == nil {
		e.logger.Warn("automation action activate_listing: listing dependencies not wired, skipping")
		return nil
	}

	return e.executeListingToggle(tenantID, event, "activate_listing",
		[]string{"inactive", "ended"}, "active",
		e.listingDeps.ListingRepo, e.listingDeps.IntegrationRepo, e.listingDeps.EncryptionKey,
		func(credJSON json.RawMessage, settings json.RawMessage, providerName, externalID string) error {
			provider, err := e.listingDeps.ProviderFactory(providerName, credJSON, settings)
			if err != nil {
				return err
			}
			return provider.ActivateOffer(context.Background(), externalID)
		},
	)
}

// executeDeactivateListing deactivates active marketplace listings for a product.
// Triggered by product.out_of_stock when stock goes from >0 → 0.
func (e *DefaultActionExecutor) executeDeactivateListing(_ context.Context, tenantID uuid.UUID, _ Action, event Event) error {
	e.logger.Info("automation action: deactivate_listing",
		"tenant_id", tenantID,
		"entity_type", event.EntityType,
		"entity_id", event.EntityID,
	)

	if event.EntityType != "product" {
		return fmt.Errorf("deactivate_listing action only supports product entities, got %q", event.EntityType)
	}

	if e.listingDeactivatorDeps == nil || e.pool == nil {
		e.logger.Warn("automation action deactivate_listing: listing dependencies not wired, skipping")
		return nil
	}

	return e.executeListingToggle(tenantID, event, "deactivate_listing",
		[]string{"active"}, "inactive",
		e.listingDeactivatorDeps.ListingRepo, e.listingDeactivatorDeps.IntegrationRepo, e.listingDeactivatorDeps.EncryptionKey,
		func(credJSON json.RawMessage, settings json.RawMessage, providerName, externalID string) error {
			provider, err := e.listingDeactivatorDeps.ProviderFactory(providerName, credJSON, settings)
			if err != nil {
				return err
			}
			return provider.DeactivateOffer(context.Background(), externalID)
		},
	)
}

// executeListingToggle is the shared implementation for activate_listing and deactivate_listing.
// It gathers eligible listings in a DB transaction, then calls the provider action outside the tx.
func (e *DefaultActionExecutor) executeListingToggle(
	tenantID uuid.UUID,
	event Event,
	actionName string,
	sourceStatuses []string,
	targetStatus string,
	repo ListingActionRepo,
	integRepo IntegrationActionRepo,
	encryptionKey []byte,
	providerAction func(credJSON json.RawMessage, settings json.RawMessage, providerName, externalID string) error,
) error {
	productID := event.EntityID
	bgCtx := context.Background()

	type toggleJob struct {
		listingID   uuid.UUID
		externalID  string
		integration *model.IntegrationWithCreds
	}
	var jobs []toggleJob

	err := database.WithTenant(bgCtx, e.pool, tenantID, func(tx pgx.Tx) error {
		listings, err := repo.ListByProduct(bgCtx, tx, productID)
		if err != nil {
			return fmt.Errorf("list listings: %w", err)
		}

		for _, listing := range listings {
			if !slices.Contains(sourceStatuses, listing.Status) {
				continue
			}
			if listing.ExternalID == nil || *listing.ExternalID == "" {
				continue
			}

			integ, err := integRepo.FindByID(bgCtx, tx, listing.IntegrationID)
			if err != nil || integ == nil {
				e.logger.Warn(actionName+": integration not found",
					"listing_id", listing.ID, "integration_id", listing.IntegrationID)
				continue
			}

			jobs = append(jobs, toggleJob{
				listingID:   listing.ID,
				externalID:  *listing.ExternalID,
				integration: integ,
			})
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("%s: gather data: %w", actionName, err)
	}

	if len(jobs) == 0 {
		e.logger.Info("automation action "+actionName+": no eligible listings",
			"product_id", productID)
		return nil
	}

	succeeded, failed := 0, 0
	for _, job := range jobs {
		if job.integration.EncryptedCredentials == "" {
			e.logger.Warn(actionName+": no credentials for integration",
				"integration_id", job.integration.ID)
			failed++
			continue
		}

		credJSON, err := crypto.Decrypt(job.integration.EncryptedCredentials, encryptionKey)
		if err != nil {
			e.logger.Error(actionName+": decrypt credentials failed",
				"integration_id", job.integration.ID, "error", err)
			failed++
			continue
		}

		if err := providerAction(credJSON, job.integration.Settings, job.integration.Provider, job.externalID); err != nil {
			e.logger.Error(actionName+": provider action failed",
				"listing_id", job.listingID, "external_id", job.externalID,
				"provider", job.integration.Provider, "error", err)
			failed++
			continue
		}

		statusStr := targetStatus
		syncOK := "synced"
		if err := database.WithTenant(bgCtx, e.pool, tenantID, func(tx pgx.Tx) error {
			return repo.Update(bgCtx, tx, job.listingID, &model.UpdateProductListingRequest{
				Status:     &statusStr,
				SyncStatus: &syncOK,
			})
		}); err != nil {
			e.logger.Warn(actionName+": failed to update listing status in DB",
				"listing_id", job.listingID, "error", err)
		}
		succeeded++
	}

	e.logger.Info("automation action "+actionName+" completed",
		"tenant_id", tenantID,
		"product_id", productID,
		"succeeded", succeeded,
		"failed", failed,
	)
	if succeeded == 0 && failed > 0 {
		return fmt.Errorf("%s: all %d listing(s) failed", actionName, failed)
	}
	return nil
}

// executeSendMarketplaceMessage sends a message to a buyer via Allegro messaging.
// It loads a message template, substitutes variables from the order context,
// finds the Allegro thread for the order, and sends the message.
//
// The caller context is intentionally discarded: Allegro messaging API calls are
// fire-and-forget and must complete even if the original request context is cancelled.
func (e *DefaultActionExecutor) executeSendMarketplaceMessage(_ context.Context, tenantID uuid.UUID, action Action, event Event) error {
	e.logger.Info("automation action: send_marketplace_message",
		"tenant_id", tenantID,
		"entity_type", event.EntityType,
		"entity_id", event.EntityID,
	)

	if e.marketplaceMessageDeps == nil || e.marketplaceMessageDeps.Pool == nil {
		e.logger.Warn("automation action send_marketplace_message: dependencies not wired, skipping")
		return nil
	}

	if event.EntityType != "order" {
		return fmt.Errorf("send_marketplace_message action only supports order entities, got %q", event.EntityType)
	}

	// Parse template_id from action params.
	templateIDStr, _ := action.Params["template_id"].(string)
	if templateIDStr == "" {
		return fmt.Errorf("send_marketplace_message action missing 'template_id' parameter")
	}
	templateID, err := uuid.Parse(templateIDStr)
	if err != nil {
		return fmt.Errorf("send_marketplace_message: invalid template_id %q: %w", templateIDStr, err)
	}

	bgCtx := context.Background()
	deps := e.marketplaceMessageDeps

	// Phase 1: Load template and order from DB.
	var tmpl *model.MessageTemplate
	var order *model.Order

	err = database.WithTenant(bgCtx, deps.Pool, tenantID, func(tx pgx.Tx) error {
		var err error
		tmpl, err = deps.TemplateRepo.FindByID(bgCtx, tx, templateID)
		if err != nil {
			return fmt.Errorf("find template: %w", err)
		}
		if tmpl == nil {
			return fmt.Errorf("template %s not found", templateID)
		}

		order, err = deps.OrderRepo.FindByID(bgCtx, tx, event.EntityID)
		if err != nil {
			return fmt.Errorf("find order: %w", err)
		}
		if order == nil {
			return fmt.Errorf("order %s not found", event.EntityID)
		}
		return nil
	})
	if err != nil {
		e.logger.Error("automation action send_marketplace_message: failed to load data",
			"tenant_id", tenantID,
			"error", err,
		)
		return fmt.Errorf("send_marketplace_message: %w", err)
	}

	// Only Allegro orders are supported (source must be "allegro" and external_id must be set).
	if order.Source != "allegro" {
		e.logger.Warn("automation action send_marketplace_message: order is not from Allegro, skipping",
			"order_id", order.ID,
			"source", order.Source,
		)
		return nil
	}
	if order.ExternalID == nil || *order.ExternalID == "" {
		e.logger.Warn("automation action send_marketplace_message: order has no external_id, skipping",
			"order_id", order.ID,
		)
		return nil
	}

	// Phase 2: Build variable data and substitute into the template body.
	// Intentionally excluded: customer_email (PII, unnecessary in outbound marketplace messages).
	// customer_name is retained: addressing the buyer by name is standard marketplace practice.
	vars := map[string]string{
		"order_id":      order.ID.String(),
		"external_id":   *order.ExternalID,
		"customer_name": order.CustomerName,
		"order_total":   fmt.Sprintf("%.2f %s", order.TotalAmount, order.Currency),
		"status":        order.Status,
	}
	// Include tracking number from event data if available.
	if tn, ok := event.Data["tracking_number"].(string); ok && tn != "" {
		vars["tracking_number"] = tn
	}
	// Include new_status from event data.
	if ns, ok := event.Data["new_status"].(string); ok && ns != "" {
		vars["new_status"] = ns
	}
	if os, ok := event.Data["old_status"].(string); ok && os != "" {
		vars["old_status"] = os
	}

	// Only substitute variables declared in the template's Variables list.
	// This prevents a template author from embedding undeclared {{keys}} to
	// exfiltrate data that was not intentionally exposed (e.g., {{customer_email}}
	// on a template that declared only ["order_id", "status"]).
	allowedVars := make(map[string]string, len(tmpl.Variables))
	for _, v := range tmpl.Variables {
		if val, ok := vars[v]; ok {
			allowedVars[v] = val
		}
	}
	messageBody := substituteVariables(tmpl.Body, allowedVars)
	if messageBody == "" {
		return fmt.Errorf("send_marketplace_message: template body is empty after substitution")
	}

	// Phase 3: Build Allegro provider from integration credentials.
	// Uses the proper provider setup (allegroIntegration.NewProvider) which handles
	// credential parsing and token configuration, instead of a bare SDK client.
	credJSON, _, err := deps.IntegrationSvc.GetDecryptedCredentialsByProvider(bgCtx, tenantID, "allegro")
	if err != nil {
		e.logger.Warn("automation action send_marketplace_message: no Allegro integration, skipping",
			"tenant_id", tenantID,
			"error", err,
		)
		return nil
	}

	provider, err := allegroIntegration.NewProvider(credJSON, nil)
	if err != nil {
		return fmt.Errorf("send_marketplace_message: create provider: %w", err)
	}
	defer provider.Close()

	client := provider.SDKClient()

	// Phase 4: Find the Allegro messaging thread for this order.
	// We list threads and look for one whose subject or interlocutor matches.
	// Allegro creates a thread per checkout-form, and the external_id is the checkout-form ID.
	threadID, err := e.findAllegroThreadForOrder(bgCtx, client, *order.ExternalID)
	if err != nil {
		e.logger.Warn("automation action send_marketplace_message: could not find thread, skipping",
			"tenant_id", tenantID,
			"order_id", order.ID,
			"external_id", *order.ExternalID,
			"error", err,
		)
		return nil
	}

	// Phase 5: Send the message.
	_, err = client.Messages.SendMessage(bgCtx, threadID, allegrosdk.SendMessageRequest{
		Text: messageBody,
	})
	if err != nil {
		e.logger.Error("automation action send_marketplace_message: failed to send",
			"tenant_id", tenantID,
			"order_id", order.ID,
			"thread_id", threadID,
			"error", err,
		)
		return fmt.Errorf("send_marketplace_message: send: %w", err)
	}

	e.logger.Info("automation action send_marketplace_message completed",
		"tenant_id", tenantID,
		"order_id", order.ID,
		"thread_id", threadID,
		"template_id", templateID,
	)
	return nil
}

// findAllegroThreadForOrder searches Allegro messaging threads to find one
// related to the given checkout-form (order) external ID.
// Allegro thread IDs are derived from the checkout-form ID, so we search
// the first pages of threads looking for a match in the thread ID or subject.
func (e *DefaultActionExecutor) findAllegroThreadForOrder(ctx context.Context, client *allegrosdk.Client, checkoutFormID string) (string, error) {
	// Allegro typically uses the checkout-form ID as the thread ID for order-related threads.
	// Try a direct GET first — if the thread exists with that ID, we're done.
	thread, err := client.Messages.GetThread(ctx, checkoutFormID)
	if err == nil && thread != nil {
		return thread.ID, nil
	}

	// Fallback: paginate through threads looking for a match.
	// Search up to 200 threads (2 pages of 100).
	for offset := 0; offset < 200; offset += 100 {
		threads, err := client.Messages.ListThreads(ctx, &allegrosdk.ListThreadsParams{
			Limit:  100,
			Offset: offset,
		})
		if err != nil {
			return "", fmt.Errorf("list threads (offset %d): %w", offset, err)
		}
		for _, t := range threads.Threads {
			if t.ID == checkoutFormID {
				return t.ID, nil
			}
		}
		if len(threads.Threads) < 100 {
			break // No more threads to fetch.
		}
	}

	return "", fmt.Errorf("no thread found for checkout-form %s", checkoutFormID)
}

// substituteVariables replaces {{key}} placeholders in a template string with values from the data map.
func substituteVariables(tmpl string, data map[string]string) string {
	result := tmpl
	for key, value := range data {
		result = strings.ReplaceAll(result, "{{"+key+"}}", value)
	}
	return result
}

func (e *DefaultActionExecutor) executeWebhook(ctx context.Context, tenantID uuid.UUID, action Action, event Event) error {
	url, _ := action.Params["url"].(string)
	if url == "" {
		return fmt.Errorf("webhook action missing url parameter")
	}

	payload := map[string]any{
		"event":       event.Type,
		"tenant_id":   tenantID.String(),
		"entity_type": event.EntityType,
		"entity_id":   event.EntityID.String(),
		"data":        event.Data,
		"fired_at":    time.Now().UTC().Format(time.RFC3339),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "OpenOMS-Automation/1.0")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}
