package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	ebayIntegration "github.com/openoms-org/openoms/apps/api-server/internal/integration/ebay"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	ebaysdk "github.com/openoms-org/openoms/packages/ebay-go-sdk"
)

// EbayImportService handles importing eBay seller offers into OpenOMS
// as Product + ProductListing records.
type EbayImportService struct {
	integrationService *IntegrationService
	productRepo        repository.ProductRepo
	listingRepo        repository.ProductListingRepo
	pool               *pgxpool.Pool
	logger             *slog.Logger
	// importLocks records tenants with an in-flight import. Entries are deleted
	// when the import finishes, so the map only grows to the number of concurrent
	// imports (not the total number of tenants that ever imported).
	importLocks sync.Map // map[uuid.UUID]struct{}
}

// NewEbayImportService creates a new EbayImportService.
func NewEbayImportService(
	integrationService *IntegrationService,
	productRepo repository.ProductRepo,
	listingRepo repository.ProductListingRepo,
	pool *pgxpool.Pool,
) *EbayImportService {
	return &EbayImportService{
		integrationService: integrationService,
		productRepo:        productRepo,
		listingRepo:        listingRepo,
		pool:               pool,
		logger:             slog.Default().With("component", "ebay_import"),
	}
}

// ImportOffers fetches all seller offers from eBay and creates/links Product + ProductListing
// records in OpenOMS. Offers that already have a listing are skipped. Existing products are
// matched by SKU. If no match is found, a new product is created.
func (s *EbayImportService) ImportOffers(ctx context.Context, tenantID uuid.UUID) (*model.EbayImportResult, error) {
	// Per-tenant concurrency guard: only one import at a time per tenant.
	// LoadOrStore is atomic — a second caller sees loaded=true and is rejected.
	if _, busy := s.importLocks.LoadOrStore(tenantID, struct{}{}); busy {
		return nil, fmt.Errorf("import already in progress for this tenant")
	}
	defer s.importLocks.Delete(tenantID)

	// Build eBay provider from encrypted credentials.
	credJSON, integration, err := s.integrationService.GetDecryptedCredentialsByProvider(ctx, tenantID, "ebay")
	if err != nil {
		return nil, fmt.Errorf("get ebay credentials: %w", err)
	}
	provider, err := ebayIntegration.NewProvider(json.RawMessage(credJSON), nil)
	if err != nil {
		return nil, fmt.Errorf("create ebay provider: %w", err)
	}

	integrationID := integration.ID
	client := provider.Client()

	// Fetch all seller offers via pagination (capped at 50 pages as safety limit).
	var allOffers []ebaysdk.Offer
	const pageSize = 100
	const maxPages = 50
	offset := 0

	for range maxPages {
		resp, err := client.Offers.GetOffers(ctx, "", pageSize, offset)
		if err != nil {
			return nil, fmt.Errorf("list ebay offers: %w", err)
		}
		allOffers = append(allOffers, resp.Offers...)
		if offset+len(resp.Offers) >= resp.Total {
			break
		}
		offset += len(resp.Offers)
	}

	result := &model.EbayImportResult{
		TotalOffers: len(allOffers),
	}

	s.logger.Info("starting offer import",
		"tenant_id", tenantID,
		"total_offers", len(allOffers),
	)

	for i := range allOffers {
		offer := &allOffers[i]

		// Bail out early if the context has been cancelled.
		if err := ctx.Err(); err != nil {
			return result, err
		}

		// Check if listing already exists BEFORE processing.
		var existingSkipped bool
		var existingProductID string
		skipErr := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
			existing, err := s.listingRepo.FindByExternalIDAndIntegration(ctx, tx, offer.OfferID, integrationID)
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
				"offer_id", offer.OfferID,
				"error", skipErr,
			)
			result.Errors++
			if len(result.Details) < maxImportDetails {
				result.Details = append(result.Details, model.EbayImportDetail{
					OfferID: offer.OfferID,
					SKU:     offer.SKU,
					Action:  "error",
					Error:   fmt.Sprintf("check listing: %v", skipErr),
				})
			}
			continue
		}
		if existingSkipped {
			result.Skipped++
			if len(result.Details) < maxImportDetails {
				result.Details = append(result.Details, model.EbayImportDetail{
					OfferID:   offer.OfferID,
					SKU:       offer.SKU,
					Action:    "skipped",
					ProductID: existingProductID,
				})
			}
			continue
		}

		// Listing does not exist — process the offer.
		detail := s.processOffer(ctx, tenantID, integrationID, offer)
		switch detail.Action {
		case "created":
			result.Created++
		case "linked":
			result.Linked++
		case "skipped":
			result.Skipped++
		case "error":
			result.Errors++
		}
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
func (s *EbayImportService) processOffer(
	ctx context.Context,
	tenantID uuid.UUID,
	integrationID uuid.UUID,
	offer *ebaysdk.Offer,
) model.EbayImportDetail {
	detail := model.EbayImportDetail{
		OfferID: offer.OfferID,
		SKU:     offer.SKU,
		Title:   offer.ListingDescription,
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Try to match an existing product by SKU.
		var matchedProduct *model.Product
		var err error
		if offer.SKU != "" {
			matchedProduct, err = s.productRepo.FindBySKU(ctx, tx, offer.SKU)
			if err != nil {
				return fmt.Errorf("find product by sku: %w", err)
			}
		}

		if matchedProduct != nil {
			// Product exists — link it to the eBay offer.
			if err := s.createListing(ctx, tx, tenantID, matchedProduct.ID, integrationID, offer); err != nil {
				return fmt.Errorf("create listing for matched product: %w", err)
			}
			detail.Action = "linked"
			detail.ProductID = matchedProduct.ID.String()
			return nil
		}

		// No existing product — create a new one from the offer data.
		product := mapEbayOfferToProduct(offer, tenantID)
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
			"offer_id", offer.OfferID,
			"error", err,
		)
		detail.Action = "error"
		detail.Error = err.Error()
	}

	return detail
}

// createListing creates a ProductListing record linking a product to an eBay offer.
func (s *EbayImportService) createListing(
	ctx context.Context,
	tx pgx.Tx,
	tenantID uuid.UUID,
	productID uuid.UUID,
	integrationID uuid.UUID,
	offer *ebaysdk.Offer,
) error {
	externalID := offer.OfferID

	// Map eBay offer status to listing status.
	status := "inactive"
	if offer.Status == "PUBLISHED" {
		status = "active"
	}

	listing := &model.ProductListing{
		ID:            uuid.New(),
		TenantID:      tenantID,
		ProductID:     productID,
		IntegrationID: integrationID,
		ExternalID:    &externalID,
		Status:        status,
		SyncStatus:    "synced",
		StockSyncMode: "manual",
		Metadata:      json.RawMessage(`{}`),
	}
	return s.listingRepo.Create(ctx, tx, listing)
}

// mapEbayOfferToProduct converts an eBay Offer into an OpenOMS Product model.
func mapEbayOfferToProduct(offer *ebaysdk.Offer, tenantID uuid.UUID) model.Product {
	product := model.Product{
		ID:       uuid.New(),
		TenantID: tenantID,
		Source:   "ebay",
		Metadata: json.RawMessage(`{}`),
		Tags:     []string{},
		Images:   json.RawMessage(`[]`),
	}

	// Use SKU as both name (fallback) and SKU field.
	if offer.SKU != "" {
		sku := offer.SKU
		product.SKU = &sku
		product.ExternalID = &sku
		product.Name = offer.SKU
	}

	// Override name with listing description if available.
	if offer.ListingDescription != "" {
		product.Name = offer.ListingDescription
	}

	// Fallback name if still empty.
	if product.Name == "" {
		product.Name = "eBay offer " + offer.OfferID
	}

	// Price from PricingSummary.
	if offer.PricingSummary != nil {
		if p, err := strconv.ParseFloat(offer.PricingSummary.Price.Value, 64); err == nil {
			product.Price = p
		}
	}

	// Stock from AvailableQuantity.
	product.StockQuantity = offer.AvailableQuantity

	// Category ID from eBay.
	if offer.CategoryID != "" {
		cat := offer.CategoryID
		product.Category = &cat
	}

	return product
}
