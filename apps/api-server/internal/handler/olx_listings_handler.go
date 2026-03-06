package handler

import (
	"context"
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
	olxintegration "github.com/openoms-org/openoms/apps/api-server/internal/integration/olx"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// OLXListingsHandler handles OLX product listing operations.
type OLXListingsHandler struct {
	integrationService *service.IntegrationService
	productService     *service.ProductService
	listingRepo        *repository.ProductListingRepository
	pool               *pgxpool.Pool
}

// NewOLXListingsHandler creates a new OLXListingsHandler.
func NewOLXListingsHandler(
	integrationService *service.IntegrationService,
	productService *service.ProductService,
	listingRepo *repository.ProductListingRepository,
	pool *pgxpool.Pool,
) *OLXListingsHandler {
	return &OLXListingsHandler{
		integrationService: integrationService,
		productService:     productService,
		listingRepo:        listingRepo,
		pool:               pool,
	}
}

type createOLXListingRequest struct {
	IntegrationID string   `json:"integration_id"`
	CategoryID    int      `json:"category_id"`
	CityID        int      `json:"city_id"`
	ContactName   string   `json:"contact_name"`
	ContactPhone  string   `json:"contact_phone,omitempty"`
	Title         string   `json:"title,omitempty"`
	Description   string   `json:"description,omitempty"`
	PriceOverride *float64 `json:"price_override,omitempty"`
	StockOverride *int     `json:"stock_override,omitempty"`
}

// CreateListing creates an OLX advert from a product and saves the listing record.
func (h *OLXListingsHandler) CreateListing(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromContext(ctx)

	productID, err := uuid.Parse(chi.URLParam(r, "productId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req createOLXListingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.IntegrationID == "" {
		writeError(w, http.StatusBadRequest, "integration_id is required")
		return
	}
	if req.CategoryID <= 0 {
		writeError(w, http.StatusBadRequest, "category_id is required")
		return
	}
	if req.CityID <= 0 {
		writeError(w, http.StatusBadRequest, "city_id is required")
		return
	}
	if req.ContactName == "" {
		writeError(w, http.StatusBadRequest, "contact_name is required")
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

	credJSON, err := h.integrationService.GetDecryptedCredentialsByID(ctx, tenantID, integrationID)
	if err != nil {
		slog.Error("olx listings: failed to get credentials", "error", err)
		writeError(w, http.StatusBadRequest, "OLX integration not configured")
		return
	}

	provider, err := olxintegration.NewProvider(credJSON, nil)
	if err != nil {
		writeServerError(w, "failed to initialize OLX client", err)
		return
	}

	listingData := map[string]any{
		"category_id":  float64(req.CategoryID),
		"city_id":      float64(req.CityID),
		"contact_name": req.ContactName,
	}
	if req.ContactPhone != "" {
		listingData["contact_phone"] = req.ContactPhone
	}
	if req.Title != "" {
		listingData["title"] = req.Title
	}
	if req.Description != "" {
		listingData["description"] = req.Description
	}
	if req.PriceOverride != nil {
		listingData["price"] = *req.PriceOverride
	}

	externalID, err := provider.PushOffer(ctx, product, listingData)
	if err != nil {
		slog.Error("olx listings: failed to push offer", "error", err, "tenant_id", tenantID)
		writeError(w, http.StatusBadGateway, "failed to publish listing on OLX")
		return
	}

	metadataJSON, err := json.Marshal(map[string]any{
		"category_id":  req.CategoryID,
		"city_id":      req.CityID,
		"contact_name": req.ContactName,
	})
	if err != nil {
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
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			slog.Warn("olx listings: duplicate listing conflict; newly created OLX advert is orphaned",
				"orphaned_external_id", externalID, "tenant_id", tenantID)
			writeError(w, http.StatusConflict, "listing already exists for this product and integration")
			return
		}
		slog.Error("olx listings: advert created on OLX but DB save failed", "external_id", externalID, "tenant_id", tenantID, "error", err)
		writeServerError(w, "failed to save listing record", err)
		return
	}

	writeJSON(w, http.StatusCreated, listing)
}

// ListCategories proxies OLX category listing for the frontend.
func (h *OLXListingsHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
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

	provider, err := h.createProvider(ctx, tenantID, iid)
	if err != nil {
		slog.Error("olx listings: failed to create provider", "error", err)
		writeError(w, http.StatusBadRequest, "OLX integration not configured")
		return
	}

	parentID := 0
	if v := r.URL.Query().Get("parent_id"); v != "" {
		parentID, _ = strconv.Atoi(v)
	}

	result, err := provider.Client().Categories.ListCategories(ctx, parentID)
	if err != nil {
		writeServerError(w, "failed to list OLX categories", err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetCategoryAttributes proxies OLX category attribute retrieval.
func (h *OLXListingsHandler) GetCategoryAttributes(w http.ResponseWriter, r *http.Request) {
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

	categoryID, err := strconv.Atoi(chi.URLParam(r, "categoryId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category ID")
		return
	}

	provider, err := h.createProvider(ctx, tenantID, iid)
	if err != nil {
		slog.Error("olx listings: failed to create provider", "error", err)
		writeError(w, http.StatusBadRequest, "OLX integration not configured")
		return
	}

	result, err := provider.Client().Categories.GetAttributes(ctx, categoryID)
	if err != nil {
		writeServerError(w, "failed to get OLX category attributes", err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// ListCities proxies OLX city search for the frontend.
func (h *OLXListingsHandler) ListCities(w http.ResponseWriter, r *http.Request) {
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

	provider, err := h.createProvider(ctx, tenantID, iid)
	if err != nil {
		slog.Error("olx listings: failed to create provider", "error", err)
		writeError(w, http.StatusBadRequest, "OLX integration not configured")
		return
	}

	query := r.URL.Query().Get("query")
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	result, err := provider.Client().Cities.ListCities(ctx, query, 0, limit)
	if err != nil {
		writeServerError(w, "failed to list OLX cities", err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// createProvider creates an OLX provider from integration credentials.
func (h *OLXListingsHandler) createProvider(ctx context.Context, tenantID, integrationID uuid.UUID) (*olxintegration.Provider, error) {
	credJSON, err := h.integrationService.GetDecryptedCredentialsByID(ctx, tenantID, integrationID)
	if err != nil {
		return nil, errors.New("OLX integration not configured")
	}

	provider, err := olxintegration.NewProvider(credJSON, nil)
	if err != nil {
		return nil, errors.New("failed to initialize OLX client")
	}

	return provider, nil
}
