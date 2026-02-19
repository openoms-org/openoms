package service

import (
	"context"
	"sync"
	"time"
)

type memTicket struct {
	data   WSTicketData
	expiry time.Time
}

// MemoryWSTicketStore implements WSTicketStore using an in-memory map.
// Suitable for single-pod deployments and local development.
type MemoryWSTicketStore struct {
	mu      sync.Mutex
	tickets map[string]*memTicket
}

// NewMemoryWSTicketStore creates a new in-memory WebSocket ticket store.
func NewMemoryWSTicketStore() *MemoryWSTicketStore {
	return &MemoryWSTicketStore{tickets: make(map[string]*memTicket)}
}

// Store saves a ticket with associated data and TTL in memory.
func (m *MemoryWSTicketStore) Store(_ context.Context, key string, data WSTicketData, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tickets[key] = &memTicket{data: data, expiry: time.Now().Add(ttl)}
	return nil
}

// Consume retrieves and deletes a ticket atomically. Returns nil if not found or expired.
func (m *MemoryWSTicketStore) Consume(_ context.Context, key string) (*WSTicketData, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tickets[key]
	if !ok || time.Now().After(t.expiry) {
		delete(m.tickets, key)
		return nil, nil
	}
	delete(m.tickets, key)
	return &t.data, nil
}
