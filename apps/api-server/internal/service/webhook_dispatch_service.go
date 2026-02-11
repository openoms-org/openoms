package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/netutil"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

type wsBroadcastFunc func(tenantID uuid.UUID, eventType string, payload any)

type WebhookDispatchService struct {
	tenantRepo   repository.TenantRepo
	deliveryRepo repository.WebhookDeliveryRepo
	pool         *pgxpool.Pool
	httpClient   *http.Client
	wsBroadcast  wsBroadcastFunc
}

// noPrivateDialer returns a DialContext function that refuses to connect to private IP addresses.
// This prevents SSRF TOCTOU attacks by checking the resolved IP at connect time (atomically).
func noPrivateDialer() func(ctx context.Context, network, addr string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid address: %w", err)
		}

		ips, err := net.DefaultResolver.LookupHost(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("DNS lookup failed: %w", err)
		}

		for _, ipStr := range ips {
			ip := net.ParseIP(ipStr)
			if ip != nil && netutil.IsPrivateIP(ip) {
				return nil, fmt.Errorf("connection to private IP %s rejected", ipStr)
			}
		}

		// Connect to the first resolved IP to avoid TOCTOU
		return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0], port))
	}
}

func NewWebhookDispatchService(
	tenantRepo repository.TenantRepo,
	deliveryRepo repository.WebhookDeliveryRepo,
	pool *pgxpool.Pool,
) *WebhookDispatchService {
	return &WebhookDispatchService{
		tenantRepo:   tenantRepo,
		deliveryRepo: deliveryRepo,
		pool:         pool,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: noPrivateDialer(),
			},
		},
	}
}

// SetWSBroadcast sets the function used to broadcast events via WebSocket.
func (s *WebhookDispatchService) SetWSBroadcast(fn func(tenantID uuid.UUID, eventType string, payload any)) {
	s.wsBroadcast = fn
}

// Dispatch sends webhook to all matching endpoints. Called as goroutine.
func (s *WebhookDispatchService) Dispatch(ctx context.Context, tenantID uuid.UUID, eventType string, payload any) {
	// Also broadcast to WebSocket clients
	if s.wsBroadcast != nil {
		s.wsBroadcast(tenantID, eventType, payload)
	}

	// 1. Load tenant settings
	var settingsRaw json.RawMessage
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var e error
		settingsRaw, e = s.tenantRepo.GetSettings(ctx, tx, tenantID)
		return e
	})
	if err != nil {
		slog.Error("webhook: failed to load settings", "error", err, "tenant_id", tenantID)
		return
	}

	// 2. Parse webhook config
	var allSettings map[string]json.RawMessage
	if err := json.Unmarshal(settingsRaw, &allSettings); err != nil {
		return
	}

	var config model.WebhookConfig
	if raw, ok := allSettings["webhooks"]; ok {
		if err := json.Unmarshal(raw, &config); err != nil {
			slog.Error("webhook: failed to parse config", "error", err)
			return
		}
	}

	if len(config.Endpoints) == 0 {
		return
	}

	// 3. Marshal payload
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		slog.Error("webhook: failed to marshal payload", "error", err)
		return
	}

	// 4. Dispatch to matching endpoints
	for _, ep := range config.Endpoints {
		if !ep.Active {
			continue
		}
		if !matchesEvent(ep.Events, eventType) {
			continue
		}
		s.sendWebhookWithRetry(ctx, tenantID, ep, eventType, payloadJSON)
	}
}

func (s *WebhookDispatchService) sendWebhookWithRetry(ctx context.Context, tenantID uuid.UUID, ep model.WebhookEndpoint, eventType string, payload []byte) {
	maxRetries := 3
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<(2*uint(attempt-1))) * time.Second // 1s, 4s, 16s
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
		}
		success := s.trySendWebhook(ctx, tenantID, ep, eventType, payload, attempt == maxRetries)
		if success {
			return
		}
	}
}

// trySendWebhook attempts a single webhook delivery. It logs the delivery only
// on success or when isFinalAttempt is true. Returns true if successful.
func (s *WebhookDispatchService) trySendWebhook(ctx context.Context, tenantID uuid.UUID, ep model.WebhookEndpoint, eventType string, payload []byte, isFinalAttempt bool) bool {
	// SSRF protection is handled atomically by the custom dialer (noPrivateDialer)
	// which checks the resolved IP at connect time, avoiding TOCTOU vulnerabilities.

	// Compute HMAC-SHA256 signature
	mac := hmac.New(sha256.New, []byte(ep.Secret))
	mac.Write(payload)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	// Send HTTP POST
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ep.URL, strings.NewReader(string(payload)))
	if err != nil {
		if isFinalAttempt {
			s.logDelivery(ctx, tenantID, ep.URL, eventType, payload, "failed", nil, err.Error())
		}
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("X-Webhook-Event", eventType)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		if isFinalAttempt {
			s.logDelivery(ctx, tenantID, ep.URL, eventType, payload, "failed", nil, err.Error())
		}
		return false
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body) // drain body

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		s.logDelivery(ctx, tenantID, ep.URL, eventType, payload, "success", &resp.StatusCode, "")
		return true
	}

	if isFinalAttempt {
		errMsg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		s.logDelivery(ctx, tenantID, ep.URL, eventType, payload, "failed", &resp.StatusCode, errMsg)
	}
	return false
}

func (s *WebhookDispatchService) logDelivery(ctx context.Context, tenantID uuid.UUID, url, eventType string, payload []byte, status string, responseCode *int, errMsg string) {
	delivery := &model.WebhookDelivery{
		ID:           uuid.New(),
		TenantID:     tenantID,
		URL:          url,
		EventType:    eventType,
		Payload:      payload,
		Status:       status,
		ResponseCode: responseCode,
		CreatedAt:    time.Now(),
	}
	if errMsg != "" {
		delivery.Error = &errMsg
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		return s.deliveryRepo.Create(ctx, tx, delivery)
	})
	if err != nil {
		slog.Error("webhook: failed to log delivery", "error", err)
	}
}

func matchesEvent(events []string, eventType string) bool {
	for _, e := range events {
		if e == eventType || e == "*" {
			return true
		}
	}
	return false
}
