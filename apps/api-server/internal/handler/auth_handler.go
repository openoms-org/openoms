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

	result, err := h.authService.Login(r.Context(), req, clientIP(r))
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

	if result.Requires2FA {
		writeJSON(w, http.StatusOK, model.LoginResponse{
			Requires2FA: true,
			TempToken:   result.TempToken,
		})
		return
	}

	h.setRefreshCookie(w, result.RefreshToken, 30*24*3600)
	writeJSON(w, http.StatusOK, result.TokenResponse)
}

func (h *AuthHandler) TwoFALogin(w http.ResponseWriter, r *http.Request) {
	var req model.TwoFALoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.TempToken == "" || req.Code == "" {
		writeError(w, http.StatusBadRequest, "temp_token and code are required")
		return
	}

	resp, refreshToken, err := h.authService.Verify2FALogin(r.Context(), req.TempToken, req.Code)
	if err != nil {
		switch err {
		case service.ErrInvalid2FAToken:
			writeError(w, http.StatusUnauthorized, "invalid or expired 2FA token")
		case service.ErrInvalid2FACode:
			writeError(w, http.StatusUnauthorized, "invalid 2FA code")
		default:
			writeError(w, http.StatusInternalServerError, "2FA verification failed")
		}
		return
	}

	h.setRefreshCookie(w, refreshToken, 30*24*3600)
	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) TwoFASetup(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	tenantID := middleware.TenantIDFromContext(r.Context())
	claims := middleware.ClaimsFromContext(r.Context())

	resp, err := h.authService.Setup2FA(r.Context(), userID, tenantID, claims.Email)
	if err != nil {
		slog.Error("2fa setup error", "error", err)
		writeError(w, http.StatusInternalServerError, "2FA setup failed")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) TwoFAVerify(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	tenantID := middleware.TenantIDFromContext(r.Context())

	var req model.TwoFAVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Code == "" {
		writeError(w, http.StatusBadRequest, "code is required")
		return
	}

	if err := h.authService.Verify2FA(r.Context(), userID, tenantID, req.Code); err != nil {
		switch err {
		case service.ErrInvalid2FACode:
			writeError(w, http.StatusBadRequest, "invalid 2FA code")
		case service.Err2FANotSetup:
			writeError(w, http.StatusBadRequest, "2FA has not been set up yet")
		default:
			slog.Error("2fa verify error", "error", err)
			writeError(w, http.StatusInternalServerError, "2FA verification failed")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "2FA enabled"})
}

func (h *AuthHandler) TwoFADisable(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	tenantID := middleware.TenantIDFromContext(r.Context())

	var req model.TwoFADisableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Password == "" || req.Code == "" {
		writeError(w, http.StatusBadRequest, "password and code are required")
		return
	}

	if err := h.authService.Disable2FA(r.Context(), userID, tenantID, req.Password, req.Code); err != nil {
		switch err {
		case service.ErrInvalidCredentials:
			writeError(w, http.StatusUnauthorized, "invalid password")
		case service.ErrInvalid2FACode:
			writeError(w, http.StatusBadRequest, "invalid 2FA code")
		default:
			slog.Error("2fa disable error", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to disable 2FA")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "2FA disabled"})
}

func (h *AuthHandler) TwoFAStatus(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	tenantID := middleware.TenantIDFromContext(r.Context())

	resp, err := h.authService.Get2FAStatus(r.Context(), userID, tenantID)
	if err != nil {
		slog.Error("2fa status error", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get 2FA status")
		return
	}

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

