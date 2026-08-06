package handler

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// ProductHandler handles HTTP requests for product management.
type ProductHandler struct {
	productService         *service.ProductService
	productImportService   *service.ProductImportService
	blProductImportService *service.BaseLinkerProductImportService
	categorySvc            *service.ProductCategoryService
	imageDownloadService   *service.ImageDownloadService
}

// NewProductHandler creates a new ProductHandler.
func NewProductHandler(
	productService *service.ProductService,
	productImportService *service.ProductImportService,
	blProductImportService *service.BaseLinkerProductImportService,
	categorySvc *service.ProductCategoryService,
	imageDownloadService *service.ImageDownloadService,
) *ProductHandler {
	return &ProductHandler{
		productService:         productService,
		productImportService:   productImportService,
		blProductImportService: blProductImportService,
		categorySvc:            categorySvc,
		imageDownloadService:   imageDownloadService,
	}
}

// List returns a paginated list of products.
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.ProductListFilter{
		PaginationParams: pagination,
	}
	if name := r.URL.Query().Get("name"); name != "" {
		filter.Name = &name
	}
	if sku := r.URL.Query().Get("sku"); sku != "" {
		filter.SKU = &sku
	}
	if t := r.URL.Query().Get("tag"); t != "" {
		filter.Tag = &t
	}
	if c := r.URL.Query().Get("category"); c != "" {
		filter.Category = &c
	}
	if s := r.URL.Query().Get("search"); s != "" {
		filter.Search = &s
	}
	if cid := r.URL.Query().Get("category_id"); cid != "" {
		id, err := uuid.Parse(cid)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid category_id")
			return
		}
		// Resolve category + all descendants for hierarchical filtering
		if h.categorySvc != nil {
			ids, err := h.categorySvc.GetDescendantIDs(r.Context(), tenantID, id)
			if err != nil {
				slog.Error("failed to resolve category descendants", "error", err)
				filter.CategoryIDs = []uuid.UUID{id}
			} else {
				filter.CategoryIDs = ids
			}
		} else {
			filter.CategoryIDs = []uuid.UUID{id}
		}
	}
	if sid := r.URL.Query().Get("supplier_id"); sid != "" {
		id, err := uuid.Parse(sid)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid supplier_id")
			return
		}
		filter.SupplierID = &id
	}
	if src := r.URL.Query().Get("source"); src != "" {
		filter.Source = &src
	}
	if mp := r.URL.Query().Get("marketplace"); mp != "" {
		filter.Marketplace = &mp
	}

	products, total, err := h.productService.List(r.Context(), tenantID, filter)
	if err != nil {
		writeServerError(w, "failed to list products", err)
		return
	}
	if products == nil {
		products = []model.Product{}
	}
	writeJSON(w, http.StatusOK, model.ListResponse[model.Product]{
		Items:  products,
		Total:  total,
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	})
}

// Get returns a single product by ID.
func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	product, err := h.productService.Get(r.Context(), tenantID, productID)
	if err != nil {
		if errors.Is(err, service.ErrProductNotFound) {
			writeError(w, http.StatusNotFound, "product not found")
		} else {
			writeServerError(w, "failed to get product", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, product)
}

// Create inserts a new product.
func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	product, err := h.productService.Create(r.Context(), tenantID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDuplicateSKU):
			writeError(w, http.StatusConflict, "product with this SKU already exists")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeServerError(w, "failed to create product", err)
			}
		}
		return
	}
	writeJSON(w, http.StatusCreated, product)
}

// Update modifies an existing product.
func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	var req model.UpdateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	product, err := h.productService.Update(r.Context(), tenantID, productID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrProductNotFound):
			writeError(w, http.StatusNotFound, "product not found")
		case errors.Is(err, service.ErrDuplicateSKU):
			writeError(w, http.StatusConflict, "product with this SKU already exists")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeServerError(w, "failed to update product", err)
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, product)
}

// Delete removes a product by ID.
func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	err = h.productService.Delete(r.Context(), tenantID, productID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrProductNotFound):
			writeError(w, http.StatusNotFound, "product not found")
		default:
			writeServerError(w, "failed to delete product", err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// productExportHeader is the column list of the product CSV export. The names are a
// customer-facing contract: ProductImportService keys off them on re-import, so they
// must not be renamed even when the underlying source of a value changes.
var productExportHeader = []string{
	"id", "name", "sku", "ean", "price", "stock_quantity",
	"category", "tags", "weight", "width", "height", "length",
	"short_description", "status",
}

// productExportRow renders a single product as a CSV row matching productExportHeader.
// The stock column carries AvailableStock (canonical, warehouse_stock derived) rather
// than the legacy products.stock_quantity column, which is never decremented on ship.
func productExportRow(p model.Product) []string {
	sku := ""
	if p.SKU != nil {
		sku = *p.SKU
	}
	ean := ""
	if p.EAN != nil {
		ean = *p.EAN
	}
	category := ""
	if p.Category != nil {
		category = *p.Category
	}
	var tags strings.Builder
	for i, t := range p.Tags {
		if i > 0 {
			tags.WriteString(",")
		}
		tags.WriteString(t)
	}
	weight := ""
	if p.Weight != nil {
		weight = fmt.Sprintf("%.2f", *p.Weight)
	}
	width := ""
	if p.Width != nil {
		width = fmt.Sprintf("%.2f", *p.Width)
	}
	height := ""
	if p.Height != nil {
		height = fmt.Sprintf("%.2f", *p.Height)
	}
	depth := ""
	if p.Depth != nil {
		depth = fmt.Sprintf("%.2f", *p.Depth)
	}

	return []string{
		p.ID.String(),
		p.Name,
		sku,
		ean,
		fmt.Sprintf("%.2f", p.Price),
		fmt.Sprintf("%d", p.AvailableStock),
		category,
		tags.String(),
		weight,
		width,
		height,
		depth,
		p.DescriptionShort,
		"active",
	}
}

// ExportCSV exports products as a CSV file.
func (h *ProductHandler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	filter := model.ProductListFilter{}
	if name := r.URL.Query().Get("name"); name != "" {
		filter.Name = &name
	}
	if c := r.URL.Query().Get("category"); c != "" {
		filter.Category = &c
	}
	if s := r.URL.Query().Get("sku"); s != "" {
		filter.SKU = &s
	}

	filename := fmt.Sprintf("products_%s.csv", time.Now().Format("2006-01-02"))
	writeCSVHeaders(w, filename)

	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writer.Write(productExportHeader); err != nil {
		slog.Error("csv export: failed to write header", "error", err)
		return
	}

	const batchSize = 500
	const maxRows = 50000
	offset := 0
	totalRows := 0

	for {
		filter.PaginationParams = model.PaginationParams{Limit: batchSize, Offset: offset}
		products, _, err := h.productService.List(r.Context(), tenantID, filter)
		if err != nil {
			slog.Error("csv export failed", "error", err, "offset", offset)
			break
		}

		if len(products) == 0 {
			break
		}

		for _, p := range products {
			if err := writer.Write(productExportRow(p)); err != nil {
				slog.Error("csv export: failed to write row", "error", err, "product_id", p.ID)
				return
			}
		}

		totalRows += len(products)
		if totalRows >= maxRows {
			break
		}
		offset += batchSize
	}
}

// ImportPreview handles POST /v1/products/import/preview
func (h *ProductHandler) ImportPreview(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "file too large or invalid multipart form")
		return
	}
	defer func() {
		if err := r.MultipartForm.RemoveAll(); err != nil {
			slog.Warn("product: failed to clean up multipart form", "error", err)
		}
	}()

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer func() { _ = file.Close() }()

	tenantID := middleware.TenantIDFromContext(r.Context())
	preview, err := h.productImportService.PreviewCSV(r.Context(), tenantID, file)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, preview)
}

// ImportCSV handles POST /v1/products/import
func (h *ProductHandler) ImportCSV(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	userID := middleware.UserIDFromContext(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "file too large or invalid multipart form")
		return
	}
	defer func() {
		if err := r.MultipartForm.RemoveAll(); err != nil {
			slog.Warn("product: failed to clean up multipart form", "error", err)
		}
	}()

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer func() { _ = file.Close() }()

	result, err := h.productImportService.ImportCSV(r.Context(), tenantID, file, userID, clientIP(r))
	if err != nil {
		writeServerError(w, "failed to export products", err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// BLImportPreview handles POST /v1/products/import/baselinker/preview
func (h *ProductHandler) BLImportPreview(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "file too large or invalid multipart form")
		return
	}
	defer func() { _ = r.MultipartForm.RemoveAll() }()

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer func() { _ = file.Close() }()

	tenantID := middleware.TenantIDFromContext(r.Context())
	preview, err := h.blProductImportService.PreviewCSV(r.Context(), tenantID, file)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

// BLImportCSV handles POST /v1/products/import/baselinker
func (h *ProductHandler) BLImportCSV(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	userID := middleware.UserIDFromContext(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "file too large or invalid multipart form")
		return
	}
	defer func() { _ = r.MultipartForm.RemoveAll() }()

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer func() { _ = file.Close() }()

	result, err := h.blProductImportService.ImportCSV(r.Context(), tenantID, file, userID, clientIP(r))
	if err != nil {
		writeServerError(w, "failed to import BaseLinker products", err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// RedownloadImages downloads external product images and re-uploads them to storage.
func (h *ProductHandler) RedownloadImages(w http.ResponseWriter, r *http.Request) {
	if h.imageDownloadService == nil {
		writeError(w, http.StatusNotImplemented, "image storage not configured")
		return
	}

	tenantID := middleware.TenantIDFromContext(r.Context())
	result, err := h.imageDownloadService.RedownloadImages(r.Context(), tenantID)
	if err != nil {
		writeServerError(w, "failed to redownload images", err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
