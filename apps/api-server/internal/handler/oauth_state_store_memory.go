package handler

import (
	"context"
	"sync"
	"time"

	"github.com/openoms-org/openoms/apps/api-server/internal/asyncutil"
)

const memoryOAuthStateSweepInterval = 5 * time.Minute

// MemoryOAuthStateStore is an in-memory implementation of OAuthStateStore.
// Suitable for single-pod deployments and local development.
type MemoryOAuthStateStore struct {
	mu            sync.Mutex
	entries       map[string]*OAuthState
	done          chan struct{}
	closeOnce     sync.Once
	sweepInterval time.Duration
}

// NewMemoryOAuthStateStore creates a new in-memory OAuth state store.
func NewMemoryOAuthStateStore() *MemoryOAuthStateStore {
	return newMemoryOAuthStateStore(memoryOAuthStateSweepInterval)
}

func newMemoryOAuthStateStore(sweepInterval time.Duration) *MemoryOAuthStateStore {
	if sweepInterval <= 0 {
		sweepInterval = memoryOAuthStateSweepInterval
	}
	s := &MemoryOAuthStateStore{
		entries:       make(map[string]*OAuthState),
		done:          make(chan struct{}),
		sweepInterval: sweepInterval,
	}
	asyncutil.SafeGo(s.cleanup)
	return s
}

// Close stops the background cleanup goroutine.
func (m *MemoryOAuthStateStore) Close() {
	m.closeOnce.Do(func() {
		close(m.done)
	})
}

func (m *MemoryOAuthStateStore) deleteExpiredLocked(now time.Time) {
	for k, s := range m.entries {
		if now.After(s.ExpiresAt) {
			delete(m.entries, k)
		}
	}
}

func (m *MemoryOAuthStateStore) deleteExpired(now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.deleteExpiredLocked(now)
}

func (m *MemoryOAuthStateStore) cleanup() {
	ticker := time.NewTicker(m.sweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.deleteExpired(time.Now())
		case <-m.done:
			return
		}
	}
}

// Save stores the OAuth state data in memory with expiration based on data.ExpiresAt.
func (m *MemoryOAuthStateStore) Save(_ context.Context, state string, data *OAuthState, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clean expired entries on each save to prevent memory growth.
	m.deleteExpiredLocked(time.Now())

	m.entries[state] = data
	return nil
}

// Load retrieves and deletes the OAuth state data for the given state key.
func (m *MemoryOAuthStateStore) Load(_ context.Context, state string) (*OAuthState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	m.deleteExpiredLocked(now)

	data, ok := m.entries[state]
	if !ok || now.After(data.ExpiresAt) {
		delete(m.entries, state)
		return nil, nil
	}

	// Consume once — prevents replay attacks.
	delete(m.entries, state)
	return data, nil
}
