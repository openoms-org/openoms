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
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

type WebhookDispatchService struct {
	tenantRepo   repository.TenantRepo
	deliveryRepo repository.WebhookDeliveryRepo
	pool         *pgxpool.Pool
	httpClient   *http.Client
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
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

// Dispatch sends webhook to all matching endpoints. Called as goroutine.
func (s *WebhookDispatchService) Dispatch(ctx context.Context, tenantID uuid.UUID, eventType string, payload any) {
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
		s.sendWebhook(ctx, tenantID, ep, eventType, payloadJSON)
	}
}

func (s *WebhookDispatchService) sendWebhook(ctx context.Context, tenantID uuid.UUID, ep model.WebhookEndpoint, eventType string, payload []byte) {
	// SSRF protection: reject private/internal URLs
	if isPrivateURL(ep.URL) {
		slog.Warn("webhook: skipping dispatch to private/internal URL", "url", ep.URL, "tenant_id", tenantID)
		return
	}

	// Compute HMAC-SHA256 signature
	mac := hmac.New(sha256.New, []byte(ep.Secret))
	mac.Write(payload)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	// Send HTTP POST
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ep.URL, strings.NewReader(string(payload)))
	if err != nil {
		s.logDelivery(ctx, tenantID, ep.URL, eventType, payload, "failed", nil, err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("X-Webhook-Event", eventType)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logDelivery(ctx, tenantID, ep.URL, eventType, payload, "failed", nil, err.Error())
		return
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body) // drain body

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		s.logDelivery(ctx, tenantID, ep.URL, eventType, payload, "success", &resp.StatusCode, "")
	} else {
		errMsg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		s.logDelivery(ctx, tenantID, ep.URL, eventType, payload, "failed", &resp.StatusCode, errMsg)
	}
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

// isPrivateURL checks whether a URL resolves to a private/internal IP address.
func isPrivateURL(urlStr string) bool {
	u, err := url.Parse(urlStr)
	if err != nil {
		return true // reject unparseable URLs
	}

	hostname := u.Hostname()
	if hostname == "" {
		return true
	}

	ips, err := net.LookupHost(hostname)
	if err != nil {
		return true // reject unresolvable hostnames
	}

	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	}

	var cidrs []*net.IPNet
	for _, cidr := range privateRanges {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		cidrs = append(cidrs, ipNet)
	}

	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		for _, cidr := range cidrs {
			if cidr.Contains(ip) {
				return true
			}
		}
	}

	return false
}

func matchesEvent(events []string, eventType string) bool {
	for _, e := range events {
		if e == eventType || e == "*" {
			return true
		}
	}
	return false
}
