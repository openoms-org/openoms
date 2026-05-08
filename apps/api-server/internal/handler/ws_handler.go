package handler

import (
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/openoms-org/openoms/apps/api-server/internal/asyncutil"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
	"github.com/openoms-org/openoms/apps/api-server/internal/ws"
)

// WSHandler handles WebSocket upgrade requests.
type WSHandler struct {
	hub           *ws.Hub
	validator     middleware.TokenValidator
	blacklist     *middleware.TokenBlacklist
	ticketSvc     *service.WSTicketService
	upgrader      websocket.Upgrader
	allowedOrigin string
}

// NewWSHandler creates a new WSHandler.
func NewWSHandler(hub *ws.Hub, validator middleware.TokenValidator, blacklist *middleware.TokenBlacklist, ticketSvc *service.WSTicketService, frontendURL string) *WSHandler {
	h := &WSHandler{hub: hub, validator: validator, blacklist: blacklist, ticketSvc: ticketSvc, allowedOrigin: frontendURL}
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
			// In development, allow localhost origins regardless of port
			if strings.Contains(parsed.Hostname(), "localhost") || parsed.Hostname() == "127.0.0.1" {
				oh := originParsed.Hostname()
				if oh == "localhost" || oh == "127.0.0.1" {
					return true
				}
			}
			return originParsed.Scheme == parsed.Scheme && originParsed.Host == parsed.Host
		},
	}
	return h
}

// ServeWS upgrades the HTTP connection to a WebSocket and registers the client.
// Authentication is performed via a short-lived ticket (?ticket=xxx) or a JWT token (?token=xxx, deprecated).
func (h *WSHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	var tokenStr string

	// Prefer ticket-based auth (keeps JWT out of URLs/logs)
	if ticket := r.URL.Query().Get("ticket"); ticket != "" && h.ticketSvc != nil {
		var err error
		tokenStr, err = h.ticketSvc.Redeem(r.Context(), ticket)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid or expired ticket")
			return
		}
	} else {
		writeError(w, http.StatusUnauthorized, "missing ticket query parameter")
		return
	}

	claims, err := h.validator.ValidateToken(tokenStr)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}

	// Check token blacklist (revoked after logout)
	if h.blacklist != nil {
		tokenHash := middleware.HashToken(tokenStr)
		if h.blacklist.IsRevoked(tokenHash) {
			writeError(w, http.StatusUnauthorized, "token has been revoked")
			return
		}
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
	asyncutil.SafeGo(func() { client.WritePump() })
	asyncutil.SafeGo(func() { client.ReadPump() })
}
