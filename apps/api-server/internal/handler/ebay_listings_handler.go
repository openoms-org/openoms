package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	ebayintegration "github.com/openoms-org/openoms/apps/api-server/internal/integration/ebay"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// EbayListingsHandler handles eBay product listing operations.
type EbayListingsHandler struct {
	integrationService *service.IntegrationService
	productService     *service.ProductService
	listingRepo        *repository.ProductListingRepository
	pool               *pgxpool.Pool
	ebayImportService  *service.EbayImportService
}

// NewEbayListingsHandler creates a new EbayListingsHandler.
func NewEbayListingsHandler(
	integrationService *service.IntegrationService,
	productService *service.ProductService,
	listingRepo *repository.ProductListingRepository,
	pool *pgxpool.Pool,
	ebayImportService *service.EbayImportService,
) *EbayListingsHandler {
	return &EbayListingsHandler{
		integrationService: integrationService,
		productService:     productService,
		listingRepo:        listingRepo,
		pool:               pool,
		ebayImportService:  ebayImportService,
	}
}

type createEbayListingRequest struct {
	IntegrationID       string   `json:"integration_id"`
	CategoryID          string   `json:"category_id"`
	MarketplaceID       string   `json:"marketplace_id,omitempty"`
	FulfillmentPolicyID string   `json:"fulfillment_policy_id"`
	ReturnPolicyID      string   `json:"return_policy_id"`
	PaymentPolicyID     string   `json:"payment_policy_id,omitempty"`
	Condition           string   `json:"condition,omitempty"`
	Title               string   `json:"title,omitempty"`
	Description         string   `json:"description,omitempty"`
	PriceOverride       *float64 `json:"price_override,omitempty"`
	StockOverride       *int     `json:"stock_override,omitempty"`
}

// CreateListing publishes a product to eBay and saves the listing record.
func (h *EbayListingsHandler) CreateListing(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromContext(ctx)

	productID, err := uuid.Parse(chi.URLParam(r, "productId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req createEbayListingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.IntegrationID == "" {
		writeError(w, http.StatusBadRequest, "integration_id is required")
		return
	}
	if req.CategoryID == "" {
		writeError(w, http.StatusBadRequest, "category_id is required")
		return
	}

	integrationID, err := uuid.Parse(req.IntegrationID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid integration_id")
		return
	}

	if req.PriceOverride != nil && *req.PriceOverride <= 0 {
		writeError(w, http.StatusBadRequest, "price_override must be greater than zero")
		return
	}
	if req.StockOverride != nil && *req.StockOverride < 0 {
		writeError(w, http.StatusBadRequest, "stock_override must be non-negative")
		return
	}

	product, err := h.productService.Get(ctx, tenantID, productID)
	if err != nil {
		if errors.Is(err, service.ErrProductNotFound) {
			writeError(w, http.StatusNotFound, "product not found")
		} else {
			writeServerError(w, "failed to get product", err)
		}
		return
	}

	// Pre-check for duplicate listing before calling eBay API.
	var existingListing *model.ProductListing
	err = database.WithTenant(ctx, h.pool, tenantID, func(tx pgx.Tx) error {
		var findErr error
		existingListing, findErr = h.listingRepo.FindByProductAndIntegration(ctx, tx, productID, integrationID)
		return findErr
	})
	if err != nil {
		writeServerError(w, "failed to check existing listing", err)
		return
	}
	if existingListing != nil {
		writeError(w, http.StatusConflict, "listing already exists for this product and integration")
		return
	}

	credJSON, err := h.integrationService.GetDecryptedCredentialsByID(ctx, tenantID, integrationID)
	if err != nil {
		slog.Error("ebay listings: failed to get credentials", "error", err)
		writeError(w, http.StatusBadRequest, "eBay integration not configured")
		return
	}

	provider, err := ebayintegration.NewProvider(credJSON, nil)
	if err != nil {
		writeServerError(w, "failed to initialize eBay client", err)
		return
	}

	listingData := map[string]any{
		"category_id":           req.CategoryID,
		"fulfillment_policy_id": req.FulfillmentPolicyID,
		"return_policy_id":      req.ReturnPolicyID,
	}
	if req.MarketplaceID != "" {
		listingData["marketplace_id"] = req.MarketplaceID
	}
	if req.PaymentPolicyID != "" {
		listingData["payment_policy_id"] = req.PaymentPolicyID
	}
	if req.Condition != "" {
		listingData["condition"] = req.Condition
	}
	if req.Title != "" {
		listingData["title"] = req.Title
	}
	if req.Description != "" {
		listingData["description"] = req.Description
	}
	if req.PriceOverride != nil {
		listingData["price_override"] = *req.PriceOverride
	}
	if req.StockOverride != nil {
		listingData["stock_override"] = *req.StockOverride
	}

	externalID, err := provider.PushOffer(ctx, product, listingData)
	if err != nil {
		slog.Error("ebay listings: failed to push offer", "error", err, "tenant_id", tenantID)
		writeError(w, http.StatusBadGateway, "failed to publish listing on eBay")
		return
	}

	metadataJSON, err := json.Marshal(map[string]any{
		"category_id":           req.CategoryID,
		"marketplace_id":        req.MarketplaceID,
		"fulfillment_policy_id": req.FulfillmentPolicyID,
		"return_policy_id":      req.ReturnPolicyID,
		"payment_policy_id":     req.PaymentPolicyID,
		"condition":             req.Condition,
	})
	if err != nil {
		slog.Warn("ebay listings: failed to marshal metadata", "error", err)
		metadataJSON = []byte("{}")
	}

	now := time.Now()
	listing := &model.ProductListing{
		ID:            uuid.New(),
		TenantID:      tenantID,
		ProductID:     productID,
		IntegrationID: integrationID,
		ExternalID:    &externalID,
		Status:        "active",
		PriceOverride: req.PriceOverride,
		StockOverride: req.StockOverride,
		SyncStatus:    "synced",
		LastSyncedAt:  &now,
		Metadata:      metadataJSON,
	}

	err = database.WithTenant(ctx, h.pool, tenantID, func(tx pgx.Tx) error {
		return h.listingRepo.Create(ctx, tx, listing)
	})
	if err != nil {
		// Best-effort cleanup: set stock to 0 to hide the eBay offer.
		if stockErr := provider.UpdateStock(ctx, externalID, 0); stockErr != nil {
			slog.Error("ebay listings: failed to zero-stock orphaned offer", "external_id", externalID, "error", stockErr)
		} else {
			slog.Info("ebay listings: zeroed stock on orphaned offer after DB failure", "external_id", externalID)
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeError(w, http.StatusConflict, "listing already exists for this product and integration")
			return
		}
		slog.Error("ebay listings: DB save failed after offer creation", "external_id", externalID, "tenant_id", tenantID, "error", err)
		writeServerError(w, "failed to save listing record", err)
		return
	}

	writeJSON(w, http.StatusCreated, listing)
}

// ListPolicies returns the seller's fulfillment, return, and payment policies from eBay.
func (h *EbayListingsHandler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromContext(ctx)

	integrationID := r.URL.Query().Get("integration_id")
	if integrationID == "" {
		writeError(w, http.StatusBadRequest, "integration_id is required")
		return
	}

	iid, err := uuid.Parse(integrationID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid integration_id")
		return
	}

	marketplaceID := r.URL.Query().Get("marketplace_id")
	if marketplaceID == "" {
		marketplaceID = "EBAY_PL"
	}

	credJSON, err := h.integrationService.GetDecryptedCredentialsByID(ctx, tenantID, iid)
	if err != nil {
		slog.Error("ebay listings: failed to get credentials", "error", err)
		writeError(w, http.StatusBadRequest, "eBay integration not configured")
		return
	}

	provider, err := ebayintegration.NewProvider(credJSON, nil)
	if err != nil {
		writeServerError(w, "failed to initialize eBay client", err)
		return
	}

	client := provider.Client()

	fulfillment, err := client.Account.GetFulfillmentPolicies(ctx, marketplaceID)
	if err != nil {
		writeServerError(w, "failed to get eBay fulfillment policies", err)
		return
	}

	returnPolicies, err := client.Account.GetReturnPolicies(ctx, marketplaceID)
	if err != nil {
		writeServerError(w, "failed to get eBay return policies", err)
		return
	}

	payment, err := client.Account.GetPaymentPolicies(ctx, marketplaceID)
	if err != nil {
		writeServerError(w, "failed to get eBay payment policies", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"fulfillment": fulfillment,
		"return":      returnPolicies,
		"payment":     payment,
	})
}

// ListOffers returns eBay offers, optionally filtered by SKU.
func (h *EbayListingsHandler) ListOffers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromContext(ctx)

	integrationID := r.URL.Query().Get("integration_id")
	if integrationID == "" {
		writeError(w, http.StatusBadRequest, "integration_id is required")
		return
	}

	iid, err := uuid.Parse(integrationID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid integration_id")
		return
	}

	sku := r.URL.Query().Get("sku")
	limit := 25
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, parseErr := strconv.Atoi(v); parseErr == nil && parsed > 0 {
			limit = parsed
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if parsed, parseErr := strconv.Atoi(v); parseErr == nil && parsed >= 0 {
			offset = parsed
		}
	}

	credJSON, err := h.integrationService.GetDecryptedCredentialsByID(ctx, tenantID, iid)
	if err != nil {
		slog.Error("ebay listings: failed to get credentials", "error", err)
		writeError(w, http.StatusBadRequest, "eBay integration not configured")
		return
	}

	provider, err := ebayintegration.NewProvider(credJSON, nil)
	if err != nil {
		writeServerError(w, "failed to initialize eBay client", err)
		return
	}

	result, err := provider.Client().Offers.GetOffers(ctx, sku, limit, offset)
	if err != nil {
		writeServerError(w, "failed to get eBay offers", err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// ImportOffers handles POST /v1/integrations/ebay/import-offers.
func (h *EbayListingsHandler) ImportOffers(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	if tenantID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	result, err := h.ebayImportService.ImportOffers(r.Context(), tenantID)
	if err != nil {
		writeServerError(w, "failed to import offers from eBay", err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}
