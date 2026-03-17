package linear

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/openoms-org/openoms/apps/orchestrator/internal/model"
	"github.com/openoms-org/openoms/apps/orchestrator/internal/store"
)

// WebhookHandler handles incoming Linear webhook events.
type WebhookHandler struct {
	secret    string
	taskStore *store.TaskStore
}

// NewWebhookHandler creates a WebhookHandler that verifies requests with the given secret.
func NewWebhookHandler(secret string, taskStore *store.TaskStore) *WebhookHandler {
	return &WebhookHandler{secret: secret, taskStore: taskStore}
}

type webhookPayload struct {
	Action string          `json:"action"`
	Type   string          `json:"type"`
	Data   json.RawMessage `json:"data"`
}

type issueData struct {
	ID          string    `json:"id"`
	Identifier  string    `json:"identifier"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Priority    int       `json:"priority"`
	State       stateData `json:"state"`
	Labels      []label   `json:"labels"`
}

type stateData struct {
	Name string `json:"name"`
}

type label struct {
	Name string `json:"name"`
}

// HandleWebhook is the HTTP handler for Linear webhook events.
func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}

	sig := r.Header.Get("Linear-Signature")
	if !h.verifySignature(body, sig) {
		slog.Warn("webhook signature verification failed")
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	var payload webhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		slog.Error("unmarshal webhook payload", "error", err)
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if payload.Type != "Issue" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var issue issueData
	if err := json.Unmarshal(payload.Data, &issue); err != nil {
		slog.Error("unmarshal issue data", "error", err)
		http.Error(w, "invalid issue data", http.StatusBadRequest)
		return
	}

	for _, l := range issue.Labels {
		if l.Name == "manual" {
			slog.Info("skipping manual ticket", "identifier", issue.Identifier)
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	switch payload.Action {
	case "create":
		h.handleCreate(r.Context(), w, &issue)
	case "update":
		h.handleUpdate(r.Context(), w, &issue)
	default:
		w.WriteHeader(http.StatusOK)
	}
}

func (h *WebhookHandler) handleCreate(ctx context.Context, w http.ResponseWriter, issue *issueData) {
	if issue.State.Name != "Todo" && issue.State.Name != "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	task := model.NewTask(issue.Identifier, issue.Title, issue.Description)
	task.Priority = issue.Priority

	err := h.taskStore.Create(ctx, task)
	if errors.Is(err, store.ErrTaskExists) {
		slog.Debug("task already exists, skipping", "identifier", issue.Identifier)
		w.WriteHeader(http.StatusOK)
		return
	}
	if err != nil {
		slog.Error("create task from webhook", "error", err, "identifier", issue.Identifier)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	slog.Info("task enqueued from webhook", "identifier", issue.Identifier, "title", issue.Title)
	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) handleUpdate(ctx context.Context, w http.ResponseWriter, issue *issueData) {
	if issue.State.Name == "Todo" {
		exists, err := h.taskStore.Exists(ctx, issue.Identifier)
		if err != nil {
			slog.Error("check task exists", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !exists {
			task := model.NewTask(issue.Identifier, issue.Title, issue.Description)
			task.Priority = issue.Priority
			if err := h.taskStore.Create(ctx, task); err != nil && !errors.Is(err, store.ErrTaskExists) {
				slog.Error("create task from update webhook", "error", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			slog.Info("task enqueued from status update", "identifier", issue.Identifier)
		}

		// Re-queue blocked tasks when moved back to Todo in Linear.
		task, err := h.taskStore.Get(ctx, issue.Identifier)
		if err == nil && task.State == model.StateBlocked {
			_ = h.taskStore.UpdateState(ctx, issue.Identifier, model.StateQueued)
			slog.Info("blocked task re-queued from Linear", "identifier", issue.Identifier)
		}
	}

	if issue.State.Name == "Cancelled" {
		_ = h.taskStore.Delete(ctx, issue.Identifier)
		slog.Info("task cancelled via Linear", "identifier", issue.Identifier)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) verifySignature(body []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
