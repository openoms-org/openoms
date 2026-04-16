package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

// Event represents a real-time event broadcast to WebSocket clients.
type Event struct {
	Type     string `json:"type"`
	TenantID string `json:"tenant_id,omitempty"`
	Payload  any    `json:"payload,omitempty"`
}

// Hub maintains the set of active clients grouped by tenant and broadcasts messages.
type Hub struct {
	mu         sync.RWMutex
	tenants    map[uuid.UUID]map[*Client]struct{}
	register   chan *Client
	unregister chan *Client
	broadcast  chan tenantEvent
}

type tenantEvent struct {
	tenantID uuid.UUID
	data     []byte
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		tenants:    make(map[uuid.UUID]map[*Client]struct{}),
		register:   make(chan *Client, 256),
		unregister: make(chan *Client, 256),
		broadcast:  make(chan tenantEvent, 256),
	}
}

// Run starts the hub event loop. Should be run as a goroutine.
// It returns when the provided context is cancelled, closing all client connections.
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			h.mu.Lock()
			for tenantID, clients := range h.tenants {
				for client := range clients {
					close(client.send)
				}
				delete(h.tenants, tenantID)
			}
			h.mu.Unlock()
			slog.Info("ws: hub stopped")
			return

		case client := <-h.register:
			h.mu.Lock()
			if _, ok := h.tenants[client.TenantID]; !ok {
				h.tenants[client.TenantID] = make(map[*Client]struct{})
			}
			h.tenants[client.TenantID][client] = struct{}{}
			h.mu.Unlock()
			slog.Debug("ws: client registered", "tenant_id", client.TenantID)

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.tenants[client.TenantID]; ok {
				if _, exists := clients[client]; exists {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.tenants, client.TenantID)
					}
				}
			}
			h.mu.Unlock()
			slog.Debug("ws: client unregistered", "tenant_id", client.TenantID)

		case evt := <-h.broadcast:
			// Copy client set under lock to avoid holding lock during sends
			h.mu.RLock()
			src := h.tenants[evt.tenantID]
			if len(src) == 0 {
				h.mu.RUnlock()
				continue
			}
			snapshot := make([]*Client, 0, len(src))
			for c := range src {
				snapshot = append(snapshot, c)
			}
			h.mu.RUnlock()

			for _, client := range snapshot {
				select {
				case client.send <- evt.data:
				default:
					// Client too slow, disconnect
					h.mu.Lock()
					if clients, ok := h.tenants[client.TenantID]; ok {
						if _, exists := clients[client]; exists {
							delete(clients, client)
							close(client.send)
							if len(clients) == 0 {
								delete(h.tenants, client.TenantID)
							}
						}
					}
					h.mu.Unlock()
				}
			}
		}
	}
}

// BroadcastToTenant sends an event to all clients of a specific tenant.
func (h *Hub) BroadcastToTenant(tenantID uuid.UUID, event Event) {
	event.TenantID = tenantID.String()
	data, err := json.Marshal(event)
	if err != nil {
		slog.Error("ws: failed to marshal event", "error", err)
		return
	}
	select {
	case h.broadcast <- tenantEvent{tenantID: tenantID, data: data}:
	default:
		slog.Warn("ws: broadcast channel full, dropping event", "type", event.Type, "tenant_id", tenantID)
	}
}

// Register adds a client to the hub.
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub.
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}
