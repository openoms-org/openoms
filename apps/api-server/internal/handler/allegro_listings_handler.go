package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"regexp"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/config"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// AllegroListingsHandler handles publishing products as Allegro marketplace listings.
type AllegroListingsHandler struct {
	integrationService *service.IntegrationService
	productService     *service.ProductService
	listingRepo        *repository.ProductListingRepository
	encryptionKey      []byte
	pool               *pgxpool.Pool
	cfg                *config.Config
}

// NewAllegroListingsHandler creates a new AllegroListingsHandler.
func NewAllegroListingsHandler(
	integrationService *service.IntegrationService,
	productService *service.ProductService,
	listingRepo *repository.ProductListingRepository,
	encryptionKey []byte,
	pool *pgxpool.Pool,
	cfg *config.Config,
) *AllegroListingsHandler {
	return &AllegroListingsHandler{
		integrationService: integrationService,
		productService:     productService,
		listingRepo:        listingRepo,
		encryptionKey:      encryptionKey,
		pool:               pool,
		cfg:                cfg,
	}
}

// createListingRequest is the request body for creating a new Allegro listing.
type createListingRequest struct {
	IntegrationID  string           `json:"integration_id"`
	CategoryID     string           `json:"category_id"`
	Parameters     []map[string]any `json:"parameters"`
	ShippingRateID string           `json:"shipping_rate_id"`
	ReturnPolicyID string           `json:"return_policy_id"`
	WarrantyID     string           `json:"warranty_id"`
	HandlingTime   string           `json:"handling_time"`
	PriceOverride  *float64         `json:"price_override"`
	StockOverride  *int             `json:"stock_override"`
	Location       *locationRequest `json:"location"`
}

type locationRequest struct {
	City        string `json:"city"`
	Province    string `json:"province"`
	PostCode    string `json:"post_code"`
	CountryCode string `json:"country_code"`
}

// newAllegroClient creates an authenticated Allegro SDK client from the integration credentials.
func (h *AllegroListingsHandler) newAllegroClient(r *http.Request) (*allegrosdk.Client, error) {
	return buildAllegroClient(r, h.integrationService, h.encryptionKey)
}

// CreateListing publishes a product as an Allegro offer and records the listing.
// POST /v1/products/{productId}/listings/allegro
func (h *AllegroListingsHandler) CreateListing(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromContext(ctx)

	productID, err := uuid.Parse(chi.URLParam(r, "productId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	var req createListingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.IntegrationID == "" {
		writeError(w, http.StatusBadRequest, "integration_id is required")
		return
	}
	if req.CategoryID == "" {
		writeError(w, http.StatusBadRequest, "category_id is required")
		return
	}

	integrationID, err := uuid.Parse(req.IntegrationID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid integration_id")
		return
	}

	// Fetch product
	product, err := h.productService.Get(ctx, tenantID, productID)
	if err != nil {
		slog.Error("allegro listings: failed to get product", "error", err)
		writeError(w, http.StatusNotFound, "product not found")
		return
	}

	// Check if listing already exists for this product + integration
	var existingListing *model.ProductListing
	err = database.WithTenant(ctx, h.pool, tenantID, func(tx pgx.Tx) error {
		var findErr error
		existingListing, findErr = h.listingRepo.FindByProductAndIntegration(ctx, tx, productID, integrationID)
		return findErr
	})
	if err != nil {
		slog.Error("allegro listings: failed to check existing listing", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to check existing listing")
		return
	}
	if existingListing != nil {
		writeError(w, http.StatusConflict, "listing already exists for this product and integration")
		return
	}

	// Create Allegro client
	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro listings: failed to create client", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	// Upload images to Allegro first
	imageURLs := buildImages(product)
	if len(imageURLs) == 0 {
		writeError(w, http.StatusBadRequest, "Produkt musi miec co najmniej jedno zdjecie aby wystawic na Allegro")
		return
	}
	var uploadedImages []string
	for _, imgURL := range imageURLs {
		hostedURL, uploadErr := h.uploadImageToAllegro(ctx, client, imgURL)
		if uploadErr != nil {
			slog.Warn("allegro listings: failed to upload image, skipping", "error", uploadErr, "url", imgURL)
			continue
		}
		uploadedImages = append(uploadedImages, hostedURL)
	}
	if len(uploadedImages) == 0 {
		writeError(w, http.StatusBadRequest, "Nie udalo sie przeslac zadnego zdjecia do Allegro. Upewnij sie ze produkt ma prawidlowe URL-e zdjec (nie placeholder)")
		return
	}

	// Fetch category parameters to split between product and offer level
	catParams, err := client.Categories.GetParameters(ctx, req.CategoryID)
	if err != nil {
		slog.Warn("allegro listings: failed to fetch category params for split, sending all as product params", "error", err)
	}
	productDescribes := map[string]bool{}
	if catParams != nil {
		for _, cp := range catParams.Parameters {
			if cp.Options != nil && cp.Options.DescribesProduct {
				productDescribes[cp.ID] = true
			}
		}
	}

	// Split parameters: describesProduct=true → productSet, false → offer level
	// Also validate EAN (GTIN) — Allegro requires 8, 10, 12, 13, or 14 chars
	var productParams, offerParams []map[string]any
	for _, p := range req.Parameters {
		id, _ := p["id"].(string)
		if id == "225693" { // EAN (GTIN)
			if vals, ok := p["values"].([]any); ok && len(vals) > 0 {
				ean, _ := vals[0].(string)
				eanLen := len(ean)
				if ean == "" || (eanLen != 8 && eanLen != 10 && eanLen != 12 && eanLen != 13 && eanLen != 14) {
					slog.Info("allegro listings: skipping EAN with invalid length", "ean", ean, "len", eanLen)
					continue
				}
			}
		}
		if productDescribes[id] {
			productParams = append(productParams, p)
		} else {
			offerParams = append(offerParams, p)
		}
	}

	// Resolve GPSR responsible producer — use existing or create from location data
	producerID, err := h.resolveResponsibleProducer(ctx, client, req)
	if err != nil {
		slog.Warn("allegro listings: failed to resolve responsible producer, offer may fail GPSR", "error", err)
	}

	// Build and create offer on Allegro
	payload := buildAllegroOfferPayload(product, req, uploadedImages, productParams, offerParams, producerID)

	offer, err := client.Offers.Create(ctx, payload)
	if err != nil {
		slog.Error("allegro listings: failed to create offer", "error", err, "product_id", productID)
		writeError(w, http.StatusUnprocessableEntity, allegroErrorMessage("Nie udało się utworzyć oferty na Allegro", err))
		return
	}

	// Build metadata to store with the listing
	metadata, _ := json.Marshal(map[string]any{
		"category_id":      req.CategoryID,
		"parameters":       req.Parameters,
		"shipping_rate_id": req.ShippingRateID,
		"return_policy_id": req.ReturnPolicyID,
		"warranty_id":      req.WarrantyID,
		"handling_time":    req.HandlingTime,
		"location":         req.Location,
	})

	now := time.Now()
	sandbox := isAllegroSandbox(r, h.integrationService, h.encryptionKey)
	offerURL := allegroOfferURL(offer.ID, sandbox)
	listing := &model.ProductListing{
		ID:            uuid.New(),
		TenantID:      tenantID,
		ProductID:     productID,
		IntegrationID: integrationID,
		ExternalID:    &offer.ID,
		Status:        "active",
		URL:           &offerURL,
		PriceOverride: req.PriceOverride,
		StockOverride: req.StockOverride,
		SyncStatus:    "synced",
		LastSyncedAt:  &now,
		Metadata:      metadata,
	}

	// Create listing record in database
	err = database.WithTenant(ctx, h.pool, tenantID, func(tx pgx.Tx) error {
		return h.listingRepo.Create(ctx, tx, listing)
	})
	if err != nil {
		slog.Error("allegro listings: failed to save listing", "error", err, "product_id", productID)
		writeError(w, http.StatusInternalServerError, "offer created on Allegro but failed to save listing record")
		return
	}

	writeJSON(w, http.StatusCreated, listing)
}

// buildAllegroOfferPayload maps product data to the Allegro sale/product-offers format.
// Product-describing parameters go inside productSet[0].product.parameters.
// Offer-level parameters (e.g. "Stan") go in top-level parameters.
func buildAllegroOfferPayload(product *model.Product, req createListingRequest, images []string, productParams, offerParams []map[string]any, producerID string) map[string]any {
	price := product.Price
	if req.PriceOverride != nil {
		price = *req.PriceOverride
	}
	stock := product.StockQuantity
	if req.StockOverride != nil {
		stock = *req.StockOverride
	}

	// Allegro limits offer name to 75 characters
	offerName := product.Name
	if len([]rune(offerName)) > 75 {
		offerName = string([]rune(offerName)[:75])
	}

	// Build product definition (goes inside productSet)
	productDef := map[string]any{
		"name":     offerName,
		"category": map[string]any{"id": req.CategoryID},
	}
	if len(productParams) > 0 {
		productDef["parameters"] = productParams
	}
	if len(images) > 0 {
		productDef["images"] = images
	}

	// Build top-level offer payload
	payload := map[string]any{
		"name": offerName,
		"productSet": []map[string]any{
			buildProductSetEntry(productDef, producerID),
		},
		"sellingMode": map[string]any{
			"format": "BUY_NOW",
			"price": map[string]any{
				"amount":   fmt.Sprintf("%.2f", price),
				"currency": "PLN",
			},
		},
		"stock": map[string]any{
			"available": stock,
			"unit":      "UNIT",
		},
		"description": buildDescription(product),
		"publication": map[string]any{"status": "ACTIVE"},
	}

	if len(offerParams) > 0 {
		payload["parameters"] = offerParams
	}
	if req.Location != nil {
		loc := map[string]any{
			"city":        req.Location.City,
			"countryCode": req.Location.CountryCode,
			"province":    req.Location.Province,
		}
		if req.Location.PostCode != "" {
			loc["postCode"] = req.Location.PostCode
		}
		payload["location"] = loc
	}
	if req.ShippingRateID != "" {
		payload["delivery"] = map[string]any{
			"shippingRates": map[string]any{"id": req.ShippingRateID},
			"handlingTime":  req.HandlingTime,
		}
	}
	afterSales := map[string]any{}
	if req.ReturnPolicyID != "" {
		afterSales["returnPolicy"] = map[string]any{"id": req.ReturnPolicyID}
	}
	if req.WarrantyID != "" {
		afterSales["impliedWarranty"] = map[string]any{"id": req.WarrantyID}
	}
	if len(afterSales) > 0 {
		payload["afterSalesServices"] = afterSales
	}
	if product.SKU != nil && *product.SKU != "" {
		payload["external"] = map[string]any{"id": *product.SKU}
	}

	return payload
}

// buildProductSetEntry creates the productSet entry with GPSR fields.
func buildProductSetEntry(productDef map[string]any, producerID string) map[string]any {
	entry := map[string]any{
		"product": productDef,
		"safetyInformation": map[string]any{
			"type":        "TEXT",
			"description": "Produkt zgodny z obowiazujacymi normami bezpieczenstwa UE.",
		},
	}
	if producerID != "" {
		entry["responsibleProducer"] = map[string]any{"id": producerID}
	}
	return entry
}

// resolveResponsibleProducer gets the first existing responsible producer, or creates one.
func (h *AllegroListingsHandler) resolveResponsibleProducer(ctx context.Context, client *allegrosdk.Client, req createListingRequest) (string, error) {
	// Try to use an existing producer
	producers, err := client.Offers.ListResponsibleProducers(ctx)
	if err == nil && len(producers) > 0 {
		return producers[0].ID, nil
	}

	// Create one from the location data
	city := "Warszawa"
	postCode := "00-001"
	if req.Location != nil {
		if req.Location.City != "" {
			city = req.Location.City
		}
		if req.Location.PostCode != "" {
			postCode = req.Location.PostCode
		}
	}

	producerName := "Sprzedawca - " + city
	producer, createErr := client.Offers.CreateResponsibleProducer(ctx, producerName, allegrosdk.ResponsibleProducerData{
		TradeName: producerName,
		Address: allegrosdk.ResponsibleProducerAddress{
			Street:      city,
			PostalCode:  postCode,
			City:        city,
			CountryCode: "PL",
		},
		Contact: allegrosdk.ResponsibleProducerContact{
			Email: "kontakt@example.com",
		},
	})
	if createErr != nil {
		return "", fmt.Errorf("create responsible producer: %w", createErr)
	}
	return producer.ID, nil
}

// uploadImageToAllegro uploads an image to Allegro. For local URLs (served from our
// upload dir), it reads the file from disk and sends binary data. For public URLs,
// it lets Allegro download from the URL directly.
func (h *AllegroListingsHandler) uploadImageToAllegro(ctx context.Context, client *allegrosdk.Client, imgURL string) (string, error) {
	// Check if this is a local upload URL (e.g. http://localhost:8080/uploads/...)
	baseURL := h.cfg.BaseURL
	if baseURL != "" && strings.HasPrefix(imgURL, baseURL+"/uploads/") {
		// Extract the relative path after /uploads/
		relPath := strings.TrimPrefix(imgURL, baseURL+"/uploads/")
		uploadDir, _ := filepath.Abs(h.cfg.UploadDir)
		filePath := filepath.Join(uploadDir, relPath)

		// Prevent path traversal — ensure resolved path stays within upload dir
		cleanPath := filepath.Clean(filePath)
		cleanDir := filepath.Clean(uploadDir)
		if !strings.HasPrefix(cleanPath, cleanDir+string(filepath.Separator)) && cleanPath != cleanDir {
			return "", fmt.Errorf("path traversal blocked: %s escapes upload dir", relPath)
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("read local image %s: %w", filePath, err)
		}

		contentType := mime.TypeByExtension(filepath.Ext(filePath))
		if contentType == "" {
			contentType = "image/jpeg"
		}

		return client.Offers.UploadImageBinary(ctx, data, contentType)
	}

	// Public URL — let Allegro download it
	return client.Offers.UploadImageURL(ctx, imgURL)
}

// buildDescription builds the Allegro description structure from product data.
func buildDescription(product *model.Product) map[string]any {
	var content string
	if product.DescriptionLong != "" {
		content = product.DescriptionLong
	} else if product.DescriptionShort != "" {
		content = product.DescriptionShort
	} else {
		content = product.Name
	}
	content = sanitizeForAllegro(content)
	return map[string]any{
		"sections": []map[string]any{
			{"items": []map[string]any{
				{"type": "TEXT", "content": content},
			}},
		},
	}
}

// sanitizeForAllegro converts HTML to Allegro-compatible format.
// Allegro only allows: h1, h2, p, ul, ol, li tags (strict XHTML subset).
// Handles supplier feed HTML (BTP/IOF) which may contain div, img, style,
// h3-h6, b, strong, em, i, hr, br, table, etc.
func sanitizeForAllegro(html string) string {
	allowed := map[string]bool{"h1": true, "h2": true, "p": true, "ul": true, "ol": true, "li": true}

	// 1. Remove <style>...</style> and <script>...</script> blocks entirely
	styleRe := regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	html = styleRe.ReplaceAllString(html, "")
	scriptRe := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	html = scriptRe.ReplaceAllString(html, "")

	// 2. Remove HTML comments
	commentRe := regexp.MustCompile(`<!--[\s\S]*?-->`)
	html = commentRe.ReplaceAllString(html, "")

	// 3. Remove void/media elements entirely (img, input, source, video, audio, iframe, object, embed)
	voidRe := regexp.MustCompile(`<(?:img|input|source|video|audio|iframe|object|embed)\b[^>]*/?>`)
	html = voidRe.ReplaceAllString(html, "")

	// 4. Convert <br> and <hr> to paragraph breaks
	brRe := regexp.MustCompile(`<br\s*/?>`)
	html = brRe.ReplaceAllString(html, "\n\n")
	hrRe := regexp.MustCompile(`<hr\s*/?>`)
	html = hrRe.ReplaceAllString(html, "\n\n")

	// 5. Convert <h3>-<h6> to <h2> (preserve heading semantics)
	for _, level := range []string{"h3", "h4", "h5", "h6"} {
		openRe := regexp.MustCompile(`<` + level + `\b[^>]*>`)
		html = openRe.ReplaceAllString(html, "<h2>")
		html = strings.ReplaceAll(html, "</"+level+">", "</h2>")
	}

	// 6. Strip remaining disallowed tags but keep their content
	tagRe := regexp.MustCompile(`</?([a-zA-Z][a-zA-Z0-9]*)\b[^>]*>`)
	html = tagRe.ReplaceAllStringFunc(html, func(m string) string {
		tagNameRe := regexp.MustCompile(`</?([a-zA-Z][a-zA-Z0-9]*)`)
		parts := tagNameRe.FindStringSubmatch(m)
		if len(parts) < 2 {
			return ""
		}
		tagName := strings.ToLower(parts[1])
		if allowed[tagName] {
			if strings.HasPrefix(m, "</") {
				return "</" + tagName + ">"
			}
			return "<" + tagName + ">"
		}
		return ""
	})

	// 7. Replace &nbsp; with regular space (not valid in strict XHTML without DTD)
	html = strings.ReplaceAll(html, "&nbsp;", " ")

	// 8. Escape bare & that are not valid XML entity references.
	// Go's regexp (RE2) lacks negative lookahead, so use a positive match + replace approach.
	entityRe := regexp.MustCompile(`&(amp|lt|gt|quot|apos|#\d+|#x[0-9a-fA-F]+);`)
	// First, protect existing valid entities by replacing & with a placeholder
	html = entityRe.ReplaceAllStringFunc(html, func(m string) string {
		return "\x00ENTITY" + m[1:] // strip leading &, prefix with placeholder
	})
	// Escape all remaining bare &
	html = strings.ReplaceAll(html, "&", "&amp;")
	// Restore protected entities
	html = strings.ReplaceAll(html, "\x00ENTITY", "&")

	// 9. Collapse whitespace: 3+ newlines → 2, multiple spaces → single
	nlCollapse := regexp.MustCompile(`\n{3,}`)
	html = nlCollapse.ReplaceAllString(html, "\n\n")
	html = strings.TrimSpace(html)

	// 10. Wrap bare text in <p> tags
	var result strings.Builder
	remaining := html

	blockOpenRe := regexp.MustCompile(`<(h1|h2|p|ul|ol|li)>`)
	anyTagRe := regexp.MustCompile(`</?(?:h1|h2|p|ul|ol|li)>`)

	for remaining != "" {
		loc := anyTagRe.FindStringIndex(remaining)
		if loc == nil {
			text := strings.TrimSpace(remaining)
			if text != "" {
				for _, para := range splitIntoParagraphs(text) {
					result.WriteString("<p>" + para + "</p>")
				}
			}
			break
		}

		before := remaining[:loc[0]]
		before = strings.TrimSpace(before)
		if before != "" {
			for _, para := range splitIntoParagraphs(before) {
				result.WriteString("<p>" + para + "</p>")
			}
		}

		tag := remaining[loc[0]:loc[1]]
		result.WriteString(tag)
		remaining = remaining[loc[1]:]

		if blockOpenRe.MatchString(tag) {
			parts := blockOpenRe.FindStringSubmatch(tag)
			tagName := parts[1]
			closeTag := "</" + tagName + ">"
			closeIdx := strings.Index(remaining, closeTag)
			if closeIdx >= 0 {
				inner := remaining[:closeIdx]
				if tagName == "ul" || tagName == "ol" {
					result.WriteString(inner)
				} else {
					inner = strings.TrimSpace(inner)
					result.WriteString(inner)
				}
				result.WriteString(closeTag)
				remaining = remaining[closeIdx+len(closeTag):]
			}
		}
	}

	out := result.String()
	if out == "" {
		return "<p></p>"
	}
	return out
}

// splitIntoParagraphs splits text by double newlines into non-empty trimmed paragraphs.
func splitIntoParagraphs(text string) []string {
	parts := strings.Split(text, "\n\n")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		// Also convert single newlines to spaces within a paragraph
		p = strings.ReplaceAll(p, "\n", " ")
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// buildImages extracts image URLs from the product for the Allegro offer.
// Allegro expects images as an array of plain URL strings.
func buildImages(product *model.Product) []string {
	var images []string
	if product.ImageURL != nil && *product.ImageURL != "" {
		images = append(images, *product.ImageURL)
	}
	// Parse Images JSON array if present
	if len(product.Images) > 0 {
		var imgList []struct {
			URL string `json:"url"`
		}
		if json.Unmarshal(product.Images, &imgList) == nil {
			for _, img := range imgList {
				if img.URL != "" {
					images = append(images, img.URL)
				}
			}
		}
	}
	return images
}

// ListByProduct returns all listings for a given product.
// GET /v1/products/{productId}/listings
func (h *AllegroListingsHandler) ListByProduct(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromContext(ctx)

	productID, err := uuid.Parse(chi.URLParam(r, "productId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	var listings []*model.ProductListing
	err = database.WithTenant(ctx, h.pool, tenantID, func(tx pgx.Tx) error {
		var listErr error
		listings, listErr = h.listingRepo.ListByProduct(ctx, tx, productID)
		return listErr
	})
	if err != nil {
		slog.Error("allegro listings: failed to list by product", "error", err, "product_id", productID)
		writeError(w, http.StatusInternalServerError, "failed to list listings")
		return
	}

	if listings == nil {
		listings = []*model.ProductListing{}
	}

	writeJSON(w, http.StatusOK, listings)
}

// GetListing returns a single listing by ID.
// GET /v1/products/{productId}/listings/{listingId}
func (h *AllegroListingsHandler) GetListing(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromContext(ctx)

	listingID, err := uuid.Parse(chi.URLParam(r, "listingId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid listing ID")
		return
	}

	var listing *model.ProductListing
	err = database.WithTenant(ctx, h.pool, tenantID, func(tx pgx.Tx) error {
		var getErr error
		listing, getErr = h.listingRepo.GetByID(ctx, tx, listingID)
		return getErr
	})
	if err != nil {
		slog.Error("allegro listings: failed to get listing", "error", err, "listing_id", listingID)
		writeError(w, http.StatusInternalServerError, "failed to get listing")
		return
	}
	if listing == nil {
		writeError(w, http.StatusNotFound, "listing not found")
		return
	}

	writeJSON(w, http.StatusOK, listing)
}

// UpdateListing updates a listing record.
// PATCH /v1/products/{productId}/listings/{listingId}
func (h *AllegroListingsHandler) UpdateListing(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromContext(ctx)

	listingID, err := uuid.Parse(chi.URLParam(r, "listingId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid listing ID")
		return
	}

	var req model.UpdateProductListingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var listing *model.ProductListing
	err = database.WithTenant(ctx, h.pool, tenantID, func(tx pgx.Tx) error {
		if updateErr := h.listingRepo.Update(ctx, tx, listingID, &req); updateErr != nil {
			return updateErr
		}
		var getErr error
		listing, getErr = h.listingRepo.GetByID(ctx, tx, listingID)
		return getErr
	})
	if err != nil {
		slog.Error("allegro listings: failed to update listing", "error", err, "listing_id", listingID)
		writeError(w, http.StatusInternalServerError, "failed to update listing")
		return
	}

	writeJSON(w, http.StatusOK, listing)
}

// DeleteListing removes a listing and optionally deactivates the Allegro offer.
// DELETE /v1/products/{productId}/listings/{listingId}
func (h *AllegroListingsHandler) DeleteListing(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromContext(ctx)

	listingID, err := uuid.Parse(chi.URLParam(r, "listingId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid listing ID")
		return
	}

	// Get the listing first to retrieve external_id
	var listing *model.ProductListing
	err = database.WithTenant(ctx, h.pool, tenantID, func(tx pgx.Tx) error {
		var getErr error
		listing, getErr = h.listingRepo.GetByID(ctx, tx, listingID)
		return getErr
	})
	if err != nil {
		slog.Error("allegro listings: failed to get listing for delete", "error", err, "listing_id", listingID)
		writeError(w, http.StatusInternalServerError, "failed to get listing")
		return
	}
	if listing == nil {
		writeError(w, http.StatusNotFound, "listing not found")
		return
	}

	// Try to deactivate the offer on Allegro (non-fatal if it fails)
	if listing.ExternalID != nil && *listing.ExternalID != "" {
		client, clientErr := h.newAllegroClient(r)
		if clientErr == nil {
			defer client.Close()
			if deactivateErr := client.Offers.Deactivate(ctx, *listing.ExternalID); deactivateErr != nil {
				slog.Warn("allegro listings: failed to deactivate offer on Allegro",
					"error", deactivateErr, "external_id", *listing.ExternalID)
			}
		} else {
			slog.Warn("allegro listings: could not create client to deactivate offer", "error", clientErr)
		}
	}

	// Delete the listing record
	err = database.WithTenant(ctx, h.pool, tenantID, func(tx pgx.Tx) error {
		return h.listingRepo.Delete(ctx, tx, listingID)
	})
	if err != nil {
		slog.Error("allegro listings: failed to delete listing", "error", err, "listing_id", listingID)
		writeError(w, http.StatusInternalServerError, "failed to delete listing")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SyncListing synchronizes stock and price from the product to the Allegro offer.
// POST /v1/products/{productId}/listings/{listingId}/sync
func (h *AllegroListingsHandler) SyncListing(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromContext(ctx)

	productID, err := uuid.Parse(chi.URLParam(r, "productId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	listingID, err := uuid.Parse(chi.URLParam(r, "listingId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid listing ID")
		return
	}

	// Get the listing
	var listing *model.ProductListing
	err = database.WithTenant(ctx, h.pool, tenantID, func(tx pgx.Tx) error {
		var getErr error
		listing, getErr = h.listingRepo.GetByID(ctx, tx, listingID)
		return getErr
	})
	if err != nil {
		slog.Error("allegro listings: failed to get listing for sync", "error", err, "listing_id", listingID)
		writeError(w, http.StatusInternalServerError, "failed to get listing")
		return
	}
	if listing == nil {
		writeError(w, http.StatusNotFound, "listing not found")
		return
	}
	if listing.ExternalID == nil || *listing.ExternalID == "" {
		writeError(w, http.StatusBadRequest, "listing has no external offer ID")
		return
	}

	// Get the product
	product, err := h.productService.Get(ctx, tenantID, productID)
	if err != nil {
		slog.Error("allegro listings: failed to get product for sync", "error", err, "product_id", productID)
		writeError(w, http.StatusNotFound, "product not found")
		return
	}

	// Create Allegro client
	client, err := h.newAllegroClient(r)
	if err != nil {
		slog.Error("allegro listings: failed to create client for sync", "error", err)
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie jest skonfigurowana")
		return
	}
	defer client.Close()

	externalID := *listing.ExternalID

	// Determine stock and price to sync
	stock := product.StockQuantity
	if listing.StockOverride != nil {
		stock = *listing.StockOverride
	}
	price := product.Price
	if listing.PriceOverride != nil {
		price = *listing.PriceOverride
	}

	// Update stock on Allegro
	if err := client.Offers.UpdateStock(ctx, externalID, stock); err != nil {
		slog.Error("allegro listings: failed to sync stock", "error", err, "external_id", externalID)
		writeError(w, http.StatusUnprocessableEntity, "Nie udało się zsynchronizować stanu magazynowego")
		return
	}

	// Update price on Allegro
	if err := client.Offers.UpdatePrice(ctx, externalID, price, "PLN"); err != nil {
		slog.Error("allegro listings: failed to sync price", "error", err, "external_id", externalID)
		writeError(w, http.StatusUnprocessableEntity, "Nie udało się zsynchronizować ceny")
		return
	}

	// Update listing sync status
	now := time.Now()
	syncStatus := "synced"
	updateReq := &model.UpdateProductListingRequest{
		SyncStatus: &syncStatus,
	}

	err = database.WithTenant(ctx, h.pool, tenantID, func(tx pgx.Tx) error {
		if updateErr := h.listingRepo.Update(ctx, tx, listingID, updateReq); updateErr != nil {
			return updateErr
		}
		// Update last_synced_at via direct SQL since UpdateProductListingRequest doesn't have it
		_, execErr := tx.Exec(ctx, "UPDATE product_listings SET last_synced_at = $1 WHERE id = $2", now, listingID)
		if execErr != nil {
			return fmt.Errorf("update last_synced_at: %w", execErr)
		}
		var getErr error
		listing, getErr = h.listingRepo.GetByID(ctx, tx, listingID)
		return getErr
	})
	if err != nil {
		slog.Error("allegro listings: failed to update sync status", "error", err, "listing_id", listingID)
		writeError(w, http.StatusInternalServerError, "synced to Allegro but failed to update listing record")
		return
	}

	writeJSON(w, http.StatusOK, listing)
}