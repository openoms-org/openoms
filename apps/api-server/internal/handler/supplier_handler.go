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
	"github.com/openoms-org/openoms/apps/api-server/internal/asyncutil"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// SupplierHandler handles HTTP requests for supplier management.
type SupplierHandler struct {
	supplierService *service.SupplierService
}

// NewSupplierHandler creates a new SupplierHandler.
func NewSupplierHandler(supplierService *service.SupplierService) *SupplierHandler {
	return &SupplierHandler{supplierService: supplierService}
}

// List returns a paginated list of suppliers.
func (h *SupplierHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.SupplierListFilter{
		PaginationParams: pagination,
	}
	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = &status
	}

	resp, err := h.supplierService.List(r.Context(), tenantID, filter)
	if err != nil {
		writeServerError(w, "failed to list suppliers", err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// Get returns a single supplier by ID.
func (h *SupplierHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	supplier, err := h.supplierService.Get(r.Context(), tenantID, supplierID)
	if err != nil {
		if errors.Is(err, service.ErrSupplierNotFound) {
			writeError(w, http.StatusNotFound, "supplier not found")
		} else {
			writeServerError(w, "failed to get supplier", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, supplier)
}

// Create inserts a new supplier.
func (h *SupplierHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.CreateSupplierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	supplier, err := h.supplierService.Create(r.Context(), tenantID, req, actorID, clientIP(r))
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
		} else {
			writeServerError(w, "failed to create supplier", err)
		}
		return
	}
	writeJSON(w, http.StatusCreated, supplier)
}

// Update modifies an existing supplier.
func (h *SupplierHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	var req model.UpdateSupplierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	supplier, err := h.supplierService.Update(r.Context(), tenantID, supplierID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSupplierNotFound):
			writeError(w, http.StatusNotFound, "supplier not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeServerError(w, "failed to update supplier", err)
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, supplier)
}

// Delete removes a supplier by ID.
func (h *SupplierHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	err = h.supplierService.Delete(r.Context(), tenantID, supplierID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSupplierNotFound):
			writeError(w, http.StatusNotFound, "supplier not found")
		default:
			writeServerError(w, "failed to delete supplier", err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Sync triggers a product feed sync for a supplier.
// Responds immediately with 202 Accepted and runs the sync in the background
// to avoid connection timeouts on large feeds.
func (h *SupplierHandler) Sync(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	// Validate supplier exists and has a feed URL before accepting
	err = h.supplierService.ValidateSyncable(r.Context(), tenantID, supplierID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSupplierNotFound):
			writeError(w, http.StatusNotFound, "supplier not found")
		case errors.Is(err, service.ErrNoFeedURL):
			writeError(w, http.StatusBadRequest, "supplier has no feed URL configured")
		default:
			writeServerError(w, "failed to validate supplier", err)
		}
		return
	}

	// Run sync in background — detached from request context to avoid
	// "conn closed" errors when HTTP client disconnects during long syncs.
	asyncutil.SafeGo(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if syncErr := h.supplierService.SyncFeed(ctx, tenantID, supplierID); syncErr != nil {
			slog.Error("background supplier sync failed",
				"tenant_id", tenantID,
				"supplier_id", supplierID,
				"error", syncErr,
			)
		}
	})

	writeJSON(w, http.StatusAccepted, map[string]string{"message": "feed sync started"})
}

// ListProducts returns a paginated list of products for a supplier.
func (h *SupplierHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	filter := model.SupplierProductListFilter{
		SupplierID:       &supplierID,
		PaginationParams: pagination,
	}
	if ean := r.URL.Query().Get("ean"); ean != "" {
		filter.EAN = &ean
	}
	if linked := r.URL.Query().Get("linked"); linked != "" {
		val := linked == "true"
		filter.Linked = &val
	}
	if search := r.URL.Query().Get("search"); search != "" {
		filter.Search = &search
	}
	if category := r.URL.Query().Get("category"); category != "" {
		filter.SourceCategory = &category
	}

	resp, err := h.supplierService.ListProducts(r.Context(), tenantID, filter)
	if err != nil {
		writeServerError(w, "failed to list supplier products", err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// ListSourceCategories handles GET /v1/suppliers/{id}/products/categories — returns distinct source categories.
func (h *SupplierHandler) ListSourceCategories(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	categories, err := h.supplierService.ListSourceCategories(r.Context(), tenantID, supplierID)
	if err != nil {
		writeServerError(w, "failed to list source categories", err)
		return
	}
	writeJSON(w, http.StatusOK, categories)
}

// DeleteProduct handles DELETE /v1/suppliers/{id}/products/{spid} — deletes a single supplier product.
func (h *SupplierHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	spID, err := uuid.Parse(chi.URLParam(r, "spid"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier product ID")
		return
	}

	err = h.supplierService.DeleteSupplierProduct(r.Context(), tenantID, supplierID, spID, actorID, clientIP(r))
	if err != nil {
		if errors.Is(err, service.ErrSupplierProductNotFound) {
			writeError(w, http.StatusNotFound, "supplier product not found")
		} else {
			writeServerError(w, "failed to delete supplier product", err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// BulkDeleteProducts handles POST /v1/suppliers/{id}/products/bulk-delete — deletes multiple supplier products.
func (h *SupplierHandler) BulkDeleteProducts(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	var req model.BulkDeleteSupplierProductsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	deleted, err := h.supplierService.BulkDeleteSupplierProducts(r.Context(), tenantID, supplierID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSupplierNotFound):
			writeError(w, http.StatusNotFound, "supplier not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeServerError(w, "failed to delete supplier products", err)
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"deleted": deleted})
}

// UnlinkProduct handles POST /v1/suppliers/{id}/products/{spid}/unlink — removes link to OMS product.
func (h *SupplierHandler) UnlinkProduct(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	spID, err := uuid.Parse(chi.URLParam(r, "spid"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier product ID")
		return
	}

	err = h.supplierService.UnlinkProduct(r.Context(), tenantID, spID, actorID, clientIP(r))
	if err != nil {
		if errors.Is(err, service.ErrSupplierProductNotFound) {
			writeError(w, http.StatusNotFound, "supplier product not found")
		} else {
			writeServerError(w, "failed to unlink product", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "product unlinked"})
}

// LinkProduct links a supplier product to a catalog product.
func (h *SupplierHandler) LinkProduct(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	supplierProductID, err := uuid.Parse(chi.URLParam(r, "spid"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier product ID")
		return
	}

	var req model.LinkProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	err = h.supplierService.LinkProduct(r.Context(), tenantID, supplierProductID, req.ProductID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSupplierProductNotFound):
			writeError(w, http.StatusNotFound, "supplier product not found")
		default:
			writeServerError(w, "failed to link product", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "product linked"})
}

// ListCategoryMappings returns supplier-to-catalog category mappings.
func (h *SupplierHandler) ListCategoryMappings(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	mappings, err := h.supplierService.ListCategoryMappings(r.Context(), tenantID, supplierID)
	if err != nil {
		if errors.Is(err, service.ErrSupplierNotFound) {
			writeError(w, http.StatusNotFound, "supplier not found")
		} else {
			writeServerError(w, "failed to list category mappings", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, mappings)
}

// UpsertCategoryMapping creates or updates a supplier category mapping.
func (h *SupplierHandler) UpsertCategoryMapping(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	var req model.UpsertCategoryMappingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	mapping, err := h.supplierService.UpsertCategoryMapping(r.Context(), tenantID, supplierID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSupplierNotFound):
			writeError(w, http.StatusNotFound, "supplier not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeServerError(w, "failed to upsert category mapping", err)
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, mapping)
}

// ImportProducts handles POST /v1/suppliers/{id}/products/import — bulk creates OMS products from supplier products.
func (h *SupplierHandler) ImportProducts(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	var req model.ImportSupplierProductsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.supplierService.ImportProducts(r.Context(), tenantID, supplierID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSupplierNotFound):
			writeError(w, http.StatusNotFound, "supplier not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeServerError(w, "failed to import products", err)
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// DeleteCategoryMapping removes a supplier category mapping.
func (h *SupplierHandler) DeleteCategoryMapping(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	mappingID, err := uuid.Parse(chi.URLParam(r, "mid"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid mapping ID")
		return
	}

	err = h.supplierService.DeleteCategoryMapping(r.Context(), tenantID, supplierID, mappingID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSupplierNotFound):
			writeError(w, http.StatusNotFound, "supplier not found")
		default:
			writeServerError(w, "failed to delete category mapping", err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListAllegroMappings handles GET /v1/suppliers/{id}/allegro-mappings?category_id=xxx
func (h *SupplierHandler) ListAllegroMappings(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	categoryID := r.URL.Query().Get("category_id")
	if categoryID == "" {
		writeError(w, http.StatusBadRequest, "category_id query parameter is required")
		return
	}

	mappings, err := h.supplierService.ListAllegroMappings(r.Context(), tenantID, supplierID, categoryID)
	if err != nil {
		if errors.Is(err, service.ErrSupplierNotFound) {
			writeError(w, http.StatusNotFound, "supplier not found")
		} else {
			writeServerError(w, "failed to list allegro mappings", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, mappings)
}

// BulkUpsertAllegroMappings handles PUT /v1/suppliers/{id}/allegro-mappings
func (h *SupplierHandler) BulkUpsertAllegroMappings(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	var req model.BulkUpsertAllegroMappingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	mappings, err := h.supplierService.BulkUpsertAllegroMappings(r.Context(), tenantID, supplierID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSupplierNotFound):
			writeError(w, http.StatusNotFound, "supplier not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeServerError(w, "failed to upsert allegro mappings", err)
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, mappings)
}

// DeleteAllegroMapping handles DELETE /v1/suppliers/{id}/allegro-mappings/{mid}
func (h *SupplierHandler) DeleteAllegroMapping(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	mappingID, err := uuid.Parse(chi.URLParam(r, "mid"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid mapping ID")
		return
	}

	err = h.supplierService.DeleteAllegroMapping(r.Context(), tenantID, supplierID, mappingID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSupplierNotFound):
			writeError(w, http.StatusNotFound, "supplier not found")
		default:
			writeServerError(w, "failed to delete allegro mapping", err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListSupplierAttributes handles GET /v1/suppliers/{id}/attributes
func (h *SupplierHandler) ListSupplierAttributes(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	attrs, err := h.supplierService.ListSupplierAttributes(r.Context(), tenantID, supplierID)
	if err != nil {
		if errors.Is(err, service.ErrSupplierNotFound) {
			writeError(w, http.StatusNotFound, "supplier not found")
		} else {
			writeServerError(w, "failed to list supplier attributes", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, attrs)
}

// ListAllegroMappingCategories handles GET /v1/suppliers/{id}/allegro-mappings/categories
func (h *SupplierHandler) ListAllegroMappingCategories(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	categories, err := h.supplierService.ListAllegroMappingCategories(r.Context(), tenantID, supplierID)
	if err != nil {
		if errors.Is(err, service.ErrSupplierNotFound) {
			writeError(w, http.StatusNotFound, "supplier not found")
		} else {
			writeServerError(w, "failed to list allegro mapping categories", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, categories)
}

// SupplierLink handles GET /v1/products/{id}/supplier-link
func (h *SupplierHandler) SupplierLink(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	supplierID, err := h.supplierService.FindSupplierForProduct(r.Context(), tenantID, productID)
	if err != nil {
		writeServerError(w, "failed to find supplier for product", err)
		return
	}
	if supplierID == nil {
		writeJSON(w, http.StatusOK, map[string]any{"supplier_id": nil})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"supplier_id": supplierID.String()})
}

// ---------------------------------------------------------------------------
// BTP Wizard
// ---------------------------------------------------------------------------

// BTPWizardStartImport starts a BTP supplier import wizard.
func (h *SupplierHandler) BTPWizardStartImport(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.BTPWizardStartImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	supplier, err := h.supplierService.BTPWizardStartImport(r.Context(), tenantID, req, actorID, clientIP(r))
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
		} else {
			writeServerError(w, "failed to start import", err)
		}
		return
	}
	writeJSON(w, http.StatusCreated, supplier)
}

// BTPWizardImportProgress returns the current import progress for a BTP supplier.
func (h *SupplierHandler) BTPWizardImportProgress(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	resp, err := h.supplierService.BTPWizardGetImportProgress(r.Context(), tenantID, supplierID)
	if err != nil {
		if errors.Is(err, service.ErrSupplierNotFound) {
			writeError(w, http.StatusNotFound, "supplier not found")
		} else {
			writeServerError(w, "failed to get import progress", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// BTPWizardSetAPIKeys sets API keys for a BTP supplier integration.
func (h *SupplierHandler) BTPWizardSetAPIKeys(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	var req model.BTPWizardSetAPIKeysRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	supplier, err := h.supplierService.BTPWizardSetAPIKeys(r.Context(), tenantID, supplierID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case isValidationError(err):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrSupplierNotFound):
			writeError(w, http.StatusNotFound, "supplier not found")
		case errors.Is(err, service.ErrDuplicateProvider):
			writeError(w, http.StatusConflict, "BTP integration already exists for this tenant")
		default:
			writeServerError(w, "failed to set API keys", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, supplier)
}

// BTPWizardCompleteSyncSettings completes the sync settings step of the BTP wizard.
func (h *SupplierHandler) BTPWizardCompleteSyncSettings(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	var req model.BTPWizardSyncSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	supplier, err := h.supplierService.BTPWizardCompleteSyncSettings(r.Context(), tenantID, supplierID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case isValidationError(err):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrSupplierNotFound):
			writeError(w, http.StatusNotFound, "supplier not found")
		default:
			writeServerError(w, "failed to complete sync settings", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, supplier)
}

// ImportSingleProduct handles POST /v1/suppliers/{id}/products/{productId}/import —
// imports a single supplier product into the OMS catalog.
func (h *SupplierHandler) ImportSingleProduct(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	spID, err := uuid.Parse(chi.URLParam(r, "spid"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier product ID")
		return
	}

	product, err := h.supplierService.ImportSingleProduct(r.Context(), tenantID, supplierID, spID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSupplierNotFound):
			writeError(w, http.StatusNotFound, "supplier not found")
		case errors.Is(err, service.ErrSupplierProductNotFound):
			writeError(w, http.StatusNotFound, "supplier product not found")
		default:
			writeServerError(w, "failed to import product", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, product)
}

// ListAllSupplierProducts handles GET /v1/supplier-products — lists supplier products across all suppliers.
func (h *SupplierHandler) ListAllSupplierProducts(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	q := r.URL.Query()

	limit := 50
	offset := 0
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	if v := q.Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	params := model.SupplierProductListAllParams{
		Search:    q.Get("search"),
		Category:  q.Get("category"),
		Linked:    q.Get("linked"),
		SortBy:    q.Get("sort_by"),
		SortOrder: q.Get("sort_order"),
		Limit:     limit,
		Offset:    offset,
	}
	if sid := q.Get("supplier_id"); sid != "" {
		if id, err := uuid.Parse(sid); err == nil {
			params.SupplierID = &id
		}
	}

	resp, err := h.supplierService.ListAllProducts(r.Context(), tenantID, params)
	if err != nil {
		writeServerError(w, "failed to list supplier products", err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
