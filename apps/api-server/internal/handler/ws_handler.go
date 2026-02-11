package handler

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/ws"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development; in production this should be restricted.
		return true
	},
}

// WSHandler handles WebSocket upgrade requests.
type WSHandler struct {
	hub       *ws.Hub
	validator middleware.TokenValidator
}

// NewWSHandler creates a new WSHandler.
func NewWSHandler(hub *ws.Hub, validator middleware.TokenValidator) *WSHandler {
	return &WSHandler{hub: hub, validator: validator}
}

// ServeWS upgrades the HTTP connection to a WebSocket and registers the client.
// Authentication is performed via a JWT token passed as a query parameter (?token=xxx).
func (h *WSHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	// Authenticate via query parameter
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		writeError(w, http.StatusUnauthorized, "missing token query parameter")
		return
	}

	claims, err := h.validator.ValidateToken(tokenStr)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid user ID in token")
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("ws: upgrade failed", "error", err)
		return
	}

	client := ws.NewClient(h.hub, conn, claims.TenantID, userID)
	h.hub.Register(client)

	// Start read/write pumps in goroutines
	go client.WritePump()
	go client.ReadPump()
}
