// Package asyncutil provides helpers for safe asynchronous execution.
package asyncutil

import (
	"log/slog"
	"runtime/debug"
)

// SafeGo runs fn in a new goroutine with panic recovery.
// If fn panics, the panic is logged and the goroutine exits cleanly.
func SafeGo(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("goroutine panicked", "error", r, "stack", string(debug.Stack()))
			}
		}()
		fn()
	}()
}
