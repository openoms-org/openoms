package handler

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

type ProductHandler struct {
	productService       *service.ProductService
	productImportService *service.ProductImportService
}

func NewProductHandler(productService *service.ProductService, productImportService *service.ProductImportService) *ProductHandler {
	return &ProductHandler{
		productService:       productService,
		productImportService: productImportService,
	}
}

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

	products, total, err := h.productService.List(r.Context(), tenantID, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list products")
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
			writeError(w, http.StatusInternalServerError, "failed to get product")
		}
		return
	}
	writeJSON(w, http.StatusOK, product)
}

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
				writeError(w, http.StatusInternalServerError, "failed to create product")
			}
		}
		return
	}
	writeJSON(w, http.StatusCreated, product)
}

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
				writeError(w, http.StatusInternalServerError, "failed to update product")
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, product)
}

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
			writeError(w, http.StatusInternalServerError, "failed to delete product")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	// BOM for Excel UTF-8 compatibility
	w.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(w)
	defer writer.Flush()

	header := []string{
		"id", "name", "sku", "ean", "price", "stock_quantity",
		"category", "tags", "weight", "width", "height", "length",
		"short_description", "status",
	}
	if err := writer.Write(header); err != nil {
		slog.Error("csv export: failed to write header", "error", err)
		return
	}

	const batchSize = 500
	offset := 0

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
			tags := ""
			if len(p.Tags) > 0 {
				for i, t := range p.Tags {
					if i > 0 {
						tags += ","
					}
					tags += t
				}
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
			status := "active"

			row := []string{
				p.ID.String(),
				p.Name,
				sku,
				ean,
				fmt.Sprintf("%.2f", p.Price),
				fmt.Sprintf("%d", p.StockQuantity),
				category,
				tags,
				weight,
				width,
				height,
				depth,
				p.DescriptionShort,
				status,
			}
			if err := writer.Write(row); err != nil {
				slog.Error("csv export: failed to write row", "error", err, "product_id", p.ID)
				return
			}
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
	defer r.MultipartForm.RemoveAll()

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close()

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
	defer r.MultipartForm.RemoveAll()

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close()

	result, err := h.productImportService.ImportCSV(r.Context(), tenantID, file, userID, clientIP(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}
