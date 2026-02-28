package automation

import (
	"bytes"
	"context"
	"encoding/json"
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
	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"
)

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

// InvoiceCreator handles auto-invoice creation on order status change.
// Implemented by service.InvoiceService.
type InvoiceCreator interface {
	HandleOrderStatusChange(ctx context.Context, tenantID uuid.UUID, order *model.Order)
}

// ListingActivatorDeps holds the dependencies needed by the activate_listing action.
type ListingActivatorDeps struct {
	ListingRepo     ListingRepoForActivation
	IntegrationRepo IntegrationRepoForActivation
	EncryptionKey   []byte
	// ProviderFactory creates a marketplace provider from decrypted credentials.
	// Returns (provider, needsClose, error). The caller must close the provider if needsClose is true.
	ProviderFactory func(provider string, credentials json.RawMessage, settings json.RawMessage) (ListingActivatorProvider, error)
}

// ListingRepoForActivation is the subset of repository.ProductListingRepo needed here.
type ListingRepoForActivation interface {
	ListByProduct(ctx context.Context, tx pgx.Tx, productID uuid.UUID) ([]*model.ProductListing, error)
	Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req *model.UpdateProductListingRequest) error
}

// IntegrationRepoForActivation is the subset of repository.IntegrationRepo needed here.
type IntegrationRepoForActivation interface {
	FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.IntegrationWithCreds, error)
}

// ListingActivatorProvider is the interface a marketplace provider must implement
// to support offer activation/relisting.
type ListingActivatorProvider interface {
	ActivateOffer(ctx context.Context, externalOfferID string) error
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
	marketplaceMessageDeps *MarketplaceMessageDeps
	pool                   *pgxpool.Pool
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

// SetListingActivatorDeps wires the dependencies needed by the activate_listing action.
func (e *DefaultActionExecutor) SetListingActivatorDeps(deps *ListingActivatorDeps) {
	e.listingDeps = deps
}

// SetMarketplaceMessageDeps wires the dependencies needed by the send_marketplace_message action.
func (e *DefaultActionExecutor) SetMarketplaceMessageDeps(deps *MarketplaceMessageDeps) {
	e.marketplaceMessageDeps = deps
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
	case "activate_listing":
		return e.executeActivateListing(ctx, tenantID, action, event)
	case "send_marketplace_message":
		return e.executeSendMarketplaceMessage(ctx, tenantID, action, event)
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
func (e *DefaultActionExecutor) executeCreateInvoice(ctx context.Context, tenantID uuid.UUID, action Action, event Event) error {
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
// This is typically triggered by the product.stock_restored event when stock goes from 0 → >0.
// It finds all inactive/ended listings for the product and calls the marketplace
// provider's ActivateOffer method to relist them.
//
// The caller context is intentionally discarded: marketplace API calls (ActivateOffer)
// are fire-and-forget and must complete even if the original HTTP request or automation
// engine context is cancelled.
//
// The Action parameter is unused because activate_listing requires no action-level
// parameters — the product is identified via event.EntityID.
//
// NOTE: Similar listing-reactivation logic exists in StockSyncService.reactivateListings
// (service package). That method serves as a direct fallback when the automation engine
// is not wired, while this one is the automation-rule-driven path. They are kept separate
// to avoid a circular dependency between the automation and service packages, and because
// they differ in provider creation (injected factory vs integration.NewMarketplaceProvider)
// and interface widths (narrow interfaces here vs full repository interfaces there).
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

	productID := event.EntityID
	bgCtx := context.Background()

	// Phase 1: Gather listings and integrations inside a DB transaction.
	type activateJob struct {
		listingID   uuid.UUID
		externalID  string
		integration *model.IntegrationWithCreds
	}
	var jobs []activateJob

	err := database.WithTenant(bgCtx, e.pool, tenantID, func(tx pgx.Tx) error {
		listings, err := e.listingDeps.ListingRepo.ListByProduct(bgCtx, tx, productID)
		if err != nil {
			return fmt.Errorf("list listings: %w", err)
		}

		for _, listing := range listings {
			// Only activate listings that are inactive or ended
			if listing.Status != "inactive" && listing.Status != "ended" {
				continue
			}
			if listing.ExternalID == nil || *listing.ExternalID == "" {
				continue
			}

			integ, err := e.listingDeps.IntegrationRepo.FindByID(bgCtx, tx, listing.IntegrationID)
			if err != nil || integ == nil {
				e.logger.Warn("activate_listing: integration not found",
					"listing_id", listing.ID, "integration_id", listing.IntegrationID)
				continue
			}

			jobs = append(jobs, activateJob{
				listingID:   listing.ID,
				externalID:  *listing.ExternalID,
				integration: integ,
			})
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("activate_listing: gather data: %w", err)
	}

	if len(jobs) == 0 {
		e.logger.Info("automation action activate_listing: no inactive listings to activate",
			"product_id", productID)
		return nil
	}

	// Phase 2: Activate via marketplace APIs outside the transaction.
	activated, failed := 0, 0
	for _, job := range jobs {
		if job.integration.EncryptedCredentials == "" {
			e.logger.Warn("activate_listing: no credentials for integration",
				"integration_id", job.integration.ID)
			failed++
			continue
		}

		credJSON, err := crypto.Decrypt(job.integration.EncryptedCredentials, e.listingDeps.EncryptionKey)
		if err != nil {
			e.logger.Error("activate_listing: decrypt credentials failed",
				"integration_id", job.integration.ID, "error", err)
			failed++
			continue
		}

		provider, err := e.listingDeps.ProviderFactory(job.integration.Provider, credJSON, job.integration.Settings)
		if err != nil {
			e.logger.Error("activate_listing: create provider failed",
				"provider", job.integration.Provider, "error", err)
			failed++
			continue
		}

		if err := provider.ActivateOffer(bgCtx, job.externalID); err != nil {
			e.logger.Error("activate_listing: activation failed",
				"listing_id", job.listingID, "external_id", job.externalID,
				"provider", job.integration.Provider, "error", err)
			failed++
			continue
		}

		// Update listing status to active (best-effort)
		activeStatus := "active"
		syncOK := "synced"
		if err := database.WithTenant(bgCtx, e.pool, tenantID, func(tx pgx.Tx) error {
			return e.listingDeps.ListingRepo.Update(bgCtx, tx, job.listingID, &model.UpdateProductListingRequest{
				Status:     &activeStatus,
				SyncStatus: &syncOK,
			})
		}); err != nil {
			e.logger.Warn("activate_listing: failed to update listing status in DB",
				"listing_id", job.listingID, "error", err)
		}
		activated++
	}

	e.logger.Info("automation action activate_listing completed",
		"tenant_id", tenantID,
		"product_id", productID,
		"activated", activated,
		"failed", failed,
	)
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
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}
