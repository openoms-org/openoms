package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

type AuthHandler struct {
	authService    *service.AuthService
	isDev          bool
	tokenBlacklist *middleware.TokenBlacklist
}

func NewAuthHandler(authService *service.AuthService, isDev bool, blacklist ...*middleware.TokenBlacklist) *AuthHandler {
	h := &AuthHandler{authService: authService, isDev: isDev}
	if len(blacklist) > 0 {
		h.tokenBlacklist = blacklist[0]
	}
	return h
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, refreshToken, err := h.authService.Register(r.Context(), req, clientIP(r))
	if err != nil {
		slog.Error("registration error", "error", err)
		switch err {
		case service.ErrSlugTaken:
			writeError(w, http.StatusConflict, "tenant slug is already taken")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "registration failed")
			}
		}
		return
	}

	h.setRefreshCookie(w, refreshToken, 30*24*3600)
	writeJSON(w, http.StatusCreated, resp)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, refreshToken, err := h.authService.Login(r.Context(), req, clientIP(r))
	if err != nil {
		switch err {
		case service.ErrInvalidCredentials:
			writeError(w, http.StatusUnauthorized, "invalid email or password")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "login failed")
			}
		}
		return
	}

	h.setRefreshCookie(w, refreshToken, 30*24*3600)
	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing refresh token")
		return
	}

	resp, newRefreshToken, err := h.authService.Refresh(r.Context(), cookie.Value)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	h.setRefreshCookie(w, newRefreshToken, 30*24*3600)
	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Revoke the access token if a blacklist is configured
	if h.tokenBlacklist != nil {
		if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenStr != "" {
				tokenHash := middleware.HashToken(tokenStr)
				h.tokenBlacklist.Revoke(tokenHash, time.Now().Add(1*time.Hour))
			}
		}
	}

	if cookie, err := r.Cookie("refresh_token"); err == nil && cookie.Value != "" {
		if claims, err := h.authService.ValidateRefreshToken(cookie.Value); err == nil {
			_ = h.authService.Logout(r.Context(), claims.UserID, claims.TenantID)
		}
	}

	h.clearRefreshCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

func (h *AuthHandler) setRefreshCookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/v1/auth",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   !h.isDev,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *AuthHandler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/v1/auth",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   !h.isDev,
		SameSite: http.SameSiteLaxMode,
	})
}

