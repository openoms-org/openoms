package allegro

import (
	"context"
	"testing"
	"time"
)

func TestRateLimiterImmediateTokens(t *testing.T) {
	rl := newRateLimiter(100)
	defer rl.Close()

	// Should be able to get a token immediately (bucket starts full).
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if err := rl.Wait(ctx); err != nil {
		t.Errorf("Wait() returned error: %v", err)
	}
}

func TestRateLimiterBlocksWhenEmpty(t *testing.T) {
	// Create a limiter with 1 token.
	rl := newRateLimiter(1)
	defer rl.Close()

	// Consume the single token.
	ctx := context.Background()
	if err := rl.Wait(ctx); err != nil {
		t.Fatalf("first Wait() error: %v", err)
	}

	// Next Wait should block; use a short timeout to verify.
	ctx2, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := rl.Wait(ctx2)
	if err == nil {
		t.Error("expected timeout error when bucket is empty, got nil")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
}

func TestRateLimiterContextCancellation(t *testing.T) {
	rl := newRateLimiter(1)
	defer rl.Close()

	// Drain the token.
	if err := rl.Wait(context.Background()); err != nil {
		t.Fatalf("first Wait() error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	err := rl.Wait(ctx)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestRateLimiterRefill(t *testing.T) {
	// 60000 requests/min = 1 token per millisecond.
	rl := newRateLimiter(60000)
	defer rl.Close()

	// Drain all tokens.
	for range 60000 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		rl.Wait(ctx)
		cancel()
	}

	// Wait for a refill (give it some time).
	time.Sleep(10 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if err := rl.Wait(ctx); err != nil {
		t.Errorf("Wait() after refill returned error: %v", err)
	}
}

func TestRateLimiterCloseIdempotent(t *testing.T) {
	rl := newRateLimiter(10)
	rl.Close()
	rl.Close() // Should not panic.
}
