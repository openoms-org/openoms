package handler

import (
	"context"
	"sync"
	"time"
)

// MemoryOAuthStateStore is an in-memory implementation of OAuthStateStore.
// Suitable for single-pod deployments and local development.
type MemoryOAuthStateStore struct {
	mu      sync.Mutex
	entries map[string]*allegroOAuthState
}

// NewMemoryOAuthStateStore creates a new in-memory OAuth state store.
func NewMemoryOAuthStateStore() *MemoryOAuthStateStore {
	return &MemoryOAuthStateStore{entries: make(map[string]*allegroOAuthState)}
}

func (m *MemoryOAuthStateStore) Save(_ context.Context, state string, data *allegroOAuthState, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clean expired entries on each save to prevent memory growth.
	now := time.Now()
	for k, s := range m.entries {
		if now.After(s.ExpiresAt) {
			delete(m.entries, k)
		}
	}

	m.entries[state] = data
	return nil
}

func (m *MemoryOAuthStateStore) Load(_ context.Context, state string) (*allegroOAuthState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, ok := m.entries[state]
	if !ok || time.Now().After(data.ExpiresAt) {
		delete(m.entries, state)
		return nil, nil
	}

	// Consume once — prevents replay attacks.
	delete(m.entries, state)
	return data, nil
}
