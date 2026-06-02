package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/getsentry/sentry-go"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// AuthHandler handles HTTP requests for authentication and session management.
type AuthHandler struct {
	authService      *service.AuthService
	invitationSvc    *service.InvitationService
	licenseSvc       *service.LicenseService
	checkoutSvc      *service.CheckoutService
	wsTicketSvc      *service.WSTicketService
	isDev            bool
	registrationMode string // "open", "invite", "closed"; "disabled" is a legacy alias for closed
	tokenBlacklist   *middleware.TokenBlacklist
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authService *service.AuthService, isDev bool, blacklist ...*middleware.TokenBlacklist) *AuthHandler {
	h := &AuthHandler{authService: authService, isDev: isDev, registrationMode: "open"}
	if len(blacklist) > 0 {
		h.tokenBlacklist = blacklist[0]
	}
	return h
}

// SetRegistrationMode updates the handler's registration mode at runtime.
func (h *AuthHandler) SetRegistrationMode(mode string) {
	h.registrationMode = mode
}

// SetInvitationService injects the invitation service dependency.
func (h *AuthHandler) SetInvitationService(svc *service.InvitationService) {
	h.invitationSvc = svc
}

// SetWSTicketService sets the WebSocket ticket service used for single-use WS authentication.
func (h *AuthHandler) SetWSTicketService(svc *service.WSTicketService) {
	h.wsTicketSvc = svc
}

// SetLicenseService sets the license verification service for cloud billing.
func (h *AuthHandler) SetLicenseService(svc *service.LicenseService) {
	h.licenseSvc = svc
}

// SetCheckoutService sets the Stripe checkout service for billing registration.
func (h *AuthHandler) SetCheckoutService(svc *service.CheckoutService) {
	h.checkoutSvc = svc
}

// Register handles new user and tenant registration.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	// Check registration mode
	switch h.registrationMode {
	case "closed", "disabled":
		writeError(w, http.StatusForbidden, "registration is disabled")
		return
	case "invite":
		// handled below after decoding
	case "open":
		// allow
	default:
		slog.Error("invalid registration mode; failing closed", "mode", h.registrationMode)
		writeError(w, http.StatusForbidden, "registration is disabled")
		return
	}

	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// In invite mode, validate the invitation, license token, or checkout session
	var licenseClaims *model.LicenseClaims
	var checkoutSessionID string
	if h.registrationMode == "invite" {
		switch {
		case req.CheckoutSessionID != "" && h.checkoutSvc != nil:
			// Stripe checkout session flow — post-payment registration
			session, err := h.checkoutSvc.GetSessionStatus(r.Context(), req.CheckoutSessionID)
			if err != nil {
				writeServerError(w, "failed to verify checkout session", err)
				return
			}
			if session == nil {
				writeError(w, http.StatusBadRequest, "checkout session not found")
				return
			}
			if session.Status != "completed" {
				writeError(w, http.StatusBadRequest, "checkout session is not completed")
				return
			}
			if !strings.EqualFold(session.Email, req.Email) {
				writeError(w, http.StatusBadRequest, "email does not match checkout session")
				return
			}
			checkoutSessionID = req.CheckoutSessionID
			req.Plan = session.Plan
			req.PlanLimits = session.Limits
			req.CheckoutSessionInterval = session.Interval
		case req.LicenseToken != "" && h.licenseSvc != nil && !h.licenseSvc.IsDisabled():
			// License token flow — cloud billing registration
			claims, err := h.licenseSvc.VerifyToken(r.Context(), req.LicenseToken)
			if err != nil {
				switch {
				case errors.Is(err, service.ErrLicenseTokenExpired):
					writeError(w, http.StatusBadRequest, "license token has expired")
				case errors.Is(err, service.ErrLicenseTokenUsed):
					writeError(w, http.StatusConflict, "license token has already been used")
				default:
					writeError(w, http.StatusBadRequest, "invalid license token")
				}
				return
			}
			if !strings.EqualFold(claims.Email, req.Email) {
				writeError(w, http.StatusBadRequest, "email does not match license token")
				return
			}
			licenseClaims = claims
			req.Email = claims.Email
			req.Plan = claims.Plan
			req.PlanLimits = &claims.Limits
		case req.InviteToken != "":
			// Existing invite token flow
			if h.invitationSvc == nil {
				writeError(w, http.StatusInternalServerError, "invitation service not configured")
				return
			}
			inv, err := h.invitationSvc.ValidateToken(r.Context(), req.InviteToken)
			if err != nil {
				switch err {
				case service.ErrInvitationNotFound:
					writeError(w, http.StatusBadRequest, "invalid invitation token")
				case service.ErrInvitationExpired:
					writeError(w, http.StatusBadRequest, "invitation has expired")
				case service.ErrInvitationUsed:
					writeError(w, http.StatusBadRequest, "invitation has already been used")
				default:
					writeServerError(w, "failed to validate invitation", err)
				}
				return
			}
			req.Email = inv.Email
		default:
			writeError(w, http.StatusBadRequest, "invite_token, license_token, or checkout_session_id is required")
			return
		}
	}

	// Atomically claim the license token BEFORE registration to prevent TOCTOU race.
	// If two concurrent requests use the same token, only one will succeed here.
	// tenant_id is initially NULL and updated after registration.
	if licenseClaims != nil {
		if err := h.licenseSvc.ClaimToken(r.Context(), licenseClaims.JTI, licenseClaims.Email, licenseClaims.Plan); err != nil {
			if errors.Is(err, service.ErrLicenseTokenUsed) {
				writeError(w, http.StatusConflict, "license token has already been used")
			} else {
				writeServerError(w, "failed to claim license token", err)
			}
			return
		}
	}

	// Atomically claim the checkout session BEFORE registration to prevent TOCTOU race.
	// Same pattern as license token: claim first, set tenant_id after registration.
	if checkoutSessionID != "" && h.checkoutSvc != nil {
		if err := h.checkoutSvc.ClaimSession(r.Context(), checkoutSessionID); err != nil {
			if errors.Is(err, service.ErrCheckoutSessionClaimed) {
				writeError(w, http.StatusConflict, "checkout session has already been used")
			} else {
				writeServerError(w, "failed to claim checkout session", err)
			}
			return
		}
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
				writeServerError(w, "registration failed", err)
			}
		}
		return
	}

	// Mark invitation as used after successful registration
	if h.registrationMode == "invite" && req.InviteToken != "" && h.invitationSvc != nil {
		if err := h.invitationSvc.MarkUsed(r.Context(), req.InviteToken); err != nil {
			slog.Warn("failed to mark invitation as used", "error", err)
		}
	}

	// Set the actual tenant_id on the claimed license token
	if licenseClaims != nil && h.licenseSvc != nil {
		if err := h.licenseSvc.UpdateClaimedTenant(r.Context(), licenseClaims.JTI, resp.Tenant.ID); err != nil {
			slog.Warn("failed to update license token tenant_id", "error", err)
		}
	}

	// Finalize checkout claim: set tenant_id, create billing customer + subscription records.
	// On failure the tenant is registered but has no billing records; alert (Sentry) so it
	// is visible, and the billing reconciliation worker re-runs the idempotent finalization.
	if checkoutSessionID != "" && h.checkoutSvc != nil {
		if err := h.checkoutSvc.FinalizeCheckoutClaim(r.Context(), checkoutSessionID, resp.Tenant.ID, req.Plan, req.CheckoutSessionInterval); err != nil {
			slog.Error("failed to finalize checkout claim after registration", "tenant_id", resp.Tenant.ID, "error", err) //nolint:gosec // G706: static message; tenant_id is server-generated UUID, no raw checkout session or request text is logged from the handler.
			sentry.CaptureException(err)
		}
	}

	h.setRefreshCookie(w, refreshToken)
	writeJSON(w, http.StatusCreated, resp)
}

// Login authenticates a user and issues access and refresh tokens.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.authService.Login(r.Context(), req, clientIP(r))
	if err != nil {
		switch err {
		case service.ErrAccountLocked:
			writeError(w, http.StatusTooManyRequests, "account temporarily locked due to too many failed attempts")
		case service.ErrInvalidCredentials:
			writeError(w, http.StatusUnauthorized, "invalid email or password")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeServerError(w, "login failed", err)
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

	h.setRefreshCookie(w, result.RefreshToken)
	writeJSON(w, http.StatusOK, result.TokenResponse)
}

// TwoFALogin completes login by verifying a TOTP code after password auth.
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
			writeServerError(w, "2FA verification failed", err)
		}
		return
	}

	h.setRefreshCookie(w, refreshToken)
	writeJSON(w, http.StatusOK, resp)
}

// TwoFASetup generates a TOTP secret and QR code for 2FA enrollment.
// Requires password re-authentication to prevent session-hijack amplification.
func (h *AuthHandler) TwoFASetup(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	tenantID := middleware.TenantIDFromContext(r.Context())
	claims := middleware.ClaimsFromContext(r.Context())

	var req model.TwoFASetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Password == "" {
		writeError(w, http.StatusBadRequest, "password is required")
		return
	}

	resp, err := h.authService.Setup2FA(r.Context(), userID, tenantID, claims.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid password")
			return
		}
		writeServerError(w, "2FA setup failed", err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// TwoFAVerify confirms a TOTP code and activates 2FA for the user.
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
			writeServerError(w, "2FA verification failed", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "2FA enabled"})
}

// TwoFADisable turns off 2FA for the authenticated user.
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
			writeServerError(w, "failed to disable 2FA", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "2FA disabled"})
}

// TwoFAStatus returns the current 2FA enrollment state for the user.
func (h *AuthHandler) TwoFAStatus(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	tenantID := middleware.TenantIDFromContext(r.Context())

	resp, err := h.authService.Get2FAStatus(r.Context(), userID, tenantID)
	if err != nil {
		writeServerError(w, "failed to get 2FA status", err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// Refresh rotates the access token using a valid refresh token cookie.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing refresh token")
		return
	}

	resp, newRefreshToken, err := h.authService.Refresh(r.Context(), cookie.Value)
	if err != nil {
		switch err {
		case service.ErrRefreshTokenReuse:
			h.clearRefreshCookie(w)
			writeError(w, http.StatusUnauthorized, "session invalidated due to token reuse")
		default:
			writeError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		}
		return
	}

	h.setRefreshCookie(w, newRefreshToken)
	writeJSON(w, http.StatusOK, resp)
}

// Logout revokes the current session and clears auth cookies.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Revoke the access token until its actual JWT expiry if a blacklist is configured.
	if h.tokenBlacklist != nil && h.authService != nil {
		if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenStr != "" {
				if expiresAt, err := h.authService.AccessTokenExpiresAt(tokenStr); err == nil {
					tokenHash := middleware.HashToken(tokenStr)
					h.tokenBlacklist.Revoke(tokenHash, expiresAt)
				}
			}
		}
	}

	if cookie, err := r.Cookie("refresh_token"); err == nil && cookie.Value != "" {
		if claims, err := h.authService.ValidateRefreshToken(cookie.Value); err == nil {
			_ = h.authService.LogoutWithRefreshToken(r.Context(), claims.UserID, claims.TenantID, cookie.Value)
		}
	}

	h.clearRefreshCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

func (h *AuthHandler) setRefreshCookie(w http.ResponseWriter, token string) {
	const maxAge = 30 * 24 * 3600 // 30 days
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

// WSTicket issues a short-lived, single-use ticket for WebSocket connections.
// This keeps the JWT access token out of URL query parameters and server/proxy logs.
func (h *AuthHandler) WSTicket(w http.ResponseWriter, r *http.Request) {
	// Get the access token from the Authorization header (already validated by JWTAuth middleware)
	tokenStr := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if tokenStr == "" || h.wsTicketSvc == nil {
		writeError(w, http.StatusInternalServerError, "ws tickets not configured")
		return
	}

	ticket, err := h.wsTicketSvc.Issue(r.Context(), tokenStr)
	if err != nil {
		writeServerError(w, "failed to create ws ticket", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"ticket": ticket})
}
