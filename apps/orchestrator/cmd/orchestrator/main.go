package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"

	"github.com/openoms-org/openoms/apps/orchestrator/internal/config"
	"github.com/openoms-org/openoms/apps/orchestrator/internal/handler"
	"github.com/openoms-org/openoms/apps/orchestrator/internal/linear"
	"github.com/openoms-org/openoms/apps/orchestrator/internal/scheduler"
	"github.com/openoms-org/openoms/apps/orchestrator/internal/store"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}

	// Logger
	logLevel := slog.LevelInfo
	if cfg.Env == "development" {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})))

	slog.Info("starting NemoClaw orchestrator", "port", cfg.Port, "env", cfg.Env)

	// Redis
	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return fmt.Errorf("parse redis url: %w", err)
	}
	rdb := redis.NewClient(redisOpts)
	defer rdb.Close()

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return fmt.Errorf("connect to redis: %w", err)
	}
	slog.Info("connected to Redis")

	taskStore := store.New(rdb)

	// Linear clients
	linearClient := linear.NewClient(cfg.LinearAPIKey, cfg.LinearTeamID)
	webhookHandler := linear.NewWebhookHandler(cfg.LinearWebhookSecret, taskStore)

	// Scheduler
	sched := scheduler.New(
		taskStore,
		linearClient,
		scheduler.WithPollInterval(time.Duration(cfg.PollIntervalSeconds)*time.Second),
	)

	// Router
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)

	healthHandler := handler.NewHealthHandler(rdb, taskStore)
	r.Get("/health", healthHandler.Health)
	r.Post("/api/webhooks/linear", webhookHandler.HandleWebhook)

	// Server with graceful shutdown
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start scheduler in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go sched.Run(ctx)

	errCh := make(chan error, 1)
	go func() {
		slog.Info("HTTP server listening", "addr", srv.Addr)
		errCh <- srv.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		slog.Info("received shutdown signal", "signal", sig)
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
	}

	// Graceful shutdown
	cancel() // Stop scheduler
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}
	slog.Info("server stopped gracefully")
	return nil
}
