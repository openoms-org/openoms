package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openoms-org/openoms/apps/api-server/internal/netutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettingsHandler_UpdateEmailSettings_InvalidJSON(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPut, "/v1/settings/email", strings.NewReader("not json"))
	rr := httptest.NewRecorder()

	h.UpdateEmailSettings(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestSettingsHandler_UpdateCompanySettings_InvalidJSON(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPut, "/v1/settings/company", strings.NewReader("not json"))
	rr := httptest.NewRecorder()

	h.UpdateCompanySettings(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestSettingsHandler_UpdateOrderStatuses_InvalidJSON(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPut, "/v1/settings/order-statuses", strings.NewReader("not json"))
	rr := httptest.NewRecorder()

	h.UpdateOrderStatuses(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestSettingsHandler_UpdateOrderStatuses_EmptyKey(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	body := `{"statuses":[{"key":"","label":"Test","color":"blue","position":1}],"transitions":{}}`
	req := httptest.NewRequest(http.MethodPut, "/v1/settings/order-statuses", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateOrderStatuses(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "key and label are required")
}

func TestSettingsHandler_UpdateOrderStatuses_DuplicateKey(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	body := `{"statuses":[{"key":"new","label":"New","color":"blue","position":1},{"key":"new","label":"Dupe","color":"red","position":2}],"transitions":{}}`
	req := httptest.NewRequest(http.MethodPut, "/v1/settings/order-statuses", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateOrderStatuses(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "duplicate")
}

func TestSettingsHandler_UpdateOrderStatuses_TransitionFromUnknown(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	body := `{"statuses":[{"key":"new","label":"New","color":"blue","position":1}],"transitions":{"bogus":["new"]}}`
	req := httptest.NewRequest(http.MethodPut, "/v1/settings/order-statuses", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateOrderStatuses(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "unknown status")
}

func TestSettingsHandler_UpdateOrderStatuses_TransitionToUnknown(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	body := `{"statuses":[{"key":"new","label":"New","color":"blue","position":1}],"transitions":{"new":["bogus"]}}`
	req := httptest.NewRequest(http.MethodPut, "/v1/settings/order-statuses", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateOrderStatuses(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "unknown status")
}

func TestSettingsHandler_UpdateCustomFields_InvalidJSON(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPut, "/v1/settings/custom-fields", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.UpdateCustomFields(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestSettingsHandler_UpdateCustomFields_EmptyKey(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	body := `{"fields":[{"key":"","label":"Test","type":"text","position":1}]}`
	req := httptest.NewRequest(http.MethodPut, "/v1/settings/custom-fields", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateCustomFields(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestSettingsHandler_UpdateCustomFields_DuplicateKey(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	body := `{"fields":[{"key":"f1","label":"F1","type":"text","position":1},{"key":"f1","label":"F1 Dupe","type":"text","position":2}]}`
	req := httptest.NewRequest(http.MethodPut, "/v1/settings/custom-fields", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateCustomFields(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "duplicate")
}

func TestSettingsHandler_UpdateCustomFields_InvalidType(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	body := `{"fields":[{"key":"f1","label":"F1","type":"invalid","position":1}]}`
	req := httptest.NewRequest(http.MethodPut, "/v1/settings/custom-fields", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateCustomFields(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "invalid field type")
}

func TestSettingsHandler_UpdateCustomFields_SelectWithoutOptions(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	body := `{"fields":[{"key":"f1","label":"F1","type":"select","position":1}]}`
	req := httptest.NewRequest(http.MethodPut, "/v1/settings/custom-fields", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateCustomFields(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "at least 1 option")
}

func TestSettingsHandler_UpdateWebhooks_InvalidJSON(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPut, "/v1/settings/webhooks", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.UpdateWebhooks(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestSettingsHandler_UpdateWebhooks_MissingName(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	body := `{"endpoints":[{"url":"https://example.com","events":["order.created"],"active":true}]}`
	req := httptest.NewRequest(http.MethodPut, "/v1/settings/webhooks", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateWebhooks(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "name is required")
}

func TestSettingsHandler_UpdateWebhooks_MissingURL(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	body := `{"endpoints":[{"name":"Test","events":["order.created"],"active":true}]}`
	req := httptest.NewRequest(http.MethodPut, "/v1/settings/webhooks", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateWebhooks(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "URL is required")
}

func TestSettingsHandler_UpdateWebhooks_NoEvents(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	body := `{"endpoints":[{"name":"Test","url":"https://example.com","events":[],"active":true}]}`
	req := httptest.NewRequest(http.MethodPut, "/v1/settings/webhooks", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateWebhooks(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "at least one event")
}

func TestSettingsHandler_UpdateWebhooks_DuplicateID(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	body := `{"endpoints":[
		{"id":"ep1","name":"A","url":"https://example.com/hook1","events":["*"],"active":true},
		{"id":"ep1","name":"B","url":"https://example.com/hook2","events":["*"],"active":true}
	]}`
	req := httptest.NewRequest(http.MethodPut, "/v1/settings/webhooks", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateWebhooks(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "duplicate")
}

func TestSettingsHandler_UpdateWebhooks_PrivateURL(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	body := `{"endpoints":[{"name":"Test","url":"http://127.0.0.1/hook","events":["*"],"active":true}]}`
	req := httptest.NewRequest(http.MethodPut, "/v1/settings/webhooks", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateWebhooks(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "private")
}

func TestSettingsHandler_UpdateProductCategories_InvalidJSON(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPut, "/v1/settings/product-categories", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.UpdateProductCategories(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestSettingsHandler_UpdateProductCategories_EmptyKey(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	body := `{"categories":[{"key":"","label":"Test","color":"blue","position":1}]}`
	req := httptest.NewRequest(http.MethodPut, "/v1/settings/product-categories", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateProductCategories(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestSettingsHandler_UpdateProductCategories_DuplicateKey(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	body := `{"categories":[{"key":"c1","label":"C1","color":"blue","position":1},{"key":"c1","label":"C1 dupe","color":"red","position":2}]}`
	req := httptest.NewRequest(http.MethodPut, "/v1/settings/product-categories", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateProductCategories(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "duplicate")
}

func TestSettingsHandler_SendTestEmail_InvalidJSON(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/settings/email/test", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.SendTestEmail(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestSettingsHandler_SendTestEmail_MissingEmail(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/settings/email/test", strings.NewReader(`{"to_email":""}`))
	rr := httptest.NewRecorder()

	h.SendTestEmail(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestSettingsHandler_UpdateInvoicingSettings_InvalidJSON(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPut, "/v1/settings/invoicing", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.UpdateInvoicingSettings(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestSettingsHandler_MaskInvoicingSettings_MasksNestedCredentials(t *testing.T) {
	cfg := map[string]any{
		"provider": "fakturownia",
		"credentials": map[string]any{
			"api_key": "secret",
			"nested": map[string]any{
				"refresh_token": "refresh-secret",
			},
			"empty": "",
		},
	}

	masked := maskInvoicingSettings(cfg)

	credentials := masked["credentials"].(map[string]any)
	assert.Equal(t, "••••••", credentials["api_key"])
	assert.Equal(t, "••••••", credentials["nested"].(map[string]any)["refresh_token"])
	assert.Equal(t, "", credentials["empty"])
	assert.Equal(t, "secret", cfg["credentials"].(map[string]any)["api_key"])
}

func TestSettingsHandler_PreserveMaskedSecretValues(t *testing.T) {
	incoming := map[string]any{
		"api_key": "••••••",
		"nested": map[string]any{
			"refresh_token": "******",
			"new_secret":    "new-value",
		},
		"empty_mask": "**REDACTED**",
	}
	existing := map[string]any{
		"api_key": "old-secret",
		"nested": map[string]any{
			"refresh_token": "old-refresh",
		},
	}

	preserved := preserveMaskedSecretValues(incoming, existing).(map[string]any)

	assert.Equal(t, "old-secret", preserved["api_key"])
	assert.Equal(t, "old-refresh", preserved["nested"].(map[string]any)["refresh_token"])
	assert.Equal(t, "new-value", preserved["nested"].(map[string]any)["new_secret"])
	assert.Equal(t, "", preserved["empty_mask"])
}

func TestSettingsHandler_MaskSensitiveSettings_MasksAllTenantSecrets(t *testing.T) {
	raw := json.RawMessage(`{
		"email":{"smtp_pass":"smtp-secret"},
		"sms":{"api_token":"sms-secret"},
		"ksef":{"token":"ksef-secret"},
		"invoicing":{"credentials":{"api_key":"invoice-secret"}},
		"webhooks":{"endpoints":[{"secret":"webhook-secret"}]}
	}`)

	masked := maskSensitiveSettings(raw)

	text := string(masked)
	assert.NotContains(t, text, "smtp-secret")
	assert.NotContains(t, text, "sms-secret")
	assert.NotContains(t, text, "ksef-secret")
	assert.NotContains(t, text, "invoice-secret")
	assert.NotContains(t, text, "webhook-secret")
	assert.Contains(t, text, "**REDACTED**")
}

func TestSettingsHandler_IsPrivateWebhookURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"http://127.0.0.1/hook", true},
		{"http://localhost/hook", true},
		{"http://10.0.0.1/hook", true},
		{"http://192.168.1.1/hook", true},
		{"https://example.com/hook", false},
		{"", true},
		{"://bad", true},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			assert.Equal(t, tt.want, netutil.IsPrivateURL(tt.url))
		})
	}
}
