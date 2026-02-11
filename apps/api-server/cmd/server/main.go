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

	// Register marketplace providers via init().
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/allegro"
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/amazon"
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/erli"
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/mirakl"
	// Register carrier providers via init().
	_ "github.com/openoms-org/openoms/apps/api-server/internal/integration/carriers"

	"github.com/openoms-org/openoms/apps/api-server/internal/config"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/handler"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/router"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
	"github.com/openoms-org/openoms/apps/api-server/internal/worker"
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

	// Create upload directory
	if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
		slog.Error("failed to create upload directory", "error", err)
		os.Exit(1)
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

	returnRepo := repository.NewReturnRepository()
	supplierRepo := repository.NewSupplierRepository()
	supplierProductRepo := repository.NewSupplierProductRepository()

	authService := service.NewAuthService(userRepo, tenantRepo, auditRepo, tokenSvc, passwordSvc, pool)
	userService := service.NewUserService(userRepo, auditRepo, passwordSvc, pool)
	emailService := service.NewEmailService(tenantRepo, pool)
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
	supplierService := service.NewSupplierService(supplierRepo, supplierProductRepo, auditRepo, pool, webhookDispatchService, slog.Default())

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService, cfg.IsDevelopment())
	userHandler := handler.NewUserHandler(userService)
	orderHandler := handler.NewOrderHandler(orderService, tenantRepo, pool)
	shipmentHandler := handler.NewShipmentHandler(shipmentService, labelService)
	productHandler := handler.NewProductHandler(productService)
	integrationHandler := handler.NewIntegrationHandler(integrationService)
	returnHandler := handler.NewReturnHandler(returnService)
	webhookHandler := handler.NewWebhookHandler(webhookService)
	statsHandler := handler.NewStatsHandler(statsService)
	uploadHandler := handler.NewUploadHandler(cfg.UploadDir, cfg.MaxUploadSize, cfg.BaseURL)
	settingsHandler := handler.NewSettingsHandler(tenantRepo, auditRepo, emailService, pool)
	auditHandler := handler.NewAuditHandler(auditRepo, pool)
	webhookDeliveryHandler := handler.NewWebhookDeliveryHandler(webhookDeliveryRepo, pool)

	// InPost point search proxy
	inpostClient := inpost.NewClient(cfg.InPostAPIToken, cfg.InPostOrgID)
	inpostPointHandler := handler.NewInPostPointHandler(inpostClient)

	// Allegro OAuth handler
	allegroAuthHandler := handler.NewAllegroAuthHandler(cfg, integrationService, encryptionKey)

	// Amazon auth handler
	amazonAuthHandler := handler.NewAmazonAuthHandler(integrationService, encryptionKey)

	// Supplier handler
	supplierHandler := handler.NewSupplierHandler(supplierService)

	// Setup router
	r := router.New(router.RouterDeps{
		Pool:            pool,
		Config:          cfg,
		TokenSvc:        tokenSvc,
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
	})

	// Start background workers
	workerMgr := worker.NewManager(pool, slog.Default())
	workerMgr.Register(worker.NewOAuthRefresher(pool, encryptionKey, slog.Default()))
	workerMgr.Register(worker.NewAllegroOrderPoller(pool, encryptionKey, orderRepo, slog.Default()))
	workerMgr.Register(worker.NewStockSyncWorker(pool, encryptionKey, slog.Default()))
	workerMgr.Register(worker.NewTrackingPoller(pool, encryptionKey, shipmentRepo, slog.Default()))
	workerMgr.Register(worker.NewAmazonOrderPoller(pool, encryptionKey, orderRepo, slog.Default()))
	workerMgr.Register(worker.NewSupplierSyncWorker(pool, supplierService, slog.Default()))
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
