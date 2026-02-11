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
}

func New(deps RouterDeps) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.Logging)
	r.Use(chimw.Recoverer)
	r.Use(middleware.CORS([]string{deps.Config.FrontendURL}))

	// Health check — no auth, no tenant required
	healthHandler := &handler.HealthHandler{DB: deps.Pool}
	r.Get("/health", healthHandler.ServeHTTP)

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

	// Authenticated routes — JWT required
	r.Route("/v1", func(r chi.Router) {
		r.Use(middleware.JWTAuth(deps.TokenSvc))

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
			})

			// Admin-only audit log and webhook deliveries
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole("admin"))
				r.Get("/audit", deps.Audit.List)
				r.Get("/webhook-deliveries", deps.WebhookDelivery.List)
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
				r.Get("/{id}", deps.Order.Get)
				r.Patch("/{id}", deps.Order.Update)
				r.Delete("/{id}", deps.Order.Delete)
				r.Post("/{id}/status", deps.Order.TransitionStatus)
				r.Get("/{id}/audit", deps.Order.GetAudit)
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
				})
			})

			// Products — any authenticated user
			r.Route("/products", func(r chi.Router) {
				r.Get("/", deps.Product.List)
				r.Post("/", deps.Product.Create)
				r.Get("/{id}", deps.Product.Get)
				r.Patch("/{id}", deps.Product.Update)
				r.Delete("/{id}", deps.Product.Delete)
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

			// Stats — any authenticated user
			r.Get("/stats/dashboard", deps.Stats.GetDashboard)

			// InPost points search (proxy)
			r.Get("/inpost/points", deps.InPostPoint.Search)
		})
	})

	return r
}
