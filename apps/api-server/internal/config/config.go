package config

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Port        string `env:"PORT" envDefault:"8080"`
	Env         string `env:"ENV" envDefault:"development"`
	BaseURL     string `env:"BASE_URL" envDefault:"http://localhost:8080"`
	FrontendURL string `env:"FRONTEND_URL" envDefault:"http://localhost:3000"`

	DatabaseURL       string `env:"DATABASE_URL,required"`
	WorkerDatabaseURL string `env:"WORKER_DATABASE_URL"` // superuser pool for cross-tenant worker queries; falls back to DATABASE_URL
	RedisURL          string `env:"REDIS_URL" envDefault:"redis://localhost:6379"`

	JWTSecret     string `env:"JWT_SECRET,required"`
	EncryptionKey string `env:"ENCRYPTION_KEY,required"`

	AllegroWebhookSecret string `env:"ALLEGRO_WEBHOOK_SECRET"`

	InPostAPIToken      string `env:"INPOST_API_TOKEN"`
	InPostOrgID         string `env:"INPOST_ORGANIZATION_ID"`
	InPostWebhookSecret string `env:"INPOST_WEBHOOK_SECRET"`

	WorkersEnabled bool `env:"WORKERS_ENABLED" envDefault:"true"`

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

	// RegistrationMode controls public registration: "open" (default), "invite" (token required), "disabled".
	RegistrationMode string `env:"REGISTRATION_MODE" envDefault:"open"`

	// LicensePublicKey is the base64-encoded Ed25519 public key for verifying license tokens.
	// Empty = license token feature disabled (self-hosted mode).
	LicensePublicKey string `env:"LICENSE_PUBLIC_KEY" envDefault:""`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return cfg, nil
}

func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
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
	case "open", "invite", "closed":
		// valid
	default:
		return fmt.Errorf("REGISTRATION_MODE must be one of: open, invite, closed (got %q)", c.RegistrationMode)
	}

	// Warn if registration is open in non-development environments.
	if c.RegistrationMode == "open" && !c.IsDevelopment() {
		slog.Warn("REGISTRATION_MODE is 'open' in non-development environment — consider using 'invite' or 'closed'", "env", c.Env)
	}

	return nil
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
