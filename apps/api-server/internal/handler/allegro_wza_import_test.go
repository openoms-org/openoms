package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func TestPlanWzAImport_CreatesAllegroRowAndLeavesInPostUntouched(t *testing.T) {
	inpostID := uuid.MustParse("5c91ec0e-0000-4000-8000-000000000001")
	inpostExt := "5c91ec0e-empty"
	oms := []model.Shipment{
		{
			ID:         inpostID,
			Provider:   "inpost",
			ExternalID: &inpostExt,
			Status:     "created",
		},
	}
	found := []allegrosdk.ExistingWzAShipment{{
		ShipmentID: "cb92efe4-1b2f-4cac-9e35-da69b0000001",
		Waybill:    "605500867604760112200733",
		Carrier:    "ALLEGRO",
		CarrierID:  "INPOST",
	}}

	plan := planWzAImport(oms, found)
	require.Len(t, plan.Creates, 1)
	assert.Equal(t, "605500867604760112200733", plan.Creates[0].Waybill)
	assert.Equal(t, "cb92efe4-1b2f-4cac-9e35-da69b0000001", plan.Creates[0].ShipmentID)
	assert.Empty(t, plan.Already)
	assert.NotEqual(t, "inpost", wzaImportProvider(plan.Creates[0]))
	assert.Equal(t, "allegro", wzaImportProvider(plan.Creates[0]))
}

func TestPlanWzAImport_IdempotentOnAlreadyImportedWaybill(t *testing.T) {
	waybill := "605500867604760112200733"
	wzaID := "cb92efe4-1b2f-4cac-9e35-da69b0000001"
	oms := []model.Shipment{
		{
			ID:             uuid.New(),
			Provider:       "allegro",
			ExternalID:     &wzaID,
			TrackingNumber: &waybill,
		},
		{
			ID:       uuid.New(),
			Provider: "inpost",
		},
	}
	found := []allegrosdk.ExistingWzAShipment{{
		ShipmentID: wzaID,
		Waybill:    waybill,
		Carrier:    "ALLEGRO",
	}}

	plan := planWzAImport(oms, found)
	assert.Empty(t, plan.Creates)
	require.Len(t, plan.Already, 1)
	assert.Equal(t, "allegro", plan.Already[0].Provider)
}

func TestPlanWzAImport_DoesNotAttachWzAWaybillToInPost(t *testing.T) {
	waybill := "605500867604760112200733"
	oms := []model.Shipment{{
		ID:             uuid.New(),
		Provider:       "inpost",
		TrackingNumber: &waybill,
	}}
	found := []allegrosdk.ExistingWzAShipment{{
		Waybill: waybill,
		Carrier: "ALLEGRO",
	}}

	plan := planWzAImport(oms, found)
	require.Len(t, plan.Creates, 1)
	assert.Equal(t, "allegro", wzaImportProvider(plan.Creates[0]))
	assert.Empty(t, plan.Already)
}

func TestWzAImportLabelFilename(t *testing.T) {
	assert.Equal(t, "wza-cb92efe4-1b2f-4cac-9e35-da69b0000001.pdf", wzaImportLabelFilename(allegrosdk.ExistingWzAShipment{
		ShipmentID: "cb92efe4-1b2f-4cac-9e35-da69b0000001",
		Waybill:    "605500867604760112200733",
	}))
	assert.Equal(t, "wza-605500867604760112200733.pdf", wzaImportLabelFilename(allegrosdk.ExistingWzAShipment{
		Waybill: "605500867604760112200733",
	}))
}

func TestAllegroShipmentHandler_ImportExistingShipments_MissingOrderID(t *testing.T) {
	h := NewAllegroShipmentHandler(nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/orders//wza-shipments/import", nil)
	rr := httptest.NewRecorder()
	h.ImportExistingShipments(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Missing order ID", resp["error"])
}

func TestAllegroShipmentHandler_ImportExistingShipments_EmptyOrderIDInRoute(t *testing.T) {
	h := NewAllegroShipmentHandler(nil, nil, nil, nil, nil, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderId", "")
	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/orders//wza-shipments/import", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h.ImportExistingShipments(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestWzAImportAuditIP_Live703LabelIsNotInet(t *testing.T) {
	// Live #703 persist: shipmentService.Create(..., "allegro-wza-import").
	// audit_log.ip_address is inet. Postgres rejects that label, the WithTenant
	// txn rolls back, and the dialog only shows "Failed to store imported shipment".
	got := wzaImportAuditIP("allegro-wza-import")
	if got == "allegro-wza-import" {
		t.Fatal(`audit_log.ip_address is inet; "allegro-wza-import" rolls back the shipment insert`)
	}
	if net.ParseIP(got) == nil {
		t.Fatalf("audit IP %q is still not inet", got)
	}
}

func TestWzAImportAuditIP_KeepsRealClientAddress(t *testing.T) {
	assert.Equal(t, "203.0.113.10", wzaImportAuditIP("203.0.113.10"))
	assert.Equal(t, "0.0.0.0", wzaImportAuditIP(""))
}

func TestWzAImportStoreError_IncludesAuditInetCause(t *testing.T) {
	err := errors.New(`audit log: ERROR: invalid input syntax for type inet: "allegro-wza-import" (SQLSTATE 22P02)`)
	got := wzaImportStoreError(err)
	assert.Contains(t, got, "Failed to store imported shipment")
	assert.Contains(t, got, `invalid input syntax for type inet: "allegro-wza-import"`)
	assert.NotEqual(t, "Failed to store imported shipment", got)
}

func TestWzAImportCreateRequest_PersistsAllegroWaybill(t *testing.T) {
	orderID := uuid.MustParse("cb6d3a51-295e-4d2e-bbdc-0ede50796501")
	integID := uuid.MustParse("aaaaaaaa-0000-4000-8000-000000000001")
	req := wzaImportCreateRequest(orderID, &integID, allegrosdk.ExistingWzAShipment{
		ShipmentID:       "cb92efe4-1b2f-4cac-9e35-da69b82b9482",
		Waybill:          "605500867604760112200733",
		Carrier:          "ALLEGRO",
		CarrierID:        "INPOST",
		DeliveryMethodID: "2488f7b7-5d1c-4d65-b85c-4cbcf253fd93",
	})

	require.NoError(t, req.Validate())
	assert.Equal(t, orderID, req.OrderID)
	assert.Equal(t, "allegro", req.Provider)
	require.NotNil(t, req.TrackingNumber)
	assert.Equal(t, "605500867604760112200733", *req.TrackingNumber)
	require.NotNil(t, req.ExternalID)
	assert.Equal(t, "cb92efe4-1b2f-4cac-9e35-da69b82b9482", *req.ExternalID)
	require.Equal(t, integID, *req.IntegrationID)

	var data map[string]any
	require.NoError(t, json.Unmarshal(req.CarrierData, &data))
	assert.Equal(t, "allegro_wza", data["billing"])
	assert.Equal(t, "cb92efe4-1b2f-4cac-9e35-da69b82b9482", data["allegro_shipment_id"])
	assert.Equal(t, true, data["imported"])
}

func TestWzAImportCreateRequest_DoesNotInventSalesCenterWaybill(t *testing.T) {
	req := wzaImportCreateRequest(uuid.New(), nil, allegrosdk.ExistingWzAShipment{
		ShipmentID: "cb92efe4-1b2f-4cac-9e35-da69b82b9482",
	})
	if req.TrackingNumber != nil && *req.TrackingNumber == "605500867604760112200733" {
		t.Fatal("must not invent Sales Center waybill 605500867604760112200733")
	}
	if req.TrackingNumber != nil && strings.TrimSpace(*req.TrackingNumber) != "" {
		t.Fatalf("empty find must not invent a tracking number, got %q", *req.TrackingNumber)
	}
}
