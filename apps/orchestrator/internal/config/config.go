package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Port string `env:"PORT" envDefault:"9090"`
	Env  string `env:"ENV" envDefault:"development"`

	RedisURL string `env:"REDIS_URL" envDefault:"redis://localhost:6379"`

	LinearAPIKey        string `env:"LINEAR_API_KEY,notEmpty"`
	LinearWebhookSecret string `env:"LINEAR_WEBHOOK_SECRET,notEmpty"`
	LinearTeamID        string `env:"LINEAR_TEAM_ID" envDefault:"OPE"`

	OpenRouterAPIKey string `env:"OPENROUTER_API_KEY"`
	GitHubToken      string `env:"GITHUB_TOKEN"`
	GitHubRepo       string `env:"GITHUB_REPO" envDefault:"openoms-org/openoms"`

	TelegramBotToken string `env:"TELEGRAM_BOT_TOKEN"`
	TelegramChatID   string `env:"TELEGRAM_CHAT_ID"`

	MaxCIRetries        int     `env:"MAX_CI_RETRIES" envDefault:"5"`
	PollIntervalSeconds int     `env:"POLL_INTERVAL_SECONDS" envDefault:"300"`
	BudgetMonthlyLimit  float64 `env:"BUDGET_MONTHLY_LIMIT" envDefault:"280"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	if c.MaxCIRetries < 1 || c.MaxCIRetries > 10 {
		return fmt.Errorf("MAX_CI_RETRIES must be between 1 and 10, got %d", c.MaxCIRetries)
	}
	if c.PollIntervalSeconds < 30 {
		return fmt.Errorf("POLL_INTERVAL_SECONDS must be >= 30, got %d", c.PollIntervalSeconds)
	}
	if c.BudgetMonthlyLimit <= 0 {
		return fmt.Errorf("BUDGET_MONTHLY_LIMIT must be positive, got %f", c.BudgetMonthlyLimit)
	}
	return nil
}
