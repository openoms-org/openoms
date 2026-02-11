package router

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/config"
	"github.com/openoms-org/openoms/apps/api-server/internal/handler"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// RouterDeps holds all dependencies needed to construct the router.
type RouterDeps struct {
	Pool            *pgxpool.Pool
	Config          *config.Config
	TokenSvc        *service.TokenService
	TokenBlacklist  *middleware.TokenBlacklist
	Auth            *handler.AuthHandler
	User            *handler.UserHandler
	Order           *handler.OrderHandler
	Shipment        *handler.ShipmentHandler
	Product         *handler.ProductHandler
	Integration     *handler.IntegrationHandler
	Webhook         *handler.WebhookHandler
	Stats           *handler.StatsHandler
	Upload          *handler.UploadHandler
	Settings        *handler.SettingsHandler
	Audit           *handler.AuditHandler
	WebhookDelivery *handler.WebhookDeliveryHandler
	Return          *handler.ReturnHandler
	InPostPoint     *handler.InPostPointHandler
	AllegroAuth     *handler.AllegroAuthHandler
	AmazonAuth      *handler.AmazonAuthHandler
	Supplier        *handler.SupplierHandler
	Invoice         *handler.InvoiceHandler
	Automation      *handler.AutomationHandler
	Import          *handler.ImportHandler
	Variant         *handler.VariantHandler
	SyncJob         *handler.SyncJobHandler
	Warehouse       *handler.WarehouseHandler
	Customer        *handler.CustomerHandler
	Print            *handler.PrintHandler
	Docs             *handler.DocsHandler
	MetricsCollector *middleware.MetricsCollector
	OrderGroup       *handler.OrderGroupHandler
	Bundle           *handler.BundleHandler
	Barcode          *handler.BarcodeHandler
	PriceList          *handler.PriceListHandler
	WarehouseDocument  *handler.WarehouseDocumentHandler
	WS                 *handler.WSHandler
	AI                 *handler.AIHandler
	Marketing          *handler.MarketingHandler
	Helpdesk           *handler.HelpdeskHandler
	PublicReturn       *handler.PublicReturnHandler
	ExchangeRate       *handler.ExchangeRateHandler
	Role               *handler.RoleHandler
	RoleService        *service.RoleService
}

func New(deps RouterDeps) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	if deps.MetricsCollector != nil {
		r.Use(deps.MetricsCollector.Middleware())
	}
	r.Use(middleware.Logging)
	r.Use(chimw.Recoverer)
	r.Use(middleware.CORS([]string{deps.Config.FrontendURL}))

	// Health check — no auth, no tenant required
	healthHandler := &handler.HealthHandler{DB: deps.Pool}
	r.Get("/health", healthHandler.ServeHTTP)

	// Prometheus metrics — no auth
	if deps.MetricsCollector != nil {
		r.Get("/metrics", deps.MetricsCollector.Handler())
	}

	// OpenAPI spec and Swagger UI — no auth
	if deps.Docs != nil {
		r.Get("/v1/openapi.yaml", deps.Docs.ServeSpec)
		r.Get("/v1/docs", deps.Docs.ServeSwaggerUI)
	}

	// Serve uploaded files (public, cached)
	fileServer := http.StripPrefix("/uploads/", http.FileServer(http.Dir(deps.Config.UploadDir)))
	uploadFileHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000")
		fileServer.ServeHTTP(w, req)
	})
	r.Get("/uploads/*", uploadFileHandler)
	r.Head("/uploads/*", uploadFileHandler)

	// Public auth routes — no JWT required, rate-limited
	r.Route("/v1/auth", func(r chi.Router) {
		r.Use(middleware.RateLimit(20, 1*time.Minute))
		r.Use(middleware.MaxBodySize(1 << 20)) // 1MB
		r.Post("/register", deps.Auth.Register)
		r.Post("/login", deps.Auth.Login)
		r.Post("/refresh", deps.Auth.Refresh)
		r.Post("/logout", deps.Auth.Logout)
	})

	// Public webhook routes — no JWT, signature-verified
	r.Post("/v1/webhooks/{provider}/{tenant_id}", deps.Webhook.Receive)

	// Public return self-service routes — no JWT, rate-limited
	r.Route("/v1/public/returns", func(r chi.Router) {
		r.Use(middleware.RateLimit(30, 1*time.Minute))
		r.Use(middleware.MaxBodySize(1 << 20))
		r.Post("/", deps.PublicReturn.CreatePublicReturn)
		r.Get("/{token}", deps.PublicReturn.GetByToken)
		r.Get("/{token}/status", deps.PublicReturn.GetStatusByToken)
	})

	// WebSocket endpoint — auth via query param, must be before JWT middleware
	r.Get("/v1/ws", deps.WS.ServeWS)

	// Authenticated routes — JWT required
	r.Route("/v1", func(r chi.Router) {
		r.Use(middleware.JWTAuth(deps.TokenSvc, deps.TokenBlacklist))

		// Upload endpoint — has its own body size limit, no global MaxBodySize
		r.Post("/uploads", deps.Upload.Upload)

		// All other authenticated routes get a 1MB body size limit
		r.Group(func(r chi.Router) {
			r.Use(middleware.MaxBodySize(1 << 20))

			r.Get("/order-statuses", deps.Settings.GetOrderStatuses)
			r.Get("/custom-fields", deps.Settings.GetCustomFields)
			r.Get("/product-categories", deps.Settings.GetProductCategories)

			// Settings — admin only
			r.Route("/settings", func(r chi.Router) {
				r.Use(middleware.RequireRole("admin"))
				r.Get("/email", deps.Settings.GetEmailSettings)
				r.Put("/email", deps.Settings.UpdateEmailSettings)
				r.Post("/email/test", deps.Settings.SendTestEmail)
				r.Get("/company", deps.Settings.GetCompanySettings)
				r.Put("/company", deps.Settings.UpdateCompanySettings)
				r.Get("/order-statuses", deps.Settings.GetOrderStatuses)
				r.Put("/order-statuses", deps.Settings.UpdateOrderStatuses)
				r.Get("/custom-fields", deps.Settings.GetCustomFields)
				r.Put("/custom-fields", deps.Settings.UpdateCustomFields)
				r.Get("/product-categories", deps.Settings.GetProductCategories)
				r.Put("/product-categories", deps.Settings.UpdateProductCategories)
				r.Get("/webhooks", deps.Settings.GetWebhooks)
				r.Put("/webhooks", deps.Settings.UpdateWebhooks)
				r.Get("/invoicing", deps.Settings.GetInvoicingSettings)
				r.Put("/invoicing", deps.Settings.UpdateInvoicingSettings)
				r.Get("/sms", deps.Settings.GetSMSSettings)
				r.Put("/sms", deps.Settings.UpdateSMSSettings)
				r.Post("/sms/test", deps.Settings.SendTestSMS)
				r.Get("/print-templates", deps.Print.GetPrintTemplates)
				r.Put("/print-templates", deps.Print.UpdatePrintTemplates)
			})

			// Admin-only audit log and webhook deliveries
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole("admin"))
				r.Get("/audit", deps.Audit.List)
				r.Get("/webhook-deliveries", deps.WebhookDelivery.List)
			})

			// Sync jobs — admin only
			r.Route("/sync-jobs", func(r chi.Router) {
				r.Use(middleware.RequireRole("admin"))
				r.Get("/", deps.SyncJob.List)
				r.Get("/{id}", deps.SyncJob.Get)
			})

			// Any authenticated user
			r.Get("/users/me", deps.User.Me)

			// Admin/owner only user management
			r.Route("/users", func(r chi.Router) {
				r.Use(middleware.RequireRole("admin"))
				r.Get("/", deps.User.List)
				r.Post("/", deps.User.Create)
				r.Patch("/{id}", deps.User.Update)
				r.Delete("/{id}", deps.User.Delete)
			})

			// Orders — any authenticated user
			r.Route("/orders", func(r chi.Router) {
				r.Get("/", deps.Order.List)
				r.Post("/", deps.Order.Create)
				r.Get("/export", deps.Order.ExportCSV)
				r.Post("/bulk-status", deps.Order.BulkTransitionStatus)
				r.Post("/merge", deps.OrderGroup.MergeOrders)
				r.Post("/import/preview", deps.Import.Preview)
				r.Post("/import", deps.Import.Import)
				r.Get("/{id}", deps.Order.Get)
				r.Patch("/{id}", deps.Order.Update)
				r.Delete("/{id}", deps.Order.Delete)
				r.Post("/{id}/status", deps.Order.TransitionStatus)
				r.Post("/{id}/split", deps.OrderGroup.SplitOrder)
				r.Get("/{id}/groups", deps.OrderGroup.ListByOrder)
				r.Get("/{id}/audit", deps.Order.GetAudit)
				r.Get("/{id}/invoices", deps.Invoice.ListByOrder)
				r.Get("/{id}/packing-slip", deps.Print.GetPackingSlip)
				r.Get("/{id}/print", deps.Print.GetOrderSummary)
				r.Post("/{id}/pack", deps.Barcode.PackOrder)
				r.Get("/{id}/tickets", deps.Helpdesk.ListOrderTickets)
				r.Post("/{id}/tickets", deps.Helpdesk.CreateOrderTicket)
			})

			// Invoices — any authenticated user
			r.Route("/invoices", func(r chi.Router) {
				r.Get("/", deps.Invoice.List)
				r.Post("/", deps.Invoice.Create)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", deps.Invoice.Get)
					r.Get("/pdf", deps.Invoice.GetPDF)
					r.Delete("/", deps.Invoice.Cancel)
				})
			})

			// Shipments — any authenticated user
			r.Route("/shipments", func(r chi.Router) {
				r.Get("/", deps.Shipment.List)
				r.Post("/", deps.Shipment.Create)
				r.Get("/{id}", deps.Shipment.Get)
				r.Patch("/{id}", deps.Shipment.Update)
				r.Delete("/{id}", deps.Shipment.Delete)
				r.Post("/{id}/status", deps.Shipment.TransitionStatus)
				r.Post("/{id}/label", deps.Shipment.GenerateLabel)
			})

			// Returns — any authenticated user
			r.Route("/returns", func(r chi.Router) {
				r.Get("/", deps.Return.List)
				r.Post("/", deps.Return.Create)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", deps.Return.Get)
					r.Patch("/", deps.Return.Update)
					r.Delete("/", deps.Return.Delete)
					r.Post("/status", deps.Return.TransitionStatus)
					r.Get("/print", deps.Print.GetReturnSlip)
				})
			})

			// Products — any authenticated user
			r.Route("/products", func(r chi.Router) {
				r.Get("/", deps.Product.List)
				r.Post("/", deps.Product.Create)
				r.Get("/{id}", deps.Product.Get)
				r.Patch("/{id}", deps.Product.Update)
				r.Delete("/{id}", deps.Product.Delete)
				r.Get("/{id}/stock", deps.Warehouse.ListProductStock)

				// Bundles
				r.Route("/{id}/bundle", func(r chi.Router) {
					r.Get("/", deps.Bundle.ListComponents)
					r.Post("/", deps.Bundle.AddComponent)
					r.Get("/stock", deps.Bundle.GetBundleStock)
					r.Put("/{componentId}", deps.Bundle.UpdateComponent)
					r.Delete("/{componentId}", deps.Bundle.RemoveComponent)
				})

				// Variants
				r.Route("/{productId}/variants", func(r chi.Router) {
					r.Get("/", deps.Variant.List)
					r.Post("/", deps.Variant.Create)
					r.Get("/{id}", deps.Variant.Get)
					r.Patch("/{id}", deps.Variant.Update)
					r.Delete("/{id}", deps.Variant.Delete)
				})
			})

			// Integrations — admin only
			r.Route("/integrations", func(r chi.Router) {
				r.Use(middleware.RequireRole("admin"))

				// Allegro OAuth2 (must be before /{id} to avoid chi treating "allegro" as an ID)
				r.Route("/allegro", func(r chi.Router) {
					r.Get("/auth-url", deps.AllegroAuth.GetAuthURL)
					r.Post("/callback", deps.AllegroAuth.HandleCallback)
				})

				// Amazon SP-API setup
				r.Post("/amazon/setup", deps.AmazonAuth.Setup)

				r.Get("/", deps.Integration.List)
				r.Post("/", deps.Integration.Create)
				r.Get("/{id}", deps.Integration.Get)
				r.Patch("/{id}", deps.Integration.Update)
				r.Delete("/{id}", deps.Integration.Delete)
			})

			// Suppliers — admin only
			r.Route("/suppliers", func(r chi.Router) {
				r.Use(middleware.RequireRole("admin"))
				r.Get("/", deps.Supplier.List)
				r.Post("/", deps.Supplier.Create)
				r.Get("/{id}", deps.Supplier.Get)
				r.Patch("/{id}", deps.Supplier.Update)
				r.Delete("/{id}", deps.Supplier.Delete)
				r.Post("/{id}/sync", deps.Supplier.Sync)
				r.Get("/{id}/products", deps.Supplier.ListProducts)
				r.Post("/{id}/products/{spid}/link", deps.Supplier.LinkProduct)
			})

			// Warehouses — admin only
			r.Route("/warehouses", func(r chi.Router) {
				r.Use(middleware.RequireRole("admin"))
				r.Get("/", deps.Warehouse.List)
				r.Post("/", deps.Warehouse.Create)
				r.Get("/{id}", deps.Warehouse.Get)
				r.Patch("/{id}", deps.Warehouse.Update)
				r.Delete("/{id}", deps.Warehouse.Delete)
				r.Get("/{id}/stock", deps.Warehouse.ListStock)
				r.Put("/{id}/stock", deps.Warehouse.UpsertStock)
			})

			// Customers — any authenticated user
			r.Route("/customers", func(r chi.Router) {
				r.Get("/", deps.Customer.List)
				r.Post("/", deps.Customer.Create)
				r.Get("/{id}", deps.Customer.Get)
				r.Patch("/{id}", deps.Customer.Update)
				r.Delete("/{id}", deps.Customer.Delete)
				r.Get("/{id}/orders", deps.Customer.ListOrders)
			})

			// Automation rules — admin only
			r.Route("/automation/rules", func(r chi.Router) {
				r.Use(middleware.RequireRole("admin"))
				r.Get("/", deps.Automation.List)
				r.Post("/", deps.Automation.Create)
				r.Get("/{id}", deps.Automation.Get)
				r.Patch("/{id}", deps.Automation.Update)
				r.Delete("/{id}", deps.Automation.Delete)
				r.Get("/{id}/logs", deps.Automation.GetLogs)
				r.Post("/{id}/test", deps.Automation.TestRule)
			})

			// Stats — any authenticated user
			r.Route("/stats", func(r chi.Router) {
				r.Get("/dashboard", deps.Stats.GetDashboard)
				r.Get("/products/top", deps.Stats.GetTopProducts)
				r.Get("/revenue/by-source", deps.Stats.GetRevenueBySource)
				r.Get("/trends", deps.Stats.GetOrderTrends)
				r.Get("/payment-methods", deps.Stats.GetPaymentMethodStats)
			})

			// Barcode lookup — any authenticated user
			r.Get("/barcode/{code}", deps.Barcode.Lookup)

			// Price lists — admin only
			r.Route("/price-lists", func(r chi.Router) {
				r.Use(middleware.RequireRole("admin"))
				r.Get("/", deps.PriceList.List)
				r.Post("/", deps.PriceList.Create)
				r.Get("/{id}", deps.PriceList.Get)
				r.Patch("/{id}", deps.PriceList.Update)
				r.Delete("/{id}", deps.PriceList.Delete)
				r.Get("/{id}/items", deps.PriceList.ListItems)
				r.Post("/{id}/items", deps.PriceList.CreateItem)
				r.Delete("/{id}/items/{itemId}", deps.PriceList.DeleteItem)
			})

			// Warehouse documents — admin only
			r.Route("/warehouse-documents", func(r chi.Router) {
				r.Use(middleware.RequireRole("admin"))
				r.Get("/", deps.WarehouseDocument.List)
				r.Post("/", deps.WarehouseDocument.Create)
				r.Get("/{id}", deps.WarehouseDocument.Get)
				r.Patch("/{id}", deps.WarehouseDocument.Update)
				r.Delete("/{id}", deps.WarehouseDocument.Delete)
				r.Post("/{id}/confirm", deps.WarehouseDocument.Confirm)
				r.Post("/{id}/cancel", deps.WarehouseDocument.Cancel)
			})

			// AI auto-categorization — any authenticated user
			r.Route("/ai", func(r chi.Router) {
				r.Post("/categorize", deps.AI.Categorize)
				r.Post("/describe", deps.AI.Describe)
				r.Post("/bulk-categorize", deps.AI.BulkCategorize)
			})

			// Marketing (Mailchimp) — admin only
			r.Route("/marketing", func(r chi.Router) {
				r.Use(middleware.RequireRole("admin"))
				r.Post("/sync", deps.Marketing.Sync)
				r.Get("/status", deps.Marketing.Status)
				r.Post("/campaigns", deps.Marketing.CreateCampaign)
			})

			// Helpdesk (Freshdesk) — any authenticated user
			r.Get("/helpdesk/tickets", deps.Helpdesk.ListAllTickets)

			// Exchange rates — admin only
			r.Route("/exchange-rates", func(r chi.Router) {
				r.Use(middleware.RequireRole("admin"))
				r.Get("/", deps.ExchangeRate.List)
				r.Post("/", deps.ExchangeRate.Create)
				r.Post("/fetch", deps.ExchangeRate.FetchNBP)
				r.Post("/convert", deps.ExchangeRate.Convert)
				r.Get("/{id}", deps.ExchangeRate.Get)
				r.Patch("/{id}", deps.ExchangeRate.Update)
				r.Delete("/{id}", deps.ExchangeRate.Delete)
			})

			// Roles (RBAC) — admin only
			r.Route("/roles", func(r chi.Router) {
				r.Use(middleware.RequireRole("admin"))
				r.Get("/", deps.Role.List)
				r.Get("/permissions", deps.Role.ListPermissions)
				r.Post("/", deps.Role.Create)
				r.Get("/{id}", deps.Role.Get)
				r.Patch("/{id}", deps.Role.Update)
				r.Delete("/{id}", deps.Role.Delete)
			})

			// InPost points search (proxy)
			r.Get("/inpost/points", deps.InPostPoint.Search)
		})
	})

	return r
}
