package main

import (
	"context"
	"encoding/hex"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	// Register marketplace providers via init().
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/allegro"
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/amazon"
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/ebay"
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/erli"
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/kaufland"
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/mirakl"
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/olx"
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/prestashop"
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/shoper"
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/shopify"
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/woocommerce"

	// Register carrier providers via init().
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/carriers"
	// Register invoicing providers via init().
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/accounting"
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/fakturownia"
	// Register supplier providers via init().
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/btp"

	"github.com/openoms-org/openoms/apps/api-server/docs"
	"github.com/openoms-org/openoms/apps/api-server/internal/automation"
	"github.com/openoms-org/openoms/apps/api-server/internal/config"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/handler"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/router"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
	"github.com/openoms-org/openoms/apps/api-server/internal/storage"
	"github.com/openoms-org/openoms/apps/api-server/internal/worker"
	"github.com/openoms-org/openoms/apps/api-server/internal/ws"
	inpost "github.com/openoms-org/openoms/packages/inpost-go-sdk"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Connect to Redis (for rate limiting and token blacklist)
	var redisClient *redis.Client
	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		slog.Warn("invalid REDIS_URL, falling back to in-memory stores", "error", err)
	} else {
		redisClient = redis.NewClient(redisOpts)
		if err := redisClient.Ping(context.Background()).Err(); err != nil {
			slog.Warn("Redis not available, falling back to in-memory stores", "error", err)
			_ = redisClient.Close()
			redisClient = nil
		} else {
			slog.Info("connected to Redis", "url", cfg.RedisURL)
		}
	}
	defer func() {
		if redisClient != nil {
			_ = redisClient.Close()
		}
	}()

	// Setup logger
	logLevel := slog.LevelInfo
	if cfg.IsDevelopment() {
		logLevel = slog.LevelDebug
	}

	var logHandler slog.Handler
	if cfg.IsDevelopment() {
		logHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})
	} else {
		logHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})
	}
	slog.SetDefault(slog.New(logHandler))

	slog.Info("starting OpenOMS API server", "port", cfg.Port, "env", cfg.Env)

	// Connect to database
	pool, err := database.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("connected to PostgreSQL")

	// Worker pool — superuser connection for cross-tenant queries (bypasses RLS).
	// Falls back to main DATABASE_URL if WORKER_DATABASE_URL is not set.
	workerDBURL := cfg.WorkerDatabaseURL
	if workerDBURL == "" {
		workerDBURL = cfg.DatabaseURL
	}
	workerPool, err := database.Connect(context.Background(), workerDBURL)
	if err != nil {
		slog.Error("failed to connect worker database", "error", err)
		pool.Close()
		os.Exit(1) //nolint:gocritic // pool closed above
	}
	defer workerPool.Close()

	// Initialize storage backend
	var objectStorage storage.ObjectStorage
	if cfg.S3Enabled {
		s3Store, err := storage.NewS3Storage(cfg.S3Region, cfg.S3Bucket, cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3PublicURL)
		if err != nil {
			slog.Error("failed to initialize S3 storage", "error", err)
			os.Exit(1)
		}
		objectStorage = s3Store
		slog.Info("using S3 storage", "bucket", cfg.S3Bucket)
	} else {
		// Create upload directory for local storage
		if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
			slog.Error("failed to create upload directory", "error", err)
			os.Exit(1)
		}
		objectStorage = storage.NewLocalStorage(cfg.UploadDir, cfg.BaseURL)
		slog.Info("using local storage", "dir", cfg.UploadDir)
	}

	// Decode encryption key
	encryptionKey, err := hex.DecodeString(cfg.EncryptionKey)
	if err != nil {
		slog.Error("invalid ENCRYPTION_KEY (must be 64-char hex string)", "error", err)
		os.Exit(1)
	}

	// Initialize token service (Ed25519 key derivation)
	tokenSvc, err := service.NewTokenService(cfg.JWTSecret)
	if err != nil {
		slog.Error("failed to initialize token service", "error", err)
		os.Exit(1)
	}
	slog.Info("token service initialized (Ed25519)")

	// Warn about missing webhook secrets
	if cfg.AllegroWebhookSecret == "" {
		slog.Warn("ALLEGRO_WEBHOOK_SECRET is empty — Allegro webhook signature verification is DISABLED")
	}
	if cfg.InPostWebhookSecret == "" {
		slog.Warn("INPOST_WEBHOOK_SECRET is empty — InPost webhook signature verification is DISABLED")
	}

	// Initialize services
	passwordSvc := service.NewPasswordService()

	tenantRepo := repository.NewTenantRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	auditRepo := repository.NewAuditRepository()
	orderRepo := repository.NewOrderRepository()
	shipmentRepo := repository.NewShipmentRepository()
	productRepo := repository.NewProductRepository()
	integrationRepo := repository.NewIntegrationRepository()
	webhookRepo := repository.NewWebhookRepository()
	webhookDeliveryRepo := repository.NewWebhookDeliveryRepository()
	statsRepo := repository.NewStatsRepository()

	orderGroupRepo := repository.NewOrderGroupRepository()
	bundleRepo := repository.NewBundleRepository()
	returnRepo := repository.NewReturnRepository()
	invoiceRepo := repository.NewInvoiceRepository()
	supplierRepo := repository.NewSupplierRepository()
	supplierProductRepo := repository.NewSupplierProductRepository()
	variantRepo := repository.NewVariantRepository()
	syncJobRepo := repository.NewSyncJobRepository()
	warehouseRepo := repository.NewWarehouseRepository()
	warehouseStockRepo := repository.NewWarehouseStockRepository()
	customerRepo := repository.NewCustomerRepository()
	priceListRepo := repository.NewPriceListRepository()
	warehouseDocRepo := repository.NewWarehouseDocumentRepository()
	warehouseDocItemRepo := repository.NewWarehouseDocItemRepository()
	exchangeRateRepo := repository.NewExchangeRateRepository()
	roleRepo := repository.NewRoleRepository()
	stocktakeRepo := repository.NewStocktakeRepository()
	stocktakeItemRepo := repository.NewStocktakeItemRepository()
	purchaseOrderRepo := repository.NewPurchaseOrderRepository()
	purchaseOrderItemRepo := repository.NewPurchaseOrderItemRepository()
	pickPackRepo := repository.NewPickPackRepository()
	dropshipRepo := repository.NewDropshipOrderRepository()
	dropshipItemRepo := repository.NewDropshipOrderItemRepository()
	recurringOrderRepo := repository.NewRecurringOrderRepository()

	authService := service.NewAuthService(userRepo, tenantRepo, auditRepo, tokenSvc, passwordSvc, pool, encryptionKey)

	// Login lockout (per-account brute-force protection)
	if redisClient != nil {
		lockoutStore := service.NewRedisLoginLockoutStore(redisClient)
		authService.SetLoginLockout(service.NewLoginLockout(lockoutStore))
		slog.Info("using Redis login lockout")
	} else {
		lockoutStore := service.NewMemoryLoginLockoutStore()
		authService.SetLoginLockout(service.NewLoginLockout(lockoutStore))
		slog.Info("using in-memory login lockout")
	}

	userService := service.NewUserService(userRepo, auditRepo, passwordSvc, pool)
	roleService := service.NewRoleService(roleRepo, auditRepo, pool)
	emailService := service.NewEmailService(tenantRepo, pool)
	smsService := service.NewSMSService(tenantRepo, pool)
	webhookDispatchService := service.NewWebhookDispatchService(tenantRepo, webhookDeliveryRepo, pool)
	orderService := service.NewOrderService(orderRepo, auditRepo, tenantRepo, pool, emailService, webhookDispatchService)
	returnService := service.NewReturnService(returnRepo, orderRepo, auditRepo, pool, webhookDispatchService)
	shipmentService := service.NewShipmentService(shipmentRepo, orderRepo, auditRepo, tenantRepo, pool, webhookDispatchService)
	productService := service.NewProductService(productRepo, auditRepo, pool, webhookDispatchService)
	integrationService := service.NewIntegrationService(integrationRepo, auditRepo, pool, encryptionKey)
	labelService := service.NewLabelService(
		shipmentRepo, orderRepo, integrationRepo, auditRepo,
		pool, encryptionKey, cfg.UploadDir, cfg.BaseURL,
	)
	webhookService := service.NewWebhookService(webhookRepo, pool, cfg.AllegroWebhookSecret, cfg.InPostWebhookSecret)
	statsService := service.NewStatsService(statsRepo, pool)
	invoiceService := service.NewInvoiceService(invoiceRepo, orderRepo, tenantRepo, auditRepo, pool, encryptionKey)
	orderService.SetInvoiceService(invoiceService)
	orderService.SetSMSService(smsService)
	orderService.SetShipmentService(shipmentService)
	shipmentService.SetSMSService(smsService)
	allegroSyncService := service.NewAllegroSyncService(integrationService)
	orderService.SetAllegroSyncService(allegroSyncService)
	shipmentService.SetAllegroSyncService(allegroSyncService)
	supplierService := service.NewSupplierService(supplierRepo, supplierProductRepo, auditRepo, pool, webhookDispatchService, integrationService, slog.Default())
	variantService := service.NewVariantService(variantRepo, productRepo, auditRepo, pool)
	warehouseService := service.NewWarehouseService(warehouseRepo, warehouseStockRepo, auditRepo, tenantRepo, pool)
	orderGroupService := service.NewOrderGroupService(orderGroupRepo, orderRepo, auditRepo, pool)
	bundleService := service.NewBundleService(bundleRepo, productRepo, auditRepo, pool)
	customerService := service.NewCustomerService(customerRepo, auditRepo, pool, webhookDispatchService, slog.Default())
	barcodeService := service.NewBarcodeService(productRepo, variantRepo, orderRepo, auditRepo, pool)
	priceListService := service.NewPriceListService(priceListRepo, productRepo, auditRepo, pool)
	warehouseDocService := service.NewWarehouseDocumentService(warehouseDocRepo, warehouseDocItemRepo, warehouseStockRepo, auditRepo, pool)
	exchangeRateService := service.NewExchangeRateService(exchangeRateRepo, auditRepo, pool)
	ksefService := service.NewKSeFService(invoiceRepo, orderRepo, tenantRepo, auditRepo, pool)
	stocktakeService := service.NewStocktakeService(stocktakeRepo, stocktakeItemRepo, warehouseStockRepo, warehouseDocRepo, warehouseDocItemRepo, auditRepo, pool, webhookDispatchService)
	purchaseOrderService := service.NewPurchaseOrderService(purchaseOrderRepo, purchaseOrderItemRepo, warehouseStockRepo, auditRepo, pool, webhookDispatchService, slog.Default())
	pickPackService := service.NewPickPackService(pickPackRepo, orderRepo, productRepo, variantRepo, auditRepo, pool)
	dropshipService := service.NewDropshipService(dropshipRepo, dropshipItemRepo, orderRepo, productRepo, supplierRepo, auditRepo, pool, webhookDispatchService, slog.Default())
	recurringOrderService := service.NewRecurringOrderService(recurringOrderRepo, orderRepo, auditRepo, pool, webhookDispatchService, slog.Default())

	// Product listing repo (needed by both stock sync and allegro listings)
	productListingRepo := repository.NewProductListingRepository()

	// Stock Sync
	stockSyncChannelRepo := repository.NewStockSyncChannelRepository()
	stockSyncEventRepo := repository.NewStockSyncEventRepository()
	stockSyncService := service.NewStockSyncService(stockSyncChannelRepo, stockSyncEventRepo, productRepo, auditRepo, productListingRepo, integrationRepo, pool, webhookDispatchService, encryptionKey, slog.Default())

	// Wire stock sync into services (setter pattern)
	orderService.SetStockSyncService(stockSyncService)
	orderService.SetWarehouseStockRepo(warehouseStockRepo)
	warehouseService.SetStockSyncService(stockSyncService)
	stocktakeService.SetStockSyncService(stockSyncService)

	// Segment & Loyalty
	segmentRepo := repository.NewCustomerSegmentRepository()
	loyaltyRepo := repository.NewLoyaltyRepository()
	segmentService := service.NewSegmentService(segmentRepo, auditRepo, pool, slog.Default())
	loyaltyService := service.NewLoyaltyService(loyaltyRepo, auditRepo, pool, slog.Default())

	// Automation engine
	automationRuleRepo := repository.NewAutomationRuleRepository()
	automationRuleLogRepo := repository.NewAutomationRuleLogRepository()
	delayedActionRepo := repository.NewDelayedActionRepository()
	automationExecutor := automation.NewDefaultActionExecutor(slog.Default())
	automationExecutor.SetOrderServices(orderService, orderService, orderRepo, pool)
	automationExecutor.SetEmailSender(emailService)
	automationExecutor.SetInvoiceCreator(invoiceService)
	automationEngine := automation.NewEngine(automationRuleRepo, automationRuleLogRepo, pool, automationExecutor, slog.Default())
	automationEngine.SetDelayedActionRepo(delayedActionRepo)
	automationService := service.NewAutomationService(automationRuleRepo, automationRuleLogRepo, pool, automationEngine, slog.Default())
	automationService.SetDelayedActionRepo(delayedActionRepo)

	// Wire automation service into entity services (setter pattern to avoid circular dependency)
	orderService.SetAutomationService(automationService)
	shipmentService.SetAutomationService(automationService)
	returnService.SetAutomationService(automationService)
	productService.SetAutomationService(automationService)

	// Invitation service (for invite-only registration mode)
	invitationRepo := repository.NewInvitationRepository()
	invitationService := service.NewInvitationService(invitationRepo, auditRepo, pool)

	// Initialize token blacklist for server-side token revocation
	var tokenBlacklist *middleware.TokenBlacklist
	if redisClient != nil {
		tokenBlacklist = middleware.NewTokenBlacklistWithStore(middleware.NewRedisTokenBlacklist(redisClient))
		slog.Info("using Redis token blacklist")
	} else {
		tokenBlacklist = middleware.NewTokenBlacklist()
		slog.Info("using in-memory token blacklist")
	}

	// WebSocket ticket service (short-lived single-use tickets for WS connections)
	var wsTicketSvc *service.WSTicketService
	if redisClient != nil {
		wsTicketSvc = service.NewWSTicketService(service.NewRedisWSTicketStore(redisClient))
		slog.Info("using Redis WS ticket store")
	} else {
		wsTicketSvc = service.NewWSTicketService(service.NewMemoryWSTicketStore())
		slog.Info("using in-memory WS ticket store")
	}

	// Initialize rate limiter
	var rateLimiter middleware.RateLimiter
	if redisClient != nil {
		rateLimiter = middleware.NewRedisRateLimiter(redisClient)
		slog.Info("using Redis rate limiter")
	} else {
		rateLimiter = middleware.NewMemoryRateLimiter()
		slog.Info("using in-memory rate limiter")
	}

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService, cfg.IsDevelopment(), tokenBlacklist)
	authHandler.SetRegistrationMode(cfg.RegistrationMode)
	authHandler.SetInvitationService(invitationService)
	authHandler.SetWSTicketService(wsTicketSvc)
	userHandler := handler.NewUserHandler(userService)
	orderHandler := handler.NewOrderHandler(orderService, tenantRepo, pool)
	shipmentHandler := handler.NewShipmentHandler(shipmentService, labelService)
	productImportService := service.NewProductImportService(productRepo, auditRepo, pool)
	productHandler := handler.NewProductHandler(productService, productImportService)
	integrationHandler := handler.NewIntegrationHandler(integrationService, integrationRepo, pool)
	returnHandler := handler.NewReturnHandler(returnService)
	webhookHandler := handler.NewWebhookHandler(webhookService)
	statsHandler := handler.NewStatsHandler(statsService)
	uploadHandler := handler.NewUploadHandler(objectStorage, cfg.MaxUploadSize)
	settingsHandler := handler.NewSettingsHandler(tenantRepo, auditRepo, emailService, smsService, pool)
	auditHandler := handler.NewAuditHandler(auditRepo, pool)
	webhookDeliveryHandler := handler.NewWebhookDeliveryHandler(webhookDeliveryRepo, pool)

	// InPost point search proxy
	inpostClient := inpost.NewClient(cfg.InPostAPIToken, cfg.InPostOrgID)
	inpostPointHandler := handler.NewInPostPointHandler(inpostClient)

	// Allegro OAuth handler
	var oauthStateStore handler.OAuthStateStore
	if redisClient != nil {
		oauthStateStore = handler.NewRedisOAuthStateStore(redisClient)
		slog.Info("using Redis OAuth state store")
	} else {
		oauthStateStore = handler.NewMemoryOAuthStateStore()
		slog.Info("using in-memory OAuth state store")
	}
	allegroAuthHandler := handler.NewAllegroAuthHandler(cfg, integrationService, encryptionKey, oauthStateStore)

	// Allegro fulfillment + tracking handler (Batch 1)
	allegroHandler := handler.NewAllegroHandler(integrationService, orderService, encryptionKey)

	// Allegro shipment management handler ("Wysyłam z Allegro")
	allegroShipmentHandler := handler.NewAllegroShipmentHandler(integrationService, shipmentService, orderRepo, shipmentRepo, pool, encryptionKey)

	// Allegro communications handler (messaging, returns, refunds)
	allegroCommsHandler := handler.NewAllegroCommsHandler(integrationService, encryptionKey)

	// Allegro webhook syncer — handles on-demand order import/update triggered by webhooks
	allegroWebhookSyncer := worker.NewAllegroWebhookSyncer(workerPool, encryptionKey, orderRepo, shipmentRepo, auditRepo, slog.Default())

	// Allegro webhook handler (public endpoint, HMAC-verified)
	allegroWebhookHandler := handler.NewAllegroWebhookHandler(cfg.AllegroWebhookSecret, allegroWebhookSyncer)

	// InPost webhook handler (public endpoint, HMAC-verified)
	inpostWebhookHandler := handler.NewInPostWebhookHandler(cfg.InPostWebhookSecret)

	// Allegro account & offers handler
	allegroAccountHandler := handler.NewAllegroAccountHandler(integrationService, encryptionKey)

	// Allegro listings handler (publish products to Allegro)
	allegroListingsHandler := handler.NewAllegroListingsHandler(integrationService, productService, productListingRepo, encryptionKey, pool, cfg)

	// Allegro catalog + finance handler
	allegroCatalogHandler := handler.NewAllegroCatalogHandler(integrationService, encryptionKey)

	// Allegro after-sales policies handler (return policies, warranties, size tables)
	allegroPoliciesHandler := handler.NewAllegroPoliciesHandler(integrationService, encryptionKey)

	// Allegro promotions handler
	allegroPromotionsHandler := handler.NewAllegroPromotionsHandler(integrationService, encryptionKey)

	// Allegro delivery settings handler
	allegroDeliveryHandler := handler.NewAllegroDeliveryHandler(integrationService, encryptionKey)

	// Allegro disputes handler
	allegroDisputesHandler := handler.NewAllegroDisputesHandler(integrationService, encryptionKey)

	// Allegro ratings handler
	allegroRatingsHandler := handler.NewAllegroRatingsHandler(integrationService, encryptionKey)

	// Amazon auth handler
	amazonAuthHandler := handler.NewAmazonAuthHandler(integrationService, encryptionKey)

	// Store platform auth handler (Shoper, PrestaShop, Shopify)
	storeAuthHandler := handler.NewStoreAuthHandler(integrationService, encryptionKey)

	// Invoice handler
	invoiceHandler := handler.NewInvoiceHandler(invoiceService)

	// KSeF handler
	ksefHandler := handler.NewKSeFHandler(ksefService)

	// Accounting handler (wFirma, inFakt provider settings)
	accountingHandler := handler.NewAccountingHandler(tenantRepo, auditRepo, pool)

	// Supplier handler
	supplierHandler := handler.NewSupplierHandler(supplierService)

	// Import service & handler
	importService := service.NewImportService(orderRepo, auditRepo, tenantRepo, pool)
	importHandler := handler.NewImportHandler(importService)

	// Automation handler
	automationHandler := handler.NewAutomationHandler(automationService)

	// Workflow builder handler
	workflowHandler := handler.NewWorkflowHandler(automationService)

	// Variant handler
	variantHandler := handler.NewVariantHandler(variantService)

	// Warehouse handler
	warehouseHandler := handler.NewWarehouseHandler(warehouseService)

	// Customer handler
	customerHandler := handler.NewCustomerHandler(customerService)

	// Order group handler
	orderGroupHandler := handler.NewOrderGroupHandler(orderGroupService)

	// Bundle handler
	bundleHandler := handler.NewBundleHandler(bundleService)

	// Barcode handler
	barcodeHandler := handler.NewBarcodeHandler(barcodeService)

	// Price list handler
	priceListHandler := handler.NewPriceListHandler(priceListService)

	// Warehouse document handler
	warehouseDocHandler := handler.NewWarehouseDocumentHandler(warehouseDocService)

	// WebSocket hub and handler
	wsHub := ws.NewHub()
	wsCtx, wsCancel := context.WithCancel(context.Background())
	go wsHub.Run(wsCtx)
	wsHandler := handler.NewWSHandler(wsHub, tokenSvc, tokenBlacklist, wsTicketSvc, cfg.FrontendURL)

	// Wire hub into webhook dispatch service for real-time events
	webhookDispatchService.SetWSBroadcast(func(tenantID uuid.UUID, eventType string, payload any) {
		wsHub.BroadcastToTenant(tenantID, ws.Event{Type: eventType, Payload: payload})
	})

	// AI service & handler (Phase 33)
	aiService := service.NewAIService(cfg.OpenAIAPIKey, cfg.OpenAIModel, productRepo, tenantRepo, pool)
	aiHandler := handler.NewAIHandler(aiService)
	if cfg.OpenAIAPIKey != "" {
		slog.Info("AI auto-categorization enabled", "model", cfg.OpenAIModel)
	}

	// Background removal service & handler
	bgRemovalService := service.NewBGRemovalService(cfg.RemoveBGAPIKey)
	bgRemovalHandler := handler.NewBGRemovalHandler(bgRemovalService, objectStorage, productRepo, pool, cfg.MaxUploadSize)
	if cfg.RemoveBGAPIKey != "" {
		slog.Info("background removal enabled (remove.bg)")
	}

	// Demand forecast service & handler
	forecastService := service.NewForecastService(pool, productRepo, tenantRepo, orderRepo, supplierRepo)
	forecastHandler := handler.NewForecastHandler(forecastService)

	// Mailchimp marketing service & handler (Phase 34)
	mailchimpService := service.NewMailchimpService(tenantRepo, customerRepo, pool, slog.Default())
	marketingHandler := handler.NewMarketingHandler(mailchimpService)

	// Freshdesk helpdesk service & handler (Phase 34)
	freshdeskService := service.NewFreshdeskService(tenantRepo, orderRepo, pool, slog.Default())
	helpdeskHandler := handler.NewHelpdeskHandler(freshdeskService)

	// Public return handler (Phase 29)
	publicReturnHandler := handler.NewPublicReturnHandler(pool, returnRepo, orderRepo)

	// Exchange rate handler (Phase 30)
	exchangeRateHandler := handler.NewExchangeRateHandler(exchangeRateService)

	// Role handler (Phase 31 — RBAC)
	roleHandler := handler.NewRoleHandler(roleService)

	// Stocktake handler (inventory counting)
	stocktakeHandler := handler.NewStocktakeHandler(stocktakeService)

	// Purchase order handler
	purchaseOrderHandler := handler.NewPurchaseOrderHandler(purchaseOrderService)

	// Dropship handler
	dropshipHandler := handler.NewDropshipHandler(dropshipService)

	// Recurring order handler
	recurringOrderHandler := handler.NewRecurringOrderHandler(recurringOrderService)

	// Segment & Loyalty handlers
	segmentHandler := handler.NewSegmentHandler(segmentService)
	loyaltyHandler := handler.NewLoyaltyHandler(loyaltyService)

	// Stock Sync handler
	stockSyncHandler := handler.NewStockSyncHandler(stockSyncService)

	// Pick & Pack handler
	pickPackHandler := handler.NewPickPackHandler(pickPackService)

	// Print handler
	printHandler := handler.NewPrintHandler(tenantRepo, orderRepo, returnRepo, pool)

	// Sync job handler
	syncJobHandler := handler.NewSyncJobHandler(syncJobRepo, pool)

	// OpenAPI docs handler
	docsHandler := handler.NewDocsHandler(docs.OpenAPISpec)

	// Public config handler
	configHandler := handler.NewConfigHandler(cfg.RegistrationMode)

	// Invitation handler (admin CRUD for invitations)
	invitationHandler := handler.NewInvitationHandler(invitationService)

	// Rate shopping service & handler
	rateService := service.NewRateService(integrationRepo, pool, encryptionKey)
	rateHandler := handler.NewRateHandler(rateService)

	// Public order tracking service & handler
	trackingService := service.NewTrackingService(tenantRepo, orderRepo, shipmentRepo, auditRepo, pool)
	trackingHandler := handler.NewTrackingHandler(trackingService)

	// Product feed service & handler (Ceneo, Google Shopping)
	feedService := service.NewFeedService(tenantRepo, productRepo, pool, cfg.BaseURL)
	feedHandler := handler.NewFeedHandler(feedService)

	// Carbon footprint service & handler
	carbonService := service.NewCarbonService(pool)
	carbonHandler := handler.NewCarbonHandler(carbonService)

	// Supplier portal service & handler
	supplierPortalTokenRepo := repository.NewSupplierPortalTokenRepository()
	supplierMessageRepo := repository.NewSupplierMessageRepository()
	supplierPortalService := service.NewSupplierPortalService(
		supplierPortalTokenRepo, supplierMessageRepo, supplierRepo,
		purchaseOrderRepo, purchaseOrderItemRepo, auditRepo,
		pool, cfg.FrontendURL, slog.Default(),
	)
	supplierPortalHandler := handler.NewSupplierPortalHandler(supplierPortalService)

	// VAT OSS service & handler
	vatOSSService := service.NewVATOSSService(tenantRepo, pool)
	vatOSSHandler := handler.NewVATOSSHandler(vatOSSService)

	// Payment reconciliation service & handler
	paymentRepo := repository.NewPaymentRepository()
	reconciliationService := service.NewReconciliationService(paymentRepo, orderRepo, auditRepo, pool)
	reconciliationHandler := handler.NewReconciliationHandler(reconciliationService)

	// Repricing engine service & handler
	repricingRepo := repository.NewRepricingRepository()
	repricingService := service.NewRepricingService(repricingRepo, productRepo, auditRepo, pool, slog.Default())
	repricingHandler := handler.NewRepricingHandler(repricingService)

	// Listing sync service & handler
	listingSyncRepo := repository.NewListingSyncRepository()
	listingSyncService := service.NewListingSyncService(listingSyncRepo, productRepo, productListingRepo, auditRepo, pool, slog.Default())
	listingSyncHandler := handler.NewListingSyncHandler(listingSyncService)

	// Prometheus metrics collector
	metricsCollector := middleware.NewMetricsCollector()

	// Setup router
	r := router.New(router.RouterDeps{
		Pool:              pool,
		Config:            cfg,
		TokenSvc:          tokenSvc,
		TokenBlacklist:    tokenBlacklist,
		RateLimiter:       rateLimiter,
		Auth:              authHandler,
		User:              userHandler,
		Order:             orderHandler,
		Shipment:          shipmentHandler,
		Product:           productHandler,
		Integration:       integrationHandler,
		Webhook:           webhookHandler,
		Stats:             statsHandler,
		Upload:            uploadHandler,
		Settings:          settingsHandler,
		Audit:             auditHandler,
		WebhookDelivery:   webhookDeliveryHandler,
		Return:            returnHandler,
		InPostPoint:       inpostPointHandler,
		AllegroAuth:       allegroAuthHandler,
		Allegro:           allegroHandler,
		AllegroShipment:   allegroShipmentHandler,
		AmazonAuth:        amazonAuthHandler,
		StoreAuth:         storeAuthHandler,
		Supplier:          supplierHandler,
		Invoice:           invoiceHandler,
		Automation:        automationHandler,
		Workflow:          workflowHandler,
		Import:            importHandler,
		Variant:           variantHandler,
		SyncJob:           syncJobHandler,
		Warehouse:         warehouseHandler,
		Customer:          customerHandler,
		Print:             printHandler,
		Docs:              docsHandler,
		MetricsCollector:  metricsCollector,
		OrderGroup:        orderGroupHandler,
		Bundle:            bundleHandler,
		Barcode:           barcodeHandler,
		PriceList:         priceListHandler,
		WarehouseDocument: warehouseDocHandler,
		WS:                wsHandler,
		AI:                aiHandler,
		Marketing:         marketingHandler,
		Helpdesk:          helpdeskHandler,
		PublicReturn:      publicReturnHandler,
		ExchangeRate:      exchangeRateHandler,
		Role:              roleHandler,
		RoleService:       roleService,
		Stocktake:         stocktakeHandler,
		KSeF:              ksefHandler,
		Rate:              rateHandler,
		AllegroComms:      allegroCommsHandler,
		AllegroWebhook:    allegroWebhookHandler,
		InPostWebhook:     inpostWebhookHandler,
		AllegroAccount:    allegroAccountHandler,
		AllegroCatalog:    allegroCatalogHandler,
		AllegroPolicies:   allegroPoliciesHandler,
		AllegroPromotions: allegroPromotionsHandler,
		AllegroDelivery:   allegroDeliveryHandler,
		AllegroDisputes:   allegroDisputesHandler,
		AllegroRatings:    allegroRatingsHandler,
		AllegroListings:   allegroListingsHandler,
		Tracking:          trackingHandler,
		Feed:              feedHandler,
		PurchaseOrder:     purchaseOrderHandler,
		PickPack:          pickPackHandler,
		Accounting:        accountingHandler,
		Reconciliation:    reconciliationHandler,
		Carbon:            carbonHandler,
		VATOSS:            vatOSSHandler,
		Dropship:          dropshipHandler,
		SupplierPortal:    supplierPortalHandler,
		RecurringOrder:    recurringOrderHandler,
		Forecast:          forecastHandler,
		Repricing:         repricingHandler,
		BGRemoval:         bgRemovalHandler,
		Segment:           segmentHandler,
		Loyalty:           loyaltyHandler,
		StockSync:         stockSyncHandler,
		ListingSync:       listingSyncHandler,
		PublicConfig:      configHandler,
		Invitation:        invitationHandler,
	})

	// Start background workers (use workerPool for cross-tenant queries)
	workerMgr := worker.NewManager(workerPool, slog.Default())
	workerMgr.Register(worker.NewOAuthRefresher(workerPool, encryptionKey, slog.Default()))
	workerMgr.Register(worker.NewAllegroOrderPoller(workerPool, encryptionKey, orderRepo, shipmentRepo, auditRepo, labelService, slog.Default()))
	workerMgr.Register(worker.NewStockSyncWorker(workerPool, encryptionKey, slog.Default()))
	workerMgr.Register(worker.NewTrackingPoller(workerPool, encryptionKey, shipmentRepo, shipmentService, slog.Default()))
	workerMgr.Register(worker.NewAmazonOrderPoller(workerPool, encryptionKey, orderRepo, shipmentRepo, auditRepo, slog.Default()))
	workerMgr.Register(worker.NewWooCommerceOrderPoller(workerPool, encryptionKey, orderRepo, shipmentRepo, auditRepo, slog.Default()))
	workerMgr.Register(worker.NewShoperOrderPoller(workerPool, encryptionKey, orderRepo, shipmentRepo, auditRepo, slog.Default()))
	workerMgr.Register(worker.NewPrestaShopOrderPoller(workerPool, encryptionKey, orderRepo, shipmentRepo, auditRepo, slog.Default()))
	workerMgr.Register(worker.NewShopifyOrderPoller(workerPool, encryptionKey, orderRepo, shipmentRepo, auditRepo, slog.Default()))
	workerMgr.Register(worker.NewSupplierSyncWorker(workerPool, supplierService, slog.Default()))
	workerMgr.Register(worker.NewExchangeRateWorker(workerPool, exchangeRateService, slog.Default()))
	workerMgr.Register(worker.NewKSeFStatusWorker(workerPool, ksefService, slog.Default()))
	workerMgr.Register(worker.NewDelayedActionWorker(workerPool, delayedActionRepo, automationExecutor, slog.Default()))
	workerMgr.Register(worker.NewRecurringOrderWorker(workerPool, recurringOrderService, slog.Default()))
	workerMgr.Register(worker.NewRepricingWorker(workerPool, repricingService, slog.Default()))
	workerMgr.Register(worker.NewListingSyncWorker(workerPool, listingSyncRepo, listingSyncService, slog.Default()))
	if cfg.WorkersEnabled {
		go workerMgr.Start(context.Background())
	}

	// Start server
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
	}

	go func() {
		slog.Info("HTTP server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("shutdown signal received", "signal", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}
	wsCancel()
	workerMgr.Stop()
	slog.Info("server stopped")
}
