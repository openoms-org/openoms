package ws

import (
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
func (h *Hub) Run() {
	for {
		select {
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
			h.mu.RLock()
			clients, ok := h.tenants[evt.tenantID]
			h.mu.RUnlock()
			if !ok {
				continue
			}
			for client := range clients {
				select {
				case client.send <- evt.data:
				default:
					// Client too slow, disconnect
					h.mu.Lock()
					if clients, ok := h.tenants[client.TenantID]; ok {
						delete(clients, client)
						close(client.send)
						if len(clients) == 0 {
							delete(h.tenants, client.TenantID)
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

// ConnectedClients returns the total number of connected clients across all tenants.
func (h *Hub) ConnectedClients() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	total := 0
	for _, clients := range h.tenants {
		total += len(clients)
	}
	return total
}
