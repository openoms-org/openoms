package handler

import (
	"log/slog"
	"net/http"

	inpost "github.com/openoms-org/openoms/packages/inpost-go-sdk"
)

// InPostPointHandler handles HTTP requests for InPost parcel locker point searches.
type InPostPointHandler struct {
	inpostClient *inpost.Client
}

// NewInPostPointHandler creates a new InPostPointHandler.
func NewInPostPointHandler(inpostClient *inpost.Client) *InPostPointHandler {
	return &InPostPointHandler{inpostClient: inpostClient}
}

// Search returns InPost parcel locker points matching the given query.
func (h *InPostPointHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		writeError(w, http.StatusBadRequest, "query parameter is required")
		return
	}
	if len(query) < 2 {
		writeError(w, http.StatusBadRequest, "query must be at least 2 characters")
		return
	}

	resp, err := h.inpostClient.Points.Search(
		r.Context(),
		query,
		inpost.PointTypeParcelLocker,
		10,
	)
	if err != nil {
		slog.Error("inpost point search failed", "error", err, "query", query) //nolint:gosec
		writeError(w, http.StatusUnprocessableEntity, "failed to search InPost points")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
