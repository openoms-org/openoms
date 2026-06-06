// Package config loads and holds all runtime configuration for the API server.
package config

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/netip"
	"strings"

	"github.com/caarlos0/env/v11"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	Port        string `env:"PORT" envDefault:"8080"`
	Env         string `env:"ENV" envDefault:"development"`
	BaseURL     string `env:"BASE_URL" envDefault:"http://localhost:8080"`
	FrontendURL string `env:"FRONTEND_URL" envDefault:"http://localhost:3000"`

	// APISurfaceMode gates non-ready features over the API. "client-ready" (default)
	// exposes only "ready" features; "full" exposes all except "blocked". Keep this in
	// sync with the dashboard's NEXT_PUBLIC_OPENOMS_DASHBOARD_SURFACE.
	APISurfaceMode string `env:"OPENOMS_API_SURFACE" envDefault:"client-ready"`

	// TrustedProxyCIDRs is a comma-separated list of immediate proxy CIDRs
	// whose X-Forwarded-For / X-Real-IP headers may update r.RemoteAddr.
	TrustedProxyCIDRs string `env:"TRUSTED_PROXY_CIDRS" envDefault:""`

	DatabaseURL        string `env:"DATABASE_URL,required"`
	WorkerDatabaseURL  string `env:"WORKER_DATABASE_URL"` // privileged pool for cross-tenant worker and webhook queries; required outside development
	RedisURL           string `env:"REDIS_URL" envDefault:"redis://localhost:6379"`
	AllowInMemoryState bool   `env:"ALLOW_IN_MEMORY_STATE" envDefault:"false"`

	JWTSecret     string `env:"JWT_SECRET,required"`
	EncryptionKey string `env:"ENCRYPTION_KEY,required"`

	AllegroWebhookSecret string `env:"ALLEGRO_WEBHOOK_SECRET"`

	InPostAPIToken      string `env:"INPOST_API_TOKEN"`
	InPostOrgID         string `env:"INPOST_ORGANIZATION_ID"`
	InPostWebhookSecret string `env:"INPOST_WEBHOOK_SECRET"`

	WorkersEnabled bool `env:"WORKERS_ENABLED" envDefault:"true"`

	// SeedProviders, when true, idempotently seeds the provider registry with the
	// class-first catalog of existing providers at startup (OPE-412).
	SeedProviders bool `env:"SEED_PROVIDERS" envDefault:"false"`

	// OrchestrationWorkerEnabled enables the fulfillment outbox worker (OPE-415).
	// Off by default until side-effect handlers are registered (OPE-416+).
	OrchestrationWorkerEnabled bool `env:"ORCHESTRATION_WORKER_ENABLED" envDefault:"false"`

	// FulfillmentProcessEnabled routes order creation through the fulfillment
	// commands (OPE-416): on each new order it creates a fulfillment process and
	// enqueues an order.created orchestration event in the order's transaction.
	// Off by default — zero behavior change until enabled.
	FulfillmentProcessEnabled bool `env:"FULFILLMENT_PROCESS_ENABLED" envDefault:"false"`

	UploadDir     string `env:"UPLOAD_DIR" envDefault:"./uploads"`
	MaxUploadSize int64  `env:"MAX_UPLOAD_SIZE" envDefault:"10485760"` // 10MB

	S3Enabled   bool   `env:"S3_ENABLED" envDefault:"false"`
	S3Bucket    string `env:"S3_BUCKET"`
	S3Region    string `env:"S3_REGION" envDefault:"eu-central-1"`
	S3Endpoint  string `env:"S3_ENDPOINT"` // for MinIO/DO Spaces
	S3AccessKey string `env:"S3_ACCESS_KEY"`
	S3SecretKey string `env:"S3_SECRET_KEY"`
	S3PublicURL string `env:"S3_PUBLIC_URL"` // CDN URL prefix

	OpenAIAPIKey string `env:"OPENAI_API_KEY"`
	OpenAIModel  string `env:"OPENAI_MODEL" envDefault:"gpt-4o-mini"`

	RemoveBGAPIKey string `env:"REMOVEBG_API_KEY"`

	MetricsToken string `env:"METRICS_TOKEN"` // Bearer token for /metrics; if empty, metrics are disabled in production

	// RegistrationMode controls public registration: "open" (default), "invite" (token required), "closed".
	// "disabled" is accepted as a legacy alias for closed registration.
	RegistrationMode string `env:"REGISTRATION_MODE" envDefault:"open"`

	// LicensePublicKey is the base64-encoded Ed25519 public key for verifying license tokens.
	// Empty = license token feature disabled (self-hosted mode).
	LicensePublicKey string `env:"LICENSE_PUBLIC_KEY" envDefault:""`

	// Stripe billing configuration. All empty = billing disabled (self-hosted mode).
	StripeSecretKey     string `env:"STRIPE_SECRET_KEY" envDefault:""`
	StripeWebhookSecret string `env:"STRIPE_WEBHOOK_SECRET" envDefault:""`
	StripePublicKey     string `env:"STRIPE_PUBLIC_KEY" envDefault:""`

	// BillingPlansJSON is a JSON array of plan configs. Parsed at startup via ParseBillingPlans().
	// Empty = billing disabled. Plan names, prices, limits are all defined here, never hardcoded.
	BillingPlansJSON string `env:"BILLING_PLANS" envDefault:""`

	// Sentry error tracking. Empty DSN = Sentry disabled (self-hosted mode).
	SentryDSN              string  `env:"SENTRY_DSN" envDefault:""`
	SentryEnvironment      string  `env:"SENTRY_ENVIRONMENT" envDefault:""`
	SentryRelease          string  `env:"SENTRY_RELEASE" envDefault:""`
	SentryTracesSampleRate float64 `env:"SENTRY_TRACES_SAMPLE_RATE" envDefault:"0"`
}

// Load parses environment variables into a Config struct.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return cfg, nil
}

// IsDevelopment reports whether the server is running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

// IsProduction reports whether the server is running in production mode.
func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

// AllowsInMemoryState reports whether auth/session/rate-limit state may use
// process-local memory when Redis is not available.
func (c *Config) AllowsInMemoryState() bool {
	return c.IsDevelopment() || c.AllowInMemoryState
}

// RequiresRedis reports whether Redis is required for shared runtime state.
func (c *Config) RequiresRedis() bool {
	return !c.AllowsInMemoryState()
}

// TrustedProxyPrefixes parses TRUSTED_PROXY_CIDRS into normalized prefixes.
func (c *Config) TrustedProxyPrefixes() ([]netip.Prefix, error) {
	raw := strings.TrimSpace(c.TrustedProxyCIDRs)
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	prefixes := make([]netip.Prefix, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			return nil, fmt.Errorf("TRUSTED_PROXY_CIDRS contains invalid CIDR %q: %w", value, err)
		}
		prefixes = append(prefixes, prefix.Masked())
	}
	return prefixes, nil
}

// RequiresWorkerDatabase reports whether cross-tenant worker/webhook queries
// must use an explicitly configured privileged database URL.
func (c *Config) RequiresWorkerDatabase() bool {
	return !c.IsDevelopment()
}

// WorkerDatabaseDSN returns the database URL for intentionally cross-tenant
// operations. Development keeps the historical local fallback; non-development
// environments must configure a separate WORKER_DATABASE_URL explicitly.
func (c *Config) WorkerDatabaseDSN() (string, error) {
	if c.WorkerDatabaseURL != "" {
		return c.WorkerDatabaseURL, nil
	}
	if !c.RequiresWorkerDatabase() {
		return c.DatabaseURL, nil
	}
	return "", fmt.Errorf("WORKER_DATABASE_URL is required outside development for cross-tenant worker and webhook queries")
}

// Validate checks critical config values and returns an error for fatal
// misconfigurations. Non-fatal issues are logged as warnings.
func (c *Config) Validate() error {
	// EncryptionKey must be exactly 64 hex chars (32 bytes).
	if len(c.EncryptionKey) != 64 {
		return fmt.Errorf("ENCRYPTION_KEY must be exactly 64 hex characters (got %d)", len(c.EncryptionKey))
	}
	if _, err := hex.DecodeString(c.EncryptionKey); err != nil {
		return fmt.Errorf("ENCRYPTION_KEY is not valid hex: %w", err)
	}

	// JWTSecret must be at least 32 characters.
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters long (got %d)", len(c.JWTSecret))
	}

	// RegistrationMode must be one of the allowed values.
	switch c.RegistrationMode {
	case "open", "invite", "closed", "disabled":
		// valid
	default:
		return fmt.Errorf("REGISTRATION_MODE must be one of: open, invite, closed, disabled (got %q)", c.RegistrationMode)
	}

	// APISurfaceMode must be one of the allowed values. Unrecognised values fall
	// through to client-ready behaviour in readiness.isVisible (fail-secure), which
	// silently disables the "full" escape hatch — reject them explicitly instead.
	switch c.APISurfaceMode {
	case "client-ready", "full":
		// valid
	default:
		return fmt.Errorf("OPENOMS_API_SURFACE must be one of: client-ready, full (got %q)", c.APISurfaceMode)
	}

	// Warn if registration is open in non-development environments.
	if c.RegistrationMode == "open" && !c.IsDevelopment() {
		slog.Warn("REGISTRATION_MODE is 'open' in non-development environment — consider using 'invite' or 'closed'", "env", c.Env)
	}

	if c.AllowInMemoryState && !c.IsDevelopment() {
		slog.Warn("ALLOW_IN_MEMORY_STATE is enabled outside development; auth, session, rate-limit, OAuth, WebSocket, and worker lock state will be process-local", "env", c.Env)
	}

	if _, err := c.TrustedProxyPrefixes(); err != nil {
		return err
	}

	if _, err := c.WorkerDatabaseDSN(); err != nil {
		return err
	}
	if c.RequiresWorkerDatabase() && c.WorkerDatabaseURL == c.DatabaseURL {
		return fmt.Errorf("WORKER_DATABASE_URL must not equal DATABASE_URL outside development; use a separate privileged role only for cross-tenant worker and webhook queries")
	}

	return nil
}

// BillingEnabled returns true when Stripe billing is configured.
func (c *Config) BillingEnabled() bool {
	return c.StripeSecretKey != "" && c.BillingPlansJSON != ""
}

// SentryEnabled returns true when Sentry error tracking is configured.
func (c *Config) SentryEnabled() bool {
	return c.SentryDSN != ""
}

// SentryEnv returns the Sentry environment, defaulting to ENV value.
func (c *Config) SentryEnv() string {
	if c.SentryEnvironment != "" {
		return c.SentryEnvironment
	}
	return c.Env
}

// PlanConfig defines a billing plan loaded from BILLING_PLANS env var.
// Stripe Price IDs are kept server-side and never exposed to frontend.
type PlanConfig struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	MonthlyPriceID string     `json:"monthly_price_id"`
	YearlyPriceID  string     `json:"yearly_price_id"`
	MonthlyAmount  int64      `json:"monthly_amount"` // in smallest currency unit (grosze)
	YearlyAmount   int64      `json:"yearly_amount"`
	Currency       string     `json:"currency"`
	TrialDays      int64      `json:"trial_days"`
	Limits         PlanLimits `json:"limits"`
	Features       []string   `json:"features"`
}

// PlanLimits defines resource limits for a plan.
type PlanLimits struct {
	MaxUsers         int `json:"max_users"`
	MaxOrdersMonthly int `json:"max_orders_monthly"`
	MaxIntegrations  int `json:"max_integrations"`
}

// ParseBillingPlans parses the BILLING_PLANS JSON env var.
// Returns nil if empty (billing disabled).
func (c *Config) ParseBillingPlans() ([]PlanConfig, error) {
	if c.BillingPlansJSON == "" {
		return nil, nil
	}
	var plans []PlanConfig
	if err := json.Unmarshal([]byte(c.BillingPlansJSON), &plans); err != nil {
		return nil, fmt.Errorf("BILLING_PLANS: invalid JSON: %w", err)
	}
	for i, p := range plans {
		if p.ID == "" || p.Name == "" {
			return nil, fmt.Errorf("BILLING_PLANS[%d]: id and name are required", i)
		}
		if p.Currency == "" {
			plans[i].Currency = "pln"
		}
	}
	return plans, nil
}

// ParseLicensePublicKey decodes the base64-encoded Ed25519 public key.
// Returns nil if the key is empty (feature disabled).
func (c *Config) ParseLicensePublicKey() (ed25519.PublicKey, error) {
	if c.LicensePublicKey == "" {
		return nil, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(c.LicensePublicKey)
	if err != nil {
		return nil, fmt.Errorf("LICENSE_PUBLIC_KEY: invalid base64: %w", err)
	}
	if len(decoded) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("LICENSE_PUBLIC_KEY: expected %d bytes, got %d", ed25519.PublicKeySize, len(decoded))
	}
	return ed25519.PublicKey(decoded), nil
}
