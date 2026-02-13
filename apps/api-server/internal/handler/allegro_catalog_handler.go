package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// AllegroCatalogHandler handles Allegro catalog browsing, product search, and fee calculation endpoints.
type AllegroCatalogHandler struct {
	integrationService *service.IntegrationService
	encryptionKey      []byte
	cache              *allegroCache
}

// NewAllegroCatalogHandler creates a new AllegroCatalogHandler.
func NewAllegroCatalogHandler(integrationService *service.IntegrationService, encryptionKey []byte) *AllegroCatalogHandler {
	return &AllegroCatalogHandler{
		integrationService: integrationService,
		encryptionKey:      encryptionKey,
		cache:              newAllegroCache(24 * time.Hour),
	}
}

// newAllegroClient creates an authenticated Allegro SDK client with auto-refresh.
func (h *AllegroCatalogHandler) newAllegroClient(r *http.Request) (*allegrosdk.Client, error) {
	return buildAllegroClient(r, h.integrationService, h.encryptionKey)
}

// ListCategories lists Allegro categories. Accepts optional query param "parent_id".
// GET /v1/integrations/allegro/categories
func (h *AllegroCatalogHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	parentID := r.URL.Query().Get("parent_id")
	tenantID := middleware.TenantIDFromContext(r.Context())
	cacheKey := fmt.Sprintf("cat:%s:%s", tenantID, parentID)

	if cached, ok := h.cache.Get(cacheKey); ok {
		writeJSON(w, http.StatusOK, cached)
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro catalog: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	categories, err := client.Categories.List(r.Context(), parentID)
	if err != nil {
		slog.Error("allegro catalog: failed to list categories", "error", err)
		writeError(w, http.StatusBadGateway, "Nie udalo sie pobrac kategorii Allegro")
		return
	}

	h.cache.Set(cacheKey, categories)
	writeJSON(w, http.StatusOK, categories)
}

// GetCategory retrieves a single Allegro category.
// GET /v1/integrations/allegro/categories/{categoryId}
func (h *AllegroCatalogHandler) GetCategory(w http.ResponseWriter, r *http.Request) {
	categoryID := chi.URLParam(r, "categoryId")
	if categoryID == "" {
		writeError(w, http.StatusBadRequest, "categoryId jest wymagane")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro catalog: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	category, err := client.Categories.Get(r.Context(), categoryID)
	if err != nil {
		slog.Error("allegro catalog: failed to get category", "error", err, "category_id", categoryID)
		writeError(w, http.StatusBadGateway, "Nie udalo sie pobrac kategorii Allegro")
		return
	}

	writeJSON(w, http.StatusOK, category)
}

// GetCategoryParameters retrieves parameters for a category.
// GET /v1/integrations/allegro/categories/{categoryId}/parameters
func (h *AllegroCatalogHandler) GetCategoryParameters(w http.ResponseWriter, r *http.Request) {
	categoryID := chi.URLParam(r, "categoryId")
	if categoryID == "" {
		writeError(w, http.StatusBadRequest, "categoryId jest wymagane")
		return
	}

	tenantID := middleware.TenantIDFromContext(r.Context())
	cacheKey := fmt.Sprintf("params:%s:%s", tenantID, categoryID)

	if cached, ok := h.cache.Get(cacheKey); ok {
		writeJSON(w, http.StatusOK, cached)
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro catalog: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	params, err := client.Categories.GetParameters(r.Context(), categoryID)
	if err != nil {
		slog.Error("allegro catalog: failed to get category parameters", "error", err, "category_id", categoryID)
		writeError(w, http.StatusBadGateway, "Nie udalo sie pobrac parametrow kategorii")
		return
	}

	h.cache.Set(cacheKey, params)
	writeJSON(w, http.StatusOK, params)
}

// SearchCategories returns category suggestions for a given phrase.
// GET /v1/integrations/allegro/categories/search?name=...
func (h *AllegroCatalogHandler) SearchCategories(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "Parametr 'name' jest wymagany")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro catalog: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	result, err := client.Categories.SearchMatching(r.Context(), name)
	if err != nil {
		slog.Error("allegro catalog: failed to search categories", "error", err, "name", name)
		writeError(w, http.StatusBadGateway, "Nie udalo sie wyszukac kategorii")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// SearchProducts searches the Allegro product catalog.
// GET /v1/integrations/allegro/products/catalog
func (h *AllegroCatalogHandler) SearchProducts(w http.ResponseWriter, r *http.Request) {
	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro catalog: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	params := &allegrosdk.SearchProductsParams{}
	if v := r.URL.Query().Get("phrase"); v != "" {
		params.Phrase = v
	}
	if v := r.URL.Query().Get("category_id"); v != "" {
		params.CategoryID = v
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		params.Limit, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		params.Offset, _ = strconv.Atoi(v)
	}

	products, err := client.ProductCatalog.Search(r.Context(), params)
	if err != nil {
		slog.Error("allegro catalog: failed to search products", "error", err)
		writeError(w, http.StatusBadGateway, "Nie udalo sie wyszukac produktow w katalogu Allegro")
		return
	}

	writeJSON(w, http.StatusOK, products)
}

// GetProduct retrieves a single product from the Allegro catalog.
// GET /v1/integrations/allegro/products/catalog/{productId}
func (h *AllegroCatalogHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "productId")
	if productID == "" {
		writeError(w, http.StatusBadRequest, "productId jest wymagane")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro catalog: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	product, err := client.ProductCatalog.Get(r.Context(), productID)
	if err != nil {
		slog.Error("allegro catalog: failed to get product", "error", err, "product_id", productID)
		writeError(w, http.StatusBadGateway, "Nie udalo sie pobrac produktu z katalogu Allegro")
		return
	}

	writeJSON(w, http.StatusOK, product)
}

// GetFeePreview calculates fees for an offer.
// GET /v1/integrations/allegro/pricing/fees
func (h *AllegroCatalogHandler) GetFeePreview(w http.ResponseWriter, r *http.Request) {
	offerID := r.URL.Query().Get("offer_id")
	if offerID == "" {
		writeError(w, http.StatusBadRequest, "offer_id jest wymagane")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro pricing: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	fees, err := client.Pricing.GetFeePreview(r.Context(), offerID)
	if err != nil {
		slog.Error("allegro pricing: failed to get fee preview", "error", err, "offer_id", offerID)
		writeError(w, http.StatusBadGateway, "Nie udalo sie obliczyc prowizji dla oferty")
		return
	}

	writeJSON(w, http.StatusOK, fees)
}

// GetCommissions lists commission rates by category.
// GET /v1/integrations/allegro/pricing/commissions
func (h *AllegroCatalogHandler) GetCommissions(w http.ResponseWriter, r *http.Request) {
	categoryID := r.URL.Query().Get("category_id")
	if categoryID == "" {
		writeError(w, http.StatusBadRequest, "category_id jest wymagane")
		return
	}

	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro pricing: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	commissions, err := client.Pricing.GetCommissions(r.Context(), categoryID)
	if err != nil {
		slog.Error("allegro pricing: failed to get commissions", "error", err, "category_id", categoryID)
		writeError(w, http.StatusBadGateway, "Nie udalo sie pobrac stawek prowizji")
		return
	}

	writeJSON(w, http.StatusOK, commissions)
}
