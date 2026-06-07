// Package router wires HTTP routes to handler implementations.
package router

import (
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/config"
	"github.com/openoms-org/openoms/apps/api-server/internal/handler"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// RouterDeps holds all dependencies needed to construct the router.
type RouterDeps struct { //nolint:revive
	Pool                       *pgxpool.Pool
	Config                     *config.Config
	TrustedProxyCIDRs          []netip.Prefix
	TokenSvc                   *service.TokenService
	TokenBlacklist             *middleware.TokenBlacklist
	RateLimiter                middleware.RateLimiter
	Auth                       *handler.AuthHandler
	User                       *handler.UserHandler
	Order                      *handler.OrderHandler
	Shipment                   *handler.ShipmentHandler
	Product                    *handler.ProductHandler
	Integration                *handler.IntegrationHandler
	Webhook                    *handler.WebhookHandler
	Stats                      *handler.StatsHandler
	Upload                     *handler.UploadHandler
	Settings                   *handler.SettingsHandler
	Audit                      *handler.AuditHandler
	WebhookDelivery            *handler.WebhookDeliveryHandler
	Return                     *handler.ReturnHandler
	InPostPoint                *handler.InPostPointHandler
	AllegroAuth                *handler.AllegroAuthHandler
	Allegro                    *handler.AllegroHandler
	AllegroShipment            *handler.AllegroShipmentHandler
	AmazonAuth                 *handler.AmazonAuthHandler
	OlxAuth                    *handler.OlxAuthHandler
	EbayAuth                   *handler.EbayAuthHandler
	Ebay                       *handler.EbayHandler
	Supplier                   *handler.SupplierHandler
	Invoice                    *handler.InvoiceHandler
	Automation                 *handler.AutomationHandler
	Import                     *handler.ImportHandler
	Variant                    *handler.VariantHandler
	SyncJob                    *handler.SyncJobHandler
	Warehouse                  *handler.WarehouseHandler
	Customer                   *handler.CustomerHandler
	Print                      *handler.PrintHandler
	Docs                       *handler.DocsHandler
	MetricsCollector           *middleware.MetricsCollector
	OrderGroup                 *handler.OrderGroupHandler
	Bundle                     *handler.BundleHandler
	Barcode                    *handler.BarcodeHandler
	PriceList                  *handler.PriceListHandler
	WarehouseDocument          *handler.WarehouseDocumentHandler
	WS                         *handler.WSHandler
	AI                         *handler.AIHandler
	Marketing                  *handler.MarketingHandler
	Helpdesk                   *handler.HelpdeskHandler
	PublicReturn               *handler.PublicReturnHandler
	ExchangeRate               *handler.ExchangeRateHandler
	Role                       *handler.RoleHandler
	RoleService                *service.RoleService
	Stocktake                  *handler.StocktakeHandler
	KSeF                       *handler.KSeFHandler
	Rate                       *handler.RateHandler
	AllegroComms               *handler.AllegroCommsHandler
	AllegroWebhook             *handler.AllegroWebhookHandler
	InPostWebhook              *handler.InPostWebhookHandler
	AllegroAccount             *handler.AllegroAccountHandler
	AllegroCatalog             *handler.AllegroCatalogHandler
	AllegroDisputes            *handler.AllegroDisputesHandler
	AllegroRatings             *handler.AllegroRatingsHandler
	AllegroPromotions          *handler.AllegroPromotionsHandler
	AllegroDelivery            *handler.AllegroDeliveryHandler
	AllegroPolicies            *handler.AllegroPoliciesHandler
	AllegroListings            *handler.AllegroListingsHandler
	WooCommerceListings        *handler.WooCommerceListingsHandler
	Tracking                   *handler.TrackingHandler
	Feed                       *handler.FeedHandler
	PurchaseOrder              *handler.PurchaseOrderHandler
	PickPack                   *handler.PickPackHandler
	Accounting                 *handler.AccountingHandler
	Reconciliation             *handler.ReconciliationHandler
	Carbon                     *handler.CarbonHandler
	VATOSS                     *handler.VATOSSHandler
	Dropship                   *handler.DropshipHandler
	SupplierPortal             *handler.SupplierPortalHandler
	RecurringOrder             *handler.RecurringOrderHandler
	Forecast                   *handler.ForecastHandler
	Repricing                  *handler.RepricingHandler
	BGRemoval                  *handler.BGRemovalHandler
	Segment                    *handler.SegmentHandler
	Loyalty                    *handler.LoyaltyHandler
	Workflow                   *handler.WorkflowHandler
	StockSync                  *handler.StockSyncHandler
	ListingSync                *handler.ListingSyncHandler
	StoreAuth                  *handler.StoreAuthHandler
	PublicConfig               *handler.ConfigHandler
	Invitation                 *handler.InvitationHandler
	Category                   *handler.ProductCategoryHandler
	MessageTemplate            *handler.MessageTemplateHandler
	MarketplaceCategoryMapping *handler.MarketplaceCategoryMappingHandler
	PlanCache                  *service.PlanCache
	Checkout                   *handler.CheckoutHandler
	StripeWebhook              *handler.StripeWebhookHandler
	ErliListings               *handler.ErliListingsHandler
	OLXListings                *handler.OLXListingsHandler
	EbayListings               *handler.EbayListingsHandler
	Platform                   *handler.PlatformHandler
	PlatformAdmin              middleware.PlatformAdminLookup
	Provider                   *handler.ProviderHandler
	ProviderValidation         *handler.ProviderValidationHandler
	Fulfillment                *handler.FulfillmentHandler
	Operations                 *handler.OperationsHandler
}

// New constructs the chi router with all routes registered.
func New(deps RouterDeps) *chi.Mux {
	r := chi.NewRouter()
	requirePermission := middleware.RequirePermission
	requireFeature := func(featureID string) func(http.Handler) http.Handler {
		return middleware.RequireFeature(featureID, deps.Config.APISurfaceMode)
	}

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(middleware.TrustedRealIP(deps.TrustedProxyCIDRs))
	if sentry.CurrentHub().Client() != nil {
		r.Use(middleware.SentryMiddleware)
	}
	if deps.MetricsCollector != nil {
		r.Use(deps.MetricsCollector.Middleware())
	}
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.Logging)
	r.Use(chimw.Recoverer)
	r.Use(middleware.CORS([]string{deps.Config.FrontendURL}))
	r.Use(middleware.CSRF(!deps.Config.IsDevelopment(), extractCookieDomain(deps.Config.FrontendURL)))

	// Health check — no auth, no tenant required
	healthHandler := &handler.HealthHandler{DB: deps.Pool}
	r.Get("/health", healthHandler.ServeHTTP)

	// Prometheus metrics — protected by Bearer token
	if deps.MetricsCollector != nil {
		r.With(middleware.MetricsAuth(deps.Config.MetricsToken)).Get("/metrics", deps.MetricsCollector.Handler())
	}

	// OpenAPI spec and Swagger UI — no auth
	if deps.Docs != nil {
		r.Get("/v1/openapi.yaml", deps.Docs.ServeSpec)
		r.Get("/v1/docs", deps.Docs.ServeSwaggerUI)
	}

	// Serve uploaded files (authenticated, tenant-scoped)
	fileServer := http.StripPrefix("/uploads/", http.FileServer(http.Dir(deps.Config.UploadDir)))
	r.Group(func(r chi.Router) {
		r.Use(middleware.JWTAuth(deps.TokenSvc, deps.TokenBlacklist))
		uploadFileHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Extract tenant from the file path and verify it matches the authenticated tenant
			pathParam := chi.URLParam(req, "*")
			parts := strings.SplitN(pathParam, "/", 2)
			if len(parts) < 2 {
				http.NotFound(w, req)
				return
			}
			fileTenantID := parts[0]
			authTenantID := middleware.TenantIDFromContext(req.Context())
			if fileTenantID != authTenantID.String() {
				http.NotFound(w, req)
				return
			}
			w.Header().Set("Cache-Control", "private, max-age=3600")
			fileServer.ServeHTTP(w, req)
		})
		r.Get("/uploads/*", uploadFileHandler)
		r.Head("/uploads/*", uploadFileHandler)
	})

	// Public auth routes — no JWT required, rate-limited
	r.Route("/v1/auth", func(r chi.Router) {
		r.Use(middleware.MaxBodySize(1 << 20)) // 1MB

		// Login/register — strict rate limit (10 req/min per IP)
		r.With(middleware.RateLimitWith(deps.RateLimiter, 10, 1*time.Minute)).Post("/register", deps.Auth.Register)
		r.With(middleware.RateLimitWith(deps.RateLimiter, 10, 1*time.Minute)).Post("/login", deps.Auth.Login)
		r.With(middleware.RateLimitWith(deps.RateLimiter, 10, 1*time.Minute)).Post("/2fa/login", deps.Auth.TwoFALogin)

		// Token refresh/logout — lighter rate limit (60 req/min per IP)
		r.With(middleware.RateLimitWith(deps.RateLimiter, 60, 1*time.Minute)).Post("/refresh", deps.Auth.Refresh)
		r.With(middleware.RateLimitWith(deps.RateLimiter, 60, 1*time.Minute)).Post("/logout", deps.Auth.Logout)

		// 2FA management — JWT required (inside /v1/auth to avoid chi prefix conflict)
		r.Route("/2fa", func(r chi.Router) {
			r.Use(middleware.JWTAuth(deps.TokenSvc, deps.TokenBlacklist))
			r.Post("/setup", deps.Auth.TwoFASetup)
			r.Post("/verify", deps.Auth.TwoFAVerify)
			r.Post("/disable", deps.Auth.TwoFADisable)
			r.Get("/status", deps.Auth.TwoFAStatus)
		})

		// WebSocket ticket — JWT required, issues short-lived single-use ticket for WS connections
		r.With(middleware.JWTAuth(deps.TokenSvc, deps.TokenBlacklist)).
			Post("/ws-ticket", deps.Auth.WSTicket)
	})

	// Public config endpoint — no auth required, rate-limited
	if deps.PublicConfig != nil {
		r.With(middleware.RateLimitWith(deps.RateLimiter, 60, 1*time.Minute)).
			Get("/v1/config/public", deps.PublicConfig.PublicConfig)
	}

	// Public billing routes — no JWT required, rate-limited
	if deps.Checkout != nil {
		r.With(middleware.RateLimitWith(deps.RateLimiter, 60, 1*time.Minute)).
			Get("/v1/billing/plans", deps.Checkout.ListPlans)
		r.With(middleware.RateLimitWith(deps.RateLimiter, 10, 1*time.Minute), middleware.MaxBodySize(1<<20)).
			Post("/v1/billing/checkout", deps.Checkout.CreateCheckoutSession)
		r.With(middleware.RateLimitWith(deps.RateLimiter, 30, 1*time.Minute)).
			Get("/v1/billing/checkout/{session_id}", deps.Checkout.GetCheckoutSessionStatus)
	}

	// Public Allegro webhook endpoints — no JWT, HMAC-verified, rate-limited (120 req/min per IP)
	if deps.AllegroWebhook != nil {
		r.With(middleware.RateLimitWith(deps.RateLimiter, 120, 1*time.Minute)).
			Post("/v1/webhooks/allegro", deps.AllegroWebhook.HandleWebhook)
		r.With(middleware.RateLimitWith(deps.RateLimiter, 120, 1*time.Minute)).
			Post("/v1/webhooks/allegro/{integrationID}", deps.AllegroWebhook.HandleWebhook)
	}

	// Public InPost webhook endpoints — no JWT, HMAC-verified, rate-limited (120 req/min per IP)
	if deps.InPostWebhook != nil {
		r.With(middleware.RateLimitWith(deps.RateLimiter, 120, 1*time.Minute)).
			Post("/v1/webhooks/inpost", deps.InPostWebhook.HandleWebhook)
		r.With(middleware.RateLimitWith(deps.RateLimiter, 120, 1*time.Minute)).
			Post("/v1/webhooks/inpost/{integrationID}", deps.InPostWebhook.HandleWebhook)
	}

	// Generic public webhook route — no JWT, signature-verified, rate-limited (120 req/min per IP)
	r.With(middleware.RateLimitWith(deps.RateLimiter, 120, 1*time.Minute)).
		Post("/v1/webhooks/{provider}/{tenant_id}", deps.Webhook.Receive)

	// Stripe webhook endpoint — no JWT, signature-verified, rate-limited (120 req/min per IP)
	if deps.StripeWebhook != nil {
		r.With(middleware.RateLimitWith(deps.RateLimiter, 120, 1*time.Minute), middleware.MaxBodySize(1<<16)).
			Post("/v1/webhooks/stripe", deps.StripeWebhook.HandleWebhook)
	}

	// Public return self-service routes — no JWT, rate-limited
	r.Route("/v1/public/returns", func(r chi.Router) {
		r.Use(middleware.RateLimitWith(deps.RateLimiter, 30, 1*time.Minute))
		r.Use(middleware.MaxBodySize(1 << 20))
		r.Post("/", deps.PublicReturn.CreatePublicReturn)
		r.Get("/{token}", deps.PublicReturn.GetByToken)
		r.Get("/{token}/status", deps.PublicReturn.GetStatusByToken)
	})

	// Public order tracking — no JWT, rate-limited (10 req/min per IP)
	// POST preferred (email in body, not in URL/logs). GET kept for backward compat.
	r.With(middleware.RateLimitWith(deps.RateLimiter, 10, 1*time.Minute)).
		Get("/v1/tracking/{tenant_slug}/{order_id}", deps.Tracking.TrackOrder)
	r.With(middleware.RateLimitWith(deps.RateLimiter, 10, 1*time.Minute), middleware.MaxBodySize(1<<20)).
		Post("/v1/tracking/{tenant_slug}/{order_id}", deps.Tracking.TrackOrder)

	// Public supplier portal routes — no JWT, token-authenticated, rate-limited (30 req/min per IP)
	if deps.SupplierPortal != nil {
		r.Route("/v1/supplier-portal", func(r chi.Router) {
			r.Use(middleware.RateLimitWith(deps.RateLimiter, 30, 1*time.Minute))
			r.Use(middleware.MaxBodySize(1 << 20))
			r.Get("/orders", deps.SupplierPortal.ListOrders)
			r.Get("/orders/{id}", deps.SupplierPortal.GetOrder)
			r.Post("/orders/{id}/confirm", deps.SupplierPortal.ConfirmOrder)
			r.Post("/orders/{id}/ship", deps.SupplierPortal.ShipOrder)
			r.Post("/orders/{id}/messages", deps.SupplierPortal.AddMessage)
			r.Get("/orders/{id}/messages", deps.SupplierPortal.ListMessages)
		})
	}

	// Public product feed routes — no JWT, token-authenticated, rate-limited (60 req/hour per IP)
	if deps.Feed != nil {
		r.Route("/v1/feeds", func(r chi.Router) {
			r.Use(middleware.RateLimitWith(deps.RateLimiter, 60, 1*time.Hour))
			r.Get("/ceneo/{tenant_id}/{token}", deps.Feed.ServeCeneoFeed)
			r.Get("/google/{tenant_id}/{token}", deps.Feed.ServeGoogleFeed)
		})
	}

	// WebSocket endpoint — auth via query param ticket, must be before JWT middleware
	r.With(middleware.RateLimitWith(deps.RateLimiter, 30, 1*time.Minute)).
		Get("/v1/ws", deps.WS.ServeWS)

	// Platform-admin routes — JWT required + platform-admin authorization.
	// Isolated from tenant routing: mounted as a sibling of /v1 so it does NOT
	// inherit TenantPlanGuard or any tenant RBAC. Backend middleware is
	// authoritative; frontend visibility is never trusted as authorization.
	// Fail loud on a half-wired boundary rather than silently dropping the guard.
	if (deps.PlatformAdmin == nil) != (deps.Platform == nil) {
		panic("router: PlatformAdmin and Platform must both be set or both be nil")
	}
	if deps.PlatformAdmin != nil && deps.Platform != nil {
		r.Route("/v1/platform", func(r chi.Router) {
			r.Use(middleware.RateLimitWith(deps.RateLimiter, 60, 1*time.Minute))
			r.Use(middleware.JWTAuth(deps.TokenSvc, deps.TokenBlacklist))
			r.Use(middleware.MaxBodySize(1 << 20))
			r.Use(middleware.RequirePlatformAdmin(deps.PlatformAdmin))
			r.Get("/me", deps.Platform.Me)

			// Provider Integration Studio registry (OPE-405). Each route is
			// additionally guarded by a specific platform permission.
			if deps.Provider != nil {
				rpp := middleware.RequirePlatformPermission
				r.Route("/providers", func(r chi.Router) {
					r.With(rpp(model.PermPlatformProvidersRead)).Get("/", deps.Provider.ListDefinitions)
					r.With(rpp(model.PermPlatformProvidersWrite)).Post("/", deps.Provider.CreateDefinition)
					r.Route("/{id}", func(r chi.Router) {
						r.With(rpp(model.PermPlatformProvidersRead)).Get("/", deps.Provider.GetDefinition)
						r.With(rpp(model.PermPlatformProvidersWrite)).Patch("/", deps.Provider.PatchDefinition)
						r.With(rpp(model.PermPlatformProvidersRead)).Get("/versions", deps.Provider.ListVersions)
						r.With(rpp(model.PermPlatformProvidersWrite)).Post("/versions", deps.Provider.CreateVersion)
						r.Route("/versions/{version_id}", func(r chi.Router) {
							r.With(rpp(model.PermPlatformProvidersRead)).Get("/", deps.Provider.GetVersion)
							r.With(rpp(model.PermPlatformProvidersRead)).Get("/publication-events", deps.Provider.ListPublicationEvents)
							r.With(rpp(model.PermPlatformProvidersRead)).Get("/schema", deps.Provider.GetSchema)
							r.With(rpp(model.PermPlatformProvidersWrite)).Patch("/schema", deps.Provider.UpdateSchema)
							r.With(rpp(model.PermPlatformProvidersRead)).Get("/capabilities", deps.Provider.GetCapabilities)
							r.With(rpp(model.PermPlatformProvidersWrite)).Patch("/capabilities", deps.Provider.UpdateCapabilities)
							r.With(rpp(model.PermPlatformProvidersRead)).Get("/status-mappings", deps.Provider.GetStatusMappings)
							r.With(rpp(model.PermPlatformProvidersWrite)).Patch("/status-mappings", deps.Provider.UpdateStatusMappings)
							r.With(rpp(model.PermPlatformProvidersRead)).Get("/gaps", deps.Provider.ListGaps)
							r.With(rpp(model.PermPlatformProvidersWrite)).Post("/gaps", deps.Provider.CreateGap)
							r.With(rpp(model.PermPlatformProvidersWrite)).Patch("/gaps/{gap_id}", deps.Provider.UpdateGap)
							r.With(rpp(model.PermPlatformProvidersPublish)).Post("/publish", deps.Provider.Publish)
							r.With(rpp(model.PermPlatformProvidersPublish)).Post("/disable", deps.Provider.EmergencyDisable)
							r.With(rpp(model.PermPlatformProvidersPublish)).Post("/enable-tenant", deps.Provider.EnableTenant)

							// Validation engine (OPE-408).
							if deps.ProviderValidation != nil {
								r.With(rpp(model.PermPlatformProvidersRead)).Get("/probes", deps.ProviderValidation.GetProbes)
								r.With(rpp(model.PermPlatformProvidersWrite)).Patch("/probes", deps.ProviderValidation.UpdateProbes)
								r.With(rpp(model.PermPlatformProvidersValidate)).Post("/validate", deps.ProviderValidation.StartRun)
								r.With(rpp(model.PermPlatformProvidersRead)).Get("/validation-runs", deps.ProviderValidation.ListRuns)
								r.With(rpp(model.PermPlatformProvidersRead)).Get("/validation-runs/{run_id}", deps.ProviderValidation.GetRun)
								r.With(rpp(model.PermPlatformProvidersValidate)).Post("/validation-runs/{run_id}/results", deps.ProviderValidation.RecordResult)
								r.With(rpp(model.PermPlatformProvidersValidate)).Post("/validation-runs/{run_id}/complete", deps.ProviderValidation.CompleteRun)
							}
						})
					})
				})
			}
		})
	}

	// Authenticated routes — JWT required
	r.Route("/v1", func(r chi.Router) {
		r.Use(middleware.JWTAuth(deps.TokenSvc, deps.TokenBlacklist))
		if sentry.CurrentHub().Client() != nil {
			r.Use(middleware.SentryContext)
		}

		// Plan enforcement — blocks suspended/past_due tenants
		if deps.PlanCache != nil {
			r.Use(middleware.TenantPlanGuard(deps.PlanCache, deps.Pool))
		}

		// Upload endpoint — has its own body size limit, no global MaxBodySize
		r.Post("/uploads", deps.Upload.Upload)

		// Background removal — has its own body size limit (accepts image uploads).
		// Gated by the "ai" feature (readiness.json lists these endpoints under "ai").
		if deps.BGRemoval != nil {
			r.With(requireFeature("ai")).Post("/images/remove-background", deps.BGRemoval.RemoveBackground)
			r.With(requireFeature("ai")).Get("/images/remove-background/status", deps.BGRemoval.Status)
		}

		// All other authenticated routes get a 1MB body size limit
		r.Group(func(r chi.Router) {
			r.Use(middleware.MaxBodySize(1 << 20))

			r.Get("/order-statuses", deps.Settings.GetOrderStatuses)
			r.Get("/custom-fields", deps.Settings.GetCustomFields)
			r.Get("/product-categories", deps.Settings.GetProductCategories)

			// Billing — subscription status (authenticated)
			if deps.Checkout != nil {
				r.Get("/billing/subscription", deps.Checkout.GetSubscription)
			}

			// Onboarding wizard
			r.Route("/onboarding", func(r chi.Router) {
				r.Get("/status", deps.Settings.GetOnboardingStatus)
				r.Group(func(r chi.Router) {
					r.Use(requirePermission(model.PermSettingsManage))
					r.Put("/step/{step}", deps.Settings.UpdateOnboardingStep)
					r.Post("/complete", deps.Settings.CompleteOnboarding)
				})
			})

			// Settings — requires settings.manage
			r.Route("/settings", func(r chi.Router) {
				r.Use(requirePermission(model.PermSettingsManage))
				r.Get("/export", deps.Settings.ExportSettings)
				r.Post("/import", deps.Settings.ImportSettings)
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
				r.Get("/inventory", deps.Settings.GetInventorySettings)
				r.Put("/inventory", deps.Settings.UpdateInventorySettings)
				r.Get("/onboarding", deps.Settings.GetOnboardingSettings)
				r.Put("/onboarding", deps.Settings.UpdateOnboardingSettings)
				r.Get("/print-templates", deps.Print.GetPrintTemplates)
				r.Put("/print-templates", deps.Print.UpdatePrintTemplates)
				r.Get("/ksef", deps.KSeF.GetSettings)
				r.Put("/ksef", deps.KSeF.UpdateSettings)
				r.Post("/ksef/test", deps.KSeF.TestConnection)
				if deps.Accounting != nil {
					r.Get("/accounting", deps.Accounting.GetAccountingSettings)
					r.Put("/accounting", deps.Accounting.UpdateAccountingSettings)
					r.Post("/accounting/test", deps.Accounting.TestConnection)
				}
				if deps.Feed != nil {
					r.Get("/feeds", deps.Feed.GetConfig)
					r.Put("/feeds", deps.Feed.UpdateConfig)
					r.Post("/feeds/regenerate-token", deps.Feed.RegenerateToken)
				}
			})

			// Audit and webhook delivery visibility
			r.Group(func(r chi.Router) {
				r.Use(requirePermission(model.PermAuditView))
				r.Get("/audit", deps.Audit.List)
				r.Get("/webhook-deliveries", deps.WebhookDelivery.List)
			})
			r.Group(func(r chi.Router) {
				r.Use(requirePermission(model.PermSettingsManage))

				// Webhook configuration — mirrors /v1/settings/webhooks
				r.Get("/webhooks", deps.Settings.GetWebhooks)
				r.Put("/webhooks", deps.Settings.UpdateWebhooks)
			})

			// Sync jobs — requires integrations.manage
			r.Route("/sync-jobs", func(r chi.Router) {
				r.Use(requireFeature("sync_jobs"))
				r.Use(requirePermission(model.PermIntegrationsManage))
				r.Get("/", deps.SyncJob.List)
				r.Get("/{id}", deps.SyncJob.Get)
			})

			// Any authenticated user
			r.Get("/users/me", deps.User.Me)
			r.Patch("/users/me", deps.User.UpdateMe)
			r.With(middleware.RateLimitByAuthenticatedUserWith(deps.RateLimiter, 5, 1*time.Minute)).
				Patch("/users/me/password", deps.User.ChangePassword)

			// User management — requires users.manage
			r.Route("/users", func(r chi.Router) {
				r.Use(requirePermission(model.PermUsersManage))
				r.Get("/", deps.User.List)
				r.Post("/", deps.User.Create)
				r.Patch("/{id}", deps.User.Update)
				r.Delete("/{id}", deps.User.Delete)
			})

			// Invitation management — requires users.manage
			if deps.Invitation != nil {
				r.Route("/invitations", func(r chi.Router) {
					r.Use(requirePermission(model.PermUsersManage))
					r.Get("/", deps.Invitation.List)
					r.Post("/", deps.Invitation.Create)
					r.Delete("/{id}", deps.Invitation.Delete)
				})
			}

			// Orders
			r.Route("/orders", func(r chi.Router) {
				r.With(requirePermission(model.PermOrdersView)).Get("/", deps.Order.List)
				r.With(requirePermission(model.PermOrdersCreate)).Post("/", deps.Order.Create)
				r.With(requirePermission(model.PermOrdersExport)).Get("/export", deps.Order.ExportCSV)
				r.With(requirePermission(model.PermOrdersView)).Get("/{id}", deps.Order.Get)
				r.With(requirePermission(model.PermOrdersEdit)).Patch("/{id}", deps.Order.Update)
				r.With(requirePermission(model.PermOrdersEdit)).Post("/{id}/status", deps.Order.TransitionStatus)
				r.With(requirePermission(model.PermOrdersView)).Get("/{id}/groups", deps.OrderGroup.ListByOrder)
				r.With(requirePermission(model.PermAuditView, model.PermOrdersView)).Get("/{id}/audit", deps.Order.GetAudit)
				r.With(requirePermission(model.PermInvoicesView)).Get("/{id}/invoices", deps.Invoice.ListByOrder)
				r.With(requirePermission(model.PermOrdersView)).Get("/{id}/packing-slip", deps.Print.GetPackingSlip)
				r.With(requirePermission(model.PermOrdersView)).Get("/{id}/print", deps.Print.GetOrderSummary)
				r.With(requirePermission(model.PermOrdersEdit)).Post("/{id}/pack", deps.Barcode.PackOrder)
				r.With(requirePermission(model.PermOrdersView)).Get("/{id}/tickets", deps.Helpdesk.ListOrderTickets)
				r.With(requirePermission(model.PermOrdersEdit)).Post("/{id}/tickets", deps.Helpdesk.CreateOrderTicket)
				r.With(requirePermission(model.PermShipmentsView)).Get("/{id}/shipments", deps.Shipment.ListByOrder)
				r.With(requirePermission(model.PermShipmentsCreate)).Post("/{id}/shipments", deps.Shipment.CreateForOrder)

				r.With(requirePermission(model.PermOrdersDelete)).Delete("/{id}", deps.Order.Delete)
				r.With(requirePermission(model.PermOrdersEdit)).Post("/bulk-status", deps.Order.BulkTransitionStatus)
				r.With(requirePermission(model.PermOrdersEdit)).Post("/merge", deps.OrderGroup.MergeOrders)
				r.With(requirePermission(model.PermOrdersCreate)).Post("/import/preview", deps.Import.Preview)
				r.With(requirePermission(model.PermOrdersCreate)).Post("/import", deps.Import.Import)
				r.With(requirePermission(model.PermOrdersCreate)).Post("/{id}/duplicate", deps.Order.DuplicateOrder)
				r.With(requirePermission(model.PermOrdersEdit)).Post("/{id}/split", deps.OrderGroup.SplitOrder)
			})

			// BaseLinker import
			r.Route("/import/baselinker", func(r chi.Router) {
				r.Use(requirePermission(model.PermOrdersCreate))
				r.Post("/orders/preview", deps.Import.BaseLinkerPreview)
				r.Post("/orders", deps.Import.BaseLinkerImport)
			})

			// Invoices
			r.Route("/invoices", func(r chi.Router) {
				r.Use(requireFeature("invoicing"))
				r.With(requirePermission(model.PermInvoicesView)).Get("/", deps.Invoice.List)
				r.With(requirePermission(model.PermInvoicesCreate)).Post("/", deps.Invoice.Create)
				r.Route("/{id}", func(r chi.Router) {
					r.With(requirePermission(model.PermInvoicesView)).Get("/", deps.Invoice.Get)
					r.With(requirePermission(model.PermInvoicesView)).Get("/pdf", deps.Invoice.GetPDF)
					r.With(requirePermission(model.PermInvoicesView)).Get("/ksef/status", deps.KSeF.CheckKSeFStatus)
					r.With(requirePermission(model.PermInvoicesView)).Get("/ksef/upo", deps.KSeF.GetUPO)
					r.With(requirePermission(model.PermInvoicesDelete)).Delete("/", deps.Invoice.Cancel)
					r.With(requirePermission(model.PermInvoicesCreate)).Post("/ksef/send", deps.KSeF.SendToKSeF)
				})

				r.With(requirePermission(model.PermInvoicesCreate)).Post("/ksef/bulk-send", deps.KSeF.BulkSendToKSeF)
			})

			// Shipments
			r.Route("/shipments", func(r chi.Router) {
				r.With(requirePermission(model.PermShipmentsView)).Get("/", deps.Shipment.List)
				r.With(requirePermission(model.PermShipmentsCreate)).Post("/", deps.Shipment.Create)
				r.With(requirePermission(model.PermShipmentsCreate)).Post("/batch-labels", deps.Shipment.BatchLabels)
				r.With(requirePermission(model.PermShipmentsCreate)).Post("/dispatch-order", deps.Shipment.CreateDispatchOrder)
				r.With(requirePermission(model.PermShipmentsView)).Get("/{id}", deps.Shipment.Get)
				r.With(requirePermission(model.PermShipmentsEdit)).Patch("/{id}", deps.Shipment.Update)
				r.With(requirePermission(model.PermShipmentsEdit)).Post("/{id}/status", deps.Shipment.TransitionStatus)
				r.With(requirePermission(model.PermShipmentsCreate)).Post("/{id}/label", deps.Shipment.GenerateLabel)
				r.With(requirePermission(model.PermShipmentsView)).Get("/{id}/tracking", deps.Shipment.GetTracking)
				r.With(requirePermission(model.PermShipmentsDelete)).Delete("/{id}", deps.Shipment.Delete)
			})

			// Returns
			r.Route("/returns", func(r chi.Router) {
				r.With(requirePermission(model.PermReturnsView)).Get("/", deps.Return.List)
				r.With(requirePermission(model.PermReturnsCreate)).Post("/", deps.Return.Create)
				r.Route("/{id}", func(r chi.Router) {
					r.With(requirePermission(model.PermReturnsView)).Get("/", deps.Return.Get)
					r.With(requirePermission(model.PermReturnsEdit)).Patch("/", deps.Return.Update)
					r.With(requirePermission(model.PermReturnsEdit)).Post("/status", deps.Return.TransitionStatus)
					r.With(requirePermission(model.PermReturnsView)).Get("/print", deps.Print.GetReturnSlip)
					r.With(requirePermission(model.PermReturnsDelete)).Delete("/", deps.Return.Delete)
				})
			})

			// Products
			r.Route("/products", func(r chi.Router) {
				r.With(requirePermission(model.PermProductsView)).Get("/", deps.Product.List)
				r.With(requirePermission(model.PermProductsCreate)).Post("/", deps.Product.Create)
				r.With(requirePermission(model.PermProductsView)).Get("/export", deps.Product.ExportCSV)
				r.With(requirePermission(model.PermProductsView)).Get("/{id}", deps.Product.Get)
				r.With(requirePermission(model.PermProductsEdit)).Patch("/{id}", deps.Product.Update)
				r.With(requirePermission(model.PermProductsView)).Get("/{id}/stock", deps.Warehouse.ListProductStock)
				r.With(requirePermission(model.PermProductsView)).Get("/{id}/supplier-link", deps.Supplier.SupplierLink)
				r.With(requirePermission(model.PermProductsDelete)).Delete("/{id}", deps.Product.Delete)
				r.With(requirePermission(model.PermProductsCreate)).Post("/import/preview", deps.Product.ImportPreview)
				r.With(requirePermission(model.PermProductsCreate)).Post("/import", deps.Product.ImportCSV)
				r.With(requirePermission(model.PermProductsCreate)).Post("/import/baselinker/preview", deps.Product.BLImportPreview)
				r.With(requirePermission(model.PermProductsCreate)).Post("/import/baselinker", deps.Product.BLImportCSV)
				r.With(requirePermission(model.PermProductsEdit)).Post("/redownload-images", deps.Product.RedownloadImages)

				// Background removal for product images
				if deps.BGRemoval != nil {
					r.With(requireFeature("ai"), requirePermission(model.PermProductsEdit)).Post("/{id}/images/{index}/remove-background", deps.BGRemoval.RemoveProductImageBackground)
				}

				// Bundles
				r.Route("/{id}/bundle", func(r chi.Router) {
					r.With(requirePermission(model.PermProductsView)).Get("/", deps.Bundle.ListComponents)
					r.With(requirePermission(model.PermProductsEdit)).Post("/", deps.Bundle.AddComponent)
					r.With(requirePermission(model.PermProductsView)).Get("/stock", deps.Bundle.GetBundleStock)
					r.With(requirePermission(model.PermProductsEdit)).Put("/{componentId}", deps.Bundle.UpdateComponent)
					r.With(requirePermission(model.PermProductsEdit)).Delete("/{componentId}", deps.Bundle.RemoveComponent)
				})

				// Variants
				r.Route("/{productId}/variants", func(r chi.Router) {
					r.Use(requireFeature("product_variants"))
					r.With(requirePermission(model.PermProductsView)).Get("/", deps.Variant.List)
					r.With(requirePermission(model.PermProductsCreate)).Post("/", deps.Variant.Create)
					r.With(requirePermission(model.PermProductsView)).Get("/{id}", deps.Variant.Get)
					r.With(requirePermission(model.PermProductsEdit)).Patch("/{id}", deps.Variant.Update)
					r.With(requirePermission(model.PermProductsDelete)).Delete("/{id}", deps.Variant.Delete)
				})

				// Marketplace listings
				r.Route("/{productId}/listings", func(r chi.Router) {
					r.Use(requireFeature("product_listings"))
					r.Use(requirePermission(model.PermIntegrationsManage))
					if deps.AllegroListings != nil {
						r.Get("/", deps.AllegroListings.ListByProduct)
						r.Post("/allegro", deps.AllegroListings.CreateListing)
						r.Get("/{listingId}", deps.AllegroListings.GetListing)
						r.Patch("/{listingId}", deps.AllegroListings.UpdateListing)
						r.Delete("/{listingId}", deps.AllegroListings.DeleteListing)
						r.Post("/{listingId}/sync", deps.AllegroListings.SyncListing)
					}
					if deps.WooCommerceListings != nil {
						r.Post("/woocommerce", deps.WooCommerceListings.CreateListing)
					}
					if deps.ErliListings != nil {
						r.Post("/erli", deps.ErliListings.CreateListing)
					}
					if deps.OLXListings != nil {
						r.Post("/olx", deps.OLXListings.CreateListing)
					}
					if deps.EbayListings != nil {
						r.Post("/ebay", deps.EbayListings.CreateListing)
					}
				})
			})

			// Integrations — requires integrations.manage
			r.Route("/integrations", func(r chi.Router) {
				r.Use(requirePermission(model.PermIntegrationsManage))

				// Allegro OAuth2 + fulfillment + shipment management + comms
				r.Route("/allegro", func(r chi.Router) {
					r.Get("/auth-url", deps.AllegroAuth.GetAuthURL)
					r.Post("/callback", deps.AllegroAuth.HandleCallback)

					// Fulfillment + tracking (Batch 1)
					r.Get("/carriers", deps.Allegro.ListCarriers)
					r.Post("/sync", deps.Allegro.SyncOrders)
					r.Post("/orders/{orderId}/fulfillment", deps.Allegro.UpdateFulfillment)
					r.Post("/orders/{orderId}/tracking", deps.Allegro.AddTracking)

					// Import Allegro offers as products + listings
					r.Post("/import-offers", deps.Allegro.ImportOffers)

					// Shipment management ("Wysyłam z Allegro")
					r.Get("/delivery-services", deps.AllegroShipment.ListDeliveryServices)
					r.Post("/shipments", deps.AllegroShipment.CreateShipment)
					r.Get("/shipments/{shipmentId}/label", deps.AllegroShipment.GetLabel)
					r.Delete("/shipments/{shipmentId}", deps.AllegroShipment.CancelShipment)
					r.Post("/pickup-proposals", deps.AllegroShipment.GetPickupProposals)
					r.Post("/pickups", deps.AllegroShipment.SchedulePickup)
					r.Post("/protocol", deps.AllegroShipment.GenerateProtocol)

					// Messaging
					r.Get("/messages", deps.AllegroComms.ListThreads)
					r.Get("/messages/{threadId}", deps.AllegroComms.GetMessages)
					r.Post("/messages/{threadId}", deps.AllegroComms.SendMessage)

					// Returns
					r.Get("/returns", deps.AllegroComms.ListAllegroReturns)
					r.Get("/returns/{returnId}", deps.AllegroComms.GetAllegroReturn)
					r.Post("/returns/{returnId}/reject", deps.AllegroComms.RejectAllegroReturn)

					// Refunds
					r.Post("/refunds", deps.AllegroComms.CreateRefund)
					r.Get("/refunds", deps.AllegroComms.ListRefunds)

					// Account & billing (Batch 4)
					if deps.AllegroAccount != nil {
						r.Get("/account", deps.AllegroAccount.GetAccount)
						r.Get("/billing", deps.AllegroAccount.GetBilling)

						// Offer management
						r.Get("/offers", deps.AllegroAccount.ListOffers)
						r.Post("/offers/{offerId}/deactivate", deps.AllegroAccount.DeactivateOffer)
						r.Post("/offers/{offerId}/activate", deps.AllegroAccount.ActivateOffer)
						r.Patch("/offers/{offerId}/stock", deps.AllegroAccount.UpdateOfferStock)
						r.Patch("/offers/{offerId}/price", deps.AllegroAccount.UpdateOfferPrice)
					}

					// Catalog + Finance (Batch 5)
					if deps.AllegroCatalog != nil {
						r.Get("/categories", deps.AllegroCatalog.ListCategories)
						r.Get("/categories/search", deps.AllegroCatalog.SearchCategories)
						r.Get("/categories/{categoryId}", deps.AllegroCatalog.GetCategory)
						r.Get("/categories/{categoryId}/parameters", deps.AllegroCatalog.GetCategoryParameters)
						r.Get("/products/catalog", deps.AllegroCatalog.SearchProducts)
						r.Get("/products/catalog/{productId}", deps.AllegroCatalog.GetProduct)
						r.Get("/offers/listing", deps.AllegroCatalog.SearchListing)
						r.Get("/pricing/fees", deps.AllegroCatalog.GetFeePreview)
						r.Get("/pricing/commissions", deps.AllegroCatalog.GetCommissions)
					}

					// After-sales policies (return policies, warranties, size tables)
					if deps.AllegroPolicies != nil {
						r.Get("/return-policies", deps.AllegroPolicies.ListReturnPolicies)
						r.Post("/return-policies", deps.AllegroPolicies.CreateReturnPolicy)
						r.Get("/return-policies/{policyId}", deps.AllegroPolicies.GetReturnPolicy)
						r.Put("/return-policies/{policyId}", deps.AllegroPolicies.UpdateReturnPolicy)

						r.Get("/warranties", deps.AllegroPolicies.ListWarranties)
						r.Post("/warranties", deps.AllegroPolicies.CreateWarranty)
						r.Get("/warranties/{warrantyId}", deps.AllegroPolicies.GetWarranty)
						r.Put("/warranties/{warrantyId}", deps.AllegroPolicies.UpdateWarranty)

						r.Get("/size-tables", deps.AllegroPolicies.ListSizeTables)
						r.Post("/size-tables", deps.AllegroPolicies.CreateSizeTable)
						r.Get("/size-tables/{tableId}", deps.AllegroPolicies.GetSizeTable)
						r.Put("/size-tables/{tableId}", deps.AllegroPolicies.UpdateSizeTable)
						r.Delete("/size-tables/{tableId}", deps.AllegroPolicies.DeleteSizeTable)
					}

					// Promotions
					if deps.AllegroPromotions != nil {
						r.Get("/promotions", deps.AllegroPromotions.ListPromotions)
						r.Post("/promotions", deps.AllegroPromotions.CreatePromotion)
						r.Get("/promotions/{promotionId}", deps.AllegroPromotions.GetPromotion)
						r.Put("/promotions/{promotionId}", deps.AllegroPromotions.UpdatePromotion)
						r.Delete("/promotions/{promotionId}", deps.AllegroPromotions.DeletePromotion)
						r.Get("/promotion-badges", deps.AllegroPromotions.ListBadges)
					}

					// Delivery settings
					if deps.AllegroDelivery != nil {
						r.Get("/delivery-settings", deps.AllegroDelivery.GetDeliverySettings)
						r.Put("/delivery-settings", deps.AllegroDelivery.UpdateDeliverySettings)
						r.Get("/shipping-rates", deps.AllegroDelivery.ListShippingRates)
						r.Post("/shipping-rates", deps.AllegroDelivery.CreateShippingRate)
						r.Post("/shipping-rates/auto-generate", deps.AllegroDelivery.AutoGenerateShippingRate)
						r.Get("/shipping-rates/{rateId}", deps.AllegroDelivery.GetShippingRate)
						r.Put("/shipping-rates/{rateId}", deps.AllegroDelivery.UpdateShippingRate)
						r.Get("/delivery-methods", deps.AllegroDelivery.ListDeliveryMethods)
					}

					// Disputes
					if deps.AllegroDisputes != nil {
						r.Get("/disputes", deps.AllegroDisputes.ListDisputes)
						r.Get("/disputes/{disputeId}", deps.AllegroDisputes.GetDispute)
						r.Get("/disputes/{disputeId}/messages", deps.AllegroDisputes.ListDisputeMessages)
						r.Post("/disputes/{disputeId}/messages", deps.AllegroDisputes.SendDisputeMessage)
					}

					// Ratings
					if deps.AllegroRatings != nil {
						r.Get("/ratings", deps.AllegroRatings.ListRatings)
						r.Get("/ratings/{ratingId}/answer", deps.AllegroRatings.GetAnswer)
						r.Put("/ratings/{ratingId}/answer", deps.AllegroRatings.CreateAnswer)
						r.Delete("/ratings/{ratingId}/answer", deps.AllegroRatings.DeleteAnswer)
						r.Post("/ratings/{ratingId}/removal", deps.AllegroRatings.RequestRemoval)
					}

				})

				// Amazon SP-API
				r.Route("/amazon", func(r chi.Router) {
					r.Post("/setup", deps.AmazonAuth.Setup)
					r.Get("/auth-url", deps.AmazonAuth.GetAuthURL)
					r.Post("/callback", deps.AmazonAuth.HandleCallback)
					r.Put("/credentials", deps.AmazonAuth.UpdateCredentials)
				})

				// OLX OAuth2 + listings proxy
				r.Route("/olx", func(r chi.Router) {
					r.Get("/auth-url", deps.OlxAuth.GetAuthURL)
					r.Post("/callback", deps.OlxAuth.HandleCallback)
					if deps.OLXListings != nil {
						r.Get("/categories", deps.OLXListings.ListCategories)
						r.Get("/categories/{categoryId}/attributes", deps.OLXListings.GetCategoryAttributes)
						r.Get("/categories/suggest", deps.OLXListings.SuggestCategory)
						r.Get("/cities", deps.OLXListings.ListCities)
						r.Get("/cities/{cityId}/districts", deps.OLXListings.ListDistricts)
					}
				})

				// eBay OAuth2 + fulfillment + tracking
				r.Route("/ebay", func(r chi.Router) {
					if deps.EbayAuth != nil {
						r.Get("/auth-url", deps.EbayAuth.GetAuthURL)
						r.Post("/callback", deps.EbayAuth.HandleCallback)
					}
					if deps.Ebay != nil {
						r.Get("/carriers", deps.Ebay.ListCarriers)
						r.Post("/orders/{orderId}/tracking", deps.Ebay.AddTracking)
						r.Post("/orders/{orderId}/refund", deps.Ebay.IssueRefund)
					}
					if deps.EbayListings != nil {
						r.Get("/policies", deps.EbayListings.ListPolicies)
						r.Get("/offers", deps.EbayListings.ListOffers)
						r.Post("/import-offers", deps.EbayListings.ImportOffers)
					}
				})

				// Store platform integrations (Shoper, PrestaShop, Shopify)
				if deps.StoreAuth != nil {
					r.Post("/shoper/setup", deps.StoreAuth.SetupShoper)
					r.Post("/prestashop/setup", deps.StoreAuth.SetupPrestaShop)
					r.Post("/shopify/setup", deps.StoreAuth.SetupShopify)
				}

				// Marketplace category mappings (per-integration)
				if deps.MarketplaceCategoryMapping != nil {
					r.Route("/{id}/category-mappings", func(r chi.Router) {
						r.Get("/", deps.MarketplaceCategoryMapping.List)
						r.Put("/", deps.MarketplaceCategoryMapping.Upsert)
						r.Delete("/{mid}", deps.MarketplaceCategoryMapping.Delete)
					})
				}

				r.Get("/", deps.Integration.List)
				r.Post("/", deps.Integration.Create)
				r.Get("/{id}", deps.Integration.Get)
				r.Patch("/{id}", deps.Integration.Update)
				r.Delete("/{id}", deps.Integration.Delete)
			})

			// Suppliers — requires integrations.manage
			r.Route("/suppliers", func(r chi.Router) {
				r.Use(requireFeature("suppliers"))
				r.Use(requirePermission(model.PermIntegrationsManage))
				r.Get("/", deps.Supplier.List)
				r.Post("/", deps.Supplier.Create)
				r.Get("/{id}", deps.Supplier.Get)
				r.Patch("/{id}", deps.Supplier.Update)
				r.Delete("/{id}", deps.Supplier.Delete)
				r.Post("/{id}/sync", deps.Supplier.Sync)
				r.Get("/{id}/products", deps.Supplier.ListProducts)
				r.Get("/{id}/products/categories", deps.Supplier.ListSourceCategories)
				r.Post("/{id}/products/import", deps.Supplier.ImportProducts)
				r.Post("/{id}/products/bulk-delete", deps.Supplier.BulkDeleteProducts)
				r.Post("/{id}/products/{spid}/link", deps.Supplier.LinkProduct)
				r.Post("/{id}/products/{spid}/unlink", deps.Supplier.UnlinkProduct)
				r.Post("/{id}/products/{spid}/import-single", deps.Supplier.ImportSingleProduct)
				r.Delete("/{id}/products/{spid}", deps.Supplier.DeleteProduct)
				r.Get("/{id}/category-mappings", deps.Supplier.ListCategoryMappings)
				r.Put("/{id}/category-mappings", deps.Supplier.UpsertCategoryMapping)
				r.Delete("/{id}/category-mappings/{mid}", deps.Supplier.DeleteCategoryMapping)
				r.Get("/{id}/allegro-mappings", deps.Supplier.ListAllegroMappings)
				r.Put("/{id}/allegro-mappings", deps.Supplier.BulkUpsertAllegroMappings)
				r.Delete("/{id}/allegro-mappings/{mid}", deps.Supplier.DeleteAllegroMapping)
				r.Get("/{id}/allegro-mappings/categories", deps.Supplier.ListAllegroMappingCategories)
				r.Get("/{id}/attributes", deps.Supplier.ListSupplierAttributes)

				// BTP wizard
				r.Post("/btp-wizard/start", deps.Supplier.BTPWizardStartImport)
				r.Get("/btp-wizard/{id}/progress", deps.Supplier.BTPWizardImportProgress)
				r.Post("/btp-wizard/{id}/api-keys", deps.Supplier.BTPWizardSetAPIKeys)
				r.Post("/btp-wizard/{id}/sync-settings", deps.Supplier.BTPWizardCompleteSyncSettings)

				// Supplier portal management
				if deps.SupplierPortal != nil {
					r.Post("/{id}/portal/generate-link", deps.SupplierPortal.GenerateLink)
					r.Post("/{id}/portal/revoke", deps.SupplierPortal.RevokeAccess)
					r.Get("/{id}/portal/status", deps.SupplierPortal.GetPortalStatus)
				}
			})

			// Cross-supplier product listing
			r.With(requireFeature("suppliers"), requirePermission(model.PermIntegrationsManage)).Get("/supplier-products", deps.Supplier.ListAllSupplierProducts)

			// Product Categories — requires settings.manage
			if deps.Category != nil {
				r.Route("/categories", func(r chi.Router) {
					r.Use(requirePermission(model.PermSettingsManage))
					r.Get("/", deps.Category.List)
					r.Post("/", deps.Category.Create)
					r.Get("/{id}", deps.Category.Get)
					r.Patch("/{id}", deps.Category.Update)
					r.Delete("/{id}", deps.Category.Delete)
					r.Get("/{id}/descendants", deps.Category.ListDescendants)
				})
			}

			// Purchase Orders
			r.Route("/purchase-orders", func(r chi.Router) {
				r.Use(requireFeature("purchase_orders"))
				r.With(requirePermission(model.PermProductsView)).Get("/", deps.PurchaseOrder.List)
				r.With(requirePermission(model.PermProductsView)).Get("/{id}", deps.PurchaseOrder.Get)
				r.With(requirePermission(model.PermProductsCreate)).Post("/", deps.PurchaseOrder.Create)
				r.With(requirePermission(model.PermProductsEdit)).Put("/{id}", deps.PurchaseOrder.Update)
				r.With(requirePermission(model.PermProductsDelete)).Delete("/{id}", deps.PurchaseOrder.Delete)
				r.With(requirePermission(model.PermProductsEdit)).Post("/{id}/send", deps.PurchaseOrder.Send)
				r.With(requirePermission(model.PermProductsEdit)).Post("/{id}/receive", deps.PurchaseOrder.Receive)
				r.With(requirePermission(model.PermProductsEdit)).Post("/{id}/cancel", deps.PurchaseOrder.Cancel)
			})

			// Dropship Orders
			r.Route("/dropship-orders", func(r chi.Router) {
				r.Use(requireFeature("dropship"))
				r.With(requirePermission(model.PermOrdersView)).Get("/", deps.Dropship.List)
				r.With(requirePermission(model.PermOrdersView)).Get("/{id}", deps.Dropship.Get)
				r.With(requirePermission(model.PermOrdersCreate)).Post("/", deps.Dropship.Create)
				r.With(requirePermission(model.PermOrdersEdit)).Put("/{id}/status", deps.Dropship.UpdateStatus)
				r.With(requirePermission(model.PermOrdersEdit)).Post("/{id}/cancel", deps.Dropship.Cancel)
			})

			// Order dropship auto-route
			r.With(requireFeature("dropship"), requirePermission(model.PermOrdersEdit)).
				Post("/orders/{order_id}/dropship", deps.Dropship.AutoRoute)
			r.With(requireFeature("dropship"), requirePermission(model.PermOrdersView)).
				Get("/orders/{order_id}/dropship-orders", deps.Dropship.GetByOrderID)

			// Warehouses — requires warehouses.manage
			r.Route("/warehouses", func(r chi.Router) {
				r.Use(requireFeature("warehouses"))
				r.Use(requirePermission(model.PermWarehousesManage))
				r.Get("/", deps.Warehouse.List)
				r.Post("/", deps.Warehouse.Create)
				r.Get("/{id}", deps.Warehouse.Get)
				r.Patch("/{id}", deps.Warehouse.Update)
				r.Delete("/{id}", deps.Warehouse.Delete)
				r.Get("/{id}/stock", deps.Warehouse.ListStock)
				r.Put("/{id}/stock", deps.Warehouse.UpsertStock)
			})

			// Customers
			r.Route("/customers", func(r chi.Router) {
				r.With(requirePermission(model.PermCustomersView)).Get("/", deps.Customer.List)
				r.With(requirePermission(model.PermCustomersCreate)).Post("/", deps.Customer.Create)
				r.With(requirePermission(model.PermCustomersView)).Get("/{id}", deps.Customer.Get)
				r.With(requirePermission(model.PermCustomersEdit)).Patch("/{id}", deps.Customer.Update)
				r.With(requirePermission(model.PermCustomersView)).Get("/{id}/orders", deps.Customer.ListOrders)
				r.With(requirePermission(model.PermCustomersDelete)).Delete("/{id}", deps.Customer.Delete)
				r.With(requirePermission(model.PermCustomersCreate)).Post("/import/preview", deps.Customer.ImportPreview)
				r.With(requirePermission(model.PermCustomersCreate)).Post("/import", deps.Customer.ImportCSV)
			})

			// Customer segments
			if deps.Segment != nil {
				r.Route("/segments", func(r chi.Router) {
					r.Use(requireFeature("segments"))
					r.With(requirePermission(model.PermCustomersView)).Get("/", deps.Segment.List)
					r.With(requirePermission(model.PermCustomersView)).Get("/customer/{customer_id}", deps.Segment.GetCustomerSegments)
					r.With(requirePermission(model.PermCustomersView)).Get("/{id}", deps.Segment.Get)
					r.With(requirePermission(model.PermCustomersView)).Get("/{id}/members", deps.Segment.ListMembers)
					r.With(requirePermission(model.PermCustomersCreate)).Post("/", deps.Segment.Create)
					r.With(requirePermission(model.PermReportsView)).Post("/rfm-analysis", deps.Segment.RunRFMAnalysis)
					r.With(requirePermission(model.PermCustomersEdit)).Put("/{id}", deps.Segment.Update)
					r.With(requirePermission(model.PermCustomersDelete)).Delete("/{id}", deps.Segment.Delete)
					r.With(requirePermission(model.PermCustomersEdit)).Post("/{id}/members", deps.Segment.AddMember)
					r.With(requirePermission(model.PermCustomersEdit)).Delete("/{id}/members/{customer_id}", deps.Segment.RemoveMember)
				})
			}

			// Loyalty programs
			if deps.Loyalty != nil {
				r.Route("/loyalty", func(r chi.Router) {
					r.Use(requireFeature("loyalty"))
					r.Route("/programs", func(r chi.Router) {
						r.With(requirePermission(model.PermCustomersView)).Get("/", deps.Loyalty.ListPrograms)
						r.With(requirePermission(model.PermCustomersView)).Get("/{id}", deps.Loyalty.GetProgram)
						r.With(requirePermission(model.PermCustomersView)).Get("/{id}/leaderboard", deps.Loyalty.GetLeaderboard)
						r.With(requirePermission(model.PermCustomersCreate)).Post("/", deps.Loyalty.CreateProgram)
						r.With(requirePermission(model.PermCustomersEdit)).Put("/{id}", deps.Loyalty.UpdateProgram)
						r.With(requirePermission(model.PermCustomersDelete)).Delete("/{id}", deps.Loyalty.DeleteProgram)
						r.With(requirePermission(model.PermCustomersEdit)).Post("/{id}/award", deps.Loyalty.AwardPoints)
						r.With(requirePermission(model.PermCustomersEdit)).Post("/{id}/redeem", deps.Loyalty.RedeemPoints)
					})
					r.With(requirePermission(model.PermCustomersView)).Get("/customers/{customer_id}", deps.Loyalty.GetCustomerLoyaltyStatus)
				})
			}

			// Recurring orders (subscriptions)
			r.Route("/recurring-orders", func(r chi.Router) {
				r.Use(requireFeature("recurring_orders"))
				r.With(requirePermission(model.PermOrdersView)).Get("/", deps.RecurringOrder.List)
				r.With(requirePermission(model.PermOrdersView)).Get("/{id}", deps.RecurringOrder.Get)
				r.With(requirePermission(model.PermOrdersCreate)).Post("/", deps.RecurringOrder.Create)
				r.With(requirePermission(model.PermOrdersEdit)).Put("/{id}", deps.RecurringOrder.Update)
				r.With(requirePermission(model.PermOrdersDelete)).Delete("/{id}", deps.RecurringOrder.Delete)
				r.With(requirePermission(model.PermOrdersEdit)).Post("/{id}/pause", deps.RecurringOrder.Pause)
				r.With(requirePermission(model.PermOrdersEdit)).Post("/{id}/resume", deps.RecurringOrder.Resume)
				r.With(requirePermission(model.PermOrdersEdit)).Post("/{id}/cancel", deps.RecurringOrder.Cancel)
			})

			// Automation rules — requires automation.manage
			r.Route("/automation", func(r chi.Router) {
				r.Use(requireFeature("automation"))
				r.Use(requirePermission(model.PermAutomationManage))
				r.Get("/delayed", deps.Automation.ListDelayed)
				r.Route("/rules", func(r chi.Router) {
					r.Get("/", deps.Automation.List)
					r.Post("/", deps.Automation.Create)
					r.Get("/{id}", deps.Automation.Get)
					r.Patch("/{id}", deps.Automation.Update)
					r.Delete("/{id}", deps.Automation.Delete)
					r.Get("/{id}/logs", deps.Automation.GetLogs)
					r.Post("/{id}/test", deps.Automation.TestRule)
				})
			})

			// Workflow builder — requires automation.manage
			r.Route("/workflows", func(r chi.Router) {
				r.Use(requireFeature("workflows"))
				r.Use(requirePermission(model.PermAutomationManage))
				r.Get("/templates", deps.Workflow.ListTemplates)
				r.Post("/validate", deps.Workflow.Validate)
				r.Post("/convert", deps.Workflow.Convert)
				r.Get("/rules/{id}/workflow", deps.Workflow.GetWorkflowForRule)
			})

			// Stats — requires reports.view
			r.Route("/stats", func(r chi.Router) {
				r.Use(requirePermission(model.PermReportsView))
				r.Get("/dashboard", deps.Stats.GetDashboard)
				r.Get("/products/top", deps.Stats.GetTopProducts)
				r.Get("/revenue/by-source", deps.Stats.GetRevenueBySource)
				r.Get("/trends", deps.Stats.GetOrderTrends)
				r.Get("/payment-methods", deps.Stats.GetPaymentMethodStats)
			})

			// Demand forecast — requires reports.view; config update requires settings.manage
			if deps.Forecast != nil {
				r.Route("/forecast", func(r chi.Router) {
					r.Use(requireFeature("forecast"))
					r.With(requirePermission(model.PermReportsView)).Get("/products", deps.Forecast.ListForecasts)
					r.With(requirePermission(model.PermReportsView)).Get("/products/{id}", deps.Forecast.GetForecast)
					r.With(requirePermission(model.PermReportsView)).Get("/reorder", deps.Forecast.GetReorderRecommendations)
					r.With(requirePermission(model.PermReportsView)).Get("/seasonality/{product_id}", deps.Forecast.GetSeasonality)
					r.With(requirePermission(model.PermReportsView)).Get("/velocity", deps.Forecast.GetVelocity)
					r.With(requirePermission(model.PermSettingsManage)).Get("/config", deps.Forecast.GetConfig)
					r.With(requirePermission(model.PermSettingsManage)).Put("/config", deps.Forecast.UpdateConfig)
				})
			}

			// Carbon footprint — requires reports.view
			if deps.Carbon != nil {
				r.Route("/carbon", func(r chi.Router) {
					r.Use(requireFeature("carbon"))
					r.Use(requirePermission(model.PermReportsView))
					r.Get("/stats", deps.Carbon.GetStats)
					r.Get("/report", deps.Carbon.GetReport)
				})
			}

			// VAT OSS — requires reports.view
			r.Route("/vat-oss", func(r chi.Router) {
				r.Use(requireFeature("vat_oss"))
				r.Use(requirePermission(model.PermReportsView))
				r.Get("/rates", deps.VATOSS.GetAllRates)
				r.Get("/rates/{country}", deps.VATOSS.GetCountryRates)
				r.Post("/calculate", deps.VATOSS.Calculate)
				r.Get("/config", deps.VATOSS.GetConfig)
				r.Put("/config", deps.VATOSS.UpdateConfig)
				r.Get("/report", deps.VATOSS.GetReport)
				r.Get("/report/csv", deps.VATOSS.GetReportCSV)
				r.Get("/threshold", deps.VATOSS.GetThreshold)
			})

			// Barcode lookup — any authenticated user, rate-limited
			r.With(middleware.RateLimitWith(deps.RateLimiter, 120, 1*time.Minute)).
				Get("/barcode/{code}", deps.Barcode.Lookup)

			// Price lists — requires settings.manage
			r.Route("/price-lists", func(r chi.Router) {
				r.Use(requireFeature("price_lists"))
				r.Use(requirePermission(model.PermSettingsManage))
				r.Get("/", deps.PriceList.List)
				r.Post("/", deps.PriceList.Create)
				r.Get("/{id}", deps.PriceList.Get)
				r.Patch("/{id}", deps.PriceList.Update)
				r.Delete("/{id}", deps.PriceList.Delete)
				r.Get("/{id}/items", deps.PriceList.ListItems)
				r.Post("/{id}/items", deps.PriceList.CreateItem)
				r.Delete("/{id}/items/{itemId}", deps.PriceList.DeleteItem)
			})

			// Message templates
			if deps.MessageTemplate != nil {
				r.Route("/message-templates", func(r chi.Router) {
					r.Use(requireFeature("message_templates"))
					r.With(requirePermission(model.PermSettingsManage)).Get("/", deps.MessageTemplate.List)
					r.With(requirePermission(model.PermSettingsManage)).Get("/{id}", deps.MessageTemplate.Get)
					r.With(requirePermission(model.PermSettingsManage)).Post("/", deps.MessageTemplate.Create)
					r.With(requirePermission(model.PermSettingsManage)).Put("/{id}", deps.MessageTemplate.Update)
					r.With(requirePermission(model.PermSettingsManage)).Delete("/{id}", deps.MessageTemplate.Delete)
				})
			}

			// Warehouse documents — requires warehouses.manage
			r.Route("/warehouse-documents", func(r chi.Router) {
				r.Use(requireFeature("warehouse_documents"))
				r.Use(requirePermission(model.PermWarehousesManage))
				r.Get("/", deps.WarehouseDocument.List)
				r.Post("/", deps.WarehouseDocument.Create)
				r.Get("/{id}", deps.WarehouseDocument.Get)
				r.Patch("/{id}", deps.WarehouseDocument.Update)
				r.Delete("/{id}", deps.WarehouseDocument.Delete)
				r.Post("/{id}/confirm", deps.WarehouseDocument.Confirm)
				r.Post("/{id}/cancel", deps.WarehouseDocument.Cancel)
			})

			// Stocktakes (inventory counting) — requires warehouses.manage
			r.Route("/stocktakes", func(r chi.Router) {
				r.Use(requireFeature("stocktakes"))
				r.Use(requirePermission(model.PermWarehousesManage))
				r.Get("/", deps.Stocktake.List)
				r.Post("/", deps.Stocktake.Create)
				r.Get("/{id}", deps.Stocktake.Get)
				r.Delete("/{id}", deps.Stocktake.Delete)
				r.Post("/{id}/start", deps.Stocktake.Start)
				r.Post("/{id}/items/{itemId}/count", deps.Stocktake.RecordCount)
				r.Post("/{id}/complete", deps.Stocktake.Complete)
				r.Post("/{id}/cancel", deps.Stocktake.Cancel)
				r.Get("/{id}/items", deps.Stocktake.ListItems)
			})

			// AI auto-categorization — requires settings.manage, rate limited (calls external APIs)
			r.Route("/ai", func(r chi.Router) {
				r.Use(requireFeature("ai"))
				r.Use(requirePermission(model.PermSettingsManage))
				r.Use(middleware.RateLimitWith(deps.RateLimiter, 20, 1*time.Minute))
				r.Post("/categorize", deps.AI.Categorize)
				r.Post("/describe", deps.AI.Describe)
				r.Post("/bulk-categorize", deps.AI.BulkCategorize)
				r.Post("/improve", deps.AI.Improve)
				r.Post("/translate", deps.AI.Translate)
			})

			// Marketing (Mailchimp) — requires integrations.manage
			r.Route("/marketing", func(r chi.Router) {
				r.Use(requireFeature("marketing"))
				r.Use(requirePermission(model.PermIntegrationsManage))
				r.Post("/sync", deps.Marketing.Sync)
				r.Get("/status", deps.Marketing.Status)
				r.Post("/campaigns", deps.Marketing.CreateCampaign)
			})

			// Helpdesk (Freshdesk) — any authenticated user
			r.Get("/helpdesk/tickets", deps.Helpdesk.ListAllTickets)

			// Exchange rates — requires settings.manage
			r.Route("/exchange-rates", func(r chi.Router) {
				r.Use(requireFeature("exchange_rates"))
				r.Use(requirePermission(model.PermSettingsManage))
				r.Get("/", deps.ExchangeRate.List)
				r.Post("/", deps.ExchangeRate.Create)
				r.Post("/fetch", deps.ExchangeRate.FetchNBP)
				r.Post("/convert", deps.ExchangeRate.Convert)
				r.Get("/{id}", deps.ExchangeRate.Get)
				r.Patch("/{id}", deps.ExchangeRate.Update)
				r.Delete("/{id}", deps.ExchangeRate.Delete)
			})

			// Roles (RBAC) — requires users.manage
			r.Route("/roles", func(r chi.Router) {
				r.Use(requirePermission(model.PermUsersManage))
				r.Get("/", deps.Role.List)
				r.Get("/permissions", deps.Role.ListPermissions)
				r.Post("/", deps.Role.Create)
				r.Get("/{id}", deps.Role.Get)
				r.Patch("/{id}", deps.Role.Update)
				r.Delete("/{id}", deps.Role.Delete)
			})

			// InPost points search (proxy)
			r.Get("/inpost/points", deps.InPostPoint.Search)
			r.Get("/inpost/geowidget-token", deps.Integration.GetGeowidgetToken)

			// Pick & Pack workflow — requires warehouses.manage while the module is validation-gated
			r.Route("/pick-pack/sessions", func(r chi.Router) {
				r.Use(requireFeature("pick_pack"))
				r.Use(requirePermission(model.PermWarehousesManage))
				r.Post("/", deps.PickPack.CreateSession)
				r.Get("/", deps.PickPack.ListSessions)
				r.Get("/{id}", deps.PickPack.GetSession)
				r.Post("/{id}/scan", deps.PickPack.ScanItem)
				r.Post("/{id}/move-to-packing", deps.PickPack.MoveToPacking)
				r.Post("/{id}/items/{itemId}/pack", deps.PickPack.MarkItemPacked)
				r.Post("/{id}/complete", deps.PickPack.CompleteSession)
				r.Post("/{id}/cancel", deps.PickPack.CancelSession)
			})

			// Shipping rate comparison — any authenticated user
			r.Post("/shipping/rates", deps.Rate.GetRates)

			// Repricing engine — requires integrations.manage
			r.Route("/repricing", func(r chi.Router) {
				r.Use(requireFeature("repricing"))
				r.Use(requirePermission(model.PermIntegrationsManage))
				r.Get("/rules", deps.Repricing.ListRules)
				r.Post("/rules", deps.Repricing.CreateRule)
				r.Get("/rules/{id}", deps.Repricing.GetRule)
				r.Put("/rules/{id}", deps.Repricing.UpdateRule)
				r.Delete("/rules/{id}", deps.Repricing.DeleteRule)
				r.Post("/rules/{id}/simulate", deps.Repricing.SimulateRule)
				r.Post("/apply", deps.Repricing.ApplyRules)
				r.Get("/log", deps.Repricing.ListLog)
				r.Get("/summary", deps.Repricing.GetSummary)
			})

			// Stock sync — requires integrations.manage
			if deps.StockSync != nil {
				r.Route("/stock-sync", func(r chi.Router) {
					r.Use(requireFeature("stock_sync"))
					r.Use(requirePermission(model.PermIntegrationsManage))
					r.Get("/channels", deps.StockSync.ListChannels)
					r.Post("/channels", deps.StockSync.CreateChannel)
					r.Get("/channels/{id}", deps.StockSync.GetChannel)
					r.Put("/channels/{id}", deps.StockSync.UpdateChannel)
					r.Delete("/channels/{id}", deps.StockSync.DeleteChannel)
					r.Post("/push", deps.StockSync.PushAll)
					r.Post("/push/channel/{channel_id}", deps.StockSync.PushChannel)
					r.Post("/push/{product_id}", deps.StockSync.PushProduct)
					r.Post("/push/listing/{listing_id}", deps.StockSync.PushListing)
					r.Post("/reconcile/{product_id}", deps.StockSync.ReconcileProduct)
					r.Get("/events", deps.StockSync.ListEvents)
					r.Get("/dashboard", deps.StockSync.GetDashboard)
					r.Get("/allocations/{product_id}", deps.StockSync.GetAllocations)
				})
			}

			// Listing sync — requires integrations.manage
			if deps.ListingSync != nil {
				r.Route("/listing-sync/configs", func(r chi.Router) {
					r.Use(requireFeature("listing_sync"))
					r.Use(requirePermission(model.PermIntegrationsManage))
					r.Get("/", deps.ListingSync.ListConfigs)
					r.Post("/", deps.ListingSync.CreateConfig)
					r.Get("/{id}", deps.ListingSync.GetConfig)
					r.Put("/{id}", deps.ListingSync.UpdateConfig)
					r.Delete("/{id}", deps.ListingSync.DeleteConfig)
					r.Post("/{id}/sync", deps.ListingSync.TriggerSync)
					r.Post("/{id}/sync-prices", deps.ListingSync.TriggerSyncPrices)
					r.Post("/{id}/sync-stock", deps.ListingSync.TriggerSyncStock)
					r.Get("/{id}/logs", deps.ListingSync.ListLogs)
				})
			}

			// Payment reconciliation — requires reports.view
			r.Route("/reconciliation", func(r chi.Router) {
				r.Use(requireFeature("reconciliation"))
				r.Use(requirePermission(model.PermReportsView))
				r.Post("/settlements", deps.Reconciliation.CreateSettlement)
				r.Get("/settlements", deps.Reconciliation.ListSettlements)
				r.Get("/settlements/{id}", deps.Reconciliation.GetSettlement)
				r.Post("/settlements/{id}/auto-match", deps.Reconciliation.AutoMatch)
				r.Get("/transactions", deps.Reconciliation.ListTransactions)
				r.Post("/transactions/{id}/match", deps.Reconciliation.ManualMatch)
				r.Post("/transactions/{id}/unmatch", deps.Reconciliation.Unmatch)
				r.Get("/summary", deps.Reconciliation.GetSummary)
			})

			// Fulfillment read API + operator actions (OPE-419) — tenant-scoped,
			// RLS-enforced. Reads guarded by orders.view, actions by orders.edit.
			if deps.Fulfillment != nil {
				r.Route("/fulfillment", func(r chi.Router) {
					r.With(requirePermission(model.PermOrdersView)).Get("/processes", deps.Fulfillment.ListProcesses)
					r.With(requirePermission(model.PermOrdersView)).Get("/processes/{id}", deps.Fulfillment.GetProcess)
					r.With(requirePermission(model.PermOrdersView)).Get("/orders/{orderID}", deps.Fulfillment.GetOrderFulfillment)
					r.With(requirePermission(model.PermOrdersEdit)).Post("/blockers/{id}/resolve", deps.Fulfillment.ResolveBlocker)
					r.With(requirePermission(model.PermOrdersEdit)).Post("/steps/{id}/retry", deps.Fulfillment.RetryStep)
				})
			}

			// Operations control tower (OPE-419) — tenant-scoped read aggregates.
			if deps.Operations != nil {
				r.Route("/operations", func(r chi.Router) {
					r.With(requirePermission(model.PermOrdersView)).Get("/summary", deps.Operations.Summary)
					r.With(requirePermission(model.PermOrdersView)).Get("/exceptions", deps.Operations.Exceptions)
					r.With(requirePermission(model.PermOrdersView)).Get("/integration-capability-summary", deps.Operations.IntegrationCapabilitySummary)
				})
			}

		})

		// CSV import for reconciliation — outside MaxBodySize group (uses its own 10MB limit)
		r.With(requireFeature("reconciliation")).With(requirePermission(model.PermReportsView)).Post("/reconciliation/import-csv", deps.Reconciliation.ImportCSV)
	})

	return r
}

// extractCookieDomain derives a cookie Domain attribute from the frontend URL.
// "https://app.openoms.org" → ".openoms.org" (cross-subdomain access).
// "http://localhost:3000" → "" (no domain restriction for local dev).
//
// NOTE: This uses a simple heuristic (last 2 hostname parts) which works for
// single-part TLDs (.org, .com, .pl) but NOT for multi-part ccTLDs like .co.uk
// or .com.pl. This is intentional — openoms.org uses a single-part TLD and we
// avoid pulling in golang.org/x/net/publicsuffix for this one call site.
func extractCookieDomain(frontendURL string) string {
	u, err := url.Parse(frontendURL)
	if err != nil {
		return ""
	}
	host := u.Hostname()
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return ""
	}
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return "" // single-label hostname
	}
	// "openoms.org" → ".openoms.org"
	// "app.openoms.org" → ".openoms.org"
	return "." + strings.Join(parts[len(parts)-2:], ".")
}
