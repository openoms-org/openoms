package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/integration/woocommerce"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// WooCommerceListingsHandler handles WooCommerce product listing operations.
type WooCommerceListingsHandler struct {
	integrationService *service.IntegrationService
	productService     *service.ProductService
	listingRepo        *repository.ProductListingRepository
	pool               *pgxpool.Pool
}

// NewWooCommerceListingsHandler creates a new WooCommerceListingsHandler.
func NewWooCommerceListingsHandler(
	integrationService *service.IntegrationService,
	productService *service.ProductService,
	listingRepo *repository.ProductListingRepository,
	pool *pgxpool.Pool,
) *WooCommerceListingsHandler {
	return &WooCommerceListingsHandler{
		integrationService: integrationService,
		productService:     productService,
		listingRepo:        listingRepo,
		pool:               pool,
	}
}

type createWooListingRequest struct {
	IntegrationID string   `json:"integration_id"`
	PriceOverride *float64 `json:"price_override,omitempty"`
	StockOverride *int     `json:"stock_override,omitempty"`
	Categories    []string `json:"categories,omitempty"`
	Description   string   `json:"description,omitempty"`
}

// CreateListing publishes a product to WooCommerce and saves the listing record.
func (h *WooCommerceListingsHandler) CreateListing(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromContext(ctx)

	productID, err := uuid.Parse(chi.URLParam(r, "productId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	var req createWooListingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.IntegrationID == "" {
		writeError(w, http.StatusBadRequest, "integration_id is required")
		return
	}

	integrationID, err := uuid.Parse(req.IntegrationID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid integration_id")
		return
	}

	product, err := h.productService.Get(ctx, tenantID, productID)
	if err != nil {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}

	// Check duplicate
	var existingListing *model.ProductListing
	err = database.WithTenant(ctx, h.pool, tenantID, func(tx pgx.Tx) error {
		var findErr error
		existingListing, findErr = h.listingRepo.FindByProductAndIntegration(ctx, tx, productID, integrationID)
		return findErr
	})
	if err != nil {
		slog.Error("woocommerce listings: failed to check existing listing", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to check existing listing")
		return
	}
	if existingListing != nil {
		writeError(w, http.StatusConflict, "listing already exists for this product and integration")
		return
	}

	// Create WooCommerce provider
	credJSON, err := h.integrationService.GetDecryptedCredentialsByID(ctx, tenantID, integrationID)
	if err != nil {
		slog.Error("woocommerce listings: failed to get credentials", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja WooCommerce nie jest skonfigurowana")
		return
	}

	provider, err := woocommerce.NewProvider(credJSON, nil)
	if err != nil {
		slog.Error("woocommerce listings: failed to create provider", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to initialize WooCommerce client")
		return
	}

	// Build listing data for PushOffer
	listingData := make(map[string]any)

	if req.Description != "" {
		listingData["description"] = req.Description
	} else if product.DescriptionLong != "" {
		listingData["description"] = product.DescriptionLong
	}
	if product.DescriptionShort != "" {
		listingData["short_description"] = product.DescriptionShort
	}

	price := product.Price
	if req.PriceOverride != nil {
		price = *req.PriceOverride
	}
	listingData["regular_price"] = fmt.Sprintf("%.2f", price)

	stock := product.StockQuantity
	if req.StockOverride != nil {
		stock = *req.StockOverride
	}
	listingData["manage_stock"] = true
	listingData["stock_quantity"] = stock

	if product.EAN != nil && *product.EAN != "" {
		listingData["sku"] = *product.EAN
	}

	if len(req.Categories) > 0 {
		cats := make([]map[string]any, 0, len(req.Categories))
		for _, c := range req.Categories {
			cats = append(cats, map[string]any{"name": c})
		}
		listingData["categories"] = cats
	}

	// Build images
	imageURLs := buildImages(product)
	if len(imageURLs) > 0 {
		imgs := make([]map[string]any, 0, len(imageURLs))
		for _, u := range imageURLs {
			imgs = append(imgs, map[string]any{"src": u})
		}
		listingData["images"] = imgs
	}

	// Push to WooCommerce
	externalID, err := provider.PushOffer(ctx, product, listingData)
	if err != nil {
		slog.Error("woocommerce listings: push offer failed", "error", err, "product_id", productID)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Błąd wystawiania na WooCommerce: %v", err))
		return
	}

	metadataJSON, _ := json.Marshal(map[string]any{
		"categories": req.Categories,
	})

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
		slog.Error("woocommerce listings: failed to save listing", "error", err, "product_id", productID)
		writeError(w, http.StatusInternalServerError, "product published to WooCommerce but failed to save listing record")
		return
	}

	writeJSON(w, http.StatusCreated, listing)
}
