package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/netutil"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
	"github.com/openoms-org/openoms/apps/api-server/internal/storage"
)

// BGRemovalHandler handles background removal endpoints.
type BGRemovalHandler struct {
	bgService   *service.BGRemovalService
	storage     storage.ObjectStorage
	productRepo repository.ProductRepo
	pool        *pgxpool.Pool
	maxSize     int64
}

// NewBGRemovalHandler creates a new background removal handler.
func NewBGRemovalHandler(
	bgService *service.BGRemovalService,
	store storage.ObjectStorage,
	productRepo repository.ProductRepo,
	pool *pgxpool.Pool,
	maxSize int64,
) *BGRemovalHandler {
	return &BGRemovalHandler{
		bgService:   bgService,
		storage:     store,
		productRepo: productRepo,
		pool:        pool,
		maxSize:     maxSize,
	}
}

var bgAllowedMimeTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// RemoveBackground handles POST /v1/images/remove-background
// Accepts a multipart file upload, removes the background, saves the result and returns the URL.
func (h *BGRemovalHandler) RemoveBackground(w http.ResponseWriter, r *http.Request) {
	if !h.bgService.IsConfigured() {
		writeError(w, http.StatusUnprocessableEntity, "Background removal not configured. Set REMOVEBG_API_KEY.")
		return
	}

	tenantID := middleware.TenantIDFromContext(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, h.maxSize)
	if err := r.ParseMultipartForm(h.maxSize); err != nil {
		writeError(w, http.StatusBadRequest, "file too large or invalid form data")
		return
	}
	defer func() {
		if err := r.MultipartForm.RemoveAll(); err != nil {
			slog.Warn("bg_removal: failed to clean up multipart form", "error", err)
		}
	}()

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing 'file' field")
		return
	}
	defer func() { _ = file.Close() }()

	imageData, err := io.ReadAll(file)
	if err != nil {
		writeServerError(w, "failed to read file", err)
		return
	}

	// Validate content type
	detectLen := min(len(imageData), 512)
	contentType := http.DetectContentType(imageData[:detectLen])
	if _, ok := bgAllowedMimeTypes[contentType]; !ok {
		writeError(w, http.StatusBadRequest, "unsupported file type — allowed: JPEG, PNG, WEBP")
		return
	}

	// Call remove.bg API
	resultBytes, resultContentType, err := h.bgService.RemoveBackground(r.Context(), imageData, header.Filename)
	if err != nil {
		writeServerError(w, "background removal failed", err)
		return
	}

	// Save processed image to storage
	ext := ".png"
	if resultContentType == "image/webp" {
		ext = ".webp"
	}
	filename := uuid.New().String() + "-nobg" + ext
	key := fmt.Sprintf("%s/%s", tenantID.String(), filename)

	url, err := h.storage.Upload(r.Context(), key, bytes.NewReader(resultBytes), resultContentType)
	if err != nil {
		writeServerError(w, "failed to save processed image", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"url":          url,
		"content_type": resultContentType,
	})
}

// RemoveProductImageBackground handles POST /v1/products/{id}/images/{index}/remove-background
// Removes background from an existing product image at the given index and updates the product.
// Use index -1 to process the main image_url, or 0+ for the images[] array.
func (h *BGRemovalHandler) RemoveProductImageBackground(w http.ResponseWriter, r *http.Request) {
	if !h.bgService.IsConfigured() {
		writeError(w, http.StatusUnprocessableEntity, "Background removal not configured. Set REMOVEBG_API_KEY.")
		return
	}

	tenantID := middleware.TenantIDFromContext(r.Context())

	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	indexStr := chi.URLParam(r, "index")
	index, err := strconv.Atoi(indexStr)
	if err != nil || index < -1 {
		writeError(w, http.StatusBadRequest, "invalid image index (use -1 for image_url or 0+ for images[])")
		return
	}

	// Fetch the product
	type productImage struct {
		URL      string `json:"url"`
		Alt      string `json:"alt,omitempty"`
		Position int    `json:"position,omitempty"`
	}

	var imageURL string
	var images []productImage
	var productName string

	err = database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		p, err := h.productRepo.FindByID(r.Context(), tx, productID)
		if err != nil {
			return err
		}
		if p == nil {
			return service.ErrProductNotFound
		}
		productName = p.Name

		if p.ImageURL != nil {
			imageURL = *p.ImageURL
		}

		if len(p.Images) > 0 {
			if err := json.Unmarshal(p.Images, &images); err != nil {
				return fmt.Errorf("unmarshal images: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		if err == service.ErrProductNotFound {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeServerError(w, "failed to fetch product", err)
		return
	}

	// Determine which image URL to process
	var targetURL string
	if index == -1 {
		if imageURL == "" {
			writeError(w, http.StatusBadRequest, "product has no main image")
			return
		}
		targetURL = imageURL
	} else {
		if index >= len(images) {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("index %d out of range (product has %d images)", index, len(images)))
			return
		}
		targetURL = images[index].URL
	}

	if targetURL == "" {
		writeError(w, http.StatusBadRequest, "no image URL to process")
		return
	}

	// Download the image
	imageData, contentType, err := h.downloadImage(r.Context(), targetURL)
	if err != nil {
		writeServerError(w, "failed to download image", err)
		return
	}

	if _, ok := bgAllowedMimeTypes[contentType]; !ok {
		writeError(w, http.StatusBadRequest, "unsupported image file type")
		return
	}

	// Remove background
	resultBytes, resultContentType, err := h.bgService.RemoveBackground(r.Context(), imageData, "image.png")
	if err != nil {
		writeServerError(w, "background removal failed", err)
		return
	}

	// Save processed image to storage
	ext := ".png"
	if resultContentType == "image/webp" {
		ext = ".webp"
	}
	filename := uuid.New().String() + "-nobg" + ext
	key := fmt.Sprintf("%s/%s", tenantID.String(), filename)

	newURL, err := h.storage.Upload(r.Context(), key, bytes.NewReader(resultBytes), resultContentType)
	if err != nil {
		writeServerError(w, "failed to save processed image", err)
		return
	}

	// Update the product. Re-fetch the images array inside the same transaction
	// so we don't overwrite concurrent edits made between the initial read and
	// now (TOCTOU race: add/remove/reorder would be lost otherwise).
	err = database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		if index == -1 {
			return h.productRepo.Update(r.Context(), tx, productID, model.UpdateProductRequest{
				ImageURL: &newURL,
			})
		}

		p, err := h.productRepo.FindByID(r.Context(), tx, productID)
		if err != nil {
			return err
		}
		if p == nil {
			return service.ErrProductNotFound
		}

		var current []productImage
		if len(p.Images) > 0 {
			if err := json.Unmarshal(p.Images, &current); err != nil {
				return fmt.Errorf("unmarshal images: %w", err)
			}
		}
		if index >= len(current) {
			return fmt.Errorf("image index %d no longer valid (product has %d images)", index, len(current))
		}

		current[index].URL = newURL
		if current[index].Alt == "" {
			current[index].Alt = productName + " (background removed)"
		}
		imagesJSON, err := json.Marshal(current)
		if err != nil {
			return fmt.Errorf("marshal images: %w", err)
		}
		raw := json.RawMessage(imagesJSON)
		return h.productRepo.Update(r.Context(), tx, productID, model.UpdateProductRequest{
			Images: &raw,
		})
	})
	if err != nil {
		writeServerError(w, "failed to update product", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"url":          newURL,
		"content_type": resultContentType,
		"message":      "Background removed successfully",
	})
}

// downloadImage fetches image bytes from a URL.
// Uses SafeHTTPClient to prevent SSRF attacks against internal services.
func (h *BGRemovalHandler) downloadImage(ctx context.Context, imageURL string) ([]byte, string, error) {
	parsedURL, err := url.Parse(imageURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return nil, "", fmt.Errorf("invalid URL scheme: only http and https are allowed")
	}

	client := netutil.SafeHTTPClient(30 * time.Second)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("download image: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, h.maxSize))
	if err != nil {
		return nil, "", fmt.Errorf("read image: %w", err)
	}

	detectLen := min(len(data), 512)
	contentType := http.DetectContentType(data[:detectLen])
	return data, contentType, nil
}

// Status handles GET /v1/images/remove-background/status
// Returns whether background removal is configured.
func (h *BGRemovalHandler) Status(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{
		"configured": h.bgService.IsConfigured(),
	})
}
