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

type MemoryWSTicketStore struct {
	mu      sync.Mutex
	tickets map[string]*memTicket
}

func NewMemoryWSTicketStore() *MemoryWSTicketStore {
	return &MemoryWSTicketStore{tickets: make(map[string]*memTicket)}
}

func (m *MemoryWSTicketStore) Store(_ context.Context, key string, data WSTicketData, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tickets[key] = &memTicket{data: data, expiry: time.Now().Add(ttl)}
	return nil
}

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
