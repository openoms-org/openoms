package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

type SupplierHandler struct {
	supplierService *service.SupplierService
}

func NewSupplierHandler(supplierService *service.SupplierService) *SupplierHandler {
	return &SupplierHandler{supplierService: supplierService}
}

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
		writeError(w, http.StatusInternalServerError, "failed to list suppliers")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

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
			writeError(w, http.StatusInternalServerError, "failed to get supplier")
		}
		return
	}
	writeJSON(w, http.StatusOK, supplier)
}

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
			writeError(w, http.StatusInternalServerError, "failed to create supplier")
		}
		return
	}
	writeJSON(w, http.StatusCreated, supplier)
}

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
				writeError(w, http.StatusInternalServerError, "failed to update supplier")
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, supplier)
}

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
			writeError(w, http.StatusInternalServerError, "failed to delete supplier")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SupplierHandler) Sync(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	err = h.supplierService.SyncFeed(r.Context(), tenantID, supplierID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSupplierNotFound):
			writeError(w, http.StatusNotFound, "supplier not found")
		case errors.Is(err, service.ErrNoFeedURL):
			writeError(w, http.StatusBadRequest, "supplier has no feed URL configured")
		default:
			writeError(w, http.StatusInternalServerError, "failed to sync feed")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "feed synced successfully"})
}

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
		writeError(w, http.StatusInternalServerError, "failed to list supplier products")
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
		writeError(w, http.StatusInternalServerError, "failed to list source categories")
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
			writeError(w, http.StatusInternalServerError, "failed to delete supplier product")
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
				writeError(w, http.StatusInternalServerError, "failed to delete supplier products")
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
			writeError(w, http.StatusInternalServerError, "failed to unlink product")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "product unlinked"})
}

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
			writeError(w, http.StatusInternalServerError, "failed to link product")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "product linked"})
}

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
			writeError(w, http.StatusInternalServerError, "failed to list category mappings")
		}
		return
	}
	writeJSON(w, http.StatusOK, mappings)
}

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
				writeError(w, http.StatusInternalServerError, "failed to upsert category mapping")
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
				writeError(w, http.StatusInternalServerError, "failed to import products")
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

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
			writeError(w, http.StatusInternalServerError, "failed to delete category mapping")
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
			writeError(w, http.StatusInternalServerError, "failed to list allegro mappings")
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
				writeError(w, http.StatusInternalServerError, "failed to upsert allegro mappings")
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
			writeError(w, http.StatusInternalServerError, "failed to delete allegro mapping")
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
			writeError(w, http.StatusInternalServerError, "failed to list supplier attributes")
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
			writeError(w, http.StatusInternalServerError, "failed to list allegro mapping categories")
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
		writeError(w, http.StatusInternalServerError, "failed to find supplier for product")
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
			writeError(w, http.StatusInternalServerError, "failed to start import")
		}
		return
	}
	writeJSON(w, http.StatusCreated, supplier)
}

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
			writeError(w, http.StatusInternalServerError, "failed to get import progress")
		}
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

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
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
		} else if errors.Is(err, service.ErrSupplierNotFound) {
			writeError(w, http.StatusNotFound, "supplier not found")
		} else if errors.Is(err, service.ErrDuplicateProvider) {
			writeError(w, http.StatusConflict, "BTP integration already exists for this tenant")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to set API keys")
		}
		return
	}
	writeJSON(w, http.StatusOK, supplier)
}

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
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
		} else if errors.Is(err, service.ErrSupplierNotFound) {
			writeError(w, http.StatusNotFound, "supplier not found")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to complete sync settings")
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
			writeError(w, http.StatusInternalServerError, "failed to import product")
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
		writeError(w, http.StatusInternalServerError, "failed to list supplier products")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
