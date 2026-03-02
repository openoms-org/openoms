// Package asyncutil provides helpers for safe asynchronous execution.
package asyncutil

import (
	"log/slog"
	"runtime/debug"

	"github.com/getsentry/sentry-go"
)

// SafeGo runs fn in a new goroutine with panic recovery.
// If fn panics, the panic is logged, reported to Sentry, and the goroutine exits cleanly.
func SafeGo(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("goroutine panicked", "error", r, "stack", string(debug.Stack()))
				sentry.CurrentHub().Recover(r)
			}
		}()
		fn()
	}()
}
