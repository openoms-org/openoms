package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// FeedService handles product feed generation (Ceneo, Google Shopping).
type FeedService struct {
	tenantRepo  repository.TenantRepo
	productRepo repository.ProductRepo
	pool        *pgxpool.Pool
	baseURL     string

	// In-memory feed cache: key = "ceneo:{tenantID}" or "google:{tenantID}"
	cacheMu    sync.RWMutex
	cacheData  map[string][]byte
	cacheExpAt map[string]time.Time
}

const feedCacheTTL = 15 * time.Minute

// NewFeedService creates a new FeedService.
func NewFeedService(tenantRepo repository.TenantRepo, productRepo repository.ProductRepo, pool *pgxpool.Pool, baseURL string) *FeedService {
	return &FeedService{
		tenantRepo:  tenantRepo,
		productRepo: productRepo,
		pool:        pool,
		baseURL:     baseURL,
		cacheData:   make(map[string][]byte),
		cacheExpAt:  make(map[string]time.Time),
	}
}

// GetFeedConfig reads the product_feeds section from tenant settings.
func (s *FeedService) GetFeedConfig(ctx context.Context, tenantID uuid.UUID) (*model.ProductFeedConfig, error) {
	cfg := model.DefaultProductFeedConfig()
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		return s.getSettingsSection(ctx, tx, tenantID, "product_feeds", &cfg)
	})
	if err != nil {
		return nil, fmt.Errorf("get feed config: %w", err)
	}
	return &cfg, nil
}

// UpdateFeedConfig persists the product_feeds section in tenant settings.
// If tokens are empty, generates new random tokens.
func (s *FeedService) UpdateFeedConfig(ctx context.Context, tenantID uuid.UUID, cfg *model.ProductFeedConfig) error {
	if cfg.CeneoEnabled && cfg.CeneoFeedToken == "" {
		token, err := generateFeedToken()
		if err != nil {
			return fmt.Errorf("generate ceneo token: %w", err)
		}
		cfg.CeneoFeedToken = token
	}
	if cfg.GoogleEnabled && cfg.GoogleFeedToken == "" {
		token, err := generateFeedToken()
		if err != nil {
			return fmt.Errorf("generate google token: %w", err)
		}
		cfg.GoogleFeedToken = token
	}

	if cfg.DefaultCurrency == "" {
		cfg.DefaultCurrency = "PLN"
	}
	if cfg.DefaultShippingCost == "" {
		cfg.DefaultShippingCost = "12.99"
	}
	if cfg.ExcludedCategories == nil {
		cfg.ExcludedCategories = []string{}
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		return s.updateSettingsSection(ctx, tx, tenantID, "product_feeds", cfg)
	})
	if err != nil {
		return fmt.Errorf("update feed config: %w", err)
	}

	// Invalidate cache for this tenant
	s.invalidateCache(tenantID)

	return nil
}

// RegenerateToken generates a new token for the specified feed type ("ceneo" or "google").
func (s *FeedService) RegenerateToken(ctx context.Context, tenantID uuid.UUID, feedType string) (*model.ProductFeedConfig, error) {
	cfg, err := s.GetFeedConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	token, err := generateFeedToken()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	switch feedType {
	case "ceneo":
		cfg.CeneoFeedToken = token
	case "google":
		cfg.GoogleFeedToken = token
	default:
		return nil, fmt.Errorf("unknown feed type: %s", feedType)
	}

	if err := s.UpdateFeedConfig(ctx, tenantID, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// ValidateFeedToken checks if the given token matches the stored token for the feed type.
func (s *FeedService) ValidateFeedToken(ctx context.Context, tenantID uuid.UUID, feedType, token string) (bool, error) {
	cfg, err := s.GetFeedConfig(ctx, tenantID)
	if err != nil {
		return false, err
	}

	switch feedType {
	case "ceneo":
		return cfg.CeneoEnabled && subtle.ConstantTimeCompare([]byte(cfg.CeneoFeedToken), []byte(token)) == 1, nil
	case "google":
		return cfg.GoogleEnabled && subtle.ConstantTimeCompare([]byte(cfg.GoogleFeedToken), []byte(token)) == 1, nil
	default:
		return false, nil
	}
}

// GenerateCeneoFeed generates the Ceneo XML feed for a tenant.
func (s *FeedService) GenerateCeneoFeed(ctx context.Context, tenantID uuid.UUID) ([]byte, error) {
	cacheKey := fmt.Sprintf("ceneo:%s", tenantID)
	if data := s.getFromCache(cacheKey); data != nil {
		return data, nil
	}

	cfg, err := s.GetFeedConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	products, err := s.fetchFilteredProducts(ctx, tenantID, cfg)
	if err != nil {
		return nil, err
	}

	data, err := s.buildCeneoXML(products, cfg)
	if err != nil {
		return nil, fmt.Errorf("build ceneo xml: %w", err)
	}

	s.putToCache(cacheKey, data)
	return data, nil
}

// GenerateGoogleFeed generates the Google Shopping RSS feed for a tenant.
func (s *FeedService) GenerateGoogleFeed(ctx context.Context, tenantID uuid.UUID) ([]byte, error) {
	cacheKey := fmt.Sprintf("google:%s", tenantID)
	if data := s.getFromCache(cacheKey); data != nil {
		return data, nil
	}

	cfg, err := s.GetFeedConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Get company name from tenant settings for Google feed title
	companyName := "OpenOMS Store"
	_ = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var company model.CompanySettings
		if err := s.getSettingsSection(ctx, tx, tenantID, "company", &company); err == nil && company.CompanyName != "" {
			companyName = company.CompanyName
		}
		return nil
	})

	products, err := s.fetchFilteredProducts(ctx, tenantID, cfg)
	if err != nil {
		return nil, err
	}

	data, err := s.buildGoogleXML(products, cfg, companyName)
	if err != nil {
		return nil, fmt.Errorf("build google xml: %w", err)
	}

	s.putToCache(cacheKey, data)
	return data, nil
}

// fetchFilteredProducts fetches products and applies feed filters.
func (s *FeedService) fetchFilteredProducts(ctx context.Context, tenantID uuid.UUID, cfg *model.ProductFeedConfig) ([]model.Product, error) {
	var products []model.Product
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Fetch all products (no pagination limit for feeds)
		allProducts, _, err := s.productRepo.List(ctx, tx, model.ProductListFilter{
			PaginationParams: model.PaginationParams{
				Limit:  10000, // practical upper bound
				Offset: 0,
			},
		})
		if err != nil {
			return fmt.Errorf("list products: %w", err)
		}
		products = allProducts
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Apply filters
	excludedSet := make(map[string]bool, len(cfg.ExcludedCategories))
	for _, cat := range cfg.ExcludedCategories {
		excludedSet[cat] = true
	}

	var filtered []model.Product
	for _, p := range products {
		// Exclude out-of-stock if configured
		if cfg.ExcludeOutOfStock && p.StockQuantity <= 0 {
			continue
		}
		// Exclude by category
		if p.Category != nil && excludedSet[*p.Category] {
			continue
		}
		filtered = append(filtered, p)
	}

	return filtered, nil
}

// ----- Ceneo XML -----

type ceneoOffers struct {
	XMLName xml.Name     `xml:"offers"`
	XMLNS   string       `xml:"xmlns:xsi,attr"`
	Version string       `xml:"version,attr"`
	Offers  []ceneoOffer `xml:"o"`
}

type ceneoOffer struct {
	ID     string     `xml:"id,attr"`
	URL    string     `xml:"url,attr"`
	Price  string     `xml:"price,attr"`
	Avail  string     `xml:"avail,attr"`
	Weight string     `xml:"weight,attr,omitempty"`
	Set    string     `xml:"set,attr"`
	Basket string     `xml:"basket,attr"`
	Cat    *cdata     `xml:"cat"`
	Name   *cdata     `xml:"name"`
	Desc   *cdata     `xml:"desc,omitempty"`
	Imgs   *ceneoImgs `xml:"imgs,omitempty"`
}

type ceneoImgs struct {
	Main *ceneoImg  `xml:"main,omitempty"`
	Imgs []ceneoImg `xml:"i,omitempty"`
}

type ceneoImg struct {
	URL string `xml:"url,attr"`
}

type cdata struct {
	Value string `xml:",cdata"`
}

func (s *FeedService) buildCeneoXML(products []model.Product, _ *model.ProductFeedConfig) ([]byte, error) {
	offers := ceneoOffers{
		XMLNS:   "http://www.w3.org/2001/XMLSchema-instance",
		Version: "1",
	}

	for _, p := range products {
		productID := s.productIdentifier(p)
		avail := "1"
		if p.StockQuantity <= 0 {
			avail = "99" // Ceneo: 99 = not available
		}

		offer := ceneoOffer{
			ID:     productID,
			URL:    fmt.Sprintf("%s/products/%s", s.baseURL, productID),
			Price:  fmt.Sprintf("%.2f", p.Price),
			Avail:  avail,
			Set:    "0",
			Basket: "0",
			Name:   &cdata{Value: p.Name},
		}

		if p.Weight != nil {
			offer.Weight = fmt.Sprintf("%.2f", *p.Weight)
		}

		if p.Category != nil && *p.Category != "" {
			offer.Cat = &cdata{Value: *p.Category}
		}

		if p.DescriptionShort != "" {
			offer.Desc = &cdata{Value: p.DescriptionShort}
		}

		imgs := s.extractImages(p)
		if len(imgs) > 0 {
			offer.Imgs = &ceneoImgs{
				Main: &ceneoImg{URL: imgs[0]},
			}
			for _, img := range imgs[1:] {
				offer.Imgs.Imgs = append(offer.Imgs.Imgs, ceneoImg{URL: img})
			}
		}

		offers.Offers = append(offers.Offers, offer)
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(offers); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ----- Google Shopping XML -----

type googleRSS struct {
	XMLName xml.Name      `xml:"rss"`
	Version string        `xml:"version,attr"`
	XMLNS   string        `xml:"xmlns:g,attr"`
	Channel googleChannel `xml:"channel"`
}

type googleChannel struct {
	Title       string       `xml:"title"`
	Link        string       `xml:"link"`
	Description string       `xml:"description"`
	Items       []googleItem `xml:"item"`
}

type googleItem struct {
	GID          string          `xml:"g:id"`
	Title        string          `xml:"title"`
	Description  string          `xml:"description"`
	Link         string          `xml:"link"`
	GImageLink   string          `xml:"g:image_link,omitempty"`
	GPrice       string          `xml:"g:price"`
	GAvail       string          `xml:"g:availability"`
	GCondition   string          `xml:"g:condition"`
	GBrand       string          `xml:"g:brand,omitempty"`
	GShipping    *googleShipping `xml:"g:shipping,omitempty"`
	GEAN         string          `xml:"g:gtin,omitempty"`
	GProductType string          `xml:"g:product_type,omitempty"`
}

type googleShipping struct {
	GCountry string `xml:"g:country"`
	GPrice   string `xml:"g:price"`
}

func (s *FeedService) buildGoogleXML(products []model.Product, cfg *model.ProductFeedConfig, storeName string) ([]byte, error) {
	rss := googleRSS{
		Version: "2.0",
		XMLNS:   "http://base.google.com/ns/1.0",
		Channel: googleChannel{
			Title:       storeName,
			Link:        s.baseURL,
			Description: "Product feed",
		},
	}

	currency := cfg.DefaultCurrency
	if currency == "" {
		currency = "PLN"
	}

	for _, p := range products {
		productID := s.productIdentifier(p)
		avail := "in_stock"
		if p.StockQuantity <= 0 {
			avail = "out_of_stock"
		}

		item := googleItem{
			GID:        productID,
			Title:      p.Name,
			Link:       fmt.Sprintf("%s/products/%s", s.baseURL, productID),
			GPrice:     fmt.Sprintf("%.2f %s", p.Price, currency),
			GAvail:     avail,
			GCondition: "new",
		}

		if p.DescriptionShort != "" {
			item.Description = p.DescriptionShort
		} else if p.DescriptionLong != "" {
			desc := p.DescriptionLong
			if len(desc) > 5000 {
				desc = desc[:5000]
			}
			item.Description = desc
		}

		imgs := s.extractImages(p)
		if len(imgs) > 0 {
			item.GImageLink = imgs[0]
		}

		if p.EAN != nil && *p.EAN != "" {
			item.GEAN = *p.EAN
		}

		if p.Category != nil && *p.Category != "" {
			item.GProductType = *p.Category
		}

		if cfg.DefaultShippingCost != "" {
			item.GShipping = &googleShipping{
				GCountry: "PL",
				GPrice:   fmt.Sprintf("%s %s", cfg.DefaultShippingCost, currency),
			}
		}

		rss.Channel.Items = append(rss.Channel.Items, item)
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(rss); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ----- Helpers -----

func (s *FeedService) productIdentifier(p model.Product) string {
	if p.SKU != nil && *p.SKU != "" {
		return *p.SKU
	}
	return p.ID.String()
}

type productImage struct {
	URL      string `json:"url"`
	Position int    `json:"position"`
}

func (s *FeedService) extractImages(p model.Product) []string {
	// Try the Images JSONB array first
	if len(p.Images) > 0 && string(p.Images) != "null" && string(p.Images) != "[]" {
		var imgs []productImage
		if err := json.Unmarshal(p.Images, &imgs); err == nil && len(imgs) > 0 {
			urls := make([]string, 0, len(imgs))
			for _, img := range imgs {
				if img.URL != "" {
					urls = append(urls, img.URL)
				}
			}
			if len(urls) > 0 {
				return urls
			}
		}
	}

	// Fall back to single image_url
	if p.ImageURL != nil && *p.ImageURL != "" {
		return []string{*p.ImageURL}
	}

	return nil
}

// getSettingsSection reads a section from tenant settings.
func (s *FeedService) getSettingsSection(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, key string, dest any) error {
	settings, err := s.tenantRepo.GetSettings(ctx, tx, tenantID)
	if err != nil {
		return err
	}
	if settings == nil {
		return nil
	}

	var allSettings map[string]json.RawMessage
	if err := json.Unmarshal(settings, &allSettings); err != nil {
		return err
	}

	raw, ok := allSettings[key]
	if !ok {
		return nil
	}

	if err := json.Unmarshal(raw, dest); err != nil {
		slog.Warn("failed to unmarshal settings section", "key", key, "error", err)
	}
	return nil
}

// updateSettingsSection merges a value into tenant settings under the given key.
func (s *FeedService) updateSettingsSection(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, key string, value any) error {
	existing, err := s.tenantRepo.GetSettings(ctx, tx, tenantID)
	if err != nil {
		return err
	}

	var allSettings map[string]json.RawMessage
	if err := json.Unmarshal(existing, &allSettings); err != nil {
		allSettings = make(map[string]json.RawMessage)
	}

	valueJSON, err := json.Marshal(value)
	if err != nil {
		return err
	}
	allSettings[key] = valueJSON

	newSettings, err := json.Marshal(allSettings)
	if err != nil {
		return err
	}

	return s.tenantRepo.UpdateSettings(ctx, tx, tenantID, newSettings)
}

// generateFeedToken generates a cryptographically random 16-byte hex token.
func generateFeedToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ----- Cache helpers -----

func (s *FeedService) getFromCache(key string) []byte {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	expAt, ok := s.cacheExpAt[key]
	if !ok || time.Now().After(expAt) {
		return nil
	}
	return s.cacheData[key]
}

func (s *FeedService) putToCache(key string, data []byte) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	s.cacheData[key] = data
	s.cacheExpAt[key] = time.Now().Add(feedCacheTTL)
}

func (s *FeedService) invalidateCache(tenantID uuid.UUID) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	delete(s.cacheData, fmt.Sprintf("ceneo:%s", tenantID))
	delete(s.cacheData, fmt.Sprintf("google:%s", tenantID))
	delete(s.cacheExpAt, fmt.Sprintf("ceneo:%s", tenantID))
	delete(s.cacheExpAt, fmt.Sprintf("google:%s", tenantID))
}
