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

	// Register marketplace providers via init().
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/allegro"
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/amazon"
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/erli"
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/mirakl"
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/woocommerce"
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/ebay"
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/kaufland"
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/olx"
	// Register carrier providers via init().
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/carriers"
	// Register invoicing providers via init().
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/fakturownia"

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

	authService := service.NewAuthService(userRepo, tenantRepo, auditRepo, tokenSvc, passwordSvc, pool)
	userService := service.NewUserService(userRepo, auditRepo, passwordSvc, pool)
	roleService := service.NewRoleService(roleRepo, auditRepo, pool)
	emailService := service.NewEmailService(tenantRepo, pool)
	smsService := service.NewSMSService(tenantRepo, pool)
	webhookDispatchService := service.NewWebhookDispatchService(tenantRepo, webhookDeliveryRepo, pool)
	orderService := service.NewOrderService(orderRepo, auditRepo, tenantRepo, pool, emailService, webhookDispatchService)
	returnService := service.NewReturnService(returnRepo, orderRepo, auditRepo, pool, webhookDispatchService)
	shipmentService := service.NewShipmentService(shipmentRepo, orderRepo, auditRepo, pool, webhookDispatchService)
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
	shipmentService.SetSMSService(smsService)
	supplierService := service.NewSupplierService(supplierRepo, supplierProductRepo, auditRepo, pool, webhookDispatchService, slog.Default())
	variantService := service.NewVariantService(variantRepo, productRepo, auditRepo, pool)
	warehouseService := service.NewWarehouseService(warehouseRepo, warehouseStockRepo, auditRepo, pool)
	orderGroupService := service.NewOrderGroupService(orderGroupRepo, orderRepo, auditRepo, pool)
	bundleService := service.NewBundleService(bundleRepo, productRepo, auditRepo, pool)
	customerService := service.NewCustomerService(customerRepo, auditRepo, pool, webhookDispatchService, slog.Default())
	barcodeService := service.NewBarcodeService(productRepo, variantRepo, orderRepo, auditRepo, pool)
	priceListService := service.NewPriceListService(priceListRepo, productRepo, auditRepo, pool)
	warehouseDocService := service.NewWarehouseDocumentService(warehouseDocRepo, warehouseDocItemRepo, warehouseStockRepo, auditRepo, pool)
	exchangeRateService := service.NewExchangeRateService(exchangeRateRepo, auditRepo, pool)
	ksefService := service.NewKSeFService(invoiceRepo, orderRepo, tenantRepo, auditRepo, pool)
	stocktakeService := service.NewStocktakeService(stocktakeRepo, stocktakeItemRepo, warehouseStockRepo, warehouseDocRepo, warehouseDocItemRepo, auditRepo, pool, webhookDispatchService)

	// Automation engine
	automationRuleRepo := repository.NewAutomationRuleRepository()
	automationRuleLogRepo := repository.NewAutomationRuleLogRepository()
	delayedActionRepo := repository.NewDelayedActionRepository()
	automationExecutor := automation.NewDefaultActionExecutor(slog.Default())
	automationEngine := automation.NewEngine(automationRuleRepo, automationRuleLogRepo, pool, automationExecutor, slog.Default())
	automationEngine.SetDelayedActionRepo(delayedActionRepo)
	automationService := service.NewAutomationService(automationRuleRepo, automationRuleLogRepo, pool, automationEngine, slog.Default())
	automationService.SetDelayedActionRepo(delayedActionRepo)

	// Wire automation service into entity services (setter pattern to avoid circular dependency)
	orderService.SetAutomationService(automationService)
	shipmentService.SetAutomationService(automationService)
	returnService.SetAutomationService(automationService)
	productService.SetAutomationService(automationService)

	// Initialize token blacklist for server-side token revocation
	tokenBlacklist := middleware.NewTokenBlacklist()

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService, cfg.IsDevelopment(), tokenBlacklist)
	userHandler := handler.NewUserHandler(userService)
	orderHandler := handler.NewOrderHandler(orderService, tenantRepo, pool)
	shipmentHandler := handler.NewShipmentHandler(shipmentService, labelService)
	productHandler := handler.NewProductHandler(productService)
	integrationHandler := handler.NewIntegrationHandler(integrationService)
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
	allegroAuthHandler := handler.NewAllegroAuthHandler(cfg, integrationService, encryptionKey)

	// Amazon auth handler
	amazonAuthHandler := handler.NewAmazonAuthHandler(integrationService, encryptionKey)

	// Invoice handler
	invoiceHandler := handler.NewInvoiceHandler(invoiceService)

	// KSeF handler
	ksefHandler := handler.NewKSeFHandler(ksefService)

	// Supplier handler
	supplierHandler := handler.NewSupplierHandler(supplierService)

	// Import service & handler
	importService := service.NewImportService(orderRepo, auditRepo, pool)
	importHandler := handler.NewImportHandler(importService)

	// Automation handler
	automationHandler := handler.NewAutomationHandler(automationService)

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
	go wsHub.Run()
	wsHandler := handler.NewWSHandler(wsHub, tokenSvc)

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

	// Print handler
	printHandler := handler.NewPrintHandler(tenantRepo, orderRepo, returnRepo, pool)

	// Sync job handler
	syncJobHandler := handler.NewSyncJobHandler(syncJobRepo, pool)

	// OpenAPI docs handler
	docsHandler := handler.NewDocsHandler(docs.OpenAPISpec)

	// Prometheus metrics collector
	metricsCollector := middleware.NewMetricsCollector()

	// Setup router
	r := router.New(router.RouterDeps{
		Pool:            pool,
		Config:          cfg,
		TokenSvc:        tokenSvc,
		TokenBlacklist:  tokenBlacklist,
		Auth:            authHandler,
		User:            userHandler,
		Order:           orderHandler,
		Shipment:        shipmentHandler,
		Product:         productHandler,
		Integration:     integrationHandler,
		Webhook:         webhookHandler,
		Stats:           statsHandler,
		Upload:          uploadHandler,
		Settings:        settingsHandler,
		Audit:           auditHandler,
		WebhookDelivery: webhookDeliveryHandler,
		Return:          returnHandler,
		InPostPoint:     inpostPointHandler,
		AllegroAuth:     allegroAuthHandler,
		AmazonAuth:      amazonAuthHandler,
		Supplier:         supplierHandler,
		Invoice:          invoiceHandler,
		Automation:       automationHandler,
		Import:           importHandler,
		Variant:          variantHandler,
		SyncJob:          syncJobHandler,
		Warehouse:        warehouseHandler,
		Customer:         customerHandler,
		Print:            printHandler,
		Docs:             docsHandler,
		MetricsCollector: metricsCollector,
		OrderGroup:       orderGroupHandler,
		Bundle:           bundleHandler,
		Barcode:          barcodeHandler,
		PriceList:          priceListHandler,
		WarehouseDocument:  warehouseDocHandler,
		WS:                 wsHandler,
		AI:                 aiHandler,
		Marketing:          marketingHandler,
		Helpdesk:           helpdeskHandler,
		PublicReturn:       publicReturnHandler,
		ExchangeRate:       exchangeRateHandler,
		Role:               roleHandler,
		RoleService:        roleService,
		Stocktake:          stocktakeHandler,
		KSeF:               ksefHandler,
	})

	// Start background workers
	workerMgr := worker.NewManager(pool, slog.Default())
	workerMgr.Register(worker.NewOAuthRefresher(pool, encryptionKey, slog.Default()))
	workerMgr.Register(worker.NewAllegroOrderPoller(pool, encryptionKey, orderRepo, slog.Default()))
	workerMgr.Register(worker.NewStockSyncWorker(pool, encryptionKey, slog.Default()))
	workerMgr.Register(worker.NewTrackingPoller(pool, encryptionKey, shipmentRepo, slog.Default()))
	workerMgr.Register(worker.NewAmazonOrderPoller(pool, encryptionKey, orderRepo, slog.Default()))
	workerMgr.Register(worker.NewWooCommerceOrderPoller(pool, encryptionKey, orderRepo, slog.Default()))
	workerMgr.Register(worker.NewSupplierSyncWorker(pool, supplierService, slog.Default()))
	workerMgr.Register(worker.NewExchangeRateWorker(pool, exchangeRateService, slog.Default()))
	workerMgr.Register(worker.NewKSeFStatusWorker(pool, ksefService, slog.Default()))
	workerMgr.Register(worker.NewDelayedActionWorker(pool, delayedActionRepo, automationExecutor, slog.Default()))
	if cfg.WorkersEnabled {
		go workerMgr.Start(context.Background())
	}

	// Start server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
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
	workerMgr.Stop()
	slog.Info("server stopped")
}
