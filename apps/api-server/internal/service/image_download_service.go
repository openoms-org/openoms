package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/netutil"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/storage"
)

// ImageDownloadService downloads external product images and re-uploads them to object storage.
type ImageDownloadService struct {
	productRepo repository.ProductRepo
	pool        *pgxpool.Pool
	storage     storage.ObjectStorage
	httpClient  *http.Client
}

// NewImageDownloadService creates a new ImageDownloadService.
func NewImageDownloadService(productRepo repository.ProductRepo, pool *pgxpool.Pool, st storage.ObjectStorage) *ImageDownloadService {
	return &ImageDownloadService{
		productRepo: productRepo,
		pool:        pool,
		storage:     st,
		httpClient: netutil.SafeHTTPClient(30 * time.Second),
	}
}

// ImageDownloadResult contains the results of the re-download operation.
type ImageDownloadResult struct {
	Total      int `json:"total"`
	Downloaded int `json:"downloaded"`
	Failed     int `json:"failed"`
	Skipped    int `json:"skipped"`
}

// RedownloadImages iterates over all products for a tenant, downloads external images,
// uploads them to object storage, and updates the product image URLs.
func (s *ImageDownloadService) RedownloadImages(ctx context.Context, tenantID uuid.UUID) (*ImageDownloadResult, error) {
	if s.storage == nil {
		return nil, fmt.Errorf("object storage not configured")
	}

	result := &ImageDownloadResult{}
	const pageSize = 100
	offset := 0

	for {
		var products []model.Product
		var total int

		err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
			var err error
			filter := model.ProductListFilter{
				PaginationParams: model.PaginationParams{Limit: pageSize, Offset: offset},
			}
			products, total, err = s.productRepo.List(ctx, tx, filter)
			return err
		})
		if err != nil {
			return nil, fmt.Errorf("listing products (offset=%d): %w", offset, err)
		}

		if len(products) == 0 {
			break
		}

		for _, p := range products {
			downloaded, failed, skipped, err := s.processProduct(ctx, tenantID, p)
			if err != nil {
				slog.Error("failed to process product images",
					"product_id", p.ID,
					"error", err,
				)
			}
			result.Downloaded += downloaded
			result.Failed += failed
			result.Skipped += skipped
			result.Total += downloaded + failed + skipped
		}

		offset += pageSize
		if offset >= total {
			break
		}
	}

	return result, nil
}

// processProduct handles image re-download for a single product.
func (s *ImageDownloadService) processProduct(ctx context.Context, tenantID uuid.UUID, p model.Product) (downloaded, failed, skipped int, err error) {
	if len(p.Images) == 0 || string(p.Images) == "null" || string(p.Images) == "[]" {
		return 0, 0, 0, nil
	}

	var imageURLs []string
	if err := json.Unmarshal(p.Images, &imageURLs); err != nil {
		return 0, 0, 0, fmt.Errorf("parsing images JSON for product %s: %w", p.ID, err)
	}

	if len(imageURLs) == 0 {
		return 0, 0, 0, nil
	}

	changed := false
	for i, imgURL := range imageURLs {
		if isLocalURL(imgURL) {
			skipped++
			continue
		}

		newURL, dlErr := s.downloadAndUpload(ctx, tenantID, p.ID, imgURL)
		if dlErr != nil {
			slog.Warn("failed to download image",
				"product_id", p.ID,
				"url", imgURL,
				"error", dlErr,
			)
			failed++
			continue
		}

		imageURLs[i] = newURL
		changed = true
		downloaded++
	}

	if !changed {
		return downloaded, failed, skipped, nil
	}

	// Update the product images in DB
	newImagesJSON, err := json.Marshal(imageURLs)
	if err != nil {
		return downloaded, failed, skipped, fmt.Errorf("marshaling new images: %w", err)
	}
	imagesRaw := json.RawMessage(newImagesJSON)

	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		return s.productRepo.Update(ctx, tx, p.ID, model.UpdateProductRequest{
			Images: &imagesRaw,
		})
	})
	if err != nil {
		return downloaded, failed, skipped, fmt.Errorf("updating product %s images: %w", p.ID, err)
	}

	return downloaded, failed, skipped, nil
}

// downloadAndUpload downloads an image from an external URL and uploads it to storage.
func (s *ImageDownloadService) downloadAndUpload(ctx context.Context, tenantID, productID uuid.UUID, imgURL string) (string, error) {
	body, contentType, err := s.downloadImage(ctx, imgURL)
	if err != nil {
		return "", fmt.Errorf("downloading %s: %w", imgURL, err)
	}
	defer body.Close()

	filename := filenameFromURL(imgURL)
	key := fmt.Sprintf("%s/products/%s/%s", tenantID, productID, filename)

	newURL, err := s.storage.Upload(ctx, key, body, contentType)
	if err != nil {
		return "", fmt.Errorf("uploading to storage: %w", err)
	}

	return newURL, nil
}

// downloadImage fetches an image from the given URL with a per-request timeout.
// The caller is responsible for closing the returned ReadCloser.
func (s *ImageDownloadService) downloadImage(ctx context.Context, rawURL string) (io.ReadCloser, string, error) {
	dlCtx, cancel := context.WithTimeout(ctx, 10*time.Second)

	req, err := http.NewRequestWithContext(dlCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		cancel()
		return nil, "", fmt.Errorf("creating request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		cancel()
		return nil, "", err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		cancel()
		return nil, "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}

	// Limit body to 10MB. Wrap in a cancelCloser so the context is
	// cancelled when the caller finishes reading and closes the body.
	limited := http.MaxBytesReader(nil, resp.Body, 10<<20)
	return &cancelCloser{ReadCloser: limited, cancel: cancel}, contentType, nil
}

// cancelCloser wraps an io.ReadCloser and calls a cancel function on Close.
type cancelCloser struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (c *cancelCloser) Close() error {
	err := c.ReadCloser.Close()
	c.cancel()
	return err
}

// filenameFromURL extracts a filename from a URL, falling back to a UUID-based name.
func filenameFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return uuid.New().String() + ".jpg"
	}
	base := path.Base(u.Path)
	if base == "" || base == "." || base == "/" {
		return uuid.New().String() + ".jpg"
	}
	return base
}

// isLocalURL checks whether the URL already points to our storage (local or S3).
// We consider a URL "local" if it contains "/uploads/" (local storage pattern)
// or does not start with "http" (relative path).
func isLocalURL(rawURL string) bool {
	if rawURL == "" {
		return true
	}
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		return true
	}
	// URLs from our own storage contain "/uploads/" (local) or are from our S3 bucket
	if strings.Contains(rawURL, "/uploads/") {
		return true
	}
	return false
}
