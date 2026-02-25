package service

import (
	"context"
	"sync"
	"time"

	"github.com/openoms-org/openoms/apps/api-server/internal/asyncutil"
)

type memRefreshFamily struct {
	family *RefreshTokenFamily
	expiry time.Time
}

type memRefreshToken struct {
	entry  *RefreshTokenEntry
	expiry time.Time
}

// MemoryRefreshTokenStore implements RefreshTokenStore using in-memory maps.
// Suitable for single-pod deployments and local development.
type MemoryRefreshTokenStore struct {
	mu       sync.Mutex
	families map[string]*memRefreshFamily
	tokens   map[string]*memRefreshToken
	done     chan struct{}
}

// NewMemoryRefreshTokenStore creates a new in-memory refresh token store
// with a background goroutine that cleans up expired entries every 5 minutes.
func NewMemoryRefreshTokenStore() *MemoryRefreshTokenStore {
	s := &MemoryRefreshTokenStore{
		families: make(map[string]*memRefreshFamily),
		tokens:   make(map[string]*memRefreshToken),
		done:     make(chan struct{}),
	}
	asyncutil.SafeGo(func() { s.cleanup() })
	return s
}

// cleanup removes expired entries every 5 minutes.
func (m *MemoryRefreshTokenStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.mu.Lock()
			now := time.Now()
			for k, f := range m.families {
				if now.After(f.expiry) {
					delete(m.families, k)
				}
			}
			for k, t := range m.tokens {
				if now.After(t.expiry) {
					delete(m.tokens, k)
				}
			}
			m.mu.Unlock()
		case <-m.done:
			return
		}
	}
}

// StoreFamily persists a token family in memory with the given TTL.
func (m *MemoryRefreshTokenStore) StoreFamily(_ context.Context, family *RefreshTokenFamily, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *family
	m.families[family.FamilyID] = &memRefreshFamily{family: &cp, expiry: time.Now().Add(ttl)}
	return nil
}

// GetFamily retrieves a token family from memory. Returns nil if not found or expired.
func (m *MemoryRefreshTokenStore) GetFamily(_ context.Context, familyID string) (*RefreshTokenFamily, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.families[familyID]
	if !ok || time.Now().After(f.expiry) {
		delete(m.families, familyID)
		return nil, nil
	}
	cp := *f.family
	return &cp, nil
}

// UpdateFamily overwrites an existing token family in memory with a new TTL.
func (m *MemoryRefreshTokenStore) UpdateFamily(ctx context.Context, family *RefreshTokenFamily, ttl time.Duration) error {
	return m.StoreFamily(ctx, family, ttl)
}

// DeleteFamily removes a token family from memory.
func (m *MemoryRefreshTokenStore) DeleteFamily(_ context.Context, familyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.families, familyID)
	return nil
}

// StoreToken persists a refresh token entry in memory with the given TTL.
func (m *MemoryRefreshTokenStore) StoreToken(_ context.Context, tokenHash string, entry *RefreshTokenEntry, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *entry
	m.tokens[tokenHash] = &memRefreshToken{entry: &cp, expiry: time.Now().Add(ttl)}
	return nil
}

// GetToken retrieves a refresh token entry from memory. Returns nil if not found or expired.
func (m *MemoryRefreshTokenStore) GetToken(_ context.Context, tokenHash string) (*RefreshTokenEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tokens[tokenHash]
	if !ok || time.Now().After(t.expiry) {
		delete(m.tokens, tokenHash)
		return nil, nil
	}
	cp := *t.entry
	return &cp, nil
}

// MarkTokenUsed sets the Used flag to true for the given token hash in memory.
func (m *MemoryRefreshTokenStore) MarkTokenUsed(_ context.Context, tokenHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tokens[tokenHash]
	if !ok || time.Now().After(t.expiry) {
		return nil
	}
	t.entry.Used = true
	return nil
}

// DeleteToken removes a refresh token entry from memory.
func (m *MemoryRefreshTokenStore) DeleteToken(_ context.Context, tokenHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tokens, tokenHash)
	return nil
}
