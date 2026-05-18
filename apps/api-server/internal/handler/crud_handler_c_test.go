package handler

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// ===========================================================================
// InPostWebhookHandler tests
// ===========================================================================

type recordedShipmentStatusUpdate struct {
	tenantID       uuid.UUID
	trackingNumber string
	provider       string
	status         string
}

type recordingShipmentStatusUpdater struct {
	updateCh chan recordedShipmentStatusUpdate
}

func (r *recordingShipmentStatusUpdater) UpdateStatusByTrackingNumber(context.Context, string, string, string) error {
	return nil
}

func (r *recordingShipmentStatusUpdater) UpdateStatusByTrackingNumberForTenant(_ context.Context, tenantID uuid.UUID, trackingNumber, provider, newStatus string) error {
	r.updateCh <- recordedShipmentStatusUpdate{
		tenantID:       tenantID,
		trackingNumber: trackingNumber,
		provider:       provider,
		status:         newStatus,
	}
	return nil
}

func TestInPostWebhookHandler_MissingWebhookSecret(t *testing.T) {
	h := NewInPostWebhookHandler("", nil)

	body := `{"type":"shipment_status_changed","payload":{}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/inpost", strings.NewReader(body))
	req.Header.Set("X-InPost-Signature", "somesignature")
	rr := httptest.NewRecorder()

	h.HandleWebhook(rr, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestInPostWebhookHandler_MissingSignatureHeader(t *testing.T) {
	h := NewInPostWebhookHandler("test-secret", nil)

	body := `{"type":"shipment_status_changed","payload":{}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/inpost", strings.NewReader(body))
	// No X-InPost-Signature header
	rr := httptest.NewRecorder()

	h.HandleWebhook(rr, req)

	// Handler returns 200 OK even on missing signature (so InPost doesn't retry)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestInPostWebhookHandler_InvalidHMACSignature(t *testing.T) {
	h := NewInPostWebhookHandler("test-secret", nil)

	body := `{"type":"shipment_status_changed","payload":{}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/inpost", strings.NewReader(body))
	req.Header.Set("X-InPost-Signature", "invalid-signature-value")
	rr := httptest.NewRecorder()

	h.HandleWebhook(rr, req)

	// Handler returns 200 OK even on invalid signature (so InPost doesn't retry)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestInPostWebhookHandler_InvalidJSONBody(t *testing.T) {
	secret := "test-secret"
	h := NewInPostWebhookHandler(secret, nil)

	// Invalid JSON that will pass HMAC verification but fail JSON parsing
	body := []byte(`not valid json`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	signature := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/inpost", strings.NewReader(string(body)))
	req.Header.Set("X-InPost-Signature", signature)
	rr := httptest.NewRecorder()

	h.HandleWebhook(rr, req)

	// Handler returns 200 OK even on parse failure (so InPost doesn't retry)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestInPostWebhookHandler_ValidSignature(t *testing.T) {
	secret := "test-secret"
	h := NewInPostWebhookHandler(secret, nil)

	body := []byte(`{"type":"shipment_created","payload":{}}`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	signature := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/inpost", strings.NewReader(string(body)))
	req.Header.Set("X-InPost-Signature", signature)
	rr := httptest.NewRecorder()

	h.HandleWebhook(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestInPostWebhookHandler_ScopedWebhookUpdatesVerifiedTenant(t *testing.T) {
	tenantID := uuid.New()
	integrationID := uuid.New()
	secret := "tenant-inpost-webhook-secret" //nolint:gosec // test credential
	updater := &recordingShipmentStatusUpdater{updateCh: make(chan recordedShipmentStatusUpdate, 1)}
	h := NewInPostWebhookHandler("legacy-secret", updater)
	h.SetProviderWebhookSecretResolver(fakeProviderWebhookSecretResolver{
		secret: secret,
		scope: &service.ProviderWebhookScope{
			TenantID:      tenantID,
			IntegrationID: integrationID,
			Provider:      "inpost",
		},
	})

	body := []byte(`{"type":"shipment_status_changed","payload":{"shipment_id":123,"tracking_number":"TRACK123","status":"delivered"}}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/inpost/"+integrationID.String(), strings.NewReader(string(body)))
	req = requestWithIntegrationID(req, integrationID)
	req.Header.Set("X-InPost-Signature", computeHMAC(secret, body))
	rr := httptest.NewRecorder()

	h.HandleWebhook(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	select {
	case got := <-updater.updateCh:
		assert.Equal(t, tenantID, got.tenantID)
		assert.Equal(t, "TRACK123", got.trackingNumber)
		assert.Equal(t, "inpost", got.provider)
		assert.Equal(t, "delivered", got.status)
	case <-time.After(time.Second):
		require.Fail(t, "expected scoped shipment status update")
	}
}

func TestInPostWebhookHandler_ScopedWebhookWrongSecretDoesNotUpdate(t *testing.T) {
	integrationID := uuid.New()
	updater := &recordingShipmentStatusUpdater{updateCh: make(chan recordedShipmentStatusUpdate, 1)}
	h := NewInPostWebhookHandler("legacy-secret", updater)
	h.SetProviderWebhookSecretResolver(fakeProviderWebhookSecretResolver{
		secret: "tenant-inpost-webhook-secret",
		scope: &service.ProviderWebhookScope{
			TenantID:      uuid.New(),
			IntegrationID: integrationID,
			Provider:      "inpost",
		},
	})

	body := []byte(`{"type":"shipment_status_changed","payload":{"shipment_id":123,"tracking_number":"TRACK123","status":"delivered"}}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/inpost/"+integrationID.String(), strings.NewReader(string(body)))
	req = requestWithIntegrationID(req, integrationID)
	req.Header.Set("X-InPost-Signature", computeHMAC("wrong-secret", body))
	rr := httptest.NewRecorder()

	h.HandleWebhook(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	select {
	case <-updater.updateCh:
		require.Fail(t, "did not expect scoped shipment status update")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestInPostWebhookHandler_ScopedWebhookWithoutResolverFailsClosed(t *testing.T) {
	integrationID := uuid.New()
	updater := &recordingShipmentStatusUpdater{updateCh: make(chan recordedShipmentStatusUpdate, 1)}
	h := NewInPostWebhookHandler("legacy-secret", updater)

	body := []byte(`{"type":"shipment_status_changed","payload":{"shipment_id":123,"tracking_number":"TRACK123","status":"delivered"}}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/inpost/"+integrationID.String(), strings.NewReader(string(body)))
	req = requestWithIntegrationID(req, integrationID)
	req.Header.Set("X-InPost-Signature", computeHMAC("legacy-secret", body))
	rr := httptest.NewRecorder()

	h.HandleWebhook(rr, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
	select {
	case <-updater.updateCh:
		require.Fail(t, "did not expect scoped shipment status update")
	case <-time.After(50 * time.Millisecond):
	}
}

// ===========================================================================
// ImportHandler tests
// ===========================================================================

func TestImportHandler_Preview_InvalidMultipartForm(t *testing.T) {
	h := NewImportHandler(nil, nil)

	// Send a non-multipart body
	req := httptest.NewRequest(http.MethodPost, "/v1/orders/import/preview", strings.NewReader("not multipart"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Preview(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "file too large or invalid multipart form", resp["error"])
}

func TestImportHandler_Import_InvalidMultipartForm(t *testing.T) {
	h := NewImportHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/orders/import", strings.NewReader("not multipart"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Import(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "file too large or invalid multipart form", resp["error"])
}

func TestImportHandler_Import_PassesPlanLimitToImportService(t *testing.T) {
	importService := &recordingCSVImportService{
		result: &model.ImportResult{TotalRows: 1, Imported: 1, Errors: []model.ImportError{}},
	}
	h := NewImportHandler(importService, nil)
	tenantID := uuid.New()
	userID := uuid.New()

	body, contentType := newOrderImportMultipartBody(t)
	req := httptest.NewRequest(http.MethodPost, "/v1/orders/import", body)
	req.Header.Set("Content-Type", contentType)
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))

	cache := service.NewPlanCache(time.Hour)
	cache.Set(tenantID, "active", json.RawMessage(`{"limits":{"max_orders_monthly":7}}`))

	rr := httptest.NewRecorder()
	middleware.TenantPlanGuard(cache, nil)(http.HandlerFunc(h.Import)).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.True(t, importService.importCalled)
	assert.Equal(t, 7, importService.options.MaxOrdersMonthly)
}

func TestImportHandler_Import_ReturnsForbiddenWhenMonthlyLimitExceeded(t *testing.T) {
	importService := &recordingCSVImportService{
		err: service.ErrOrderLimitExceeded,
	}
	h := NewImportHandler(importService, nil)
	tenantID := uuid.New()
	userID := uuid.New()

	body, contentType := newOrderImportMultipartBody(t)
	req := httptest.NewRequest(http.MethodPost, "/v1/orders/import", body)
	req.Header.Set("Content-Type", contentType)
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))

	cache := service.NewPlanCache(time.Hour)
	cache.Set(tenantID, "active", json.RawMessage(`{"limits":{"max_orders_monthly":7}}`))

	rr := httptest.NewRecorder()
	middleware.TenantPlanGuard(cache, nil)(http.HandlerFunc(h.Import)).ServeHTTP(rr, req)

	require.Equal(t, http.StatusForbidden, rr.Code)
	require.True(t, importService.importCalled)
	assert.Contains(t, rr.Body.String(), "Monthly order limit reached for current plan (max: 7)")
}

type recordingCSVImportService struct {
	result       *model.ImportResult
	err          error
	options      service.ImportOrdersOptions
	importCalled bool
}

func (s *recordingCSVImportService) ParseCSV(_ io.Reader) (*model.ImportPreviewResponse, error) {
	return nil, nil
}

func (s *recordingCSVImportService) ImportOrders(
	_ context.Context,
	_ uuid.UUID,
	_ io.Reader,
	_ []model.ImportColumnMapping,
	_ uuid.UUID,
	_ string,
	options service.ImportOrdersOptions,
) (*model.ImportResult, error) {
	s.importCalled = true
	s.options = options
	return s.result, s.err
}

func newOrderImportMultipartBody(t *testing.T) (*bytes.Buffer, string) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("file", "orders.csv")
	require.NoError(t, err)
	_, err = file.Write([]byte("customer_name,total_amount\nJan Kowalski,100\n"))
	require.NoError(t, err)
	require.NoError(t, writer.WriteField("mappings", `[{"csv_column":"customer_name","order_field":"customer_name"},{"csv_column":"total_amount","order_field":"total_amount"}]`))
	require.NoError(t, writer.Close())

	return &body, writer.FormDataContentType()
}

// ===========================================================================
// FeedHandler tests
// ===========================================================================

func TestFeedHandler_UpdateConfig_InvalidJSON(t *testing.T) {
	h := NewFeedHandler(nil)

	req := httptest.NewRequest(http.MethodPut, "/v1/feeds/config", strings.NewReader("bad json"))
	rr := httptest.NewRecorder()

	h.UpdateConfig(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestFeedHandler_RegenerateToken_InvalidJSON(t *testing.T) {
	h := NewFeedHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/feeds/regenerate-token", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.RegenerateToken(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "feed_type must be 'ceneo' or 'google'", resp["error"])
}

func TestFeedHandler_RegenerateToken_InvalidFeedType(t *testing.T) {
	h := NewFeedHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/feeds/regenerate-token", strings.NewReader(`{"feed_type":"invalid"}`))
	rr := httptest.NewRecorder()

	h.RegenerateToken(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "feed_type must be 'ceneo' or 'google'", resp["error"])
}

func TestFeedHandler_ServeCeneoFeed_InvalidTenantID(t *testing.T) {
	h := NewFeedHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("tenant_id", "bad")
	rctx.URLParams.Add("token", "some-token")

	req := httptest.NewRequest(http.MethodGet, "/v1/feeds/ceneo/bad/some-token", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ServeCeneoFeed(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid tenant_id", resp["error"])
}

func TestFeedHandler_ServeCeneoFeed_MissingToken(t *testing.T) {
	h := NewFeedHandler(nil)

	tenantID := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("tenant_id", tenantID.String())
	rctx.URLParams.Add("token", "")

	req := httptest.NewRequest(http.MethodGet, "/v1/feeds/ceneo/"+tenantID.String()+"/", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ServeCeneoFeed(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "token is required", resp["error"])
}

func TestFeedHandler_ServeGoogleFeed_InvalidTenantID(t *testing.T) {
	h := NewFeedHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("tenant_id", "not-uuid")
	rctx.URLParams.Add("token", "some-token")

	req := httptest.NewRequest(http.MethodGet, "/v1/feeds/google/not-uuid/some-token", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ServeGoogleFeed(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid tenant_id", resp["error"])
}

func TestFeedHandler_ServeGoogleFeed_MissingToken(t *testing.T) {
	h := NewFeedHandler(nil)

	tenantID := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("tenant_id", tenantID.String())
	rctx.URLParams.Add("token", "")

	req := httptest.NewRequest(http.MethodGet, "/v1/feeds/google/"+tenantID.String()+"/", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ServeGoogleFeed(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "token is required", resp["error"])
}

// ===========================================================================
// ForecastHandler tests
// ===========================================================================

func TestForecastHandler_GetForecast_InvalidProductID(t *testing.T) {
	h := NewForecastHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad-uuid")

	req := httptest.NewRequest(http.MethodGet, "/v1/forecast/products/bad-uuid", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetForecast(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid product ID", resp["error"])
}

func TestForecastHandler_GetSeasonality_InvalidProductID(t *testing.T) {
	h := NewForecastHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("product_id", "not-a-uuid")

	req := httptest.NewRequest(http.MethodGet, "/v1/forecast/seasonality/not-a-uuid", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetSeasonality(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid product ID", resp["error"])
}

func TestForecastHandler_UpdateConfig_InvalidJSON(t *testing.T) {
	h := NewForecastHandler(nil)

	req := httptest.NewRequest(http.MethodPut, "/v1/forecast/config", strings.NewReader("not json"))
	rr := httptest.NewRecorder()

	h.UpdateConfig(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid input data", resp["error"])
}

func TestForecastHandler_UpdateConfig_LeadTimeOutOfRange(t *testing.T) {
	h := NewForecastHandler(nil)

	body := `{"default_lead_time_days":0,"safety_stock_days":7,"forecast_days_ahead":30}`
	req := httptest.NewRequest(http.MethodPut, "/v1/forecast/config", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateConfig(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "lead time must be between 1 and 365 days", resp["error"])
}

func TestForecastHandler_UpdateConfig_LeadTimeTooHigh(t *testing.T) {
	h := NewForecastHandler(nil)

	body := `{"default_lead_time_days":500,"safety_stock_days":7,"forecast_days_ahead":30}`
	req := httptest.NewRequest(http.MethodPut, "/v1/forecast/config", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateConfig(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "lead time must be between 1 and 365 days", resp["error"])
}

func TestForecastHandler_UpdateConfig_SafetyStockNegative(t *testing.T) {
	h := NewForecastHandler(nil)

	body := `{"default_lead_time_days":14,"safety_stock_days":-1,"forecast_days_ahead":30}`
	req := httptest.NewRequest(http.MethodPut, "/v1/forecast/config", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateConfig(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "safety stock must be between 0 and 365 days", resp["error"])
}

func TestForecastHandler_UpdateConfig_SafetyStockTooHigh(t *testing.T) {
	h := NewForecastHandler(nil)

	body := `{"default_lead_time_days":14,"safety_stock_days":400,"forecast_days_ahead":30}`
	req := httptest.NewRequest(http.MethodPut, "/v1/forecast/config", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateConfig(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "safety stock must be between 0 and 365 days", resp["error"])
}

func TestForecastHandler_UpdateConfig_ForecastHorizonZero(t *testing.T) {
	h := NewForecastHandler(nil)

	body := `{"default_lead_time_days":14,"safety_stock_days":7,"forecast_days_ahead":0}`
	req := httptest.NewRequest(http.MethodPut, "/v1/forecast/config", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateConfig(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "forecast horizon must be between 1 and 365 days", resp["error"])
}

func TestForecastHandler_UpdateConfig_ForecastHorizonTooHigh(t *testing.T) {
	h := NewForecastHandler(nil)

	body := `{"default_lead_time_days":14,"safety_stock_days":7,"forecast_days_ahead":999}`
	req := httptest.NewRequest(http.MethodPut, "/v1/forecast/config", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateConfig(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "forecast horizon must be between 1 and 365 days", resp["error"])
}

// ===========================================================================
// TrackingHandler tests
// ===========================================================================

func TestTrackingHandler_MissingAllParams(t *testing.T) {
	h := NewTrackingHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("tenant_slug", "")
	rctx.URLParams.Add("order_id", "")

	req := httptest.NewRequest(http.MethodGet, "/v1/tracking//", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.TrackOrder(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "tenant_slug, order_id, and email are required", resp["error"])
}

func TestTrackingHandler_MissingEmail(t *testing.T) {
	h := NewTrackingHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("tenant_slug", "mercpart")
	rctx.URLParams.Add("order_id", "ORD-001")

	req := httptest.NewRequest(http.MethodGet, "/v1/tracking/mercpart/ORD-001", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.TrackOrder(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "tenant_slug, order_id, and email are required", resp["error"])
}

func TestTrackingHandler_MissingTenantSlug(t *testing.T) {
	h := NewTrackingHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("tenant_slug", "")
	rctx.URLParams.Add("order_id", "ORD-001")

	req := httptest.NewRequest(http.MethodGet, "/v1/tracking//ORD-001?email=test@example.com", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.TrackOrder(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "tenant_slug, order_id, and email are required", resp["error"])
}

func TestTrackingHandler_MissingOrderID(t *testing.T) {
	h := NewTrackingHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("tenant_slug", "mercpart")
	rctx.URLParams.Add("order_id", "")

	req := httptest.NewRequest(http.MethodGet, "/v1/tracking/mercpart/?email=test@example.com", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.TrackOrder(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "tenant_slug, order_id, and email are required", resp["error"])
}

// ===========================================================================
// InvitationHandler tests
// ===========================================================================

func TestInvitationHandler_Create_InvalidJSON(t *testing.T) {
	h := NewInvitationHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/invitations", strings.NewReader("bad json"))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestInvitationHandler_Delete_InvalidID(t *testing.T) {
	h := NewInvitationHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodDelete, "/v1/invitations/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid invitation ID", resp["error"])
}

// ===========================================================================
// WarehouseDocumentHandler tests
// ===========================================================================

func TestWarehouseDocumentHandler_Get_InvalidID(t *testing.T) {
	h := NewWarehouseDocumentHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/warehouse-documents/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid document ID", resp["error"])
}

func TestWarehouseDocumentHandler_Create_InvalidJSON(t *testing.T) {
	h := NewWarehouseDocumentHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/warehouse-documents", strings.NewReader("bad json"))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestWarehouseDocumentHandler_Update_InvalidID(t *testing.T) {
	h := NewWarehouseDocumentHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPatch, "/v1/warehouse-documents/bad", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid document ID", resp["error"])
}

func TestWarehouseDocumentHandler_Update_InvalidJSON(t *testing.T) {
	h := NewWarehouseDocumentHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPatch, "/v1/warehouse-documents/"+id.String(), strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestWarehouseDocumentHandler_Delete_InvalidID(t *testing.T) {
	h := NewWarehouseDocumentHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodDelete, "/v1/warehouse-documents/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid document ID", resp["error"])
}

func TestWarehouseDocumentHandler_Confirm_InvalidID(t *testing.T) {
	h := NewWarehouseDocumentHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/warehouse-documents/bad/confirm", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Confirm(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid document ID", resp["error"])
}

func TestWarehouseDocumentHandler_Cancel_InvalidID(t *testing.T) {
	h := NewWarehouseDocumentHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/warehouse-documents/bad/cancel", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Cancel(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid document ID", resp["error"])
}

// ===========================================================================
// WebhookDeliveryHandler tests
// ===========================================================================

// Note: WebhookDeliveryHandler only has a List method which requires a database
// connection (pgxpool) for the WithTenant call. There are no UUID-parsing
// validation paths. We verify that constructing with nil is possible.
func TestWebhookDeliveryHandler_Constructor(t *testing.T) {
	h := NewWebhookDeliveryHandler(nil, nil)
	assert.NotNil(t, h)
}

// ===========================================================================
// DropshipHandler tests
// ===========================================================================

func TestDropshipHandler_Get_InvalidID(t *testing.T) {
	h := NewDropshipHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/dropship/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid dropship order ID", resp["error"])
}

func TestDropshipHandler_Create_InvalidJSON(t *testing.T) {
	h := NewDropshipHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/dropship", strings.NewReader("bad json"))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestDropshipHandler_AutoRoute_InvalidOrderID(t *testing.T) {
	h := NewDropshipHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("order_id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/dropship/auto-route/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.AutoRoute(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid order ID", resp["error"])
}

func TestDropshipHandler_GetByOrderID_InvalidOrderID(t *testing.T) {
	h := NewDropshipHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("order_id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/dropship/order/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetByOrderID(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid order ID", resp["error"])
}

func TestDropshipHandler_UpdateStatus_InvalidID(t *testing.T) {
	h := NewDropshipHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPatch, "/v1/dropship/bad/status", strings.NewReader(`{"status":"shipped"}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.UpdateStatus(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid dropship order ID", resp["error"])
}

func TestDropshipHandler_UpdateStatus_InvalidJSON(t *testing.T) {
	h := NewDropshipHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPatch, "/v1/dropship/"+id.String()+"/status", strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.UpdateStatus(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestDropshipHandler_Cancel_InvalidID(t *testing.T) {
	h := NewDropshipHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/dropship/bad/cancel", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Cancel(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid dropship order ID", resp["error"])
}
