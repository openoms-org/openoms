package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	ebaysdk "github.com/openoms-org/openoms/packages/ebay-go-sdk"

	ebayIntegration "github.com/openoms-org/openoms/apps/api-server/internal/integration/ebay"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// EbayCarrier represents an eBay shipping carrier code.
type EbayCarrier struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

var ebayCarriers = []EbayCarrier{
	{"DHL", "DHL"},
	{"DPD", "DPD"},
	{"GLS", "GLS"},
	{"HERMES", "Hermes"},
	{"UPS", "UPS"},
	{"FEDEX", "FedEx"},
	{"USPS", "USPS"},
	{"ROYAL_MAIL", "Royal Mail"},
	{"TNT", "TNT"},
	{"POCZTA_POLSKA", "Poczta Polska"},
	{"INPOST", "InPost"},
	{"OTHER", "Inny"},
}

// EbayHandler handles eBay-specific API endpoints (fulfillment, tracking, carriers).
type EbayHandler struct {
	integrationService *service.IntegrationService
	orderService       *service.OrderService
	encryptionKey      []byte
}

// NewEbayHandler creates a new EbayHandler.
func NewEbayHandler(integrationService *service.IntegrationService, orderService *service.OrderService, encryptionKey []byte) *EbayHandler {
	return &EbayHandler{
		integrationService: integrationService,
		orderService:       orderService,
		encryptionKey:      encryptionKey,
	}
}

// getClient creates an eBay SDK client from the integration credentials for the given tenant.
func (h *EbayHandler) getClient(ctx context.Context, tenantID uuid.UUID) (*ebaysdk.Client, error) {
	credJSON, _, err := h.integrationService.GetDecryptedCredentialsByProvider(ctx, tenantID, "ebay")
	if err != nil {
		return nil, err
	}

	var creds ebayIntegration.EbayCredentials
	if err := json.Unmarshal(credJSON, &creds); err != nil {
		return nil, err
	}

	var opts []ebaysdk.Option
	if creds.Sandbox {
		opts = append(opts, ebaysdk.WithSandbox())
	}

	client := ebaysdk.NewClient(creds.AppID, creds.CertID, creds.DevID, creds.RefreshToken, opts...)
	return client, nil
}

// AddTracking handles POST /v1/integrations/ebay/orders/{orderId}/tracking.
// It pushes tracking/carrier information to eBay for an order.
func (h *EbayHandler) AddTracking(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromContext(ctx)
	orderIDStr := chi.URLParam(r, "orderId")

	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	var body struct {
		CarrierCode    string `json:"carrier_code"`
		TrackingNumber string `json:"tracking_number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.CarrierCode == "" || body.TrackingNumber == "" {
		writeError(w, http.StatusBadRequest, "carrier_code and tracking_number are required")
		return
	}

	// Get order from DB to find external_id
	order, err := h.orderService.Get(ctx, tenantID, orderID)
	if err != nil {
		slog.Error("ebay tracking: get order failed", "error", err)
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	if order.ExternalID == nil || *order.ExternalID == "" {
		writeError(w, http.StatusBadRequest, "order has no external eBay ID")
		return
	}

	client, err := h.getClient(ctx, tenantID)
	if err != nil {
		writeServerError(w, "failed to connect to eBay", err)
		return
	}

	// Build line items from the order's Items JSONB.
	// Items stored in DB contain external_id (eBay lineItemId) and quantity.
	lineItems, err := h.buildLineItems(ctx, client, *order.ExternalID, order.Items)
	if err != nil {
		slog.Error("ebay tracking: build line items failed", "order_id", orderIDStr, "error", err)
		writeServerError(w, "failed to resolve order line items", err)
		return
	}

	fulfillmentReq := ebaysdk.ShippingFulfillmentRequest{
		LineItems:           lineItems,
		ShippingCarrierCode: body.CarrierCode,
		TrackingNumber:      body.TrackingNumber,
		ShippedDate:         time.Now().UTC().Format(time.RFC3339),
	}

	result, err := client.Fulfillment.CreateShippingFulfillment(ctx, *order.ExternalID, fulfillmentReq)
	if err != nil {
		slog.Error("ebay tracking: create fulfillment failed", "order_id", orderIDStr, "error", err)
		writeServerError(w, "failed to push tracking to eBay", err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// orderItem is a minimal struct for parsing order items from JSONB.
type orderItem struct {
	ExternalID string `json:"external_id"`
	Quantity   int    `json:"quantity"`
}

// buildLineItems attempts to parse line items from the order's Items JSONB.
// If parsing fails or yields no items, it falls back to fetching the order from eBay API.
func (h *EbayHandler) buildLineItems(ctx context.Context, client *ebaysdk.Client, externalOrderID string, itemsJSON json.RawMessage) ([]ebaysdk.FulfillmentLineItem, error) {
	// Try parsing from stored order items first
	if len(itemsJSON) > 0 {
		var items []orderItem
		if err := json.Unmarshal(itemsJSON, &items); err == nil && len(items) > 0 {
			var lineItems []ebaysdk.FulfillmentLineItem
			allHaveExternalID := true
			for _, item := range items {
				if item.ExternalID == "" {
					allHaveExternalID = false
					break
				}
				lineItems = append(lineItems, ebaysdk.FulfillmentLineItem{
					LineItemID: item.ExternalID,
					Quantity:   item.Quantity,
				})
			}
			if allHaveExternalID && len(lineItems) > 0 {
				return lineItems, nil
			}
		}
	}

	// Fall back to fetching line items from eBay API
	ebayOrder, err := client.Orders.GetOrder(ctx, externalOrderID)
	if err != nil {
		return nil, err
	}

	var lineItems []ebaysdk.FulfillmentLineItem
	for _, li := range ebayOrder.LineItems {
		lineItems = append(lineItems, ebaysdk.FulfillmentLineItem{
			LineItemID: li.LineItemID,
			Quantity:   li.Quantity,
		})
	}
	return lineItems, nil
}

// IssueRefund handles POST /v1/integrations/ebay/orders/{orderId}/refund.
// It issues a refund on eBay for the given order.
func (h *EbayHandler) IssueRefund(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromContext(ctx)
	orderIDStr := chi.URLParam(r, "orderId")

	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	var body struct {
		Reason  string               `json:"reason"`
		Comment string               `json:"comment,omitempty"`
		Items   []ebaysdk.RefundItem `json:"items,omitempty"`
		Amount  *ebaysdk.Amount      `json:"amount,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Reason == "" {
		writeError(w, http.StatusBadRequest, "reason is required")
		return
	}

	order, err := h.orderService.Get(ctx, tenantID, orderID)
	if err != nil {
		slog.Error("ebay refund: get order failed", "error", err)
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	if order.ExternalID == nil || *order.ExternalID == "" {
		writeError(w, http.StatusBadRequest, "order has no external eBay ID")
		return
	}

	client, err := h.getClient(ctx, tenantID)
	if err != nil {
		writeServerError(w, "failed to connect to eBay", err)
		return
	}

	refundReq := ebaysdk.IssueRefundRequest{
		ReasonForRefund:        body.Reason,
		Comment:                body.Comment,
		RefundItems:            body.Items,
		OrderLevelRefundAmount: body.Amount,
	}

	result, err := client.Fulfillment.IssueRefund(ctx, *order.ExternalID, refundReq)
	if err != nil {
		slog.Error("ebay refund: issue refund failed", "order_id", orderIDStr, "error", err)
		writeServerError(w, "failed to issue refund on eBay", err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// ListCarriers handles GET /v1/integrations/ebay/carriers.
// It returns the list of available eBay shipping carrier codes.
func (h *EbayHandler) ListCarriers(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string][]EbayCarrier{"carriers": ebayCarriers})
}
