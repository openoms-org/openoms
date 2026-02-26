package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/integration/erli"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// ErliListingsHandler handles Erli.pl product listing operations.
type ErliListingsHandler struct {
	integrationService *service.IntegrationService
	productService     *service.ProductService
	listingRepo        *repository.ProductListingRepository
	pool               *pgxpool.Pool
}

// NewErliListingsHandler creates a new ErliListingsHandler.
func NewErliListingsHandler(
	integrationService *service.IntegrationService,
	productService *service.ProductService,
	listingRepo *repository.ProductListingRepository,
	pool *pgxpool.Pool,
) *ErliListingsHandler {
	return &ErliListingsHandler{
		integrationService: integrationService,
		productService:     productService,
		listingRepo:        listingRepo,
		pool:               pool,
	}
}

type createErliListingRequest struct {
	IntegrationID string   `json:"integration_id"`
	CategoryID    string   `json:"category_id,omitempty"`
	PriceOverride *float64 `json:"price_override,omitempty"`
	StockOverride *int     `json:"stock_override,omitempty"`
}

// CreateListing publishes a product to Erli and saves the listing record.
func (h *ErliListingsHandler) CreateListing(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromContext(ctx)

	productID, err := uuid.Parse(chi.URLParam(r, "productId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req createErliListingRequest
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
		if errors.Is(err, service.ErrProductNotFound) {
			writeError(w, http.StatusNotFound, "product not found")
		} else {
			writeServerError(w, "failed to get product", err)
		}
		return
	}

	// Check for duplicate listing
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
		slog.Error("erli listings: failed to get credentials", "error", err)
		writeError(w, http.StatusBadRequest, "Erli integration not configured")
		return
	}

	provider, err := erli.NewProvider(credJSON, nil)
	if err != nil {
		writeServerError(w, "failed to initialize Erli client", err)
		return
	}

	listingData := make(map[string]any)
	if req.CategoryID != "" {
		listingData["category_id"] = req.CategoryID
	}
	if req.PriceOverride != nil {
		listingData["price"] = *req.PriceOverride
	}
	if req.StockOverride != nil {
		listingData["stock"] = *req.StockOverride
	}

	externalID, err := provider.PushOffer(ctx, product, listingData)
	if err != nil {
		writeServerError(w, "Erli listing error", err)
		return
	}

	metadataJSON, _ := json.Marshal(map[string]any{
		"category_id": req.CategoryID,
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
		writeServerError(w, "product published to Erli but failed to save listing record", err)
		return
	}

	writeJSON(w, http.StatusCreated, listing)
}
