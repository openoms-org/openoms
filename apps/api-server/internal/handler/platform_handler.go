package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// PlatformAuditLogger records platform-admin audit entries.
// Implemented by repository.PlatformAuditRepository.
type PlatformAuditLogger interface {
	Log(ctx context.Context, entry model.PlatformAuditEntry) error
}

// PlatformHandler serves platform-admin (cross-tenant) endpoints. Every route
// is gated by middleware.RequirePlatformAdmin, so a resolved platform admin is
// expected in the request context.
type PlatformHandler struct {
	audit PlatformAuditLogger
}

// NewPlatformHandler creates a PlatformHandler. audit may be nil (access
// auditing is then skipped).
func NewPlatformHandler(audit PlatformAuditLogger) *PlatformHandler {
	return &PlatformHandler{audit: audit}
}

type platformMeResponse struct {
	UserID      string   `json:"user_id"`
	Permissions []string `json:"permissions"`
}

// Me returns the authenticated platform admin's identity and permissions, and
// records the console access in the platform audit log. The audit write is
// best-effort: a failure is logged but does not block the response.
func (h *PlatformHandler) Me(w http.ResponseWriter, r *http.Request) {
	admin := middleware.PlatformAdminFromContext(r.Context())
	if admin == nil {
		// Defensive: RequirePlatformAdmin should guarantee a non-nil admin.
		writeError(w, http.StatusUnauthorized, "platform administration access required")
		return
	}

	if h.audit != nil {
		if err := h.audit.Log(r.Context(), model.PlatformAuditEntry{
			ActorUserID: admin.UserID,
			Action:      "platform.session.accessed",
			IPAddress:   clientIP(r),
		}); err != nil {
			slog.Warn("failed to write platform audit entry", "error", err, "action", "platform.session.accessed")
		}
	}

	perms := admin.Permissions
	if perms == nil {
		perms = []string{}
	}
	writeJSON(w, http.StatusOK, platformMeResponse{
		UserID:      admin.UserID.String(),
		Permissions: perms,
	})
}
