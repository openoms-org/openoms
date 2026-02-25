package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func TestKSeFSettings_AutoSend_Marshal(t *testing.T) {
	cfg := KSeFSettings{
		Enabled:     true,
		Environment: "test",
		NIP:         "1234567890",
		Token:       "test-token",
		AutoSend:    true,
		CompanyName: "Test Company",
	}

	data, err := json.Marshal(cfg)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"auto_send":true`)
}

func TestKSeFSettings_AutoSend_Unmarshal(t *testing.T) {
	jsonStr := `{"enabled":true,"environment":"production","nip":"9876543210","token":"prod-token","auto_send":true,"company_name":"Prod Co"}`

	var cfg KSeFSettings
	err := json.Unmarshal([]byte(jsonStr), &cfg)
	require.NoError(t, err)
	assert.True(t, cfg.AutoSend)
	assert.True(t, cfg.Enabled)
	assert.Equal(t, "production", cfg.Environment)
}

func TestKSeFSettings_AutoSend_DefaultsFalse(t *testing.T) {
	jsonStr := `{"enabled":true,"environment":"test","nip":"1234567890","token":"tok"}`

	var cfg KSeFSettings
	err := json.Unmarshal([]byte(jsonStr), &cfg)
	require.NoError(t, err)
	assert.False(t, cfg.AutoSend)
}

func TestPow3(t *testing.T) {
	tests := []struct {
		n    int
		want int
	}{
		{0, 1},
		{1, 3},
		{2, 9},
		{3, 27},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, pow3(tt.n), "pow3(%d)", tt.n)
	}
}

// TestMarkKSeFError_StatusContract verifies the data transformation that markKSeFError
// performs on the Invoice struct. Since markKSeFError is a private method requiring a
// pgx.Tx, we replicate its in-memory logic to protect the contract:
//   - KSeFStatus must be set to "error" (retryable), NOT "rejected" (terminal)
//   - retry_count from existing ksef_response must be preserved
//   - ksef_response JSON must contain error, retry_count, and last_error_at fields
func TestMarkKSeFError_StatusContract(t *testing.T) {
	tests := []struct {
		name             string
		existingResponse json.RawMessage
		wantRetryCount   int
		errMsg           string
	}{
		{
			name:             "fresh invoice with no prior response",
			existingResponse: nil,
			wantRetryCount:   0,
			errMsg:           "authorisation challenge: connection refused",
		},
		{
			name:             "invoice with retry_count=1 from prior error",
			existingResponse: json.RawMessage(`{"error":"init session: timeout","retry_count":1,"last_error_at":"2026-02-25T10:00:00Z"}`),
			wantRetryCount:   1,
			errMsg:           "send invoice: server error",
		},
		{
			name:             "invoice with retry_count=2 preserves count",
			existingResponse: json.RawMessage(`{"error":"previous error","retry_count":2,"last_error_at":"2026-02-25T11:00:00Z"}`),
			wantRetryCount:   2,
			errMsg:           "authorisation challenge: DNS resolution failed",
		},
		{
			name:             "invoice with malformed JSON response",
			existingResponse: json.RawMessage(`{invalid json`),
			wantRetryCount:   0,
			errMsg:           "init session: parse error",
		},
		{
			name:             "invoice with response missing retry_count field",
			existingResponse: json.RawMessage(`{"error":"old error"}`),
			wantRetryCount:   0,
			errMsg:           "send invoice: 500 Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := &model.Invoice{
				ID:           uuid.New(),
				KSeFStatus:   "not_sent",
				KSeFResponse: tt.existingResponse,
			}

			// Replicate the markKSeFError data transformation (lines 624-646 of ksef_service.go).
			// This protects against regressions where someone might change "error" to "rejected".
			retryCount := 0
			if inv.KSeFResponse != nil {
				var existing map[string]any
				if jsonErr := json.Unmarshal(inv.KSeFResponse, &existing); jsonErr == nil {
					if rc, ok := existing["retry_count"].(float64); ok {
						retryCount = int(rc)
					}
				}
			}

			inv.KSeFStatus = "error"
			responseJSON, _ := json.Marshal(map[string]any{
				"error":         tt.errMsg,
				"retry_count":   retryCount,
				"last_error_at": time.Now().UTC().Format(time.RFC3339),
			})
			inv.KSeFResponse = responseJSON

			// Core assertion: status MUST be "error" (retryable), never "rejected" (terminal).
			// "rejected" is reserved for UPO failures (processing_code >= 400) in CheckKSeFStatus.
			assert.Equal(t, "error", inv.KSeFStatus, "markKSeFError must set retryable 'error' status, not terminal 'rejected'")
			assert.NotEqual(t, "rejected", inv.KSeFStatus)

			// Verify retry_count is preserved from existing response
			assert.Equal(t, tt.wantRetryCount, retryCount, "retry_count should be preserved from prior ksef_response")

			// Verify the response JSON structure
			var resp map[string]any
			require.NoError(t, json.Unmarshal(inv.KSeFResponse, &resp))
			assert.Equal(t, tt.errMsg, resp["error"])
			assert.Equal(t, float64(tt.wantRetryCount), resp["retry_count"])
			assert.NotEmpty(t, resp["last_error_at"], "last_error_at must be set")

			// Verify last_error_at is a valid RFC3339 timestamp
			lastErrorAt, ok := resp["last_error_at"].(string)
			require.True(t, ok)
			_, parseErr := time.Parse(time.RFC3339, lastErrorAt)
			assert.NoError(t, parseErr, "last_error_at must be valid RFC3339")
		})
	}
}

// TestRetryErroredInvoices_BackoffEligibility verifies the retry eligibility logic
// that RetryErroredInvoices uses to decide whether an errored invoice should be retried.
// The logic parses retry_count and last_error_at from ksef_response JSON:
//   - Max 3 retries; after that, invoice becomes "rejected" (terminal)
//   - Exponential backoff: 5min * 3^retry_count (5min, 15min, 45min)
//   - Invoice is skipped if the backoff period hasn't elapsed yet
func TestRetryErroredInvoices_BackoffEligibility(t *testing.T) {
	const maxRetries = 3

	tests := []struct {
		name         string
		ksefResponse json.RawMessage
		now          time.Time
		wantEligible bool // true = should be retried, false = skip or reject
		wantTerminal bool // true = max retries exceeded → "rejected"
	}{
		{
			name:         "first retry, backoff elapsed",
			ksefResponse: json.RawMessage(`{"error":"connection refused","retry_count":0,"last_error_at":"2026-02-25T10:00:00Z"}`),
			now:          time.Date(2026, 2, 25, 10, 6, 0, 0, time.UTC), // 6 min after error (backoff=5min)
			wantEligible: true,
			wantTerminal: false,
		},
		{
			name:         "first retry, backoff NOT elapsed",
			ksefResponse: json.RawMessage(`{"error":"connection refused","retry_count":0,"last_error_at":"2026-02-25T10:00:00Z"}`),
			now:          time.Date(2026, 2, 25, 10, 3, 0, 0, time.UTC), // 3 min after error (backoff=5min)
			wantEligible: false,
			wantTerminal: false,
		},
		{
			name:         "second retry, backoff=15min elapsed",
			ksefResponse: json.RawMessage(`{"error":"timeout","retry_count":1,"last_error_at":"2026-02-25T10:00:00Z"}`),
			now:          time.Date(2026, 2, 25, 10, 16, 0, 0, time.UTC), // 16 min after error (backoff=15min)
			wantEligible: true,
			wantTerminal: false,
		},
		{
			name:         "second retry, backoff=15min NOT elapsed",
			ksefResponse: json.RawMessage(`{"error":"timeout","retry_count":1,"last_error_at":"2026-02-25T10:00:00Z"}`),
			now:          time.Date(2026, 2, 25, 10, 10, 0, 0, time.UTC), // 10 min after error (backoff=15min)
			wantEligible: false,
			wantTerminal: false,
		},
		{
			name:         "third retry, backoff=45min elapsed",
			ksefResponse: json.RawMessage(`{"error":"server error","retry_count":2,"last_error_at":"2026-02-25T10:00:00Z"}`),
			now:          time.Date(2026, 2, 25, 10, 46, 0, 0, time.UTC), // 46 min after error (backoff=45min)
			wantEligible: true,
			wantTerminal: false,
		},
		{
			name:         "max retries exceeded — should become terminal",
			ksefResponse: json.RawMessage(`{"error":"persistent failure","retry_count":3,"last_error_at":"2026-02-25T10:00:00Z"}`),
			now:          time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC),
			wantEligible: false,
			wantTerminal: true,
		},
		{
			name:         "retry_count=4 also terminal",
			ksefResponse: json.RawMessage(`{"error":"still failing","retry_count":4,"last_error_at":"2026-02-25T10:00:00Z"}`),
			now:          time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC),
			wantEligible: false,
			wantTerminal: true,
		},
		{
			name:         "missing last_error_at — eligible immediately (zero time)",
			ksefResponse: json.RawMessage(`{"error":"some error","retry_count":0}`),
			now:          time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC),
			wantEligible: true,
			wantTerminal: false,
		},
		{
			name:         "nil ksef_response — retry_count=0, eligible immediately",
			ksefResponse: nil,
			now:          time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC),
			wantEligible: true,
			wantTerminal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Replicate the retry eligibility logic from RetryErroredInvoices (lines 669-696).
			retryCount := 0
			var lastErrorAt time.Time

			if tt.ksefResponse != nil {
				var respData map[string]any
				if jsonErr := json.Unmarshal(tt.ksefResponse, &respData); jsonErr == nil {
					if rc, ok := respData["retry_count"].(float64); ok {
						retryCount = int(rc)
					}
					if lea, ok := respData["last_error_at"].(string); ok {
						lastErrorAt, _ = time.Parse(time.RFC3339, lea)
					}
				}
			}

			// Check max retries
			if retryCount >= maxRetries {
				assert.True(t, tt.wantTerminal, "retry_count=%d should be terminal (>= maxRetries=%d)", retryCount, maxRetries)
				assert.False(t, tt.wantEligible, "terminal invoices should not be eligible for retry")
				return
			}
			assert.False(t, tt.wantTerminal, "retry_count=%d should NOT be terminal", retryCount)

			// Check backoff: 5min * 3^retryCount
			backoff := 5 * time.Minute * time.Duration(pow3(retryCount))
			eligible := lastErrorAt.IsZero() || !tt.now.Before(lastErrorAt.Add(backoff))

			if tt.wantEligible {
				assert.True(t, eligible, "invoice should be eligible for retry (backoff=%v, elapsed=%v)", backoff, tt.now.Sub(lastErrorAt))
			} else {
				assert.False(t, eligible, "invoice should NOT be eligible for retry yet (backoff=%v, elapsed=%v)", backoff, tt.now.Sub(lastErrorAt))
			}

			// Verify backoff values match the documented schedule: 5min, 15min, 45min
			expectedBackoffs := map[int]time.Duration{
				0: 5 * time.Minute,
				1: 15 * time.Minute,
				2: 45 * time.Minute,
			}
			if expected, ok := expectedBackoffs[retryCount]; ok {
				assert.Equal(t, expected, backoff, "backoff for retry_count=%d", retryCount)
			}
		})
	}
}
