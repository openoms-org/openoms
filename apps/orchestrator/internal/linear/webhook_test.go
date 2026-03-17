package linear_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	redisclient "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/orchestrator/internal/linear"
	"github.com/openoms-org/openoms/apps/orchestrator/internal/store"
)

const testWebhookSecret = "whsec_test_secret_123"

func setupWebhookTest(t *testing.T) (*linear.WebhookHandler, *store.TaskStore) {
	t.Helper()
	url := os.Getenv("REDIS_URL")
	if url == "" {
		url = "redis://localhost:6379/1"
	}
	opts, err := redisclient.ParseURL(url)
	require.NoError(t, err)
	rdb := redisclient.NewClient(opts)
	require.NoError(t, rdb.Ping(context.Background()).Err())
	rdb.FlushDB(context.Background())
	t.Cleanup(func() {
		rdb.FlushDB(context.Background())
		rdb.Close()
	})

	taskStore := store.New(rdb)
	wh := linear.NewWebhookHandler(testWebhookSecret, taskStore)
	return wh, taskStore
}

func signPayload(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestWebhook_IssueCreate(t *testing.T) {
	wh, taskStore := setupWebhookTest(t)

	payload := map[string]interface{}{
		"action": "create",
		"type":   "Issue",
		"data": map[string]interface{}{
			"id":          "issue-uuid-123",
			"identifier":  "OPE-142",
			"title":       "Add bulk export",
			"description": "Implement CSV export for orders",
			"priority":    2,
			"state":       map[string]interface{}{"name": "Todo"},
		},
	}
	body, _ := json.Marshal(payload)
	sig := signPayload(testWebhookSecret, body)

	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/linear", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Linear-Signature", sig)

	w := httptest.NewRecorder()
	wh.HandleWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	task, err := taskStore.Get(context.Background(), "OPE-142")
	require.NoError(t, err)
	assert.Equal(t, "Add bulk export", task.Title)
}

func TestWebhook_InvalidSignature(t *testing.T) {
	wh, _ := setupWebhookTest(t)

	body := []byte(`{"action":"create","type":"Issue","data":{}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/linear", bytes.NewReader(body))
	req.Header.Set("Linear-Signature", "invalid")

	w := httptest.NewRecorder()
	wh.HandleWebhook(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestWebhook_DuplicateIssue(t *testing.T) {
	wh, _ := setupWebhookTest(t)

	payload := map[string]interface{}{
		"action": "create",
		"type":   "Issue",
		"data": map[string]interface{}{
			"identifier":  "OPE-142",
			"title":       "Dup",
			"description": "",
			"priority":    2,
			"state":       map[string]interface{}{"name": "Todo"},
		},
	}
	body, _ := json.Marshal(payload)
	sig := signPayload(testWebhookSecret, body)

	req1 := httptest.NewRequest(http.MethodPost, "/api/webhooks/linear", bytes.NewReader(body))
	req1.Header.Set("Linear-Signature", sig)
	w1 := httptest.NewRecorder()
	wh.HandleWebhook(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	req2 := httptest.NewRequest(http.MethodPost, "/api/webhooks/linear", bytes.NewReader(body))
	req2.Header.Set("Linear-Signature", sig)
	w2 := httptest.NewRecorder()
	wh.HandleWebhook(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code) // idempotent
}

func TestWebhook_IgnoresNonIssue(t *testing.T) {
	wh, _ := setupWebhookTest(t)

	payload := map[string]interface{}{
		"action": "create",
		"type":   "Comment",
		"data":   map[string]interface{}{},
	}
	body, _ := json.Marshal(payload)
	sig := signPayload(testWebhookSecret, body)

	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/linear", bytes.NewReader(body))
	req.Header.Set("Linear-Signature", sig)
	w := httptest.NewRecorder()
	wh.HandleWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code) // ack but ignore
}
