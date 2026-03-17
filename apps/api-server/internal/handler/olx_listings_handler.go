package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	olxsdk "github.com/openoms-org/openoms/packages/olx-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	olxintegration "github.com/openoms-org/openoms/apps/api-server/internal/integration/olx"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// olxCityCache caches OLX cities in memory (OLX API does not support search).
type olxCityCache struct {
	mu        sync.RWMutex
	cities    []olxsdk.City
	fetchedAt time.Time
}

var globalOLXCityCache = &olxCityCache{}

const olxCityCacheTTL = 24 * time.Hour

func (c *olxCityCache) get() ([]olxsdk.City, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.cities) == 0 || time.Since(c.fetchedAt) > olxCityCacheTTL {
		return nil, false
	}
	return c.cities, true
}

func (c *olxCityCache) set(cities []olxsdk.City) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cities = cities
	c.fetchedAt = time.Now()
}

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

type olxAttributeValue struct {
	Code  string `json:"code"`
	Value any    `json:"value,omitempty"`
}

type createOLXListingRequest struct {
	IntegrationID string              `json:"integration_id"`
	CategoryID    int                 `json:"category_id"`
	CityID        int                 `json:"city_id"`
	DistrictID    int                 `json:"district_id,omitempty"`
	ContactName   string              `json:"contact_name"`
	ContactPhone  string              `json:"contact_phone,omitempty"`
	Title         string              `json:"title,omitempty"`
	Description   string              `json:"description,omitempty"`
	PriceOverride *float64            `json:"price_override,omitempty"`
	StockOverride *int                `json:"stock_override,omitempty"`
	Attributes    []olxAttributeValue `json:"attributes,omitempty"`
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

	// Pre-check for duplicate listing before calling OLX API.
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
	if req.DistrictID > 0 {
		listingData["district_id"] = float64(req.DistrictID)
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
	if len(req.Attributes) > 0 {
		var attrs []olxsdk.AdvertAttribute
		for _, a := range req.Attributes {
			attrs = append(attrs, olxsdk.AdvertAttribute{Code: a.Code, Value: a.Value})
		}
		listingData["attributes"] = attrs
	}

	externalID, err := provider.PushOffer(ctx, product, listingData)
	if err != nil {
		slog.Error("olx listings: failed to push offer", "error", err, "tenant_id", tenantID)
		var apiErr *olxsdk.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode < 500 {
			// Do not forward raw OLX API error messages to client — may contain internal details
			slog.Warn("olx listings: client error from OLX API", "status", apiErr.StatusCode, "message", apiErr.Message)
			writeError(w, http.StatusUnprocessableEntity, "OLX rejected the listing — check listing data and try again")
			return
		}
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
		// Best-effort cleanup: deactivate the OLX advert we just created.
		if deactErr := provider.DeactivateOffer(ctx, externalID); deactErr != nil {
			slog.Error("olx listings: failed to deactivate orphaned advert", "external_id", externalID, "error", deactErr) //nolint:gosec
		} else {
			slog.Info("olx listings: deactivated orphaned advert after DB failure", "external_id", externalID) //nolint:gosec
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeError(w, http.StatusConflict, "listing already exists for this product and integration")
			return
		}
		slog.Error("olx listings: DB save failed after advert creation", "external_id", externalID, "tenant_id", tenantID, "error", err) //nolint:gosec
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

// ListCities returns OLX cities filtered by query.
// OLX Partner API does not support search on /cities, so we fetch all cities,
// cache them in memory (24h TTL), and filter server-side.
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

	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("query")))
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	allCities, ok := globalOLXCityCache.get()
	if !ok {
		provider, err := h.createProvider(ctx, tenantID, iid)
		if err != nil {
			slog.Error("olx listings: failed to create provider", "error", err)
			writeError(w, http.StatusBadRequest, "OLX integration not configured")
			return
		}

		// Fetch all cities from OLX (paginate through all results).
		var fetched []olxsdk.City
		offset := 0
		batchSize := 1000
		for {
			batch, err := provider.Client().Cities.ListCities(ctx, "", offset, batchSize)
			if err != nil {
				writeServerError(w, "failed to list OLX cities", err)
				return
			}
			fetched = append(fetched, batch.Data...)
			if len(batch.Data) < batchSize {
				break
			}
			offset += batchSize
		}
		slog.Info("olx: cached cities from OLX API", "count", len(fetched)) //nolint:gosec
		globalOLXCityCache.set(fetched)
		allCities = fetched
	}

	// Filter by query.
	var filtered []olxsdk.City
	if query == "" {
		filtered = allCities
	} else {
		for _, c := range allCities {
			if strings.Contains(strings.ToLower(c.Name), query) {
				filtered = append(filtered, c)
				if len(filtered) >= limit {
					break
				}
			}
		}
	}

	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	writeJSON(w, http.StatusOK, &olxsdk.CityListResponse{
		Data:  filtered,
		Total: len(filtered),
	})
}

// ListDistricts returns districts for a city.
func (h *OLXListingsHandler) ListDistricts(w http.ResponseWriter, r *http.Request) {
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

	cityID, err := strconv.Atoi(chi.URLParam(r, "cityId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid city ID")
		return
	}

	provider, err := h.createProvider(ctx, tenantID, iid)
	if err != nil {
		slog.Error("olx listings: failed to create provider", "error", err)
		writeError(w, http.StatusBadRequest, "OLX integration not configured")
		return
	}

	result, err := provider.Client().Cities.ListDistricts(ctx, cityID)
	if err != nil {
		writeServerError(w, "failed to list OLX districts", err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// SuggestCategory proxies OLX category suggestion based on a query string.
func (h *OLXListingsHandler) SuggestCategory(w http.ResponseWriter, r *http.Request) {
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

	query := r.URL.Query().Get("query")
	if query == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}

	provider, err := h.createProvider(ctx, tenantID, iid)
	if err != nil {
		slog.Error("olx listings: failed to create provider", "error", err)
		writeError(w, http.StatusBadRequest, "OLX integration not configured")
		return
	}

	result, err := provider.Client().Categories.SuggestCategory(ctx, query)
	if err != nil {
		writeServerError(w, "failed to suggest OLX category", err)
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
