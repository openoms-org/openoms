package automation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// ActionExecutor executes a single automation action.
type ActionExecutor interface {
	ExecuteAction(ctx context.Context, tenantID uuid.UUID, action Action, event Event) error
}

// DefaultActionExecutor is a basic action executor that handles webhook actions
// and logs other action types. Real integrations (set_status, send_email, etc.)
// would be wired through service-layer callbacks in production.
type DefaultActionExecutor struct {
	logger     *slog.Logger
	httpClient *http.Client
}

func NewDefaultActionExecutor(logger *slog.Logger) *DefaultActionExecutor {
	return &DefaultActionExecutor{
		logger: logger,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (e *DefaultActionExecutor) ExecuteAction(ctx context.Context, tenantID uuid.UUID, action Action, event Event) error {
	switch action.Type {
	case "webhook":
		return e.executeWebhook(ctx, tenantID, action, event)
	case "set_status":
		e.logger.Info("automation action: set_status",
			"tenant_id", tenantID,
			"entity_type", event.EntityType,
			"entity_id", event.EntityID,
			"params", action.Params,
		)
		return nil
	case "add_tag":
		e.logger.Info("automation action: add_tag",
			"tenant_id", tenantID,
			"entity_type", event.EntityType,
			"entity_id", event.EntityID,
			"params", action.Params,
		)
		return nil
	case "send_email":
		e.logger.Info("automation action: send_email",
			"tenant_id", tenantID,
			"entity_type", event.EntityType,
			"entity_id", event.EntityID,
			"params", action.Params,
		)
		return nil
	case "create_invoice":
		e.logger.Info("automation action: create_invoice",
			"tenant_id", tenantID,
			"entity_type", event.EntityType,
			"entity_id", event.EntityID,
			"params", action.Params,
		)
		return nil
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

func (e *DefaultActionExecutor) executeWebhook(ctx context.Context, tenantID uuid.UUID, action Action, event Event) error {
	url, _ := action.Params["url"].(string)
	if url == "" {
		return fmt.Errorf("webhook action missing url parameter")
	}

	payload := map[string]any{
		"event":       event.Type,
		"tenant_id":   tenantID.String(),
		"entity_type": event.EntityType,
		"entity_id":   event.EntityID.String(),
		"data":        event.Data,
		"fired_at":    time.Now().UTC().Format(time.RFC3339),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "OpenOMS-Automation/1.0")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}
