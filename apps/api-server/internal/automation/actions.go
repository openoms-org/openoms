package automation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/netutil"
)

// ActionExecutor executes a single automation action.
type ActionExecutor interface {
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

// InvoiceCreator handles auto-invoice creation on order status change.
// Implemented by service.InvoiceService.
type InvoiceCreator interface {
	HandleOrderStatusChange(ctx context.Context, tenantID uuid.UUID, order *model.Order)
}

// DefaultActionExecutor is a basic action executor that handles webhook actions
// and executes set_status, send_email, add_tag, and create_invoice actions
// via injected service dependencies.
type DefaultActionExecutor struct {
	logger     *slog.Logger
	httpClient *http.Client

	// Service dependencies (set via setters to avoid changing constructor for DelayedActionWorker).
	orderTransitioner OrderStatusTransitioner
	orderGetter       OrderGetter
	orderRepo         OrderTagUpdater
	emailSender       EmailSender
	invoiceCreator    InvoiceCreator
	pool              *pgxpool.Pool
}

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
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

// executeSetStatus transitions an order to the target status specified in action params.
func (e *DefaultActionExecutor) executeSetStatus(_ context.Context, tenantID uuid.UUID, action Action, event Event) error {
	e.logger.Info("automation action: set_status",
		"tenant_id", tenantID,
		"entity_type", event.EntityType,
		"entity_id", event.EntityID,
		"params", action.Params,
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

	// Use Force=true because automation rules are system-driven and should
	// bypass normal transition validation rules.
	req := model.StatusTransitionRequest{
		Status: targetStatus,
		Force:  true,
	}

	// Use a fresh context in case the original is cancelled.
	bgCtx := context.Background()

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

// executeAddTag appends a tag to an order's tags array, skipping duplicates.
func (e *DefaultActionExecutor) executeAddTag(_ context.Context, tenantID uuid.UUID, action Action, event Event) error {
	e.logger.Info("automation action: add_tag",
		"tenant_id", tenantID,
		"entity_type", event.EntityType,
		"entity_id", event.EntityID,
		"params", action.Params,
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
		"params", action.Params,
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
func (e *DefaultActionExecutor) executeCreateInvoice(ctx context.Context, tenantID uuid.UUID, action Action, event Event) error {
	e.logger.Info("automation action: create_invoice",
		"tenant_id", tenantID,
		"entity_type", event.EntityType,
		"entity_id", event.EntityID,
		"params", action.Params,
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
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}
