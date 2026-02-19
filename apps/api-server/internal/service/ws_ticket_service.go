package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// WSTicketStore abstracts storage for WebSocket tickets.
type WSTicketStore interface {
	// Store saves a ticket with associated user data and TTL. Returns error if fails.
	Store(ctx context.Context, ticket string, data WSTicketData, ttl time.Duration) error
	// Consume retrieves and deletes a ticket atomically. Returns nil data if not found.
	Consume(ctx context.Context, ticket string) (*WSTicketData, error)
}

type WSTicketData struct {
	Token string // the original JWT access token
}

type WSTicketService struct {
	store WSTicketStore
}

func NewWSTicketService(store WSTicketStore) *WSTicketService {
	return &WSTicketService{store: store}
}

// Issue creates a new single-use ticket valid for 30 seconds.
func (s *WSTicketService) Issue(ctx context.Context, accessToken string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate ticket: %w", err)
	}
	ticket := hex.EncodeToString(b)
	err := s.store.Store(ctx, "ws_ticket:"+ticket, WSTicketData{Token: accessToken}, 30*time.Second)
	if err != nil {
		return "", fmt.Errorf("store ticket: %w", err)
	}
	return ticket, nil
}

// Redeem consumes a ticket and returns the original access token.
func (s *WSTicketService) Redeem(ctx context.Context, ticket string) (string, error) {
	data, err := s.store.Consume(ctx, "ws_ticket:"+ticket)
	if err != nil {
		return "", fmt.Errorf("consume ticket: %w", err)
	}
	if data == nil {
		return "", fmt.Errorf("invalid or expired ticket")
	}
	return data.Token, nil
}
