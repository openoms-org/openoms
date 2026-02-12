package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Port        string `env:"PORT" envDefault:"8080"`
	Env         string `env:"ENV" envDefault:"development"`
	BaseURL     string `env:"BASE_URL" envDefault:"http://localhost:8080"`
	FrontendURL string `env:"FRONTEND_URL" envDefault:"http://localhost:3000"`

	DatabaseURL string `env:"DATABASE_URL,required"`
	RedisURL    string `env:"REDIS_URL" envDefault:"redis://localhost:6379"`

	JWTSecret     string `env:"JWT_SECRET,required"`
	EncryptionKey string `env:"ENCRYPTION_KEY,required"`

	AllegroWebhookSecret string `env:"ALLEGRO_WEBHOOK_SECRET"`

	InPostAPIToken      string `env:"INPOST_API_TOKEN"`
	InPostOrgID         string `env:"INPOST_ORGANIZATION_ID"`
	InPostWebhookSecret string `env:"INPOST_WEBHOOK_SECRET"`

	FeatureAllegro bool `env:"FEATURE_ALLEGRO" envDefault:"true"`
	FeatureInPost  bool `env:"FEATURE_INPOST" envDefault:"true"`

	WorkersEnabled bool   `env:"WORKERS_ENABLED" envDefault:"true"`

	UploadDir     string `env:"UPLOAD_DIR" envDefault:"./uploads"`
	MaxUploadSize int64  `env:"MAX_UPLOAD_SIZE" envDefault:"10485760"` // 10MB

	S3Enabled   bool   `env:"S3_ENABLED" envDefault:"false"`
	S3Bucket    string `env:"S3_BUCKET"`
	S3Region    string `env:"S3_REGION" envDefault:"eu-central-1"`
	S3Endpoint  string `env:"S3_ENDPOINT"`  // for MinIO/DO Spaces
	S3AccessKey string `env:"S3_ACCESS_KEY"`
	S3SecretKey string `env:"S3_SECRET_KEY"`
	S3PublicURL string `env:"S3_PUBLIC_URL"` // CDN URL prefix

	OpenAIAPIKey string `env:"OPENAI_API_KEY"`
	OpenAIModel  string `env:"OPENAI_MODEL" envDefault:"gpt-4o-mini"`
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
