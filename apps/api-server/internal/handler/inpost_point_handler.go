package handler

import (
	"log/slog"
	"net/http"

	inpost "github.com/openoms-org/openoms/packages/inpost-go-sdk"
)

type InPostPointHandler struct {
	inpostClient *inpost.Client
}

func NewInPostPointHandler(inpostClient *inpost.Client) *InPostPointHandler {
	return &InPostPointHandler{inpostClient: inpostClient}
}

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
		slog.Error("inpost point search failed", "error", err, "query", query)
		writeError(w, http.StatusBadGateway, "failed to search InPost points")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
