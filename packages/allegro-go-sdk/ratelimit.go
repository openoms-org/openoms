package allegro

import (
	"context"
	"time"
)

type rateLimiter struct {
	tokens chan struct{}
	done   chan struct{}
}

func newRateLimiter(requestsPerMinute int) *rateLimiter {
	rl := &rateLimiter{
		tokens: make(chan struct{}, requestsPerMinute),
		done:   make(chan struct{}),
	}

	// Fill the bucket initially.
	for range requestsPerMinute {
		rl.tokens <- struct{}{}
	}

	interval := time.Minute / time.Duration(requestsPerMinute)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-rl.done:
				return
			case <-ticker.C:
				select {
				case rl.tokens <- struct{}{}:
				default:
					// Bucket is full, discard.
				}
			}
		}
	}()

	return rl
}

// Wait blocks until a token is available or the context is cancelled.
func (r *rateLimiter) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-r.tokens:
		return nil
	}
}

// Close stops the background goroutine that refills tokens.
func (r *rateLimiter) Close() {
	select {
	case <-r.done:
		// Already closed.
	default:
		close(r.done)
	}
}
