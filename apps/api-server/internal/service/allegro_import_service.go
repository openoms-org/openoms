package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/asyncutil"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	allegroIntegration "github.com/openoms-org/openoms/apps/api-server/internal/integration/allegro"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"
)

// maxImportDetails caps the number of per-offer details returned in the import result
// to avoid unbounded memory growth. Counters are always incremented for all offers.
const maxImportDetails = 100

// AllegroImportService handles importing Allegro seller offers into OpenOMS
// as Product + ProductListing records.
type AllegroImportService struct {
	integrationService *IntegrationService
	productRepo        repository.ProductRepo
	listingRepo        repository.ProductListingRepo
	categoryService    *ProductCategoryService
	pool               *pgxpool.Pool
	stockSyncService   *StockSyncService
	logger             *slog.Logger
	// importLocks records tenants with an in-flight import. Entries are deleted
	// when the import finishes, so the map only grows to the number of concurrent
	// imports (not the total number of tenants that ever imported).
	importLocks sync.Map // map[uuid.UUID]struct{}
}

// SetStockSyncService sets the stock sync service for propagating stock changes after import.
func (s *AllegroImportService) SetStockSyncService(svc *StockSyncService) {
	s.stockSyncService = svc
}

// NewAllegroImportService creates a new AllegroImportService.
func NewAllegroImportService(
	integrationService *IntegrationService,
	productRepo repository.ProductRepo,
	listingRepo repository.ProductListingRepo,
	categoryService *ProductCategoryService,
	pool *pgxpool.Pool,
) *AllegroImportService {
	return &AllegroImportService{
		integrationService: integrationService,
		productRepo:        productRepo,
		listingRepo:        listingRepo,
		categoryService:    categoryService,
		pool:               pool,
		logger:             slog.Default().With("component", "allegro_import"),
	}
}

// ImportOffers fetches all seller offers from Allegro and creates/links Product + ProductListing
// records in OpenOMS. Offers that already have a listing are skipped. Existing products are
// matched by SKU (External.ID). If no match is found, a new product is created.
//
// To avoid N+1 API calls, the full offer (Get) is only fetched when the listing does not
// already exist — re-imports where most offers are already linked skip the Get entirely.
func (s *AllegroImportService) ImportOffers(ctx context.Context, tenantID uuid.UUID) (*model.AllegroImportResult, error) {
	// Per-tenant concurrency guard: only one import at a time per tenant.
	// LoadOrStore is atomic — a second caller sees loaded=true and is rejected.
	if _, busy := s.importLocks.LoadOrStore(tenantID, struct{}{}); busy {
		return nil, fmt.Errorf("import already in progress for this tenant")
	}
	defer s.importLocks.Delete(tenantID)

	// Build Allegro provider from encrypted credentials.
	credJSON, integration, err := s.integrationService.GetDecryptedCredentialsByProvider(ctx, tenantID, "allegro")
	if err != nil {
		return nil, fmt.Errorf("get allegro credentials: %w", err)
	}
	provider, err := allegroIntegration.NewProvider(json.RawMessage(credJSON), nil)
	if err != nil {
		return nil, fmt.Errorf("create allegro provider: %w", err)
	}
	defer provider.Close()

	integrationID := integration.ID
	client := provider.SDKClient()

	// Fetch all seller offer summaries (auto-paginated).
	summaries, err := client.Offers.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list allegro offers: %w", err)
	}

	result := &model.AllegroImportResult{
		TotalOffers: len(summaries),
	}

	s.logger.Info("starting offer import",
		"tenant_id", tenantID,
		"total_offers", len(summaries),
	)

	for _, summary := range summaries {
		// Fix 4: Bail out early if the context has been cancelled.
		if err := ctx.Err(); err != nil {
			return result, err
		}

		// Fix 1: Check if listing already exists BEFORE calling Get().
		// This avoids N+1 API calls on re-imports where most offers are already linked.
		var existingSkipped bool
		var existingProductID string
		skipErr := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
			existing, err := s.listingRepo.FindByExternalIDAndIntegration(ctx, tx, summary.ID, integrationID)
			if err != nil {
				return fmt.Errorf("check existing listing: %w", err)
			}
			if existing != nil {
				existingSkipped = true
				existingProductID = existing.ProductID.String()
			}
			return nil
		})
		if skipErr != nil {
			s.logger.Error("failed to check existing listing",
				"tenant_id", tenantID,
				"offer_id", summary.ID,
				"error", skipErr,
			)
			result.Errors++
			if len(result.Details) < maxImportDetails {
				result.Details = append(result.Details, model.AllegroImportDetail{
					OfferID:   summary.ID,
					OfferName: summary.Name,
					Action:    "error",
					Error:     fmt.Sprintf("check listing: %v", skipErr),
				})
			}
			continue
		}
		if existingSkipped {
			result.Skipped++
			if len(result.Details) < maxImportDetails {
				result.Details = append(result.Details, model.AllegroImportDetail{
					OfferID:   summary.ID,
					OfferName: summary.Name,
					Action:    "skipped",
					ProductID: existingProductID,
				})
			}
			continue
		}

		// Listing does not exist — fetch full offer for SKU matching + product creation.
		offer, err := client.Offers.Get(ctx, summary.ID)
		if err != nil {
			s.logger.Error("failed to fetch offer details",
				"tenant_id", tenantID,
				"offer_id", summary.ID,
				"error", err,
			)
			result.Errors++
			if len(result.Details) < maxImportDetails {
				result.Details = append(result.Details, model.AllegroImportDetail{
					OfferID:   summary.ID,
					OfferName: summary.Name,
					Action:    "error",
					Error:     fmt.Sprintf("fetch offer: %v", err),
				})
			}
			continue
		}

		detail := s.processOffer(ctx, tenantID, integrationID, offer, client)
		switch detail.Action {
		case "created":
			result.Created++
			// Trigger stock sync for newly created products with stock
			if s.stockSyncService != nil && detail.ProductID != "" {
				if pid, err := uuid.Parse(detail.ProductID); err == nil && offer.Stock != nil && offer.Stock.Available > 0 {
					asyncutil.SafeGo(func() {
						s.stockSyncService.OnStockChange(context.Background(), tenantID, pid, "allegro_import", 0, offer.Stock.Available)
					})
				}
			}
		case "linked":
			result.Linked++
		case "skipped":
			result.Skipped++
		case "error":
			result.Errors++
		}
		// Fix 2: Cap the details slice.
		if len(result.Details) < maxImportDetails {
			result.Details = append(result.Details, detail)
		}
	}

	s.logger.Info("offer import completed",
		"tenant_id", tenantID,
		"total", result.TotalOffers,
		"created", result.Created,
		"linked", result.Linked,
		"skipped", result.Skipped,
		"errors", result.Errors,
	)

	return result, nil
}

// processOffer handles a single offer that needs linking or creation: matches by SKU
// or creates a new product + listing. Each offer is processed in its own transaction.
// This is only called when the listing does NOT already exist (checked in ImportOffers).
func (s *AllegroImportService) processOffer(
	ctx context.Context,
	tenantID uuid.UUID,
	integrationID uuid.UUID,
	offer *allegrosdk.Offer,
	client *allegrosdk.Client,
) model.AllegroImportDetail {
	detail := model.AllegroImportDetail{
		OfferID:   offer.ID,
		OfferName: offer.Name,
	}

	// Fetch Allegro category name BEFORE the transaction to avoid holding a DB
	// connection during an external HTTP call.
	var categoryName string
	if s.categoryService != nil && offer.Category != nil && offer.Category.ID != "" {
		categoryName = s.fetchAllegroCategoryName(ctx, client, offer.Category.ID)
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Try to match an existing product by SKU.
		var matchedProduct *model.Product
		var err error
		if offer.External != nil && offer.External.ID != "" {
			matchedProduct, err = s.productRepo.FindBySKU(ctx, tx, offer.External.ID)
			if err != nil {
				return fmt.Errorf("find product by sku: %w", err)
			}
		}

		if matchedProduct != nil {
			// Product exists — link it to the Allegro offer.
			if err := s.createListing(ctx, tx, tenantID, matchedProduct.ID, integrationID, offer); err != nil {
				return fmt.Errorf("create listing for matched product: %w", err)
			}

			// Backfill category_id if the existing product doesn't have one yet.
			if s.categoryService != nil && matchedProduct.CategoryID == nil && offer.Category != nil && offer.Category.ID != "" {
				categoryID, err := s.categoryService.ResolveMarketplaceCategory(
					ctx, tx, tenantID, integrationID,
					offer.Category.ID, categoryName,
					true, nil,
				)
				if err == nil && categoryID != nil {
					_ = s.productRepo.Update(ctx, tx, matchedProduct.ID, model.UpdateProductRequest{CategoryID: categoryID})
				}
			}

			detail.Action = "linked"
			detail.ProductID = matchedProduct.ID.String()
			return nil
		}

		// No existing product — create a new one from the offer data.
		product := mapAllegroOfferToProduct(*offer, tenantID)

		// Resolve Allegro category to an internal product category via mapping system.
		if s.categoryService != nil && offer.Category != nil && offer.Category.ID != "" {
			categoryID, err := s.categoryService.ResolveMarketplaceCategory(
				ctx, tx, tenantID, integrationID,
				offer.Category.ID, categoryName,
				true, nil,
			)
			if err == nil && categoryID != nil {
				product.CategoryID = categoryID
			}
		}

		if err := s.productRepo.Create(ctx, tx, &product); err != nil {
			return fmt.Errorf("create product: %w", err)
		}
		if err := s.createListing(ctx, tx, tenantID, product.ID, integrationID, offer); err != nil {
			return fmt.Errorf("create listing for new product: %w", err)
		}
		detail.Action = "created"
		detail.ProductID = product.ID.String()
		return nil
	})

	if err != nil {
		s.logger.Error("failed to process offer",
			"tenant_id", tenantID,
			"offer_id", offer.ID,
			"error", err,
		)
		detail.Action = "error"
		detail.Error = err.Error()
	}

	return detail
}

// fetchAllegroCategoryName fetches a category name from Allegro API.
// Best-effort: returns empty string on error (category will still be mapped without name).
func (s *AllegroImportService) fetchAllegroCategoryName(ctx context.Context, client *allegrosdk.Client, categoryID string) string {
	cat, err := client.Categories.Get(ctx, categoryID)
	if err != nil {
		s.logger.Warn("failed to fetch Allegro category name",
			"category_id", categoryID, "error", err)
		return ""
	}
	return cat.Name
}

// createListing creates a ProductListing record linking a product to an Allegro offer.
func (s *AllegroImportService) createListing(
	ctx context.Context,
	tx pgx.Tx,
	tenantID uuid.UUID,
	productID uuid.UUID,
	integrationID uuid.UUID,
	offer *allegrosdk.Offer,
) error {
	externalID := offer.ID
	listing := &model.ProductListing{
		ID:            uuid.New(),
		TenantID:      tenantID,
		ProductID:     productID,
		IntegrationID: integrationID,
		ExternalID:    &externalID,
		Status:        "active",
		SyncStatus:    "synced",
		StockSyncMode: "manual",
		Metadata:      json.RawMessage(`{}`),
	}
	return s.listingRepo.Create(ctx, tx, listing)
}

// mapAllegroOfferToProduct converts an Allegro Offer into an OpenOMS Product model.
// Fix 3: ExternalID is set to the seller's SKU (offer.External.ID), not Allegro's offer ID.
// The Allegro offer ID belongs on the ProductListing (as listing.ExternalID), not on the Product.
func mapAllegroOfferToProduct(offer allegrosdk.Offer, tenantID uuid.UUID) model.Product {
	product := model.Product{
		ID:       uuid.New(),
		TenantID: tenantID,
		Name:     offer.Name,
		Source:   "allegro",
		Metadata: json.RawMessage(`{}`),
		Tags:     []string{},
		Images:   json.RawMessage(`[]`),
	}

	// External.ID is the seller's own SKU — use it as both SKU and ExternalID.
	if offer.External != nil && offer.External.ID != "" {
		sku := offer.External.ID
		product.SKU = &sku
		product.ExternalID = &sku
	}

	// Preserve the Allegro category ID on the product for reference/filtering.
	if offer.Category != nil && offer.Category.ID != "" {
		cat := offer.Category.ID
		product.Category = &cat
	}

	if offer.SellingMode != nil {
		if p, err := strconv.ParseFloat(offer.SellingMode.Price.Amount, 64); err == nil {
			product.Price = p
		}
	}
	if offer.Stock != nil {
		product.StockQuantity = offer.Stock.Available
	}

	// Images: offer.Images is []string of URLs from Allegro API.
	if len(offer.Images) > 0 {
		first := offer.Images[0]
		product.ImageURL = &first
		if data, err := json.Marshal(offer.Images); err == nil {
			product.Images = data
		}
	} else if offer.PrimaryImage != nil && offer.PrimaryImage.URL != "" {
		product.ImageURL = &offer.PrimaryImage.URL
	}

	// Category from Allegro category ID.
	if offer.Category != nil && offer.Category.ID != "" {
		cat := offer.Category.ID
		product.Category = &cat
	}

	// EAN from offer parameters (match by name containing "EAN" or "GTIN").
	for _, p := range offer.Parameters {
		nameLower := strings.ToLower(p.Name)
		if (strings.Contains(nameLower, "ean") || strings.Contains(nameLower, "gtin")) && len(p.Values) > 0 && p.Values[0] != "" {
			ean := p.Values[0]
			product.EAN = &ean
			break
		}
	}

	// Description from offer description sections (TEXT items).
	if offer.Description != nil && len(offer.Description.Sections) > 0 {
		var parts []string
		for _, section := range offer.Description.Sections {
			for _, item := range section.Items {
				if item.Type == "TEXT" && strings.TrimSpace(item.Content) != "" {
					parts = append(parts, item.Content)
				}
			}
		}
		if len(parts) > 0 {
			product.DescriptionLong = strings.Join(parts, "\n")
		}
	}

	return product
}
