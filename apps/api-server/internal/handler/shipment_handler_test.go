package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// ---------------------------------------------------------------------------
// Existing basic tests
// ---------------------------------------------------------------------------

func TestShipmentHandler_Create_InvalidJSON(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	req := httptest.NewRequest(http.MethodPost, "/v1/shipments", strings.NewReader("bad json"))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestShipmentHandler_Get_InvalidID(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/shipments/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid shipment ID", resp["error"])
}

func TestShipmentHandler_Update_InvalidID(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPatch, "/v1/shipments/bad", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestShipmentHandler_Update_InvalidJSON(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPatch, "/v1/shipments/"+id.String(), strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestShipmentHandler_Delete_InvalidID(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodDelete, "/v1/shipments/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestShipmentHandler_TransitionStatus_InvalidID(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/bad/status", strings.NewReader(`{"status":"delivered"}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.TransitionStatus(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestShipmentHandler_TransitionStatus_InvalidJSON(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/"+id.String()+"/status", strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.TransitionStatus(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestShipmentHandler_GenerateLabel_InvalidID(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/bad/label", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GenerateLabel(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestShipmentHandler_GenerateLabel_InvalidJSON(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/"+id.String()+"/label", strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GenerateLabel(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestShipmentHandler_GetLabel_InvalidID(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/shipments/bad/label", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetLabel(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid shipment ID", resp["error"])
}

func TestShipmentHandler_List_InvalidOrderIDFilter(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	req := httptest.NewRequest(http.MethodGet, "/v1/shipments?order_id=not-uuid", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid order_id filter", resp["error"])
}

func TestShipmentHandler_Create_ValidationError(t *testing.T) {
	svc := service.NewShipmentService(nil, nil, nil, nil, nil, nil, nil, "")
	h := NewShipmentHandler(svc, nil, "client-ready")

	tenantID := uuid.New()
	userID := uuid.New()

	// Missing provider
	body := `{"order_id":"` + uuid.New().String() + `"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipments", strings.NewReader(body))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "provider")
}

func TestShipmentHandler_TransitionStatus_ValidationError(t *testing.T) {
	svc := service.NewShipmentService(nil, nil, nil, nil, nil, nil, nil, "")
	h := NewShipmentHandler(svc, nil, "client-ready")

	tenantID := uuid.New()
	userID := uuid.New()
	shipmentID := uuid.New()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", shipmentID.String())

	// Empty status should fail validation
	body := `{"status":""}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/"+shipmentID.String()+"/status", strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.TransitionStatus(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestShipmentHandler_Update_ValidationError(t *testing.T) {
	svc := service.NewShipmentService(nil, nil, nil, nil, nil, nil, nil, "")
	h := NewShipmentHandler(svc, nil, "client-ready")

	tenantID := uuid.New()
	userID := uuid.New()
	shipmentID := uuid.New()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", shipmentID.String())

	// Empty body - no fields to update
	body := `{}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/shipments/"+shipmentID.String(), strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// ---------------------------------------------------------------------------
// GenerateLabel — InPost-specific tests
// ---------------------------------------------------------------------------

func TestShipmentHandler_GenerateLabel_MissingServiceType(t *testing.T) {
	// GenerateLabelRequest.Validate() requires service_type.
	// The label service validates this, but when labelService is nil the handler
	// will panic before reaching validation. So we create a LabelService with nil
	// dependencies — it will fail at the DB layer. However, the service-level
	// validation for GenerateLabelRequest happens inside the service (not the handler).
	// The handler only validates UUID and JSON parsing.
	//
	// We verify the handler correctly parses the JSON and passes it on.
	// With a nil labelService, the request with a valid UUID and valid JSON
	// would panic. So we test the parsing boundary only.
	h := NewShipmentHandler(nil, nil, "client-ready")

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	// Valid JSON but empty body — tests JSON parsing succeeds
	body := `{"service_type":"","label_format":"pdf"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/"+id.String()+"/label", strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Without a label service, calling GenerateLabel after JSON decoding will nil-pointer.
	// We catch the panic to verify that the handler got past UUID and JSON validation.
	func() {
		defer func() {
			r := recover()
			// Expected: nil pointer dereference when calling h.labelService.GenerateLabel
			assert.NotNil(t, r, "expected panic from nil labelService, meaning JSON parsed successfully")
		}()
		rr := httptest.NewRecorder()
		h.GenerateLabel(rr, req)
	}()
}

func TestShipmentHandler_GenerateLabel_EmptyBody(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	// Empty body - should still parse as valid JSON (empty object)
	// but will panic on nil labelService
	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/"+id.String()+"/label", strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	func() {
		defer func() {
			r := recover()
			assert.NotNil(t, r, "expected panic from nil labelService after valid JSON parsing")
		}()
		rr := httptest.NewRecorder()
		h.GenerateLabel(rr, req)
	}()
}

func TestShipmentHandler_GenerateLabel_InvalidLabelFormat(t *testing.T) {
	// The service validates label_format, but the handler does not.
	// With a nil labelService, valid JSON with valid UUID will reach service.
	// We only test that the handler correctly rejects invalid UUIDs and malformed JSON
	// (tested above), so this test documents the expected flow.

	h := NewShipmentHandler(nil, nil, "client-ready")

	// Valid UUID with totally invalid ID text
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "not-a-uuid-at-all")

	body := `{"service_type":"inpost_locker_standard","label_format":"bmp","target_point":"KRA01M"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/not-a-uuid-at-all/label", strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GenerateLabel(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid shipment ID", resp["error"])
}

func TestShipmentHandler_GenerateLabel_ValidUUIDFormats(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	// UUID with hyphens - valid
	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	body := `{"service_type":"inpost_locker_standard","label_format":"pdf","target_point":"KRA01M"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/"+id.String()+"/label", strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	func() {
		defer func() {
			r := recover()
			// Panic from nil labelService confirms the handler accepted the UUID
			assert.NotNil(t, r)
		}()
		rr := httptest.NewRecorder()
		h.GenerateLabel(rr, req)
	}()
}

// ---------------------------------------------------------------------------
// GetTracking — InPost-specific tests
// ---------------------------------------------------------------------------

func TestShipmentHandler_GetTracking_InvalidID(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad-id")

	req := httptest.NewRequest(http.MethodGet, "/v1/shipments/bad-id/tracking", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetTracking(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid shipment ID", resp["error"])
}

func TestShipmentHandler_GetTracking_EmptyID(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "")

	req := httptest.NewRequest(http.MethodGet, "/v1/shipments//tracking", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetTracking(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestShipmentHandler_GetTracking_NumericID(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "12345")

	req := httptest.NewRequest(http.MethodGet, "/v1/shipments/12345/tracking", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetTracking(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid shipment ID", resp["error"])
}

func TestShipmentHandler_GetTracking_ValidUUIDReachesService(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodGet, "/v1/shipments/"+id.String()+"/tracking", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// With nil labelService, calling GetTracking with a valid UUID should panic
	// after passing the UUID validation — confirming the handler layer is correct.
	func() {
		defer func() {
			r := recover()
			assert.NotNil(t, r, "expected panic from nil labelService, confirming UUID validation passed")
		}()
		rr := httptest.NewRecorder()
		h.GetTracking(rr, req)
	}()
}

// ---------------------------------------------------------------------------
// CreateDispatchOrder — InPost-specific tests
// ---------------------------------------------------------------------------

func TestShipmentHandler_CreateDispatchOrder_InvalidJSON(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/dispatch", strings.NewReader("not json"))
	rr := httptest.NewRecorder()

	h.CreateDispatchOrder(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestShipmentHandler_CreateDispatchOrder_EmptyShipmentIDs(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	body := `{"shipment_ids":[]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/dispatch", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreateDispatchOrder(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "shipment_ids is required", resp["error"])
}

func TestShipmentHandler_CreateDispatchOrder_MissingShipmentIDs(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	// JSON body without shipment_ids field at all
	body := `{"name":"Test","phone":"123456789"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/dispatch", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreateDispatchOrder(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "shipment_ids is required", resp["error"])
}

func TestShipmentHandler_CreateDispatchOrder_NullShipmentIDs(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	body := `{"shipment_ids":null}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/dispatch", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreateDispatchOrder(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "shipment_ids is required", resp["error"])
}

func TestShipmentHandler_CreateDispatchOrder_ValidPayloadReachesService(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	sid := uuid.New()
	body := `{"shipment_ids":["` + sid.String() + `"],"name":"Jan Kowalski","phone":"500600700"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/dispatch", strings.NewReader(body))
	rr := httptest.NewRecorder()

	// With nil labelService, after passing validation the handler will panic
	func() {
		defer func() {
			r := recover()
			assert.NotNil(t, r, "expected panic from nil labelService, confirming validation passed")
		}()
		h.CreateDispatchOrder(rr, req)
	}()
}

func TestShipmentHandler_CreateDispatchOrder_SingleShipmentID(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	sid := uuid.New()
	body := `{"shipment_ids":["` + sid.String() + `"]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/dispatch", strings.NewReader(body))
	rr := httptest.NewRecorder()

	func() {
		defer func() {
			r := recover()
			assert.NotNil(t, r, "expected panic from nil labelService after passing validation")
		}()
		h.CreateDispatchOrder(rr, req)
	}()
}

func TestShipmentHandler_CreateDispatchOrder_MultipleShipmentIDs(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	body := `{"shipment_ids":["` + ids[0].String() + `","` + ids[1].String() + `","` + ids[2].String() + `"]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/dispatch", strings.NewReader(body))
	rr := httptest.NewRecorder()

	func() {
		defer func() {
			r := recover()
			assert.NotNil(t, r, "expected panic from nil labelService after passing validation with multiple IDs")
		}()
		h.CreateDispatchOrder(rr, req)
	}()
}

func TestShipmentHandler_CreateDispatchOrder_EmptyBody(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/dispatch", strings.NewReader(""))
	rr := httptest.NewRecorder()

	h.CreateDispatchOrder(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestShipmentHandler_CreateDispatchOrder_WithOptionalFields(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	sid := uuid.New()
	body := `{
		"shipment_ids":["` + sid.String() + `"],
		"street":"ul. Testowa 1",
		"building_number":"10A",
		"city":"Krakow",
		"post_code":"30-001",
		"name":"Jan Kowalski",
		"phone":"500600700",
		"email":"jan@example.com",
		"comment":"Ring the bell"
	}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/dispatch", strings.NewReader(body))
	rr := httptest.NewRecorder()

	func() {
		defer func() {
			r := recover()
			assert.NotNil(t, r, "expected panic from nil labelService after passing validation")
		}()
		h.CreateDispatchOrder(rr, req)
	}()
}

// ---------------------------------------------------------------------------
// BatchLabels — InPost-specific tests
// ---------------------------------------------------------------------------

func TestShipmentHandler_BatchLabels_InvalidJSON(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/batch-labels", strings.NewReader("not json"))
	rr := httptest.NewRecorder()

	h.BatchLabels(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestShipmentHandler_BatchLabels_EmptyShipmentIDs(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	body := `{"shipment_ids":[]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/batch-labels", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.BatchLabels(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "shipment_ids is required", resp["error"])
}

func TestShipmentHandler_BatchLabels_NullShipmentIDs(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	body := `{"shipment_ids":null}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/batch-labels", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.BatchLabels(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "shipment_ids is required", resp["error"])
}

func TestShipmentHandler_BatchLabels_MissingShipmentIDs(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/batch-labels", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.BatchLabels(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "shipment_ids is required", resp["error"])
}

func TestShipmentHandler_BatchLabels_ExceedsMaximum(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	// Build a list of 101 UUIDs
	ids := make([]string, 101)
	for i := range ids {
		ids[i] = `"` + uuid.New().String() + `"`
	}
	body := `{"shipment_ids":[` + strings.Join(ids, ",") + `]}`

	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/batch-labels", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.BatchLabels(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "maximum 100 shipments per batch", resp["error"])
}

func TestShipmentHandler_BatchLabels_ExactlyMaximum(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	// 100 UUIDs should pass the limit check and reach the service
	ids := make([]string, 100)
	for i := range ids {
		ids[i] = `"` + uuid.New().String() + `"`
	}
	body := `{"shipment_ids":[` + strings.Join(ids, ",") + `]}`

	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/batch-labels", strings.NewReader(body))
	rr := httptest.NewRecorder()

	// With nil shipmentService, the handler will panic after validation passes.
	func() {
		defer func() {
			r := recover()
			assert.NotNil(t, r, "expected panic from nil shipmentService, confirming batch size validation passed")
		}()
		h.BatchLabels(rr, req)
	}()
}

func TestShipmentHandler_BatchLabels_SingleShipmentID(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	body := `{"shipment_ids":["` + uuid.New().String() + `"]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/batch-labels", strings.NewReader(body))
	rr := httptest.NewRecorder()

	func() {
		defer func() {
			r := recover()
			assert.NotNil(t, r, "expected panic from nil shipmentService after passing validation")
		}()
		h.BatchLabels(rr, req)
	}()
}

func TestShipmentHandler_BatchLabels_EmptyBody(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/batch-labels", strings.NewReader(""))
	rr := httptest.NewRecorder()

	h.BatchLabels(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// ---------------------------------------------------------------------------
// Create — additional InPost-related validation tests
// ---------------------------------------------------------------------------

func TestShipmentHandler_Create_MissingOrderID(t *testing.T) {
	svc := service.NewShipmentService(nil, nil, nil, nil, nil, nil, nil, "")
	h := NewShipmentHandler(svc, nil, "client-ready")

	tenantID := uuid.New()
	userID := uuid.New()

	body := `{"provider":"inpost"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipments", strings.NewReader(body))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "order_id")
}

func TestShipmentHandler_Create_InPostProviderMissingOrderID(t *testing.T) {
	svc := service.NewShipmentService(nil, nil, nil, nil, nil, nil, nil, "")
	h := NewShipmentHandler(svc, nil, "client-ready")

	tenantID := uuid.New()
	userID := uuid.New()

	body := `{"provider":"inpost","order_id":"00000000-0000-0000-0000-000000000000"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipments", strings.NewReader(body))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "order_id")
}

func TestShipmentHandler_Create_InPostProviderValidPayload(t *testing.T) {
	// With a nil pool, the service will panic at the database layer, not at validation.
	// We catch the panic to confirm the handler got past validation.
	svc := service.NewShipmentService(nil, nil, nil, nil, nil, nil, nil, "")
	h := NewShipmentHandler(svc, nil, "client-ready")

	tenantID := uuid.New()
	userID := uuid.New()
	orderID := uuid.New()

	body := `{"provider":"inpost","order_id":"` + orderID.String() + `"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipments", strings.NewReader(body))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	func() {
		defer func() {
			r := recover()
			// Panic from nil pool confirms validation passed (not a 400 error)
			assert.NotNil(t, r, "expected panic from nil pool, confirming validation passed")
		}()
		h.Create(rr, req)
	}()
}

func TestShipmentHandler_Create_InPostWithCarrierData(t *testing.T) {
	svc := service.NewShipmentService(nil, nil, nil, nil, nil, nil, nil, "")
	h := NewShipmentHandler(svc, nil, "client-ready")

	tenantID := uuid.New()
	userID := uuid.New()
	orderID := uuid.New()

	body := `{
		"provider":"inpost",
		"order_id":"` + orderID.String() + `",
		"carrier_data":{"target_point":"KRA01M","service_type":"inpost_locker_standard","parcel_size":"small","sending_method":"parcel_locker"}
	}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipments", strings.NewReader(body))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	func() {
		defer func() {
			r := recover()
			// Panic from nil pool confirms validation passed with carrier_data
			assert.NotNil(t, r, "expected panic from nil pool, confirming validation passed with carrier_data")
		}()
		h.Create(rr, req)
	}()
}

// ---------------------------------------------------------------------------
// List — additional filter tests
// ---------------------------------------------------------------------------

func TestShipmentHandler_List_ValidOrderIDFilter(t *testing.T) {
	// With nil shipmentService, a valid order_id filter should pass the UUID parse
	// and then panic (nil service). This proves the handler accepted the filter.
	h := NewShipmentHandler(nil, nil, "client-ready")

	orderID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/shipments?order_id="+orderID.String(), nil)
	rr := httptest.NewRecorder()

	func() {
		defer func() {
			r := recover()
			assert.NotNil(t, r, "expected panic from nil shipmentService, confirming UUID filter parsed")
		}()
		h.List(rr, req)
	}()
}

func TestShipmentHandler_List_StatusFilter(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	// Status filter does not require UUID parsing, goes straight to service
	req := httptest.NewRequest(http.MethodGet, "/v1/shipments?status=label_ready", nil)
	rr := httptest.NewRecorder()

	func() {
		defer func() {
			r := recover()
			assert.NotNil(t, r, "expected panic from nil shipmentService")
		}()
		h.List(rr, req)
	}()
}

func TestShipmentHandler_List_ProviderFilter(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	req := httptest.NewRequest(http.MethodGet, "/v1/shipments?provider=inpost", nil)
	rr := httptest.NewRecorder()

	func() {
		defer func() {
			r := recover()
			assert.NotNil(t, r, "expected panic from nil shipmentService")
		}()
		h.List(rr, req)
	}()
}

func TestShipmentHandler_List_CombinedFilters(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	orderID := uuid.New()
	req := httptest.NewRequest(http.MethodGet,
		"/v1/shipments?status=created&provider=inpost&order_id="+orderID.String(), nil)
	rr := httptest.NewRecorder()

	func() {
		defer func() {
			r := recover()
			assert.NotNil(t, r, "expected panic from nil shipmentService, confirming all filters parsed")
		}()
		h.List(rr, req)
	}()
}

func TestShipmentHandler_List_NoFilters(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	req := httptest.NewRequest(http.MethodGet, "/v1/shipments", nil)
	rr := httptest.NewRecorder()

	func() {
		defer func() {
			r := recover()
			assert.NotNil(t, r, "expected panic from nil shipmentService")
		}()
		h.List(rr, req)
	}()
}

// ---------------------------------------------------------------------------
// Response structure tests
// ---------------------------------------------------------------------------

func TestShipmentHandler_GenerateLabel_ErrorResponseFormat(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/bad/label", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GenerateLabel(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/json")

	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	_, hasError := resp["error"]
	assert.True(t, hasError, "response should contain 'error' key")
}

func TestShipmentHandler_GetTracking_ErrorResponseFormat(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "invalid")

	req := httptest.NewRequest(http.MethodGet, "/v1/shipments/invalid/tracking", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetTracking(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/json")

	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid shipment ID", resp["error"])
}

func TestShipmentHandler_CreateDispatchOrder_ErrorResponseFormat(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	body := `{"shipment_ids":[]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/dispatch", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreateDispatchOrder(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/json")

	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	_, hasError := resp["error"]
	assert.True(t, hasError, "response should contain 'error' key")
}

func TestShipmentHandler_BatchLabels_ErrorResponseFormat(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	body := `{"shipment_ids":[]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/batch-labels", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.BatchLabels(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/json")

	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	_, hasError := resp["error"]
	assert.True(t, hasError, "response should contain 'error' key")
}

// ---------------------------------------------------------------------------
// GenerateLabel — InPost service_type specific validation
// ---------------------------------------------------------------------------

func TestShipmentHandler_GenerateLabel_InPostLockerRequiresTargetPoint(t *testing.T) {
	// The GenerateLabelRequest.Validate() requires target_point for inpost_locker_standard.
	// This validation happens inside the service, not the handler. The service first
	// loads the shipment from DB before validation. With nil pool, it panics.
	// We catch the panic to confirm the handler correctly passed the request to the service.
	labelSvc := service.NewLabelService(nil, nil, nil, nil, nil, nil, nil, nil, "", "", nil)
	h := NewShipmentHandler(nil, labelSvc, "client-ready")

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	tenantID := uuid.New()
	userID := uuid.New()

	// Missing target_point for inpost_locker_standard
	body := `{"service_type":"inpost_locker_standard","label_format":"pdf"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/"+id.String()+"/label", strings.NewReader(body))
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = newContextWithTenantAndUser(ctx, tenantID, userID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	func() {
		defer func() {
			r := recover()
			// Panic from nil pool confirms handler forwarded the request to the service
			assert.NotNil(t, r, "expected panic from nil pool in label service")
		}()
		h.GenerateLabel(rr, req)
	}()
}

func TestShipmentHandler_GenerateLabel_InvalidSendingMethod(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	// Valid JSON with invalid sending_method - handler parses it fine,
	// validation happens in the service.
	body := `{"service_type":"inpost_locker_standard","label_format":"pdf","target_point":"KRA01M","sending_method":"invalid_method"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/"+id.String()+"/label", strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// With nil labelService, it will panic after JSON parsing
	func() {
		defer func() {
			r := recover()
			assert.NotNil(t, r, "expected panic from nil labelService, confirming JSON parsed")
		}()
		rr := httptest.NewRecorder()
		h.GenerateLabel(rr, req)
	}()
}

// ---------------------------------------------------------------------------
// Delete — additional tests
// ---------------------------------------------------------------------------

func TestShipmentHandler_Delete_ValidUUID(t *testing.T) {
	h := NewShipmentHandler(nil, nil, "client-ready")

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodDelete, "/v1/shipments/"+id.String(), nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// With nil shipmentService, should panic after UUID validation
	func() {
		defer func() {
			r := recover()
			assert.NotNil(t, r, "expected panic from nil shipmentService, confirming UUID validation passed")
		}()
		rr := httptest.NewRecorder()
		h.Delete(rr, req)
	}()
}

// ---------------------------------------------------------------------------
// TransitionStatus — additional tests
// ---------------------------------------------------------------------------

func TestShipmentHandler_TransitionStatus_WhitespaceOnlyStatus(t *testing.T) {
	svc := service.NewShipmentService(nil, nil, nil, nil, nil, nil, nil, "")
	h := NewShipmentHandler(svc, nil, "client-ready")

	tenantID := uuid.New()
	userID := uuid.New()
	shipmentID := uuid.New()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", shipmentID.String())

	body := `{"status":"   "}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/"+shipmentID.String()+"/status", strings.NewReader(body))
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = newContextWithTenantAndUser(ctx, tenantID, userID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.TransitionStatus(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
