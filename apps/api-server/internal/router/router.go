package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/config"
	"github.com/openoms-org/openoms/apps/api-server/internal/handler"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

func New(
	pool *pgxpool.Pool,
	cfg *config.Config,
	tokenSvc *service.TokenService,
	authHandler *handler.AuthHandler,
	userHandler *handler.UserHandler,
	orderHandler *handler.OrderHandler,
	shipmentHandler *handler.ShipmentHandler,
	productHandler *handler.ProductHandler,
	integrationHandler *handler.IntegrationHandler,
	webhookHandler *handler.WebhookHandler,
	statsHandler *handler.StatsHandler,
	uploadHandler *handler.UploadHandler,
	settingsHandler *handler.SettingsHandler,
) *chi.Mux {

	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.Logging)
	r.Use(chimw.Recoverer)
	r.Use(middleware.CORS([]string{cfg.FrontendURL}))

	// Health check — no auth, no tenant required
	healthHandler := &handler.HealthHandler{DB: pool}
	r.Get("/health", healthHandler.ServeHTTP)

	// Serve uploaded files (public, cached)
	fileServer := http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.UploadDir)))
	uploadFileHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000")
		fileServer.ServeHTTP(w, req)
	})
	r.Get("/uploads/*", uploadFileHandler)
	r.Head("/uploads/*", uploadFileHandler)

	// Public auth routes — no JWT required
	r.Route("/v1/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Post("/refresh", authHandler.Refresh)
		r.Post("/logout", authHandler.Logout)
	})

	// Public webhook routes — no JWT, signature-verified
	r.Post("/v1/webhooks/{provider}/{tenant_id}", webhookHandler.Receive)

	// Authenticated routes — JWT required
	r.Route("/v1", func(r chi.Router) {
		r.Use(middleware.JWTAuth(tokenSvc))

		r.Post("/uploads", uploadHandler.Upload)

		// Settings — admin only
		r.Route("/settings", func(r chi.Router) {
			r.Use(middleware.RequireRole("admin"))
			r.Get("/email", settingsHandler.GetEmailSettings)
			r.Put("/email", settingsHandler.UpdateEmailSettings)
			r.Post("/email/test", settingsHandler.SendTestEmail)
			r.Get("/company", settingsHandler.GetCompanySettings)
			r.Put("/company", settingsHandler.UpdateCompanySettings)
		})

		// Any authenticated user
		r.Get("/users/me", userHandler.Me)

		// Admin/owner only user management
		r.Route("/users", func(r chi.Router) {
			r.Use(middleware.RequireRole("admin"))
			r.Get("/", userHandler.List)
			r.Post("/", userHandler.Create)
			r.Patch("/{id}", userHandler.Update)
			r.Delete("/{id}", userHandler.Delete)
		})

		// Orders — any authenticated user
		r.Route("/orders", func(r chi.Router) {
			r.Get("/", orderHandler.List)
			r.Post("/", orderHandler.Create)
			r.Get("/export", orderHandler.ExportCSV)
			r.Post("/bulk-status", orderHandler.BulkTransitionStatus)
			r.Get("/{id}", orderHandler.Get)
			r.Patch("/{id}", orderHandler.Update)
			r.Delete("/{id}", orderHandler.Delete)
			r.Post("/{id}/status", orderHandler.TransitionStatus)
			r.Get("/{id}/audit", orderHandler.GetAudit)
		})

		// Shipments — any authenticated user
		r.Route("/shipments", func(r chi.Router) {
			r.Get("/", shipmentHandler.List)
			r.Post("/", shipmentHandler.Create)
			r.Get("/{id}", shipmentHandler.Get)
			r.Patch("/{id}", shipmentHandler.Update)
			r.Delete("/{id}", shipmentHandler.Delete)
			r.Post("/{id}/status", shipmentHandler.TransitionStatus)
			r.Post("/{id}/label", shipmentHandler.GenerateLabel)
		})

		// Products — any authenticated user
		r.Route("/products", func(r chi.Router) {
			r.Get("/", productHandler.List)
			r.Post("/", productHandler.Create)
			r.Get("/{id}", productHandler.Get)
			r.Patch("/{id}", productHandler.Update)
			r.Delete("/{id}", productHandler.Delete)
		})

		// Integrations — admin only
		r.Route("/integrations", func(r chi.Router) {
			r.Use(middleware.RequireRole("admin"))
			r.Get("/", integrationHandler.List)
			r.Post("/", integrationHandler.Create)
			r.Get("/{id}", integrationHandler.Get)
			r.Patch("/{id}", integrationHandler.Update)
			r.Delete("/{id}", integrationHandler.Delete)
		})

		// Stats — any authenticated user
		r.Get("/stats/dashboard", statsHandler.GetDashboard)
	})

	return r
}
