package util

import (
	"sync"
	"testing"
	"time"
)

func TestSafeGo_RunsFunction(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	executed := false
	SafeGo(func() {
		defer wg.Done()
		executed = true
	})

	wg.Wait()
	if !executed {
		t.Fatal("expected SafeGo to execute the function")
	}
}

func TestSafeGo_RecoversPanic(t *testing.T) {
	done := make(chan struct{})

	SafeGo(func() {
		defer close(done)
		panic("test panic")
	})

	select {
	case <-done:
		// goroutine completed without crashing the process
	case <-time.After(2 * time.Second):
		t.Fatal("SafeGo did not recover from panic in time")
	}
}

func TestSafeGo_RunsAsynchronously(t *testing.T) {
	ch := make(chan struct{})

	SafeGo(func() {
		<-ch // block until signaled
	})

	// If SafeGo blocked, we would never reach this line.
	// Signal the goroutine to finish.
	close(ch)
}
