package handler

import (
	"log/slog"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/ws"
)

// WSHandler handles WebSocket upgrade requests.
type WSHandler struct {
	hub           *ws.Hub
	validator     middleware.TokenValidator
	upgrader      websocket.Upgrader
	allowedOrigin string
}

// NewWSHandler creates a new WSHandler.
func NewWSHandler(hub *ws.Hub, validator middleware.TokenValidator, frontendURL string) *WSHandler {
	h := &WSHandler{hub: hub, validator: validator, allowedOrigin: frontendURL}
	h.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true // non-browser clients (curl, etc.)
			}
			parsed, err := url.Parse(h.allowedOrigin)
			if err != nil || parsed.Host == "" {
				return false
			}
			originParsed, err := url.Parse(origin)
			if err != nil {
				return false
			}
			return originParsed.Scheme == parsed.Scheme && originParsed.Host == parsed.Host
		},
	}
	return h
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

	// Reject non-access tokens (refresh, 2fa_pending, etc.)
	if claims.Type != "" && claims.Type != "access" {
		writeError(w, http.StatusUnauthorized, "invalid token type")
		return
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid user ID in token")
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
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
