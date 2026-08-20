package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
