// Package main is the entry point for the api-server.
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/netutil"
	"github.com/openoms-org/openoms/apps/api-server/internal/obsmetrics"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/router"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
	"github.com/openoms-org/openoms/apps/api-server/internal/storage"
	"github.com/openoms-org/openoms/apps/api-server/internal/worker"
	"github.com/openoms-org/openoms/apps/api-server/internal/ws"
	inpost "github.com/openoms-org/openoms/packages/inpost-go-sdk"
	stripe "github.com/stripe/stripe-go/v82"
)

const redisConnectTimeout = 5 * time.Second

const (
	// startupConnectAttempts / startupConnectBaseDelay bound the retry of the
	// initial database pool connects. A brief pooler blip (the case we care about)
	// fails fast — a TCP reset, not the full ConnectWithOptions 10s ping timeout —
	// so it is absorbed within a retry or two. A genuine outage hits the full 10s
	// timeout each attempt: worst case per pool is 4*10s + (1+2+4)s backoff = 47s,
	// which stays under the pod's ~60s startup-probe budget so run() still returns
	// a logged error (rather than the probe killing the pod mid-retry) and a real
	// outage is not masked. Do not raise attempts without re-checking that bound.
	startupConnectAttempts  = 4
	startupConnectBaseDelay = time.Second
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func setupLogger(cfg *config.Config) {
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
}

func connectRedis(ctx context.Context, cfg *config.Config) (*redis.Client, error) {
	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		if cfg.RequiresRedis() {
			return nil, fmt.Errorf("REDIS_URL is invalid and in-memory state is disabled: %w", err)
		}
		slog.Warn("invalid REDIS_URL, using in-memory state stores", "error", err)
		return nil, nil
	}

	redisClient := redis.NewClient(redisOpts)
	pingCtx, cancel := context.WithTimeout(ctx, redisConnectTimeout)
	defer cancel()
	if err := redisClient.Ping(pingCtx).Err(); err != nil {
		_ = redisClient.Close()
		if cfg.RequiresRedis() {
			return nil, fmt.Errorf("redis is required but unavailable: %w", err)
		}
		slog.Warn("Redis not available, using in-memory state stores", "error", err)
		return nil, nil
	}

	slog.Info("connected to Redis", "addr", redisOpts.Addr)
	return redisClient, nil
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		return fmt.Errorf("failed to load config: %w", err)
	}

	setupLogger(cfg)

	if err := cfg.Validate(); err != nil {
		slog.Error("invalid configuration", "error", err)
		return fmt.Errorf("invalid configuration: %w", err)
	}

	trustedProxyCIDRs, err := cfg.TrustedProxyPrefixes()
	if err != nil {
		slog.Error("failed to parse TRUSTED_PROXY_CIDRS", "error", err)
		return fmt.Errorf("failed to parse TRUSTED_PROXY_CIDRS: %w", err)
	}

	// Redis backs shared auth/session/rate-limit state and worker locks.
	redisClient, err := connectRedis(context.Background(), cfg)
	if err != nil {
		slog.Error("failed to connect to Redis", "error", err)
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	defer func() {
		if redisClient != nil {
			_ = redisClient.Close()
		}
	}()

	// Initialize Sentry error tracking (optional — disabled when DSN is empty)
	if cfg.SentryEnabled() {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              cfg.SentryDSN,
			Environment:      cfg.SentryEnv(),
			Release:          cfg.SentryRelease,
			TracesSampleRate: cfg.SentryTracesSampleRate,
			EnableTracing:    cfg.SentryTracesSampleRate > 0,
			SendDefaultPII:   false,
			BeforeSend:       middleware.ScrubSentryEvent,
		})
		if err != nil {
			slog.Error("failed to initialize Sentry", "error", err)
		} else {
			slog.Info("Sentry initialized", "environment", cfg.SentryEnv())
		}
		defer sentry.Flush(2 * time.Second)
	}

	slog.Info("starting OpenOMS API server", "port", cfg.Port, "env", cfg.Env)

	// Connect to database. Retry transient failures so a brief pooler blip at
	// startup (common during blue-green rollouts when extra pods connect at once)
	// does not crash the pod and abort the deploy's pre-promotion gate.
	pool, err := database.ConnectWithRetry(context.Background(), cfg.DatabaseURL, database.DefaultPoolOptions(), startupConnectAttempts, startupConnectBaseDelay)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer pool.Close()
	slog.Info("connected to PostgreSQL")

	// Worker pool — privileged connection for intentionally cross-tenant queries.
	// Non-development environments must configure this explicitly so app DATABASE_URL
	// can stay RLS-scoped and least-privileged.
	workerDBURL, err := cfg.WorkerDatabaseDSN()
	if err != nil {
		slog.Error("invalid worker database configuration", "error", err)
		return fmt.Errorf("invalid worker database configuration: %w", err)
	}
	workerPool, err := database.ConnectWithRetry(context.Background(), workerDBURL, database.WorkerPoolOptions(), startupConnectAttempts, startupConnectBaseDelay)
	if err != nil {
		slog.Error("failed to connect worker database", "error", err)
		return fmt.Errorf("failed to connect worker database: %w", err)
	}
	defer workerPool.Close()

	// Initialize storage backend
	var objectStorage storage.ObjectStorage
	if cfg.S3Enabled {
		s3Store, err := storage.NewS3Storage(cfg.S3Region, cfg.S3Bucket, cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3PublicURL)
		if err != nil {
			slog.Error("failed to initialize S3 storage", "error", err)
			return fmt.Errorf("failed to initialize S3 storage: %w", err)
		}
		objectStorage = s3Store
		slog.Info("using S3 storage", "bucket", cfg.S3Bucket)
	} else {
		// Create upload directory for local storage
		if err := os.MkdirAll(cfg.UploadDir, 0750); err != nil {
			slog.Error("failed to create upload directory", "error", err)
			return fmt.Errorf("failed to create upload directory: %w", err)
		}
		objectStorage = storage.NewLocalStorage(cfg.UploadDir, cfg.BaseURL)
		slog.Info("using local storage", "dir", cfg.UploadDir)
	}

	// Decode encryption key
	encryptionKey, err := hex.DecodeString(cfg.EncryptionKey)
	if err != nil {
		slog.Error("invalid ENCRYPTION_KEY (must be 64-char hex string)", "error", err)
		return fmt.Errorf("invalid ENCRYPTION_KEY: %w", err)
	}

	// Initialize token service (Ed25519 key derivation)
	tokenSvc, err := service.NewTokenService(cfg.JWTSecret)
	if err != nil {
		slog.Error("failed to initialize token service", "error", err)
		return fmt.Errorf("failed to initialize token service: %w", err)
	}
	slog.Info("token service initialized (Ed25519)")

	// Warn about missing webhook secrets — handlers already reject unsigned requests
	if cfg.AllegroWebhookSecret == "" {
		slog.Warn("ALLEGRO_WEBHOOK_SECRET is empty — Allegro webhook signature verification is DISABLED")
	}
	if cfg.InPostWebhookSecret == "" {
		slog.Warn("INPOST_WEBHOOK_SECRET is empty — InPost webhook signature verification is DISABLED")
	}

	// Initialize services
	passwordSvc := service.NewPasswordService()

	tenantRepo := repository.NewTenantRepository(pool, encryptionKey)
	if cfg.WorkersEnabled {
		backfillCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		encryptedTenants, err := tenantRepo.BackfillSettingsSecretEncryption(backfillCtx, workerPool)
		cancel()
		if err != nil {
			if cfg.IsDevelopment() {
				slog.Warn("tenant settings secret encryption backfill failed", "error", err)
			} else {
				slog.Error("tenant settings secret encryption backfill failed", "error", err)
				return fmt.Errorf("tenant settings secret encryption backfill: %w", err)
			}
		} else if encryptedTenants > 0 {
			slog.Info("encrypted legacy tenant settings secrets", "tenants_updated", encryptedTenants)
		}
	}
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
	productCategoryRepo := repository.NewProductCategoryRepository()
	supplierCategoryMappingRepo := repository.NewSupplierCategoryMappingRepository()
	allegroParameterMappingRepo := repository.NewAllegroParameterMappingRepository()
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

	messageTemplateRepo := repository.NewMessageTemplateRepository()

	authService := service.NewAuthService(userRepo, tenantRepo, auditRepo, tokenSvc, passwordSvc, pool, encryptionKey)
	authService.SetRoleRepo(roleRepo)

	apiTokenRepo := repository.NewAPITokenRepository(pool)
	apiTokenService := service.NewAPITokenService(
		apiTokenRepo,
		repository.TenantUserLookup{Pool: pool, Users: userRepo},
		repository.TenantRoleLookup{Pool: pool, Roles: roleRepo},
	)
	apiTokenHandler := handler.NewAPITokenHandler(apiTokenService)

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

	// Refresh token rotation with reuse detection
	if redisClient != nil {
		refreshStore := service.NewRedisRefreshTokenStore(redisClient)
		authService.SetRefreshTokenStore(refreshStore)
		slog.Info("using Redis refresh token rotation")
	} else {
		refreshStore := service.NewMemoryRefreshTokenStore()
		defer refreshStore.Close()
		authService.SetRefreshTokenStore(refreshStore)
		slog.Info("using in-memory refresh token rotation")
	}

	userService := service.NewUserService(userRepo, auditRepo, passwordSvc, pool)
	roleService := service.NewRoleService(roleRepo, auditRepo, pool)
	// Seed default system roles (Owner/Administrator/Employee) for every newly
	// registered tenant via AuthService.Register (DEAD-02). Best-effort inside Register.
	authService.SetRoleService(roleService)
	// Backfill: ensure system roles for tenants created before the seeding above was wired into
	// registration (OPE-561). Idempotent; gated on WorkersEnabled so only the worker instance runs
	// it, not every API pod.
	if cfg.WorkersEnabled {
		backfillCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		tenantIDs, listErr := tenantRepo.ListAllTenantIDs(backfillCtx, workerPool)
		var processed int
		var ensureErr error
		if listErr == nil {
			processed, ensureErr = roleService.EnsureSystemRolesForAll(backfillCtx, tenantIDs)
		}
		cancel()
		switch {
		case listErr != nil:
			slog.Warn("system roles backfill: list tenants failed", "error", listErr)
		case ensureErr != nil:
			slog.Warn("system roles backfill failed", "error", ensureErr)
		case processed > 0:
			slog.Info("system roles backfill complete", "tenants_checked", processed)
		}
	}
	emailService := service.NewEmailService(tenantRepo, pool)
	smsService := service.NewSMSService(tenantRepo, pool)
	webhookDispatchService := service.NewWebhookDispatchService(tenantRepo, webhookDeliveryRepo, pool)
	// Fulfillment commands (OPE-416): gated bridge from order creation to the
	// fulfillment process + orchestration outbox. A no-op when disabled.
	fulfillmentRepo := repository.NewFulfillmentRepository()
	orchestrationRepo := repository.NewOrchestrationRepository()
	fulfillmentAttemptRepo := repository.NewFulfillmentAttemptRepository()
	// OPE-422: additive, best-effort observability collector for the fulfillment /
	// orchestration / provider-validation paths. Injected (nil-safe) into the
	// services + worker below and registered with the Prometheus MetricsCollector so
	// the existing /metrics handler exposes it. Uses ONLY bounded enum labels.
	fulfillmentMetrics := obsmetrics.NewFulfillmentMetrics()
	// OPE-417: best-effort provider-attempt recording is wired via WithRecording.
	// It stays a no-op until FULFILLMENT_PROCESS_ENABLED is set.
	fulfillmentService := service.NewFulfillmentService(cfg.FulfillmentProcessEnabled, fulfillmentRepo, orchestrationRepo).
		WithRecording(pool, fulfillmentAttemptRepo).
		WithMetrics(fulfillmentMetrics)
	// OPE-418: gated supplier-availability read-model. The supplier sync writes snapshots
	// and dropship routing / stock propagation consult the resolver. A complete no-op
	// (Enabled()==false) until SUPPLIER_AVAILABILITY_ENABLED is set.
	supplierAvailabilityRepo := repository.NewSupplierAvailabilityRepository()
	supplierAvailabilityService := service.NewSupplierAvailabilityService(
		cfg.SupplierAvailabilityEnabled, pool, supplierAvailabilityRepo, auditRepo)
	// OPE-419: tenant-safe operations/fulfillment READ API over the canonical
	// fulfillment model. Always active (read endpoints return empty results until
	// fulfillment data is recorded); reuses the OPE-414..418 repositories.
	// OPE-422: best-effort stuck/blocked gauges + operator-action audit.
	fulfillmentReadService := service.NewFulfillmentReadService(pool, fulfillmentRepo, fulfillmentAttemptRepo, orchestrationRepo).
		WithMetrics(fulfillmentMetrics).
		WithAudit(auditRepo)
	fulfillmentHandler := handler.NewFulfillmentHandler(fulfillmentReadService)
	operationsHandler := handler.NewOperationsHandler(fulfillmentReadService)
	orderService := service.NewOrderService(orderRepo, auditRepo, tenantRepo, pool, emailService, webhookDispatchService, fulfillmentService)
	returnService := service.NewReturnService(returnRepo, orderRepo, auditRepo, pool, webhookDispatchService)
	shipmentService := service.NewShipmentService(shipmentRepo, orderRepo, productRepo, auditRepo, tenantRepo, pool, webhookDispatchService, objectStorage)
	shipmentService.SetWorkerPool(workerPool)
	shipmentService.SetFulfillmentService(fulfillmentService) // OPE-417: gated best-effort recording
	productService := service.NewProductService(productRepo, auditRepo, pool, webhookDispatchService)
	integrationService := service.NewIntegrationService(integrationRepo, auditRepo, pool, encryptionKey)
	labelService := service.NewLabelService(
		shipmentRepo, orderRepo, integrationRepo, auditRepo,
		warehouseRepo, tenantRepo,
		pool, encryptionKey, cfg.BaseURL,
		objectStorage,
	)
	labelService.SetFulfillmentService(fulfillmentService) // OPE-417: gated best-effort recording
	webhookService := service.NewWebhookService(webhookRepo, tenantRepo, pool, cfg.AllegroWebhookSecret, cfg.InPostWebhookSecret)
	providerWebhookSecretResolver := service.NewProviderWebhookSecretResolver(workerPool, encryptionKey)
	statsService := service.NewStatsService(statsRepo, pool)
	invoiceService := service.NewInvoiceService(invoiceRepo, orderRepo, tenantRepo, auditRepo, pool, encryptionKey)
	orderService.SetInvoiceService(invoiceService)
	orderService.SetSMSService(smsService)
	orderService.SetShipmentService(shipmentService)
	// Carrier-driven order status changes (package picked up / delivered) fan out the
	// same side effects as an operator-driven transition.
	shipmentService.SetOrderStatusSideEffects(orderService)
	shipmentService.SetSMSService(smsService)
	allegroSyncService := service.NewAllegroSyncService(integrationService).
		WithFulfillment(fulfillmentService) // OPE-417 followup: best-effort marketplace-sync provider attempts
	orderService.SetAllegroSyncService(allegroSyncService)
	shipmentService.SetAllegroSyncService(allegroSyncService)
	supplierService := service.NewSupplierService(supplierRepo, supplierProductRepo, supplierCategoryMappingRepo, allegroParameterMappingRepo, productCategoryRepo, productRepo, auditRepo, pool, webhookDispatchService, integrationService, slog.Default()).
		WithAvailability(supplierAvailabilityService) // OPE-418: gated snapshot upsert during catalog sync
	marketplaceCategoryMappingRepo := repository.NewMarketplaceCategoryMappingRepository()
	productCategoryService := service.NewProductCategoryService(productCategoryRepo, marketplaceCategoryMappingRepo, auditRepo, pool)
	variantService := service.NewVariantService(variantRepo, productRepo, auditRepo, pool)
	warehouseService := service.NewWarehouseService(warehouseRepo, warehouseStockRepo, auditRepo, tenantRepo, pool)
	orderGroupService := service.NewOrderGroupService(orderGroupRepo, orderRepo, auditRepo, pool)
	bundleService := service.NewBundleService(bundleRepo, productRepo, auditRepo, pool)
	customerService := service.NewCustomerService(customerRepo, auditRepo, pool, webhookDispatchService, slog.Default())
	customerImportService := service.NewCustomerImportService(customerRepo, auditRepo, pool)
	barcodeService := service.NewBarcodeService(productRepo, variantRepo, orderRepo, auditRepo, pool)
	priceListService := service.NewPriceListService(priceListRepo, productRepo, auditRepo, pool)
	messageTemplateService := service.NewMessageTemplateService(messageTemplateRepo, pool)
	warehouseDocService := service.NewWarehouseDocumentService(warehouseDocRepo, warehouseDocItemRepo, warehouseStockRepo, auditRepo, pool)
	warehouseDocService.SetFulfillmentService(fulfillmentService) // OPE-418: gated best-effort unit/step recording
	exchangeRateService := service.NewExchangeRateService(exchangeRateRepo, auditRepo, pool)
	ksefService := service.NewKSeFService(invoiceRepo, orderRepo, tenantRepo, auditRepo, pool)
	invoiceService.SetKSeFService(ksefService)
	stocktakeService := service.NewStocktakeService(stocktakeRepo, stocktakeItemRepo, warehouseStockRepo, warehouseDocRepo, warehouseDocItemRepo, auditRepo, pool, webhookDispatchService)
	purchaseOrderService := service.NewPurchaseOrderService(purchaseOrderRepo, purchaseOrderItemRepo, warehouseStockRepo, auditRepo, pool, webhookDispatchService, slog.Default())
	purchaseOrderService.SetFulfillmentService(fulfillmentService) // OPE-418: gated best-effort recording
	pickPackService := service.NewPickPackService(pickPackRepo, orderRepo, productRepo, variantRepo, auditRepo, pool)
	pickPackService.SetFulfillmentService(fulfillmentService) // OPE-418: gated best-effort unit/step recording
	dropshipService := service.NewDropshipService(dropshipRepo, dropshipItemRepo, orderRepo, productRepo, supplierRepo, auditRepo, integrationService, pool, webhookDispatchService, slog.Default())
	dropshipService.SetFulfillmentService(fulfillmentService)           // OPE-418: gated best-effort unit/step recording
	dropshipService.SetAvailabilityService(supplierAvailabilityService) // OPE-418: gated availability-based auto-routing gate
	// OPE-418/Phase-7: gated supplier-order engine. When SUPPLIER_ORDER_ENABLED is off the
	// service's Enabled() is false, so the gate's API branch keeps its current behavior (mark
	// the create_dropship_order step ready, no auto-submit) and no manual blocker is added.
	supplierOrderService := service.NewSupplierOrderService(cfg.SupplierOrderEnabled, pool, fulfillmentService, orchestrationRepo)
	dropshipService.SetSupplierOrderService(supplierOrderService)
	// newSupplierProvider builds a SupplierProvider for a supplier inside a tenant tx: it loads
	// the supplier, resolves the provider name (xml feed -> btp), decrypts the linked
	// integration credentials, and constructs the adapter. Returns (nil, nil) when the supplier
	// has no API provider — mirrors DropshipService.submitToSupplierAPI's provider construction.
	// Shared by the supplier-order handler (submit) and the status poller (reconcile).
	newSupplierProvider := func(ctx context.Context, tx pgx.Tx, tenantID, supplierID uuid.UUID) (integration.SupplierProvider, error) {
		supplier, err := supplierRepo.FindByID(ctx, tx, supplierID)
		if err != nil {
			return nil, fmt.Errorf("load supplier: %w", err)
		}
		if supplier == nil || supplier.IntegrationID == nil {
			return nil, nil // no integration — manual process
		}
		providerName := supplier.FeedFormat
		if providerName == "xml" {
			providerName = "btp"
		}
		if !integration.HasSupplierProvider(providerName) {
			return nil, nil // no provider registered for this format
		}
		credJSON, err := integrationService.GetDecryptedCredentialsByID(ctx, tenantID, *supplier.IntegrationID)
		if err != nil {
			return nil, fmt.Errorf("decrypt credentials: %w", err)
		}
		provider, err := integration.NewSupplierProvider(providerName, credJSON, supplier.Settings)
		if err != nil {
			return nil, fmt.Errorf("create provider: %w", err)
		}
		return provider, nil
	}
	// dropshipItemsLoader resolves the dropship lines for an (order, supplier) into the
	// supplier-order builder's input shape (OPE-516): the line's ItemID is the SUPPLIER's
	// catalogue identity from the supplier_products mapping, the EAN comes from the tenant
	// product, and the tenant's internal SKU is never sent. Runs inside the handler's tx.
	dropshipItemsLoader := service.NewSupplierOrderItemsLoader(dropshipRepo, dropshipItemRepo, productRepo, supplierProductRepo)
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
	productService.SetStockSyncService(stockSyncService)
	warehouseDocService.SetStockSyncService(stockSyncService)
	supplierService.SetStockSyncService(stockSyncService)

	// Segment & Loyalty
	segmentRepo := repository.NewCustomerSegmentRepository()
	loyaltyRepo := repository.NewLoyaltyRepository()
	segmentService := service.NewSegmentService(segmentRepo, auditRepo, pool, slog.Default())
	loyaltyService := service.NewLoyaltyService(loyaltyRepo, auditRepo, pool, slog.Default())

	// OPE-538: order-lifecycle dependencies wired into the order service via setters
	// (customer link/stats, B2B pricing, loyalty accrual, bundle stock).
	orderService.SetCustomerRepo(customerRepo)
	orderService.SetPriceListService(priceListService)
	orderService.SetLoyaltyService(loyaltyService)
	orderService.SetBundleService(bundleService)

	// Automation engine
	automationRuleRepo := repository.NewAutomationRuleRepository()
	automationRuleLogRepo := repository.NewAutomationRuleLogRepository()
	delayedActionRepo := repository.NewDelayedActionRepository()
	automationExecutor := automation.NewDefaultActionExecutor(slog.Default())
	automationExecutor.SetOrderServices(orderService, orderService, orderRepo, pool)
	automationExecutor.SetEmailSender(emailService)
	automationExecutor.SetInvoiceCreator(invoiceService)
	automationExecutor.SetListingActivatorDeps(&automation.ListingActivatorDeps{
		ListingRepo:     productListingRepo,
		IntegrationRepo: integrationRepo,
		EncryptionKey:   encryptionKey,
		ProviderFactory: newListingActivatorFactory(),
	})
	automationExecutor.SetListingDeactivatorDeps(&automation.ListingDeactivatorDeps{
		ListingRepo:     productListingRepo,
		IntegrationRepo: integrationRepo,
		EncryptionKey:   encryptionKey,
		ProviderFactory: newListingDeactivatorFactory(),
	})
	automationExecutor.SetMarketplaceMessageDeps(&automation.MarketplaceMessageDeps{
		TemplateRepo:   messageTemplateRepo,
		OrderRepo:      orderRepo,
		Pool:           pool,
		IntegrationSvc: integrationService,
	})
	// OPE-421: gate set_status routing through the orchestration outbox. When
	// AUTOMATION_ORCHESTRATION_ENABLED is on, executeSetStatus ensures the order's
	// fulfillment process and enqueues an automation.set_status event instead of
	// calling TransitionStatus directly; the handler (registered in the
	// ORCHESTRATION_WORKER_ENABLED block below) drains it. This is the ENQUEUE half
	// of the dual-flag dependency — processing additionally needs the worker flag.
	// When the flag is off this is a no-op and automation behaviour is unchanged.
	automationExecutor.SetOrchestration(cfg.AutomationOrchestrationEnabled, orchestrationRepo, fulfillmentRepo)

	// OPE-421/Phase-13 external-workflow connector (gated by EXTERNAL_WORKFLOW_ENABLED).
	// When the flag is off the service's Enabled() is false, so the external_workflow action
	// is a no-op, the callback route is unregistered, and the dispatcher handlers are not
	// registered — the default build is byte-for-byte unchanged.
	externalWorkflowTokenRepo := repository.NewExternalWorkflowTokenRepository()
	externalWorkflowService := service.NewExternalWorkflowService(
		cfg.ExternalWorkflowEnabled, pool, workerPool, fulfillmentService, orchestrationRepo,
		externalWorkflowTokenRepo, auditRepo)
	// loadExternalWorkflowConfig decrypts the integration's external-workflow credentials JSONB
	// (outbound_url, signing_secret, timeout_seconds, criticality, outbound_field_allowlist).
	loadExternalWorkflowConfig := func(ctx context.Context, tenantID, integrationID uuid.UUID) (service.ExternalWorkflowConfig, error) {
		var cfgOut service.ExternalWorkflowConfig
		credJSON, err := integrationService.GetDecryptedCredentialsByID(ctx, tenantID, integrationID)
		if err != nil {
			return cfgOut, err
		}
		if err := json.Unmarshal(credJSON, &cfgOut); err != nil {
			return cfgOut, fmt.Errorf("parse external workflow config: %w", err)
		}
		return cfgOut, nil
	}
	externalWorkflowService.SetConfigLoader(loadExternalWorkflowConfig)
	externalWorkflowService.SetOrderLoader(func(ctx context.Context, tx pgx.Tx, orderID uuid.UUID) (*model.Order, error) {
		return orderRepo.FindByID(ctx, tx, orderID)
	})
	automationExecutor.SetExternalWorkflow(externalWorkflowService)
	externalWorkflowCallbackHandler := handler.NewExternalWorkflowCallbackHandler(externalWorkflowService)
	// SSRF-safe HTTP client for the outbound signed dispatch (same protection as webhook dispatch).
	externalWorkflowHTTPClient := netutil.SafeHTTPClient(15 * time.Second)

	automationEngine := automation.NewEngine(automationRuleRepo, automationRuleLogRepo, pool, automationExecutor, slog.Default())
	automationEngine.SetDelayedActionRepo(delayedActionRepo)
	automationService := service.NewAutomationService(automationRuleRepo, automationRuleLogRepo, pool, automationEngine, slog.Default())
	automationService.SetDelayedActionRepo(delayedActionRepo)

	// Wire automation service into entity services (setter pattern to avoid circular dependency)
	orderService.SetAutomationService(automationService)
	shipmentService.SetAutomationService(automationService)
	returnService.SetAutomationService(automationService)
	productService.SetAutomationService(automationService)
	stockSyncService.SetAutomationService(automationService)

	// Invitation service (for invite-only registration mode)
	invitationRepo := repository.NewInvitationRepository()
	invitationService := service.NewInvitationService(invitationRepo, auditRepo, pool)

	// Initialize token blacklist for server-side token revocation.
	// Uses a composite store: writes to both Redis and memory, reads from either.
	// Redis is the shared production store; memory only protects the local process.
	memBlacklist := middleware.NewMemoryTokenBlacklist()
	var tokenBlacklist *middleware.TokenBlacklist
	if redisClient != nil {
		redisBlacklist := middleware.NewRedisTokenBlacklist(redisClient)
		composite := middleware.NewCompositeTokenBlacklist(redisBlacklist, memBlacklist)
		tokenBlacklist = middleware.NewTokenBlacklistWithStore(composite)
		slog.Info("using composite (Redis + memory) token blacklist")
	} else {
		tokenBlacklist = middleware.NewTokenBlacklistWithStore(memBlacklist)
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

	// License service for cloud billing
	licensePublicKey, err := cfg.ParseLicensePublicKey()
	if err != nil {
		slog.Error("failed to parse LICENSE_PUBLIC_KEY", "error", err)
		return fmt.Errorf("failed to parse LICENSE_PUBLIC_KEY: %w", err)
	}
	if licensePublicKey != nil {
		licenseRepo := repository.NewLicenseRepository()
		licenseSvc := service.NewLicenseService(licensePublicKey, pool)
		licenseSvc.SetRepository(licenseRepo)
		authHandler.SetLicenseService(licenseSvc)
		slog.Info("license token verification enabled")
	} else {
		slog.Info("license token verification disabled (no LICENSE_PUBLIC_KEY)")
	}

	// Plan cache for enforcement middleware (5 min TTL)
	planCache := service.NewPlanCache(5 * time.Minute)

	userHandler := handler.NewUserHandler(userService)
	orderHandler := handler.NewOrderHandler(orderService, tenantRepo, pool)
	shipmentHandler := handler.NewShipmentHandler(shipmentService, labelService, cfg.APISurfaceMode)
	productImportService := service.NewProductImportService(productRepo, auditRepo, pool)
	productImportService.SetStockSyncService(stockSyncService)
	blProductImportService := service.NewBaseLinkerProductImportService(productRepo, variantRepo, productCategoryRepo, auditRepo, pool)
	imageDownloadService := service.NewImageDownloadService(productRepo, pool, objectStorage)
	productHandler := handler.NewProductHandler(productService, productImportService, blProductImportService, productCategoryService, imageDownloadService)
	integrationHandler := handler.NewIntegrationHandler(integrationService, integrationRepo, pool, cfg.APISurfaceMode)
	returnHandler := handler.NewReturnHandler(returnService)
	webhookHandler := handler.NewWebhookHandler(webhookService)
	statsHandler := handler.NewStatsHandler(statsService)
	uploadHandler := handler.NewUploadHandler(objectStorage, cfg.MaxUploadSize)
	settingsHandler := handler.NewSettingsHandler(tenantRepo, auditRepo, productCategoryRepo, emailService, smsService, pool)
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
		memoryOAuthStateStore := handler.NewMemoryOAuthStateStore()
		defer memoryOAuthStateStore.Close()
		oauthStateStore = memoryOAuthStateStore
		slog.Info("using in-memory OAuth state store")
	}
	allegroAuthHandler := handler.NewAllegroAuthHandler(cfg, integrationService, encryptionKey, oauthStateStore)
	olxAuthHandler := handler.NewOlxAuthHandler(cfg, integrationService, oauthStateStore)
	ebayAuthHandler := handler.NewEbayAuthHandler(cfg, integrationService, oauthStateStore)
	ebayHandler := handler.NewEbayHandler(integrationService, orderService, encryptionKey)

	// Allegro import service (import Allegro offers as products + listings)
	allegroImportService := service.NewAllegroImportService(integrationService, productRepo, productListingRepo, productCategoryService, pool)
	allegroImportService.SetStockSyncService(stockSyncService)

	// Allegro fulfillment + tracking handler (Batch 1)
	allegroHandler := handler.NewAllegroHandler(integrationService, orderService, allegroImportService, encryptionKey)
	allegroOrderInbound := service.NewAllegroOrderInboundService(integrationService, orderRepo, auditRepo, pool)
	allegroHandler.SetOrderInbound(allegroOrderInbound)

	// Allegro shipment management handler ("Wysyłam z Allegro")
	allegroShipmentHandler := handler.NewAllegroShipmentHandler(integrationService, shipmentService, orderRepo, shipmentRepo, pool, encryptionKey)
	allegroShipmentHandler.SetLabelStore(objectStorage, cfg.BaseURL)

	// Allegro communications handler (messaging, returns, refunds)
	allegroCommsHandler := handler.NewAllegroCommsHandler(integrationService, encryptionKey)

	// Allegro webhook syncer — handles on-demand order import/update triggered by webhooks
	allegroWebhookSyncer := worker.NewAllegroWebhookSyncer(workerPool, encryptionKey, orderRepo, shipmentRepo, auditRepo, slog.Default())

	// Allegro webhook handler (public endpoint, HMAC-verified)
	allegroWebhookHandler := handler.NewAllegroWebhookHandler(cfg.AllegroWebhookSecret, allegroWebhookSyncer)
	allegroWebhookHandler.SetProviderWebhookSecretResolver(providerWebhookSecretResolver)

	// InPost webhook handler (public endpoint, HMAC-verified)
	inpostWebhookHandler := handler.NewInPostWebhookHandler(cfg.InPostWebhookSecret, shipmentService)
	inpostWebhookHandler.SetProviderWebhookSecretResolver(providerWebhookSecretResolver)

	// Allegro account & offers handler
	allegroAccountHandler := handler.NewAllegroAccountHandler(integrationService, encryptionKey)

	// Allegro listings handler (publish products to Allegro)
	allegroListingsHandler := handler.NewAllegroListingsHandler(integrationService, productService, productListingRepo, encryptionKey, pool, cfg)

	// WooCommerce listings handler (publish products to WooCommerce)
	wooCommerceListingsHandler := handler.NewWooCommerceListingsHandler(integrationService, productService, productListingRepo, pool)

	// Erli listings handler (publish products to Erli.pl)
	erliListingsHandler := handler.NewErliListingsHandler(integrationService, productService, productListingRepo, pool)

	// OLX listings handler (publish products to OLX.pl)
	olxListingsHandler := handler.NewOLXListingsHandler(integrationService, productService, productListingRepo, pool)

	// eBay import service (import eBay offers as products + listings)
	ebayImportService := service.NewEbayImportService(integrationService, productRepo, productListingRepo, pool)

	// eBay listings handler (publish products to eBay)
	ebayListingsHandler := handler.NewEbayListingsHandler(integrationService, productService, productListingRepo, pool, ebayImportService)

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
	amazonAuthHandler := handler.NewAmazonAuthHandler(cfg, integrationService, encryptionKey, oauthStateStore)

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
	productCategoryHandler := handler.NewProductCategoryHandler(productCategoryService)
	marketplaceCategoryMappingHandler := handler.NewMarketplaceCategoryMappingHandler(marketplaceCategoryMappingRepo, pool)

	// Import service & handler
	importService := service.NewImportService(orderRepo, auditRepo, tenantRepo, pool, fulfillmentService)
	baseLinkerImportService := service.NewBaseLinkerImportService(orderRepo, customerRepo, auditRepo, tenantRepo, pool, fulfillmentService)
	importHandler := handler.NewImportHandler(importService, baseLinkerImportService)

	// Automation handler
	automationHandler := handler.NewAutomationHandler(automationService)

	// Workflow builder handler
	workflowHandler := handler.NewWorkflowHandler(automationService)

	// Variant handler
	variantHandler := handler.NewVariantHandler(variantService)

	// Warehouse handler
	warehouseHandler := handler.NewWarehouseHandler(warehouseService)

	// Customer handler
	customerHandler := handler.NewCustomerHandler(customerService, customerImportService)

	// Order group handler
	orderGroupHandler := handler.NewOrderGroupHandler(orderGroupService)

	// Bundle handler
	bundleHandler := handler.NewBundleHandler(bundleService)

	// Barcode handler
	barcodeHandler := handler.NewBarcodeHandler(barcodeService)

	// Price list handler
	priceListHandler := handler.NewPriceListHandler(priceListService)

	// Message template handler
	messageTemplateHandler := handler.NewMessageTemplateHandler(messageTemplateService)

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
	aiService := service.NewAIService(cfg.OpenAIAPIKey, cfg.OpenAIModel, productRepo, tenantRepo, productCategoryRepo, pool)
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
	configHandler := handler.NewConfigHandler(cfg.RegistrationMode, licensePublicKey != nil, cfg.BillingEnabled(), cfg.StripePublicKey)

	// Stripe billing (conditional — disabled if no STRIPE_SECRET_KEY)
	var checkoutHandler *handler.CheckoutHandler
	var stripeWebhookHandler *handler.StripeWebhookHandler
	var checkoutSvc *service.CheckoutService
	if cfg.BillingEnabled() {
		stripe.Key = cfg.StripeSecretKey

		billingPlans, err := cfg.ParseBillingPlans()
		if err != nil {
			slog.Error("failed to parse BILLING_PLANS", "error", err)
			wsCancel()
			return fmt.Errorf("failed to parse BILLING_PLANS: %w", err)
		}

		billingRepo := repository.NewBillingRepository()
		checkoutSvc = service.NewCheckoutService(billingRepo, pool, billingPlans)
		checkoutHandler = handler.NewCheckoutHandler(checkoutSvc, planCache, pool, cfg.FrontendURL)
		authHandler.SetCheckoutService(checkoutSvc)
		slog.Info("stripe billing enabled", "plans", len(billingPlans))

		if cfg.StripeWebhookSecret != "" {
			webhookSvc := service.NewStripeWebhookService(cfg.StripeWebhookSecret, billingRepo, pool, planCache)
			stripeWebhookHandler = handler.NewStripeWebhookHandler(webhookSvc)
			slog.Info("stripe webhook handler enabled")
		}
	} else {
		slog.Info("stripe billing disabled (no STRIPE_SECRET_KEY or BILLING_PLANS)")
	}

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
	listingSyncService := service.NewListingSyncService(listingSyncRepo, productRepo, productListingRepo, auditRepo, integrationRepo, pool, encryptionKey, slog.Default())
	listingSyncService.SetStockSyncService(stockSyncService)
	listingSyncHandler := handler.NewListingSyncHandler(listingSyncService)

	// Prometheus metrics collector
	metricsCollector := middleware.NewMetricsCollector()
	// OPE-422: expose the additive fulfillment/orchestration/validation metrics
	// through the same /metrics handler.
	metricsCollector.Register(fulfillmentMetrics)

	// Platform-admin boundary (OPE-404): separate from tenant RBAC, not tenant-scoped.
	platformAdminRepo := repository.NewPlatformAdminRepository(pool)
	platformAuditRepo := repository.NewPlatformAuditRepository(pool)
	platformHandler := handler.NewPlatformHandler(platformAuditRepo)

	// Provider Integration Studio registry (OPE-405): platform-managed, not tenant-scoped.
	providerDefinitionRepo := repository.NewProviderDefinitionRepository(pool)
	providerVersionRepo := repository.NewProviderVersionRepository(pool)
	providerPublicationRepo := repository.NewProviderPublicationRepository(pool)
	providerSchemaRepo := repository.NewProviderSchemaRepository(pool)
	providerCapabilityRepo := repository.NewProviderCapabilityRepository(pool)
	providerRegistryService := service.NewProviderRegistryService(pool, providerDefinitionRepo, providerVersionRepo, providerPublicationRepo, providerSchemaRepo, providerCapabilityRepo).
		WithMetrics(fulfillmentMetrics) // OPE-422: best-effort publication-transition metrics
	providerHandler := handler.NewProviderHandler(providerRegistryService, platformAuditRepo)
	providerValidationRepo := repository.NewProviderValidationRepository(pool)
	providerValidationService := service.NewProviderValidationService(pool, providerVersionRepo, providerCapabilityRepo, providerValidationRepo).
		WithMetrics(fulfillmentMetrics) // OPE-422: best-effort validation-run + failure metrics
	providerValidationHandler := handler.NewProviderValidationHandler(providerValidationService, platformAuditRepo)

	// Provider registry seed (OPE-412): idempotently create draft registry
	// definitions for existing providers. Additive — tenant integrations untouched.
	if cfg.SeedProviders {
		seedCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		res, err := service.NewProviderRegistrySeeder(providerRegistryService).Seed(seedCtx, service.ProviderRegistryCatalog())
		cancel()
		if err != nil {
			if cfg.IsDevelopment() {
				slog.Warn("provider registry seed failed", "error", err)
			} else {
				slog.Error("provider registry seed failed", "error", err)
				wsCancel()
				return fmt.Errorf("provider registry seed: %w", err)
			}
		} else {
			slog.Info("provider registry seeded", "created", res.Created, "skipped", res.Skipped)
		}
	}

	// Setup router
	r := router.New(router.RouterDeps{
		Pool:                       pool,
		Config:                     cfg,
		TrustedProxyCIDRs:          trustedProxyCIDRs,
		TokenSvc:                   tokenSvc,
		TokenBlacklist:             tokenBlacklist,
		RateLimiter:                rateLimiter,
		Auth:                       authHandler,
		User:                       userHandler,
		Order:                      orderHandler,
		Shipment:                   shipmentHandler,
		Product:                    productHandler,
		Integration:                integrationHandler,
		Webhook:                    webhookHandler,
		Stats:                      statsHandler,
		Upload:                     uploadHandler,
		Settings:                   settingsHandler,
		Audit:                      auditHandler,
		WebhookDelivery:            webhookDeliveryHandler,
		Return:                     returnHandler,
		InPostPoint:                inpostPointHandler,
		AllegroAuth:                allegroAuthHandler,
		Allegro:                    allegroHandler,
		AllegroShipment:            allegroShipmentHandler,
		AmazonAuth:                 amazonAuthHandler,
		OlxAuth:                    olxAuthHandler,
		EbayAuth:                   ebayAuthHandler,
		Ebay:                       ebayHandler,
		StoreAuth:                  storeAuthHandler,
		Supplier:                   supplierHandler,
		Category:                   productCategoryHandler,
		Invoice:                    invoiceHandler,
		Automation:                 automationHandler,
		Workflow:                   workflowHandler,
		Import:                     importHandler,
		Variant:                    variantHandler,
		SyncJob:                    syncJobHandler,
		Warehouse:                  warehouseHandler,
		Customer:                   customerHandler,
		Print:                      printHandler,
		Docs:                       docsHandler,
		MetricsCollector:           metricsCollector,
		OrderGroup:                 orderGroupHandler,
		Bundle:                     bundleHandler,
		Barcode:                    barcodeHandler,
		PriceList:                  priceListHandler,
		WarehouseDocument:          warehouseDocHandler,
		WS:                         wsHandler,
		AI:                         aiHandler,
		Marketing:                  marketingHandler,
		Helpdesk:                   helpdeskHandler,
		PublicReturn:               publicReturnHandler,
		ExchangeRate:               exchangeRateHandler,
		Role:                       roleHandler,
		RoleService:                roleService,
		Stocktake:                  stocktakeHandler,
		KSeF:                       ksefHandler,
		Rate:                       rateHandler,
		AllegroComms:               allegroCommsHandler,
		AllegroWebhook:             allegroWebhookHandler,
		InPostWebhook:              inpostWebhookHandler,
		AllegroAccount:             allegroAccountHandler,
		AllegroCatalog:             allegroCatalogHandler,
		AllegroPolicies:            allegroPoliciesHandler,
		AllegroPromotions:          allegroPromotionsHandler,
		AllegroDelivery:            allegroDeliveryHandler,
		AllegroDisputes:            allegroDisputesHandler,
		AllegroRatings:             allegroRatingsHandler,
		AllegroListings:            allegroListingsHandler,
		WooCommerceListings:        wooCommerceListingsHandler,
		Tracking:                   trackingHandler,
		Feed:                       feedHandler,
		PurchaseOrder:              purchaseOrderHandler,
		PickPack:                   pickPackHandler,
		Accounting:                 accountingHandler,
		Reconciliation:             reconciliationHandler,
		Carbon:                     carbonHandler,
		VATOSS:                     vatOSSHandler,
		Dropship:                   dropshipHandler,
		SupplierPortal:             supplierPortalHandler,
		RecurringOrder:             recurringOrderHandler,
		Forecast:                   forecastHandler,
		Repricing:                  repricingHandler,
		BGRemoval:                  bgRemovalHandler,
		Segment:                    segmentHandler,
		Loyalty:                    loyaltyHandler,
		StockSync:                  stockSyncHandler,
		ListingSync:                listingSyncHandler,
		PublicConfig:               configHandler,
		Invitation:                 invitationHandler,
		MessageTemplate:            messageTemplateHandler,
		MarketplaceCategoryMapping: marketplaceCategoryMappingHandler,
		PlanCache:                  planCache,
		Checkout:                   checkoutHandler,
		StripeWebhook:              stripeWebhookHandler,
		ErliListings:               erliListingsHandler,
		OLXListings:                olxListingsHandler,
		EbayListings:               ebayListingsHandler,
		Platform:                   platformHandler,
		PlatformAdmin:              platformAdminRepo,
		Provider:                   providerHandler,
		ProviderValidation:         providerValidationHandler,
		Fulfillment:                fulfillmentHandler,
		Operations:                 operationsHandler,
		ExternalWorkflowCallback:   externalWorkflowCallbackHandler,
		APIToken:                   apiTokenHandler,
		APITokenAuth:               apiTokenService,
	})

	// Start background workers (use workerPool for cross-tenant queries).
	// Pass Redis client for distributed locking — prevents duplicate worker
	// execution when HPA scales to multiple pods.
	workerMgr := worker.NewManager(workerPool, slog.Default(), redisClient)
	workerMgr.Register(worker.NewOAuthRefresher(workerPool, encryptionKey, slog.Default()))
	workerMgr.Register(worker.NewAllegroOrderPoller(workerPool, encryptionKey, orderRepo, shipmentRepo, auditRepo, labelService, slog.Default()).WithFulfillment(fulfillmentService))
	workerMgr.Register(worker.NewStockSyncWorker(workerPool, encryptionKey, slog.Default()).WithAvailability(supplierAvailabilityService)) // OPE-418: gated channel-increase gate
	workerMgr.Register(worker.NewPriceSyncWorker(workerPool, encryptionKey, slog.Default()))
	trackingPoller := worker.NewTrackingPoller(workerPool, encryptionKey, shipmentRepo, shipmentService, slog.Default())
	trackingPoller.SetFulfillmentRecorder(fulfillmentService) // OPE-417: gated best-effort recording
	workerMgr.Register(trackingPoller)
	workerMgr.Register(worker.NewErliOrderPoller(workerPool, encryptionKey, orderRepo, shipmentRepo, auditRepo, slog.Default()).WithFulfillment(fulfillmentService))
	workerMgr.Register(worker.NewAmazonOrderPoller(workerPool, encryptionKey, orderRepo, shipmentRepo, auditRepo, slog.Default()).WithFulfillment(fulfillmentService))
	workerMgr.Register(worker.NewAmazonFeedStatusWorker(workerPool, encryptionKey, slog.Default()))
	workerMgr.Register(worker.NewWooCommerceOrderPoller(workerPool, encryptionKey, orderRepo, shipmentRepo, auditRepo, slog.Default()).WithFulfillment(fulfillmentService))
	workerMgr.Register(worker.NewShoperOrderPoller(workerPool, encryptionKey, orderRepo, shipmentRepo, auditRepo, slog.Default()).WithFulfillment(fulfillmentService))
	workerMgr.Register(worker.NewPrestaShopOrderPoller(workerPool, encryptionKey, orderRepo, shipmentRepo, auditRepo, slog.Default()).WithFulfillment(fulfillmentService))
	workerMgr.Register(worker.NewShopifyOrderPoller(workerPool, encryptionKey, orderRepo, shipmentRepo, auditRepo, slog.Default()).WithFulfillment(fulfillmentService))
	workerMgr.Register(worker.NewOLXOrderPoller(workerPool, encryptionKey, orderRepo, shipmentRepo, auditRepo, slog.Default()).WithFulfillment(fulfillmentService))
	workerMgr.Register(worker.NewEbayOrderPoller(workerPool, encryptionKey, orderRepo, shipmentRepo, auditRepo, slog.Default()).WithFulfillment(fulfillmentService))
	// Gated: the supplier module is behind a readiness checklist, so its background catalog
	// sync only runs where SUPPLIER_SYNC_ENABLED is set (off in production until GA).
	if cfg.SupplierSyncEnabled {
		workerMgr.Register(worker.NewSupplierSyncWorker(workerPool, supplierService, slog.Default()))
	}
	workerMgr.Register(worker.NewExchangeRateWorker(workerPool, exchangeRateService, slog.Default()))
	workerMgr.Register(worker.NewKSeFStatusWorker(workerPool, ksefService, slog.Default()))
	workerMgr.Register(worker.NewDelayedActionWorker(workerPool, delayedActionRepo, automationExecutor, slog.Default()))
	workerMgr.Register(worker.NewRecurringOrderWorker(workerPool, recurringOrderService, slog.Default()))
	workerMgr.Register(worker.NewSegmentRefreshWorker(workerPool, segmentService, slog.Default()))
	workerMgr.Register(worker.NewRepricingWorker(workerPool, repricingService, slog.Default()))
	workerMgr.Register(worker.NewListingSyncWorker(workerPool, listingSyncRepo, listingSyncService, slog.Default()))
	if cfg.BillingEnabled() && checkoutSvc != nil {
		workerMgr.Register(worker.NewBillingReconciliationWorker(workerPool, checkoutSvc, slog.Default()))
	}
	// Fulfillment orchestration outbox worker (OPE-415). Gated off by default.
	// OPE-416 registers the first real handler (order.created) on the dispatcher.
	if cfg.OrchestrationWorkerEnabled {
		orchestrationDispatcher := service.NewOrchestrationDispatcher()
		orchestrationDispatcher.Register(service.EventOrderCreated, service.NewOrderCreatedHandler(pool, fulfillmentRepo))
		// OPE-513: ack fulfillment.step events. EmitFulfillmentStep enqueues them on
		// the SUCCESS paths of shipment/label/tracking operations whenever
		// FULFILLMENT_PROCESS_ENABLED recording is on, so this handler is registered
		// UNCONDITIONALLY within the worker block — without it the dispatcher would
		// fail every fulfillment.step permanently and open spurious blockers on
		// healthy operations. The events are observability-only (the step was already
		// recorded by the emitter's caller), so the handler is a no-op ack.
		orchestrationDispatcher.Register(service.EventFulfillmentStep, service.NewFulfillmentStepHandler())
		// OPE-421: register the automation.set_status handler only when BOTH the
		// orchestration worker AND automation orchestration routing are enabled. This
		// is the PROCESSING half of the dual-flag dependency: the executor enqueues
		// automation.set_status events when AUTOMATION_ORCHESTRATION_ENABLED is on, and
		// this handler (which applies the idempotent transition via the order service)
		// drains them only when ORCHESTRATION_WORKER_ENABLED is also on. With routing
		// on but the worker off, events are durably enqueued but left unprocessed
		// (expected, opt-in). orderService's Get/TransitionStatus set tenant context
		// internally, matching the worker handler contract.
		if cfg.AutomationOrchestrationEnabled {
			orchestrationDispatcher.Register(automation.EventAutomationSetStatus, service.NewAutomationStatusTransitionHandler(orderService))
		}
		// OPE-421/Phase-13: register the external-workflow dispatch + follow-on-command
		// handlers only when EXTERNAL_WORKFLOW_ENABLED is also on. Off => unregistered, so a
		// stray external_workflow event would become a visible blocker rather than dispatch.
		if cfg.ExternalWorkflowEnabled {
			orchestrationDispatcher.Register(service.EventExternalWorkflow,
				service.NewExternalWorkflowHandler(externalWorkflowHTTPClient, loadExternalWorkflowConfig, workerPool, orchestrationRepo))
			orchestrationDispatcher.Register(service.EventExternalWorkflowCommand,
				service.NewExternalWorkflowCommandHandler(orderService))
		}
		// OPE-418/Phase-7: register the supplier-order submit handler + the reconcile poller
		// only when SUPPLIER_ORDER_ENABLED is also on. Off => unregistered, so a stray
		// supplier.order.submit event becomes a visible blocker rather than dispatching, and the
		// poller never runs — the default build is byte-for-byte unchanged.
		if cfg.SupplierOrderEnabled {
			orchestrationDispatcher.Register(service.EventSupplierOrderSubmit,
				service.NewSupplierOrderHandler(pool, fulfillmentService, dropshipRepo, orderRepo, dropshipItemsLoader, newSupplierProvider))
			workerMgr.Register(worker.NewSupplierOrderStatusPoller(
				workerPool, cfg.SupplierOrderEnabled, fulfillmentService, dropshipRepo, newSupplierProvider, slog.Default()))
		}
		// OPE-422: best-effort outbox metrics (claimed/processed/failed + queue depth).
		workerMgr.Register(worker.NewOrchestrationWorker(workerPool, orchestrationRepo, orchestrationDispatcher, fulfillmentRepo, 0, slog.Default()).
			WithMetrics(fulfillmentMetrics))
	}
	// Fulfillment-process backfill worker (OPE-423a). DOUBLE gated: it is a complete
	// no-op unless FULFILLMENT_BACKFILL_ENABLED is set, and even then only COUNTS
	// (dry run) unless FULFILLMENT_BACKFILL_DRY_RUN is explicitly false. Registered
	// only when enabled so a disabled deployment carries zero behaviour change. It
	// backfills processes for legacy non-terminal orders so they join process-backed
	// orchestration; the order.created events it enqueues are drained by the
	// orchestration worker above (enable ORCHESTRATION_WORKER_ENABLED to process them).
	if cfg.FulfillmentBackfillEnabled {
		fulfillmentBackfillService := service.NewFulfillmentBackfillService(fulfillmentRepo, orchestrationRepo)
		workerMgr.Register(worker.NewFulfillmentBackfillWorker(
			workerPool, fulfillmentBackfillService, cfg.FulfillmentBackfillEnabled, cfg.FulfillmentBackfillDryRun, slog.Default()))
	}
	// Global stuck/blocked process gauge sweeper (OPE-422 followup). Registered only
	// when fulfillment recording is on (there are processes to count); it counts across
	// all tenants on the privileged worker pool and publishes a single label-free
	// aggregate per tick. Read-only, best-effort.
	if cfg.FulfillmentProcessEnabled {
		workerMgr.Register(worker.NewFulfillmentGaugeSweeper(workerPool, fulfillmentReadService, slog.Default()))
	}
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
	if closer, ok := rateLimiter.(interface{ Close() }); ok {
		closer.Close()
	}
	slog.Info("server stopped")
	return nil
}

// newListingActivatorFactory returns a factory function that creates ListingActivatorProvider
// instances from marketplace provider credentials. It bridges the automation package's
// ListingActivatorProvider interface to the integration package's MarketplaceProvider.
func newListingActivatorFactory() func(provider string, credentials json.RawMessage, settings json.RawMessage) (automation.ListingActivatorProvider, error) {
	return func(providerName string, credentials json.RawMessage, settings json.RawMessage) (automation.ListingActivatorProvider, error) {
		mp, err := integration.NewMarketplaceProvider(providerName, credentials, settings)
		if err != nil {
			return nil, err
		}
		activator, ok := mp.(automation.ListingActivatorProvider)
		if !ok {
			closeProviderIfNeeded(mp)
			return nil, fmt.Errorf("provider %q does not support offer activation", providerName)
		}
		return activator, nil
	}
}

// newListingDeactivatorFactory returns a factory function that creates ListingDeactivatorProvider
// instances from marketplace provider credentials.
func newListingDeactivatorFactory() func(provider string, credentials json.RawMessage, settings json.RawMessage) (automation.ListingDeactivatorProvider, error) {
	return func(providerName string, credentials json.RawMessage, settings json.RawMessage) (automation.ListingDeactivatorProvider, error) {
		mp, err := integration.NewMarketplaceProvider(providerName, credentials, settings)
		if err != nil {
			return nil, err
		}
		deactivator, ok := mp.(automation.ListingDeactivatorProvider)
		if !ok {
			closeProviderIfNeeded(mp)
			return nil, fmt.Errorf("provider %q does not support offer deactivation", providerName)
		}
		return deactivator, nil
	}
}

// closeProviderIfNeeded closes a provider that implements Close() to prevent goroutine leaks.
func closeProviderIfNeeded(p any) {
	type closer interface{ Close() }
	if c, ok := p.(closer); ok {
		c.Close()
	}
}
