package main

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/openoms-org/openoms/apps/api-server/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectRedisInvalidURLRequiresRedis(t *testing.T) {
	cfg := &config.Config{
		Env:      "production",
		RedisURL: "not-a-redis-url",
	}

	client, err := connectRedis(context.Background(), cfg)

	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "REDIS_URL is invalid")
}

func TestConnectRedisInvalidURLFallsBackWhenAllowed(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
	}{
		{
			name: "development",
			cfg:  config.Config{Env: "development", RedisURL: "not-a-redis-url"},
		},
		{
			name: "explicit production opt-in",
			cfg:  config.Config{Env: "production", RedisURL: "not-a-redis-url", AllowInMemoryState: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := connectRedis(context.Background(), &tt.cfg)

			require.NoError(t, err)
			assert.Nil(t, client)
		})
	}
}

func TestConnectRedisUnavailableRequiresRedis(t *testing.T) {
	cfg := &config.Config{
		Env:      "production",
		RedisURL: "redis://" + unusedLocalAddress(t),
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	client, err := connectRedis(ctx, cfg)

	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "redis is required but unavailable")
}

func TestConnectRedisUnavailableFallsBackInDevelopment(t *testing.T) {
	cfg := &config.Config{
		Env:      "development",
		RedisURL: "redis://" + unusedLocalAddress(t),
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	client, err := connectRedis(ctx, cfg)

	require.NoError(t, err)
	assert.Nil(t, client)
}

func unusedLocalAddress(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())
	return addr
}
